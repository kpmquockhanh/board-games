package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ping/game"
	"ping/models"
	"ping/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// liveServer puts the handler behind a real HTTP server so tests can open real
// sockets. Reload behaviour is a race between a closing socket and an arriving
// one, and only the real thing puts those two in the same order as a browser.
func liveServer(t *testing.T) (*Handler, *gin.Engine, *httptest.Server) {
	t.Helper()
	h, r := newTestHandler(t)
	r.GET("/ws", h.HandleWS)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return h, r, srv
}

// dial opens a socket the way a page does, identifying the tab behind it.
func dial(t *testing.T, srv *httptest.Server, roomKey, player, session string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") +
		"/ws?room=" + roomKey + "&player=" + player + "&session=" + session
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func postJoinAs(t *testing.T, r *gin.Engine, roomKey, player, session string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"room_key": roomKey, "player_name": player, "color": "red", "session": session,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/ek/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// eventually polls, since the hub registers and unregisters on its own
// goroutine and a socket's teardown reaches the handler some time after close.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// await reads from a socket until the given message type turns up.
func await(t *testing.T, conn *websocket.Conn, msgType string) models.WSMessage {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("waiting for %s: %v", msgType, err)
		}
		var msg models.WSMessage
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.Type == msgType {
			return msg
		}
	}
}

func startedGame(t *testing.T, h *Handler, roomKey string, names ...string) {
	t.Helper()
	h.setEKState(roomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       names[0],
		TurnOrder:  names,
		Players:    livePlayers(names...),
		CancelFunc: make(chan struct{}),
	})
}

// Reloading mid-game used to be refused: the socket the old page left behind
// was still registered, so the returning player looked like a second person
// claiming the name and was sent back to the join screen.
func TestReloadIsNotRefusedWhileTheOldSocketLingers(t *testing.T) {
	h, r, srv := liveServer(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo"} {
		if w := postJoinAs(t, r, room.RoomKey, name, "tab-"+name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}
	startedGame(t, h, room.RoomKey, "ana", "bo")

	old := dial(t, srv, room.RoomKey, "ana", "tab-ana")
	eventually(t, "ana's socket to register", func() bool {
		return h.hub.IsPlayerConnected(room.RoomKey, "ana")
	})

	// The reload's join, arriving before the server has noticed the old socket.
	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("rejoining after a reload: %d %s", w.Code, w.Body)
	}

	dial(t, srv, room.RoomKey, "ana", "tab-ana")
	old.Close()

	eventually(t, "the stale socket to be dropped", func() bool {
		return h.hub.RoomCount(room.RoomKey) == 1
	})
	if !seated(t, h, room.ID, "ana") {
		t.Fatal("reloading cost ana her seat")
	}
	if got := playerSeat(t, h, room.ID, "ana").DisconnectedAt; got != nil {
		t.Fatalf("ana is marked offline after reloading: %v", got)
	}
}

// The teardown of the old page's socket can land after the new page has already
// joined. Acted on, it takes a player who is sitting right there out of the
// room — in a lobby by releasing their seat outright.
func TestStaleTeardownAfterAReloadIsIgnored(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "lobby")
	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("seating ana: %d", w.Code)
	}
	stale := h.currentEpoch(room.RoomKey, "ana")

	// The reload lands first.
	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("rejoining after a reload: %d %s", w.Code, w.Body)
	}

	h.HandleDisconnect(room.RoomKey, "ana", stale)

	if !seated(t, h, room.ID, "ana") {
		t.Fatal("the old page's teardown released the seat of a player who had already come back")
	}
}

// Same seat, different browser: still refused. Reload has to be told apart from
// a second person, not waved through.
func TestASecondBrowserOnTheSameNameIsStillRefused(t *testing.T) {
	h, r, srv := liveServer(t)
	room := postCreate(t, h, r, "in progress")
	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("seating ana: %d", w.Code)
	}
	startedGame(t, h, room.RoomKey, "ana")

	dial(t, srv, room.RoomKey, "ana", "tab-ana")
	eventually(t, "ana's socket to register", func() bool {
		return h.hub.IsPlayerConnected(room.RoomKey, "ana")
	})

	if w := postJoinAs(t, r, room.RoomKey, "ana", "someone-elses-tab"); w.Code != http.StatusConflict {
		t.Fatalf("a second browser on ana's name got %d, want 409", w.Code)
	}
}

