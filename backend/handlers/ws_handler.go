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
	if room == "" {
		room = "default"
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &ws.Client{
		Conn:   conn,
		Room:   room,
		Player: player,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
	}

	h.hub.Register(client)

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

	players, _ := h.store.GetRoomPlayers(room.ID)
	playerNames := make([]string, 0, len(players))
	playerColors := make(map[string]string)
	for _, p := range players {
		playerNames = append(playerNames, p.PlayerName)
		playerColors[p.PlayerName] = p.Color
	}

	if room.Game == "ek" {
		gs := h.getEKState(roomKey)
		if gs != nil {
			payload, _ := json.Marshal(gs)
			msg := models.WSMessage{
				Type:    "state_updated",
				Room:    roomKey,
				Payload: payload,
			}
			data, _ := json.Marshal(msg)
			select {
			case client.Send <- data:
			default:
			}
			return
		}
	}

	stateJSON, ok, _ := h.store.GetLatestSnapshot(room.ID)
	if ok && stateJSON != "" {
		var payload []byte
		switch room.Game {
		case "ek":
			var raw models.EKRoomState
			if json.Unmarshal([]byte(stateJSON), &raw) == nil && raw.GameState != nil {
				payload, _ = json.Marshal(raw.GameState)
			}
		case "hotpot":
			var raw models.HotpotState
			if json.Unmarshal([]byte(stateJSON), &raw) == nil {
				payload, _ = json.Marshal(raw)
			}
		default:
			payload = []byte(stateJSON)
		}
		if payload != nil {
			msg := models.WSMessage{
				Type:    "state_updated",
				Room:    roomKey,
				Payload: payload,
			}
			data, _ := json.Marshal(msg)
			select {
			case client.Send <- data:
			default:
			}
		}
	}
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
