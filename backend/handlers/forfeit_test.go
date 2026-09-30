package handlers

import (
	"net/http"
	"testing"
	"time"

	"ping/game"
	"ping/models"
)

// dropSocket tears down the socket the player's current page holds, the way the
// hub does when a connection goes away. The epoch is what a real client would
// have been given when it arrived, so the drop is not mistaken for the stale
// teardown of a page that has since been replaced.
func dropSocket(h *Handler, roomKey, player string) {
	h.HandleDisconnect(roomKey, player, h.currentEpoch(roomKey, player))
}

func playerSeat(t *testing.T, h *Handler, roomID int64, name string) models.RoomPlayer {
	t.Helper()
	players, err := h.store.GetRoomPlayers(roomID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range players {
		if p.PlayerName == name {
			return p
		}
	}
	t.Fatalf("%s is not seated in room %d", name, roomID)
	return models.RoomPlayer{}
}

func seated(t *testing.T, h *Handler, roomID int64, name string) bool {
	t.Helper()
	players, _ := h.store.GetRoomPlayers(roomID)
	for _, p := range players {
		if p.PlayerName == name {
			return true
		}
	}
	return false
}

func livePlayers(names ...string) map[string]*game.PlayerState {
	players := map[string]*game.PlayerState{}
	for _, n := range names {
		players[n] = &game.PlayerState{Color: "#fff", Alive: true, Hand: []string{"sk_001"}}
	}
	return players
}

// A mid-game drop holds the seat and starts a clock on it, rather than either
// releasing it immediately or holding it forever.
func TestDisconnectMidGameStampsTheSeatInsteadOfReleasingIt(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo", "cy"} {
		if w := postJoin(t, r, room.RoomKey, name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       "ana",
		TurnOrder:  []string{"ana", "bo", "cy"},
		Players:    livePlayers("ana", "bo", "cy"),
		CancelFunc: make(chan struct{}),
	})

	dropSocket(h, room.RoomKey, "bo")

	if !seated(t, h, room.ID, "bo") {
		t.Fatal("the seat was released on a mid-game drop")
	}
	if playerSeat(t, h, room.ID, "bo").DisconnectedAt == nil {
		t.Fatal("the drop was never stamped, so nothing can ever reclaim the seat")
	}
	if n, _ := h.store.GetActivePlayerCount(room.ID); n != 3 {
		t.Fatalf("active players = %d, want 3 — the seat is still theirs for now", n)
	}
}

// In a lobby a drop just means leaving, and no clock is needed.
// A lobby drop releases the seat, once the grace a reload needs has run out.
func TestDisconnectInLobbyReleasesTheSeatAfterTheGrace(t *testing.T) {
	h, r := newTestHandler(t)
	h.lobbyGrace = 50 * time.Millisecond
	room := postCreate(t, h, r, "lobby")
	postJoin(t, r, room.RoomKey, "ana")

	dropSocket(h, room.RoomKey, "ana")
	if !seated(t, h, room.ID, "ana") {
		t.Fatal("a lobby drop released the seat before a reload could come back for it")
	}

	time.Sleep(4 * h.lobbyGrace)
	if seated(t, h, room.ID, "ana") {
		t.Fatal("a lobby drop left the seat held after the grace")
	}
}

// Reloading in the lobby closes the old socket before the new page joins.
// Releasing the seat in between seated the player afresh, and not ready.
func TestReloadingInTheLobbyKeepsTheSeatAndItsReadyFlag(t *testing.T) {
	h, r, _ := seatServer(t)
	h.lobbyGrace = 50 * time.Millisecond
	room := postCreate(t, h, r, "lobby")
	postJoinAs(t, r, room.RoomKey, "ana", "tab-ana")
	ready := map[string]any{"action": "toggleReady", "data": map[string]any{}, "player": "ana"}
	if w := request(r, http.MethodPost, "/api/ek/rooms/"+room.RoomKey+"/state", seatTokenOf(room.RoomKey, "ana"), ready); w.Code != http.StatusOK {
		t.Fatalf("ready: got %d; body %s", w.Code, w.Body)
	}

	dropSocket(h, room.RoomKey, "ana")
	if w := postJoinAs(t, r, room.RoomKey, "ana", "tab-ana"); w.Code != http.StatusOK {
		t.Fatalf("rejoin: got %d; body %s", w.Code, w.Body)
	}

	time.Sleep(4 * h.lobbyGrace)
	if !seated(t, h, room.ID, "ana") {
		t.Fatal("the reload lost ana her seat")
	}
	if !playerSeat(t, h, room.ID, "ana").Ready {
		t.Fatal("the reload lost ana her ready flag")
	}
}

