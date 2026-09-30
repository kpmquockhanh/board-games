package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"ping/models"
	"ping/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Handler) HandleWS(c *gin.Context) {
	room := c.Query("room")
	player := c.Query("player")
	// The browser tab behind this socket, so the one its previous page left
	// open can be recognised as the same player and dropped rather than
	// treated as a second one.
	session := c.Query("session")
	if room == "" {
		room = "default"
	}

	// A socket for a player receives that player's view and private prompts,
	// so it has to carry their seat's token. A left seat's token still counts:
	// a lobby drop takes the player out of the room, and their page reconnects
	// the socket before it rejoins. Once someone else takes the name, the old
	// token names nobody.
	relay := false
	if player != "" {
		r, err := h.store.GetRoom(room)
		if err != nil || r == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}
		relay = r.Game == "hotpot"
		holder, _, err := h.store.SeatHolder(r.ID, hashSecret(presentedSeatToken(c)))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if holder != player {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "not your seat"})
			return
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &ws.Client{
		Conn:    conn,
		Room:    room,
		Player:  player,
		Session: session,
		// Claimed before registering, so this socket's own teardown carries an
		// epoch no later than the seat's and a reload that arrives afterwards
		// still supersedes it.
		Epoch: h.arrive(room, player),
		Send:  make(chan []byte, 256),
		Hub:   h.hub,
	}

	h.hub.Register(client)

	// Opening a socket is activity in itself: a table sitting in the lobby
	// between games sends no actions, and the reaper must not mistake that
	// for an empty room.
	if r, err := h.store.GetRoom(room); err == nil && r != nil {
		h.touch(r.ID)
		// The socket is the real signal that someone is back, and it arrives
		// whether or not they went through the join endpoint first. Either one
		// may be the first to find the seat stamped, so both announce it.
		if player != "" {
			h.markBack(r.ID, room, player)
		}
	}

	if player != "" {
		h.sendCurrentStateToClient(client, room, player)
	}

	go clientWritePump(client)
	go clientReadPump(client, relay)
}

func (h *Handler) sendCurrentStateToClient(client *ws.Client, roomKey, playerName string) {
	room, err := h.store.GetRoom(roomKey)
	if err != nil || room == nil {
		return
	}

	var payload []byte

	if room.Game == "ek" {
		// Always a view: the stored snapshot holds every hand and the deck.
		// The room lock is held while reading, since actions mutate in place.
		mu := h.getEKMutex(roomKey)
		mu.Lock()
		gs := h.loadEKState(roomKey, room.ID)
		if gs != nil {
			payload, _ = json.Marshal(gs.ViewFor(playerName))
		}
		mu.Unlock()
		if gs == nil {
			return
		}
	} else {
		stateJSON, ok, _ := h.store.GetLatestSnapshot(room.ID)
		if !ok || stateJSON == "" {
			return
		}
		switch room.Game {
		case "hotpot":
			var raw models.HotpotState
			if json.Unmarshal([]byte(stateJSON), &raw) == nil {
				payload, _ = json.Marshal(raw)
			}
		default:
			payload = []byte(stateJSON)
		}
	}

	if payload == nil {
		return
	}

	data, _ := json.Marshal(models.WSMessage{
		Type:    "state_updated",
		Room:    roomKey,
		Player:  playerName,
		Payload: payload,
	})
	h.hub.Send(client, data)
}

// relayedTypes are the messages a client may send to the rest of its room:
// Hotpot's live feed, which is also saved through the state endpoint. Nothing
// else is passed on. Everything a client used to be able to send went to the
// whole room as is, so any page could fake a state_updated, a prompt, or a
// room_deleted for everyone else.
var relayedTypes = map[string]bool{
	"join":  true,
	"leave": true,
	"drop":  true,
	"cheer": true,
	"chat":  true,
}

// relayable returns the message as it may go to the room, or false if it may
// not. The name in the payload is what the feed shows, so it is set to the
// socket's own player, whose seat token was checked when it opened.
func relayable(msg models.WSMessage, player string) (models.WSMessage, bool) {
	if player == "" || !relayedTypes[msg.Type] {
		return msg, false
	}
	payload := map[string]any{}
	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return msg, false
		}
	}
	payload["name"] = player
	raw, err := json.Marshal(payload)
	if err != nil {
		return msg, false
	}
	msg.Payload = raw
	return msg, true
}

// clientReadPump reads until the socket goes away. Only sockets in rooms that
// use the relay (relay) pass anything on; the rest are read to notice pongs
// and the close.
func clientReadPump(c *ws.Client, relay bool) {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(65536)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}

		var msg models.WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[ws] unmarshal error: %v", err)
			continue
		}

		if !relay {
			continue
		}
		msg.Room = c.Room
		msg.Player = c.Player
		out, ok := relayable(msg, c.Player)
		if !ok {
			continue
		}
		c.Hub.BroadcastToRoom(c.Room, out)
	}
}

func clientWritePump(c *ws.Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
