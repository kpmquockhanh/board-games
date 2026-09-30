package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// seatServer is liveServer with the state routes wired too.
func seatServer(t *testing.T) (*Handler, *gin.Engine, *httptest.Server) {
	t.Helper()
	h, r, srv := liveServer(t)
	r.GET("/api/:game/rooms/:roomKey/state", h.GetState)
	r.POST("/api/:game/rooms/:roomKey/state", h.SaveState)
	r.POST("/api/:game/rooms/:roomKey/leave", h.LeaveRoom)
	return h, r, srv
}

func request(r *gin.Engine, method, url, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(seatTokenHeader, token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// joinWithToken joins as a stranger would: with whatever token they hold,
// usually none, and without remembering what comes back.
func joinWithToken(r *gin.Engine, roomKey, player, token string) *httptest.ResponseRecorder {
	return request(r, http.MethodPost, "/api/ek/join", token, map[string]string{
		"room_key": roomKey, "player_name": player, "color": "red", "session": "stranger",
	})
}

// A dropped player's seat used to be anyone's for the typing of their name,
// hand included.
func TestTypingASeatedPlayersNameDoesNotSeatYouInTheirPlace(t *testing.T) {
	h, r, _ := seatServer(t)
	room := postCreate(t, h, r, "table")
	postJoin(t, r, room.RoomKey, "ana")

	if w := joinWithToken(r, room.RoomKey, "ana", ""); w.Code != http.StatusConflict {
		t.Fatalf("no token: got %d, want 409; body %s", w.Code, w.Body)
	}
	if w := joinWithToken(r, room.RoomKey, "ana", "guessed"); w.Code != http.StatusConflict {
		t.Fatalf("wrong token: got %d, want 409; body %s", w.Code, w.Body)
	}
	if w := postJoin(t, r, room.RoomKey, "ana"); w.Code != http.StatusOK {
		t.Fatalf("ana herself: got %d, want 200; body %s", w.Code, w.Body)
	}
}

func TestStateShowsOnlyTheHandTheTokenProves(t *testing.T) {
	h, r, _ := seatServer(t)
	room := postCreate(t, h, r, "table")
	postJoin(t, r, room.RoomKey, "ana")
	postJoin(t, r, room.RoomKey, "bo")
	startedGame(t, h, room.RoomKey, "ana", "bo")

	hands := func(w *httptest.ResponseRecorder) map[string]int {
		t.Helper()
		var out struct {
			State struct {
				GameState struct {
					Players map[string]struct {
						Hand []string `json:"hand"`
					} `json:"players"`
				} `json:"gameState"`
			} `json:"state"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatalf("state: %v; body %s", err, w.Body)
		}
		seen := map[string]int{}
		for name, p := range out.State.GameState.Players {
			seen[name] = len(p.Hand)
		}
		return seen
	}

	url := "/api/ek/rooms/" + room.RoomKey + "/state?player=ana"
	if seen := hands(request(r, http.MethodGet, url, seatTokenOf(room.RoomKey, "bo"), nil)); seen["ana"] != 0 || seen["bo"] == 0 {
		t.Fatalf("bo asking for ana's view saw hands %v", seen)
	}
	if seen := hands(request(r, http.MethodGet, url, "", nil)); seen["ana"] != 0 || seen["bo"] != 0 {
		t.Fatalf("no token saw hands %v", seen)
	}
	if seen := hands(request(r, http.MethodGet, url, seatTokenOf(room.RoomKey, "ana"), nil)); seen["ana"] == 0 {
		t.Fatalf("ana did not see her own hand: %v", seen)
	}
}

func TestActionsAreTakenAsTheTokensPlayer(t *testing.T) {
	h, r, _ := seatServer(t)
	room := postCreate(t, h, r, "table")
	postJoin(t, r, room.RoomKey, "ana")
	postJoin(t, r, room.RoomKey, "bo")
	url := "/api/ek/rooms/" + room.RoomKey + "/state"
	act := map[string]any{"action": "toggleReady", "data": map[string]any{}, "player": "ana"}

	if w := request(r, http.MethodPost, url, "", act); w.Code != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401; body %s", w.Code, w.Body)
	}
	if w := request(r, http.MethodPost, url, seatTokenOf(room.RoomKey, "bo"), act); w.Code != http.StatusForbidden {
		t.Fatalf("bo acting as ana: got %d, want 403; body %s", w.Code, w.Body)
	}
	if w := request(r, http.MethodPost, url, seatTokenOf(room.RoomKey, "ana"), act); w.Code != http.StatusOK {
		t.Fatalf("ana: got %d, want 200; body %s", w.Code, w.Body)
	}
	if !playerSeat(t, h, room.ID, "ana").Ready || playerSeat(t, h, room.ID, "bo").Ready {
		t.Fatal("the ready flag landed on the wrong seat")
	}

	leave := map[string]string{"player_name": "ana"}
	if w := request(r, http.MethodPost, "/api/ek/rooms/"+room.RoomKey+"/leave", seatTokenOf(room.RoomKey, "bo"), leave); w.Code == http.StatusOK {
		t.Fatal("bo made ana leave")
	}
}

func TestASocketForAPlayerNeedsTheirToken(t *testing.T) {
	h, r, srv := seatServer(t)
	room := postCreate(t, h, r, "table")
	postJoin(t, r, room.RoomKey, "ana")

	base := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?room=" + room.RoomKey + "&player=ana&session=x"
	for _, token := range []string{"", "guessed"} {
		conn, resp, err := websocket.DefaultDialer.Dial(base+"&token="+token, nil)
		if err == nil {
			conn.Close()
			t.Fatalf("socket as ana opened with token %q", token)
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: want 401, got %v", token, resp)
		}
	}
	dial(t, srv, room.RoomKey, "ana", "tab-ana")
}

// A lobby drop releases the seat, and the page reconnects its socket before
// it rejoins. The old token has to open that socket, and the rejoin has to
// keep it, or the page would be locked out of the room it was in.
func TestALobbyDropCanReconnectAndRejoinWithTheSameToken(t *testing.T) {
	h, r, srv := seatServer(t)
	room := postCreate(t, h, r, "lobby")
	postJoin(t, r, room.RoomKey, "ana")
	token := seatTokenOf(room.RoomKey, "ana")

	h.store.RemovePlayer(room.ID, "ana")
	dial(t, srv, room.RoomKey, "ana", "tab-ana")

	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("rejoin: got %d; body %s", w.Code, w.Body)
	}
	if got := seatTokenOf(room.RoomKey, "ana"); got != token {
		t.Fatal("rejoining handed out a new token; the page's socket URL goes stale")
	}
}

// Once someone else takes a released name, the old token is worth nothing.
func TestAReleasedNameTakenByAnotherVoidsTheOldToken(t *testing.T) {
	h, r, srv := seatServer(t)
	room := postCreate(t, h, r, "lobby")
	postJoin(t, r, room.RoomKey, "ana")
	old := seatTokenOf(room.RoomKey, "ana")
	h.store.RemovePlayer(room.ID, "ana")

	if w := joinWithToken(r, room.RoomKey, "ana", ""); w.Code != http.StatusOK {
		t.Fatalf("new ana: got %d; body %s", w.Code, w.Body)
	}
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?room=" + room.RoomKey + "&player=ana&session=x&token=" + old
	if conn, _, err := websocket.DefaultDialer.Dial(url, nil); err == nil {
		conn.Close()
		t.Fatal("the old token still opens ana's socket")
	}
}

// Seats taken before tokens existed have none. The first to rejoin gets one,
// and after that the seat is as protected as any other.
func TestAnUntokenedSeatGoesToTheFirstToComeBack(t *testing.T) {
	h, r, _ := seatServer(t)
	room := postCreate(t, h, r, "old")
	h.store.AddPlayer(room.ID, "ana", "red", "", "")

	first := joinWithToken(r, room.RoomKey, "ana", "")
	if first.Code != http.StatusOK {
		t.Fatalf("claiming: got %d; body %s", first.Code, first.Body)
	}
	if w := joinWithToken(r, room.RoomKey, "ana", ""); w.Code != http.StatusConflict {
		t.Fatalf("second claim: got %d, want 409", w.Code)
	}
}
