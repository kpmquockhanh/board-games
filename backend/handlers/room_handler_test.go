package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"ping/models"
	"ping/storage"
	"ping/ws"

	"github.com/gin-gonic/gin"
)

func newTestHandler(t *testing.T) (*Handler, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store, err := storage.NewSQLite(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	hub := ws.NewHub()
	go hub.Run()

	h := NewHandler(store, hub)
	// Wired as main.go wires it, so a socket closing in a test goes down the
	// same path it does in the running server.
	hub.OnDisconnect(h.HandleDisconnect)
	r := gin.New()
	r.Use(h.Identify)
	h.RegisterAccount(r.Group("/api"))
	h.RegisterAuth(r.Group("/api"))
	r.POST("/api/:game/create", h.CreateRoom)
	r.POST("/api/:game/join", h.JoinRoom)
	return h, r
}

// postCreate makes a room the way the app does, through the handler, so tests
// get the initial settings snapshot that CreateRoom writes.
func postCreate(t *testing.T, h *Handler, r *gin.Engine, name string) *models.Room {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"name": name})
	req := httptest.NewRequest(http.MethodPost, "/api/ek/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create room: %d %s", w.Code, w.Body)
	}
	var out struct {
		RoomKey string `json:"room_key"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	room, err := h.store.GetRoom(out.RoomKey)
	if err != nil || room == nil {
		t.Fatalf("room %q not readable back: %v", out.RoomKey, err)
	}
	return room
}

func postJoin(t *testing.T, r *gin.Engine, roomKey, player string) *httptest.ResponseRecorder {
	t.Helper()
	return postJoinAs(t, r, roomKey, player, "")
}

// seatTokens stands in for a browser's storage: tests join, reload, and dial
// as the same player the way a tab does, carrying the token its seat was given.
var seatTokens sync.Map

func seatKey(roomKey, player string) string { return roomKey + "\x00" + player }

func seatTokenOf(roomKey, player string) string {
	if v, ok := seatTokens.Load(seatKey(roomKey, player)); ok {
		return v.(string)
	}
	return ""
}

func rememberSeatToken(t *testing.T, roomKey, player string, w *httptest.ResponseRecorder) {
	t.Helper()
	if w.Code != http.StatusOK {
		return
	}
	var out struct {
		SeatToken string `json:"seat_token"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || out.SeatToken == "" {
		t.Fatalf("join gave %s no seat token: %s", player, w.Body)
	}
	seatTokens.Store(seatKey(roomKey, player), out.SeatToken)
}

// A player still seated in a full room is rejoining, not taking a new seat.
// The capacity check used to run first and count them against the limit, so a
// mid-game disconnect at a full table locked them out of their own game.
func TestJoinRoomFullTableAllowsReturningPlayer(t *testing.T) {
	h, r := newTestHandler(t)

	room := postCreate(t, h, r, "table")

	// Cap the room at two so the table is full with two players.
	settings := defaultEKSettings()
	settings.MaxPlayers = 2
	blob, _ := json.Marshal(models.EKRoomState{RoomSettings: settings})
	if err := h.store.UpdateSnapshot(room.ID, "", string(blob)); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"ana", "bo"} {
		if w := postJoin(t, r, room.RoomKey, name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: got %d, body %s", name, w.Code, w.Body)
		}
	}

	// A third player must still be turned away.
	if w := postJoin(t, r, room.RoomKey, "cy"); w.Code != http.StatusConflict {
		t.Fatalf("third player: got %d, want 409; body %s", w.Code, w.Body)
	}

	// Ana dropped mid-game, so HandleDisconnect kept her seat: her row is
	// still active and still counted. Rejoining must succeed.
	w := postJoin(t, r, room.RoomKey, "ana")
	if w.Code != http.StatusOK {
		t.Fatalf("returning player: got %d, want 200; body %s", w.Code, w.Body)
	}
}
