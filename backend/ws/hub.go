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
	// Session identifies the browser tab behind the socket. A reload opens a
	// new socket while the one the old page left behind is still registered,
	// and the two are indistinguishable by room and name alone.
	Session string
	// Epoch orders arrivals on a seat. A teardown carrying an old epoch comes
	// from a page that has already been replaced, so it says nothing about
	// whether the player is still here.
	Epoch uint64
	Send  chan []byte
	Hub   *Hub
}

type Hub struct {
	clients      map[*Client]bool
	rooms        map[string]map[*Client]bool
	broadcast    chan []byte
	mu           sync.RWMutex
	onDisconnect func(room, player string, epoch uint64)
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[*Client]bool),
		rooms:     make(map[string]map[*Client]bool),
		broadcast: make(chan []byte, 256),
	}
}

func (h *Hub) OnDisconnect(fn func(room, player string, epoch uint64)) {
	h.onDisconnect = fn
}

// removeLocked takes a client out of the hub's maps. It reports whether the
// client was still registered, so a caller only acts once on a socket that two
// paths may be tearing down at the same time. h.mu must be held for writing.
func (h *Hub) removeLocked(client *Client) bool {
	if _, ok := h.clients[client]; !ok {
		return false
	}
	delete(h.clients, client)
	close(client.Send)
	if room, ok := h.rooms[client.Room]; ok {
		delete(room, client)
		if len(room) == 0 {
			delete(h.rooms, client.Room)
		}
	}
	return true
}

// EvictSession closes the sockets this player left behind from the same
// browser tab. A reload leaves one registered until the server notices the old
// connection is gone, and until then the returning player looks like a second
// person using their name. Evicting is deliberately not a disconnect: the page
// that owned the socket is gone, but the player is right here.
func (h *Hub) EvictSession(room, player, session string) int {
	if player == "" || session == "" {
		return 0
	}

	h.mu.Lock()
	var ghosts []*Client
	for c := range h.rooms[room] {
		if c.Player == player && c.Session == session {
			ghosts = append(ghosts, c)
		}
	}
	evicted := 0
	for _, c := range ghosts {
		if h.removeLocked(c) {
			evicted++
		}
	}
	h.mu.Unlock()

	for _, c := range ghosts {
		c.Conn.Close()
	}
	if evicted > 0 {
		log.Printf("[ws] dropped %d stale socket(s) for player=%s room=%s (same tab reconnected)", evicted, player, room)
	}
	return evicted
}

// IsPlayerConnectedFromElsewhere reports whether some other page is live on
// this seat. The socket a reloading tab left behind presents the same session
// id, so it does not count as somebody else holding the name.
func (h *Hub) IsPlayerConnectedFromElsewhere(room, player, session string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[room] {
		if c.Player != player {
			continue
		}
		if session != "" && c.Session == session {
			continue
		}
		return true
	}
	return false
}

// Run fans out room-wide broadcasts. Joining and leaving are not handled here:
// they happen inline in Register and Unregister, so that they have taken effect
// by the time those return.
func (h *Hub) Run() {
	for message := range h.broadcast {
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
			var toNotify []*Client
			h.mu.Lock()
			for _, client := range stale {
				if h.removeLocked(client) && h.onDisconnect != nil && client.Player != "" {
					toNotify = append(toNotify, client)
				}
			}
			h.mu.Unlock()
			for _, c := range toNotify {
				h.onDisconnect(c.Room, c.Player, c.Epoch)
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
		var toNotify []*Client
		h.mu.Lock()
		for _, client := range stale {
			if h.removeLocked(client) && h.onDisconnect != nil && client.Player != "" {
				toNotify = append(toNotify, client)
			}
		}
		h.mu.Unlock()
		for _, c := range toNotify {
			h.onDisconnect(c.Room, c.Player, c.Epoch)
		}
	}
}

// Send delivers to one client, unless it has already been dropped from the
// hub. Checking under the lock is what makes it safe: a client's channel is
// only ever closed while the hub is held for writing, so a socket evicted
// while its own handler is still setting it up cannot be written to afterwards.
func (h *Hub) Send(client *Client, data []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if _, ok := h.clients[client]; !ok {
		return false
	}
	select {
	case client.Send <- data:
		return true
	default:
		return false
	}
}

// Register adds a client, holding the lock itself rather than handing the
// client to Run. It used to be a channel send, which returned before the client
// was actually in the maps: the state sent to a reconnecting page immediately
// afterwards was then dropped as "not registered", and a reloaded tab sat in
// the lobby while its game was running, until some later broadcast reached it.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	// The same tab reconnecting replaces its own socket rather than joining
	// alongside it, so a reload never leaves two registered.
	var ghosts []*Client
	if client.Player != "" && client.Session != "" {
		for c := range h.rooms[client.Room] {
			if c != client && c.Player == client.Player && c.Session == client.Session {
				ghosts = append(ghosts, c)
			}
		}
		for _, c := range ghosts {
			h.removeLocked(c)
		}
	}
	h.clients[client] = true
	if h.rooms[client.Room] == nil {
		h.rooms[client.Room] = make(map[*Client]bool)
	}
	h.rooms[client.Room][client] = true
	total := len(h.rooms[client.Room])
	h.mu.Unlock()

	for _, c := range ghosts {
		c.Conn.Close()
	}
	log.Printf("[ws] client joined room=%s player=%s (total in room: %d)", client.Room, client.Player, total)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	// Already gone means it was evicted: a newer page owns the seat and nobody
	// should hear that this socket's player disconnected.
	dropped := h.removeLocked(client)
	h.mu.Unlock()

	if dropped && h.onDisconnect != nil && client.Player != "" {
		h.onDisconnect(client.Room, client.Player, client.Epoch)
	}
	log.Printf("[ws] client left room=%s player=%s", client.Room, client.Player)
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