// Everyone else greys a dropped player out. Nothing ever took that mark off
// again, so a player who came back stayed "offline" for the rest of the game.
func TestComingBackIsAnnouncedToTheTable(t *testing.T) {
	h, r, srv := liveServer(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo"} {
		if w := postJoinAs(t, r, room.RoomKey, name, "tab-"+name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}
	startedGame(t, h, room.RoomKey, "ana", "bo")

	watcher := dial(t, srv, room.RoomKey, "bo", "tab-bo")
	anaConn := dial(t, srv, room.RoomKey, "ana", "tab-ana")
	eventually(t, "both sockets to register", func() bool {
		return h.hub.RoomCount(room.RoomKey) == 2
	})

	anaConn.Close()
	if msg := await(t, watcher, "player_disconnected"); msg.Player != "ana" {
		t.Fatalf("player_disconnected named %q", msg.Player)
	}
	eventually(t, "ana's seat to be stamped", func() bool {
		return playerSeat(t, h, room.ID, "ana").DisconnectedAt != nil
	})

	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("ana coming back: %d %s", w.Code, w.Body)
	}

	if msg := await(t, watcher, "player_reconnected"); msg.Player != "ana" {
		t.Fatalf("player_reconnected named %q", msg.Player)
	}
	if got := playerSeat(t, h, room.ID, "ana").DisconnectedAt; got != nil {
		t.Fatalf("ana is still stamped offline after coming back: %v", got)
	}
}

// The socket, not the join endpoint, may be the first to find the seat stamped
// — a socket that drops and comes back on its own never goes through join.
func TestASocketComingBackAloneAnnouncesIt(t *testing.T) {
	h, r, srv := liveServer(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo"} {
		if w := postJoinAs(t, r, room.RoomKey, name, "tab-"+name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}
	startedGame(t, h, room.RoomKey, "ana", "bo")
	watcher := dial(t, srv, room.RoomKey, "bo", "tab-bo")

	dropSocket(h, room.RoomKey, "ana")
	await(t, watcher, "player_disconnected")

	dial(t, srv, room.RoomKey, "ana", "tab-ana")

	if msg := await(t, watcher, "player_reconnected"); msg.Player != "ana" {
		t.Fatalf("player_reconnected named %q", msg.Player)
	}
}

// The page that comes back mid-game is sent the game it left. Registering a
// socket used to be a hand-off to the hub's own goroutine, so the state pushed
// immediately afterwards could find the socket not yet registered and drop it.
// The reloaded page then sat in the lobby, waiting for some later broadcast —
// clicking Ready made one, which is why the table came back only after that.
func TestReloadingMidGameIsSentTheRunningGame(t *testing.T) {
	h, r, srv := liveServer(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo"} {
		if w := postJoinAs(t, r, room.RoomKey, name, "tab-"+name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}
	startedGame(t, h, room.RoomKey, "ana", "bo")

	old := dial(t, srv, room.RoomKey, "ana", "tab-ana")
	eventually(t, "ana's socket to register", func() bool {
		return h.hub.IsPlayerConnected(room.RoomKey, "ana")
	})
	old.Close()

	// Nothing else happens at the table: the state has to arrive because the
	// socket opened, not because someone acted.
	fresh := dial(t, srv, room.RoomKey, "ana", "tab-ana")
	msg := await(t, fresh, "state_updated")

	var view struct {
		Phase string `json:"phase"`
		Turn  string `json:"turn"`
	}
	if err := json.Unmarshal(msg.Payload, &view); err != nil {
		t.Fatalf("state sent on reconnect: %v", err)
	}
	if view.Phase != "playing" {
		t.Fatalf("reconnect was sent phase %q, want playing", view.Phase)
	}
	if view.Turn != "ana" {
		t.Fatalf("reconnect was sent turn %q, want ana", view.Turn)
	}
}

// Register must have taken effect by the time it returns: the socket's own
// handler sends to it straight afterwards.
func TestASocketCanBeSentToAsSoonAsItIsRegistered(t *testing.T) {
	h, _, srv := liveServer(t)
	_ = srv
	hub := h.hub
	client := &ws.Client{Room: "room", Player: "ana", Session: "tab", Send: make(chan []byte, 4)}
	hub.Register(client)
	if !hub.Send(client, []byte("hello")) {
		t.Fatal("a client just registered could not be sent to")
	}
	if !hub.IsPlayerConnected("room", "ana") {
		t.Fatal("a client just registered is not connected")
	}
}
