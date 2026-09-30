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

	"github.com/gin-gonic/gin"
)

// joinWithToken joins from a browser's tab holding the given seat token, ""
// for a tab that has none, such as one on another device.
func (b browser) joinWithToken(r *gin.Engine, roomKey, player, token string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{
		"room_key": roomKey, "player_name": player, "color": "red", "session": "tab-" + player + "-" + token,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/ek/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set(seatTokenHeader, token)
	}
	return b.do(r, req)
}

func (b browser) seats(t *testing.T, r *gin.Engine) []models.Seat {
	t.Helper()
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/me/seats", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("seats: %d %s", w.Code, w.Body)
	}
	var out struct{ Seats []models.Seat }
	json.Unmarshal(w.Body.Bytes(), &out)
	return out.Seats
}

func (b browser) matches(t *testing.T, r *gin.Engine) []models.Match {
	t.Helper()
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/me/matches", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("matches: %d %s", w.Code, w.Body)
	}
	var out struct{ Matches []models.Match }
	json.Unmarshal(w.Body.Bytes(), &out)
	return out.Matches
}

func seatTokenOwner(t *testing.T, h *Handler, roomID int64, token string) string {
	t.Helper()
	name, _, err := h.store.SeatHolder(roomID, hashSecret(token))
	if err != nil {
		t.Fatal(err)
	}
	return name
}

// playAndForfeit deals ana and bo a game and ends it by forfeiting bo, the
// way the reaper does, so ana wins.
func playAndForfeit(t *testing.T, h *Handler, room *models.Room) {
	t.Helper()
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       "ana",
		TurnOrder:  []string{"ana", "bo"},
		Players:    livePlayers("ana", "bo"),
		StartedAt:  time.Now().Add(-10 * time.Minute),
		CancelFunc: make(chan struct{}),
	})
	h.forfeitPlayer(models.StalePlayer{RoomID: room.ID, RoomKey: room.RoomKey, Game: "ek", PlayerName: "bo"})
	if gs := h.getEKState(room.RoomKey); gs.Phase != "ended" {
		t.Fatalf("setup: the game is %q after the forfeit, want ended", gs.Phase)
	}
}

func TestAFinishedGameIsInEveryPlayersHistory(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "friday")
	ana, bo := browser{}, browser{}
	ana.join(t, r, room.RoomKey, "ana")
	bo.login(t, r, f, "sub-bo", "/")
	bo.join(t, r, room.RoomKey, "bo")

	playAndForfeit(t, h, room)
	// Ending the same game again, as a timer racing the last action might,
	// records nothing new.
	h.recordEKMatch(room.ID, room.RoomKey, h.getEKState(room.RoomKey))

	for _, tc := range []struct {
		who     string
		browser browser
	}{{"ana", ana}, {"bo", bo}} {
		ms := tc.browser.matches(t, r)
		if len(ms) != 1 {
			t.Fatalf("%s has %d matches, want 1: %+v", tc.who, len(ms), ms)
		}
		m := ms[0]
		if m.Game != "ek" || m.RoomName != "friday" || len(m.Players) != 2 {
			t.Fatalf("%s's match: %+v", tc.who, m)
		}
		for _, p := range m.Players {
			if p.Won != (p.Name == "ana") {
				t.Errorf("%s's match says %s won=%v", tc.who, p.Name, p.Won)
			}
			if p.You != (p.Name == tc.who) {
				t.Errorf("%s's match marks %s as you=%v", tc.who, p.Name, p.You)
			}
		}
	}

	if ms := (browser{}).matches(t, r); len(ms) != 0 {
		t.Fatalf("a browser with no account sees %d matches", len(ms))
	}
}

// The history outlives the room, which the reaper deletes a day after.
func TestHistoryOutlivesTheRoom(t *testing.T) {
	h, r, _ := authServer(t)
	room := postCreate(t, h, r, "gone")
	ana := browser{}
	ana.join(t, r, room.RoomKey, "ana")
	postJoin(t, r, room.RoomKey, "bo")
	playAndForfeit(t, h, room)

	if err := h.store.DeleteRoom(room.RoomKey); err != nil {
		t.Fatal(err)
	}
	if ms := ana.matches(t, r); len(ms) != 1 || ms[0].RoomName != "gone" {
		t.Fatalf("after the room was deleted: %+v", ms)
	}
}

