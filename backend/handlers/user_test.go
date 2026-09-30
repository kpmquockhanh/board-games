package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ping/models"

	"github.com/gin-gonic/gin"
)

// joinFromBrowser joins as a tab of a browser holding the given session
// cookie ("" for a browser that has none yet).
func joinFromBrowser(t *testing.T, r *gin.Engine, roomKey, player, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"room_key": roomKey, "player_name": player, "color": "red", "session": "tab-" + player,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/ek/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(seatTokenHeader, seatTokenOf(roomKey, player))
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("join %s: %d %s", player, w.Code, w.Body)
	}
	rememberSeatToken(t, roomKey, player, w)
	return w
}

func sessionCookieSet(w *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			return c
		}
	}
	return nil
}

func me(t *testing.T, r *gin.Engine, cookie string) (*models.User, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: sessionCookie, Value: cookie})
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me: %d %s", w.Code, w.Body)
	}
	var u models.User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil {
		t.Fatal(err)
	}
	return &u, w
}

func seatUser(t *testing.T, h *Handler, roomID int64, name string) string {
	t.Helper()
	if id := playerSeat(t, h, roomID, name).UserID; id != nil {
		return *id
	}
	return ""
}

func TestSittingDownMakesAGuestForTheBrowser(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "table")

	cookie := sessionCookieSet(joinFromBrowser(t, r, room.RoomKey, "ana", ""))
	if cookie == nil || cookie.Value == "" {
		t.Fatal("joining set no session cookie")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Fatalf("session cookie is %+v; want HttpOnly, SameSite=Lax, Path=/", cookie)
	}

	u, _ := me(t, r, cookie.Value)
	if !u.Guest || u.DisplayName != "ana" || u.Color != "red" {
		t.Fatalf("me: %+v", u)
	}
	if got := seatUser(t, h, room.ID, "ana"); got != u.ID {
		t.Fatalf("ana's seat is %q's, want %q's", got, u.ID)
	}
}

// Two tabs of one browser share the cookie: one person in two seats. The
// seat tokens, one per tab, still tell the seats apart.
func TestTwoTabsOfOneBrowserAreOneGuestInTwoSeats(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "table")
	cookie := sessionCookieSet(joinFromBrowser(t, r, room.RoomKey, "ana", "")).Value

	second := joinFromBrowser(t, r, room.RoomKey, "bo", cookie)
	if renewed := sessionCookieSet(second); renewed != nil && renewed.Value != cookie {
		t.Fatal("the second tab was given a different account")
	}
	if a, b := seatUser(t, h, room.ID, "ana"), seatUser(t, h, room.ID, "bo"); a == "" || a != b {
		t.Fatalf("seats belong to %q and %q, want one guest", a, b)
	}
	if seatTokenOf(room.RoomKey, "ana") == seatTokenOf(room.RoomKey, "bo") {
		t.Fatal("the two seats share a token")
	}
}

// A reload rejoins through the existing seat, and must keep the same account.
func TestRejoiningKeepsTheAccount(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "table")
	cookie := sessionCookieSet(joinFromBrowser(t, r, room.RoomKey, "ana", "")).Value
	before := seatUser(t, h, room.ID, "ana")

	joinFromBrowser(t, r, room.RoomKey, "ana", cookie)
	if after := seatUser(t, h, room.ID, "ana"); after != before {
		t.Fatalf("rejoining moved the seat from %q to %q", before, after)
	}
}

// A cookie that names no session, such as one for a guest since pruned, is
// replaced rather than trusted.
func TestAnUnknownCookieGetsAFreshGuest(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "table")

	cookie := sessionCookieSet(joinFromBrowser(t, r, room.RoomKey, "ana", "made-up"))
	if cookie == nil || cookie.Value == "made-up" {
		t.Fatalf("an unknown cookie was kept: %+v", cookie)
	}
	if seatUser(t, h, room.ID, "ana") == "" {
		t.Fatal("the seat has no account")
	}
}

// Only sitting down, or asking, makes an account. Reads by a page that is
// only looking must not, or its parallel first requests would make several.
func TestLookingAroundMakesNoAccount(t *testing.T) {
	h, r, _ := seatServer(t)
	room := postCreate(t, h, r, "table")
	w := request(r, http.MethodGet, "/api/ek/rooms/"+room.RoomKey+"/state", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("state: %d %s", w.Code, w.Body)
	}
	if sessionCookieSet(w) != nil {
		t.Fatal("reading a room's state made an account")
	}

	u, w := me(t, r, "")
	if u.ID == "" || sessionCookieSet(w) == nil {
		t.Fatal("asking who I am made no account")
	}
	if again, _ := me(t, r, sessionCookieSet(w).Value); again.ID != u.ID {
		t.Fatal("asking again with the cookie gave another account")
	}
}
