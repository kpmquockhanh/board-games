package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ping/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// hotpotRoom makes a Hotpot room and seats the given players in it.
func hotpotRoom(t *testing.T, h *Handler, r *gin.Engine, players ...string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": "pot"})
	req := httptest.NewRequest(http.MethodPost, "/api/hotpot/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var out struct {
		RoomKey string `json:"room_key"`
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &out) != nil {
		t.Fatalf("create hotpot room: %d %s", w.Code, w.Body)
	}
	for _, name := range players {
		body, _ := json.Marshal(map[string]string{"room_key": out.RoomKey, "player_name": name, "color": "red", "session": "tab-" + name})
		req := httptest.NewRequest(http.MethodPost, "/api/hotpot/join", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("join %s: %d %s", name, w.Code, w.Body)
		}
		rememberSeatToken(t, out.RoomKey, name, w)
	}
	return out.RoomKey
}

func sendWS(t *testing.T, conn *websocket.Conn, msgType string, payload any) {
	t.Helper()
	raw, _ := json.Marshal(payload)
	if err := conn.WriteJSON(models.WSMessage{Type: msgType, Payload: raw}); err != nil {
		t.Fatalf("send %s: %v", msgType, err)
	}
}

// relayedUntil reads everything that reaches conn up to the first message of
// the given type, and returns the ones seen before it along with it.
func relayedUntil(t *testing.T, conn *websocket.Conn, msgType string) ([]models.WSMessage, models.WSMessage) {
	t.Helper()
	var seen []models.WSMessage
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %s after %v: %v", msgType, seen, err)
		}
		var msg models.WSMessage
		json.Unmarshal(data, &msg)
		if msg.Type == msgType {
			return seen, msg
		}
		seen = append(seen, msg)
	}
}

// Whatever a client sent used to go to the whole room as is, so any page could
// hand the others a fake state, prompt, or room_deleted.
func TestAClientCannotRelayServerMessages(t *testing.T) {
	h, r, srv := liveServer(t)
	room := hotpotRoom(t, h, r, "ana", "bo")
	ana := dial(t, srv, room, "ana", "tab-ana")
	bo := dial(t, srv, room, "bo", "tab-bo")
	eventually(t, "both sockets registered", func() bool { return len(h.hub.ConnectedPlayers(room)) == 2 })

	for _, forged := range []string{"state_updated", "room_deleted", "prompt_defuse", "player_left"} {
		sendWS(t, ana, forged, map[string]any{})
	}
	sendWS(t, ana, "chat", map[string]any{"text": "hi"})

	// Bo's own socket is sent his state when it opens, so what matters is
	// whether anything came from ana.
	seen, _ := relayedUntil(t, bo, "chat")
	for _, got := range seen {
		if got.Player == "ana" {
			t.Fatalf("bo was sent a forged %s", got.Type)
		}
	}
}

// The feed shows the name in the payload, so it has to be the sender's own.
func TestARelayedMessageCarriesTheSendersOwnName(t *testing.T) {
	h, r, srv := liveServer(t)
	room := hotpotRoom(t, h, r, "ana", "bo")
	ana := dial(t, srv, room, "ana", "tab-ana")
	bo := dial(t, srv, room, "bo", "tab-bo")
	eventually(t, "both sockets registered", func() bool { return len(h.hub.ConnectedPlayers(room)) == 2 })

	sendWS(t, ana, "chat", map[string]any{"name": "bo", "color": "red", "text": "it was me"})

	_, msg := relayedUntil(t, bo, "chat")
	var payload map[string]any
	json.Unmarshal(msg.Payload, &payload)
	if msg.Player != "ana" || payload["name"] != "ana" {
		t.Fatalf("chat relayed as player %q, name %v", msg.Player, payload["name"])
	}
	if payload["text"] != "it was me" {
		t.Fatalf("the rest of the payload was lost: %v", payload)
	}
}

// EK clients send nothing over the socket; everything goes through actions.
func TestAnEKSocketRelaysNothing(t *testing.T) {
	h, r, srv := liveServer(t)
	room := postCreate(t, h, r, "table")
	postJoin(t, r, room.RoomKey, "ana")
	postJoin(t, r, room.RoomKey, "bo")
	ana := dial(t, srv, room.RoomKey, "ana", "tab-ana")
	bo := dial(t, srv, room.RoomKey, "bo", "tab-bo")
	eventually(t, "both sockets registered", func() bool { return len(h.hub.ConnectedPlayers(room.RoomKey)) == 2 })

	sendWS(t, ana, "chat", map[string]any{"text": "hi"})

	bo.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	for {
		_, data, err := bo.ReadMessage()
		if err != nil {
			return
		}
		var msg models.WSMessage
		json.Unmarshal(data, &msg)
		if msg.Type == "chat" {
			t.Fatal("an EK socket relayed a chat to the room")
		}
	}
}