// A guest's games come along when its browser signs in to an account that
// already exists, the same as its seats do.
func TestSigningInBringsTheGuestsMatches(t *testing.T) {
	h, r, f := authServer(t)
	laptop := browser{}
	laptop.login(t, r, f, "sub-ana", "/")

	room := postCreate(t, h, r, "table")
	phone := browser{}
	phone.join(t, r, room.RoomKey, "ana")
	postJoin(t, r, room.RoomKey, "bo")
	playAndForfeit(t, h, room)

	phone.login(t, r, f, "sub-ana", "/")
	if ms := laptop.matches(t, r); len(ms) != 1 || !ms[0].Players[0].You {
		t.Fatalf("the account's matches after the guest folded in: %+v", ms)
	}
}

// Signed in, a player can sit back down in their seat from another device:
// the seat gets a new token, and the old device's one stops working.
func TestASignedInAccountTakesItsSeatToAnotherDevice(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")
	laptop, phone := browser{}, browser{}
	laptop.login(t, r, f, "sub-ana", "/")
	laptop.join(t, r, room.RoomKey, "ana")
	oldToken := seatTokenOf(room.RoomKey, "ana")
	phone.login(t, r, f, "sub-ana", "/")

	seats := phone.seats(t, r)
	if len(seats) != 1 || seats[0].RoomKey != room.RoomKey || seats[0].PlayerName != "ana" || seats[0].Game != "ek" {
		t.Fatalf("the phone's list of seats: %+v", seats)
	}

	w := phone.joinWithToken(r, room.RoomKey, "ana", "")
	if w.Code != http.StatusOK {
		t.Fatalf("taking the seat to the phone: %d %s", w.Code, w.Body)
	}
	var out struct {
		SeatToken string `json:"seat_token"`
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if seatTokenOwner(t, h, room.ID, out.SeatToken) != "ana" {
		t.Fatal("the phone's new token does not hold the seat")
	}
	if seatTokenOwner(t, h, room.ID, oldToken) != "" {
		t.Fatal("the laptop's token still holds the seat")
	}
}

// Nobody else can do that: not another account, and not a guest, whose
// cookie two people sharing a browser in two tabs have in common.
func TestOnlyTheSeatsSignedInAccountCanTakeItWithoutTheToken(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")

	guest := browser{}
	guest.join(t, r, room.RoomKey, "ana")
	if w := guest.joinWithToken(r, room.RoomKey, "ana", ""); w.Code != http.StatusConflict {
		t.Fatalf("another tab of the guest's browser took ana's seat: %d %s", w.Code, w.Body)
	}
	if seats := guest.seats(t, r); len(seats) != 0 {
		t.Fatalf("a guest was offered seats to take elsewhere: %+v", seats)
	}

	owner, stranger := browser{}, browser{}
	owner.login(t, r, f, "sub-bo", "/")
	owner.join(t, r, room.RoomKey, "bo")
	stranger.login(t, r, f, "sub-cy", "/")
	w := stranger.joinWithToken(r, room.RoomKey, "bo", "")
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "already taken") {
		t.Fatalf("another account took bo's seat: %d %s", w.Code, w.Body)
	}
	if seatTokenOwner(t, h, room.ID, seatTokenOf(room.RoomKey, "bo")) != "bo" {
		t.Fatal("bo's own token stopped working")
	}
}