func TestReapForfeitsPlayerGoneTooLong(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo", "cy"} {
		postJoin(t, r, room.RoomKey, name)
	}
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       "bo",
		TurnOrder:  []string{"ana", "bo", "cy"},
		Players:    livePlayers("ana", "bo", "cy"),
		CancelFunc: make(chan struct{}),
	})
	// The others are still at the table; only bo's socket went away. Without
	// them on the hub the room itself would be abandoned by the same sweep,
	// and there would be no game left to forfeit anyone out of.
	connect(t, h, room.RoomKey, "ana")
	connect(t, h, room.RoomKey, "cy")
	dropSocket(h, room.RoomKey, "bo")

	// Too soon: the seat is still being held for them.
	if got := h.ReapOnce(time.Now(), testCfg()); got.Forfeited != 0 {
		t.Fatalf("forfeited a player who just dropped: %+v", got)
	}
	if !seated(t, h, room.ID, "bo") {
		t.Fatal("the seat was released early")
	}

	stats := h.ReapOnce(later(time.Hour), testCfg())
	if stats.Forfeited != 1 {
		t.Fatalf("forfeited = %d, want 1 (%+v)", stats.Forfeited, stats)
	}

	gs := h.getEKState(room.RoomKey)
	if gs.Players["bo"].Alive {
		t.Error("bo is still alive in the game")
	}
	if gs.Turn == "bo" {
		t.Error("the turn is still on the player who was forfeited")
	}
	if seated(t, h, room.ID, "bo") {
		t.Error("the seat was not released")
	}
	// The others play on — this is the point of forfeiting one player rather
	// than abandoning the whole room.
	if gs.Phase != "playing" {
		t.Errorf("phase = %q, want playing", gs.Phase)
	}
	after, _ := h.store.GetRoom(room.RoomKey)
	if after.Status != StatusActive {
		t.Errorf("room status = %q, want %q", after.Status, StatusActive)
	}
}

// Coming back inside the window costs nothing.
func TestReapSparesPlayerWhoReconnected(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "in progress")
	for _, name := range []string{"ana", "bo", "cy"} {
		postJoin(t, r, room.RoomKey, name)
	}
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       "ana",
		TurnOrder:  []string{"ana", "bo", "cy"},
		Players:    livePlayers("ana", "bo", "cy"),
		CancelFunc: make(chan struct{}),
	})

	connect(t, h, room.RoomKey, "ana")
	connect(t, h, room.RoomKey, "cy")
	dropSocket(h, room.RoomKey, "bo")
	if playerSeat(t, h, room.ID, "bo").DisconnectedAt == nil {
		t.Fatal("setup: the drop was not stamped")
	}

	// Back through the join endpoint, the way the client rejoins.
	if w := postJoin(t, r, room.RoomKey, "bo"); w.Code != http.StatusOK {
		t.Fatalf("rejoin: %d %s", w.Code, w.Body)
	}
	if playerSeat(t, h, room.ID, "bo").DisconnectedAt != nil {
		t.Fatal("the stamp survived the rejoin")
	}

	if got := h.ReapOnce(later(time.Hour), testCfg()); got.Forfeited != 0 {
		t.Fatalf("forfeited a player who came back: %+v", got)
	}
	if gs := h.getEKState(room.RoomKey); !gs.Players["bo"].Alive {
		t.Error("bo was taken out of the game despite reconnecting")
	}
}

// A forfeit that leaves one player standing ends the game properly, rather
// than leaving a room running with nobody able to act.
func TestReapForfeitEndingTheGameClosesTheRoom(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "duel")
	for _, name := range []string{"ana", "bo"} {
		postJoin(t, r, room.RoomKey, name)
	}
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       "bo",
		TurnOrder:  []string{"ana", "bo"},
		Players:    livePlayers("ana", "bo"),
		CancelFunc: make(chan struct{}),
	})
	connect(t, h, room.RoomKey, "ana")
	dropSocket(h, room.RoomKey, "bo")

	if got := h.ReapOnce(later(time.Hour), testCfg()); got.Forfeited != 1 {
		t.Fatalf("forfeited = %d, want 1", got.Forfeited)
	}

	gs := h.getEKState(room.RoomKey)
	if gs.Phase != "ended" {
		t.Fatalf("phase = %q, want ended", gs.Phase)
	}
	if gs.Winner == nil || *gs.Winner != "ana" {
		t.Fatalf("winner = %v, want ana", gs.Winner)
	}
	after, _ := h.store.GetRoom(room.RoomKey)
	if after.Status != StatusEnded {
		t.Fatalf("room status = %q, want %q", after.Status, StatusEnded)
	}
}
