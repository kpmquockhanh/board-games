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
	go clientReadPump(client)
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

func clientReadPump(c *ws.Client) {
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

		msg.Room = c.Room
		msg.Player = c.Player

		c.Hub.BroadcastToRoom(c.Room, msg)
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
