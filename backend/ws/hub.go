package ws

import (
	"encoding/json"
	"log"
	"sync"

	"ping/models"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn   *websocket.Conn
	Room   string
	Player string
	Send   chan []byte
	Hub    *Hub
}

type Hub struct {
	clients      map[*Client]bool
	rooms        map[string]map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
	onDisconnect func(room, player string)
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) OnDisconnect(fn func(room, player string)) {
	h.onDisconnect = fn
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			log.Printf("[ws] client joined room=%s player=%s (total in room: %d)", client.Room, client.Player, len(h.rooms[client.Room]))
			h.clients[client] = true
			if h.rooms[client.Room] == nil {
				h.rooms[client.Room] = make(map[*Client]bool)
			}
			h.rooms[client.Room][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			var toNotify []struct{ room, player string }
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				if room, ok := h.rooms[client.Room]; ok {
					delete(room, client)
					if len(room) == 0 {
						delete(h.rooms, client.Room)
					}
				}
				if h.onDisconnect != nil && client.Player != "" {
					toNotify = append(toNotify, struct{ room, player string }{client.Room, client.Player})
				}
			}
			h.mu.Unlock()
			for _, n := range toNotify {
				h.onDisconnect(n.room, n.player)
			}
			log.Printf("[ws] client left room=%s player=%s", client.Room, client.Player)

		case message := <-h.broadcast:
			h.mu.RLock()
			var stale []*Client
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					stale = append(stale, client)
				}
			}
			h.mu.RUnlock()

			if len(stale) > 0 {
				var toNotify []struct{ room, player string }
				h.mu.Lock()
				for _, client := range stale {
					if _, ok := h.clients[client]; ok {
						close(client.Send)
						delete(h.clients, client)
						if room, ok := h.rooms[client.Room]; ok {
							delete(room, client)
							if len(room) == 0 {
								delete(h.rooms, client.Room)
							}
						}
						if h.onDisconnect != nil && client.Player != "" {
							toNotify = append(toNotify, struct{ room, player string }{client.Room, client.Player})
						}
					}
				}
				h.mu.Unlock()
				for _, n := range toNotify {
					h.onDisconnect(n.room, n.player)
				}
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(room string, msg models.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] marshal error: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.rooms[room]
	var stale []*Client
	if ok {
		for client := range clients {
			select {
			case client.Send <- data:
			default:
				stale = append(stale, client)
			}
		}
	}
	h.mu.RUnlock()

	if len(stale) > 0 {
		var toNotify []struct{ room, player string }
		h.mu.Lock()
		for _, client := range stale {
			if _, ok := h.clients[client]; ok {
				close(client.Send)
				delete(h.clients, client)
				if room, ok := h.rooms[client.Room]; ok {
					delete(room, client)
					if len(room) == 0 {
						delete(h.rooms, client.Room)
					}
				}
				if h.onDisconnect != nil && client.Player != "" {
					toNotify = append(toNotify, struct{ room, player string }{client.Room, client.Player})
				}
			}
		}
		h.mu.Unlock()
		for _, n := range toNotify {
			h.onDisconnect(n.room, n.player)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) RoomCount(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}

func (h *Hub) IsPlayerConnected(room, player string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients, ok := h.rooms[room]
	if !ok {
		return false
	}
	for c := range clients {
		if c.Player == player {
			return true
		}
	}
	return false
}

func (h *Hub) ConnectedPlayers(room string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	players := make([]string, 0)
	clients, ok := h.rooms[room]
	if !ok {
		return players
	}
	for c := range clients {
		if c.Player != "" {
			players = append(players, c.Player)
		}
	}
	return players
}

func (h *Hub) SendToPlayer(room, player string, msg models.WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] marshal error: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.rooms[room]
	if ok {
		for client := range clients {
			if client.Player == player {
				select {
				case client.Send <- data:
				default:
				}
				break
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) CloseRoom(room string) {
	msg := models.WSMessage{
		Type: "room_deleted",
		Room: room,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] marshal error: %v", err)
		return
	}

	h.mu.Lock()
	clients, ok := h.rooms[room]
	var toClose []*Client
	if ok {
		for client := range clients {
			toClose = append(toClose, client)
		}
		// Clean up hub state directly under the lock
		for _, client := range toClose {
			delete(h.clients, client)
		}
		delete(h.rooms, room)
	}
	h.mu.Unlock()

	for _, client := range toClose {
		// Write directly to the conn, bypassing the buffered channel
		client.Conn.WriteMessage(websocket.TextMessage, data)
		client.Conn.Close()
		log.Printf("[ws] kicked player=%s from room=%s (room deleted)", client.Player, room)
	}
}