// A seat whose player is connected on the other device stays there. Taking
// it would leave two devices fighting over one seat.
func TestTheSeatStaysWhileTheOtherDeviceIsConnected(t *testing.T) {
	h, r, f := authServer(t)
	room := postCreate(t, h, r, "table")
	laptop, phone := browser{}, browser{}
	laptop.login(t, r, f, "sub-ana", "/")
	laptop.join(t, r, room.RoomKey, "ana")
	connect(t, h, room.RoomKey, "ana")
	phone.login(t, r, f, "sub-ana", "/")

	w := phone.joinWithToken(r, room.RoomKey, "ana", "")
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "another device") {
		t.Fatalf("taking a connected seat: %d %s", w.Code, w.Body)
	}
	if seatTokenOwner(t, h, room.ID, seatTokenOf(room.RoomKey, "ana")) != "ana" {
		t.Fatal("the connected laptop lost its token")
	}
}

func (b browser) playedWith(t *testing.T, r *gin.Engine) []models.Companion {
	t.Helper()
	w := b.do(r, httptest.NewRequest(http.MethodGet, "/api/me/played-with", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("played-with: %d %s", w.Code, w.Body)
	}
	var out struct {
		PlayedWith []models.Companion `json:"played_with"`
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	return out.PlayedWith
}

// finishGame deals the room's seated players a game and forfeits everyone
// but the first, so the first wins.
func finishGame(t *testing.T, h *Handler, room *models.Room, players ...string) {
	t.Helper()
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       players[0],
		TurnOrder:  players,
		Players:    livePlayers(players...),
		StartedAt:  time.Now().Add(-10 * time.Minute),
		CancelFunc: make(chan struct{}),
	})
	for _, p := range players[1:] {
		h.forfeitPlayer(models.StalePlayer{RoomID: room.ID, RoomKey: room.RoomKey, Game: "ek", PlayerName: p})
	}
	if gs := h.getEKState(room.RoomKey); gs.Phase != "ended" {
		t.Fatalf("setup: the game is %q, want ended", gs.Phase)
	}
}

// Everyone an account has finished a game with is listed once, known by
// their account rather than their name, under the name they last used.
func TestPlayedWithListsEachPersonOnceUnderTheirLatestName(t *testing.T) {
	h, r, _ := authServer(t)
	ana, bo, cy := browser{}, browser{}, browser{}

	first := postCreate(t, h, r, "first")
	ana.join(t, r, first.RoomKey, "ana")
	bo.join(t, r, first.RoomKey, "bo")
	finishGame(t, h, first, "ana", "bo")
	time.Sleep(1100 * time.Millisecond) // the games' ends are stored to the second

	second := postCreate(t, h, r, "second")
	ana.join(t, r, second.RoomKey, "ana")
	bo.join(t, r, second.RoomKey, "bobby")
	cy.join(t, r, second.RoomKey, "cy")
	finishGame(t, h, second, "ana", "bobby", "cy")

	got := ana.playedWith(t, r)
	if len(got) != 2 {
		t.Fatalf("ana played with %+v, want bo and cy once each", got)
	}
	byName := map[string]models.Companion{}
	for _, c := range got {
		byName[c.Name] = c
	}
	if byName["bobby"].Games != 2 || byName["cy"].Games != 1 {
		t.Fatalf("ana's companions: %+v", got)
	}
	if _, ok := byName["bo"]; ok {
		t.Fatal("bo is listed under the name he no longer uses")
	}
	if seen := cy.playedWith(t, r); len(seen) != 2 {
		t.Fatalf("cy played with %+v, want ana and bobby", seen)
	}
	if none := (browser{}).playedWith(t, r); len(none) != 0 {
		t.Fatalf("a browser with no account played with %+v", none)
	}
}

// A browser's two seats in one game are one account, and it did not play
// with itself.
func TestPlayedWithLeavesOutTheAccountsOwnSeats(t *testing.T) {
	h, r, _ := authServer(t)
	room := postCreate(t, h, r, "table")
	shared := browser{}
	shared.join(t, r, room.RoomKey, "ana")
	shared.join(t, r, room.RoomKey, "bo")
	finishGame(t, h, room, "ana", "bo")

	if got := shared.playedWith(t, r); len(got) != 0 {
		t.Fatalf("a browser played with its own seats: %+v", got)
	}
}
