package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"ping/game"
	"ping/models"
)

// restart writes gs to the room's snapshot the way the running server would
// have, then hands back a handler that has never seen the room in memory —
// which is exactly the position the process is in after being restarted.
func restart(t *testing.T, h *Handler, roomID int64, gs *game.EKGameState, turnTimer int) *Handler {
	t.Helper()
	settings := defaultEKSettings()
	settings.TurnTimer = turnTimer
	stateJSON, err := json.Marshal(models.EKRoomState{RoomSettings: settings, GameState: gs})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.store.UpdateSnapshot(roomID, "", string(stateJSON)); err != nil {
		t.Fatal(err)
	}
	return NewHandler(h.store, h.hub)
}

// rehydrate loads a room the way a request would: under the room lock.
func rehydrate(t *testing.T, h *Handler, roomKey string, roomID int64) *game.EKGameState {
	t.Helper()
	mu := h.getEKMutex(roomKey)
	mu.Lock()
	defer mu.Unlock()
	gs := h.loadEKState(roomKey, roomID)
	if gs == nil {
		t.Fatal("the game did not come back from the snapshot")
	}
	return gs
}

// waitFor polls a condition under the room lock, since the timer callbacks
// mutate the state while holding it.
func waitFor(t *testing.T, h *Handler, roomKey string, d time.Duration, cond func() bool) bool {
	t.Helper()
	mu := h.getEKMutex(roomKey)
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		mu.Lock()
		ok := cond()
		mu.Unlock()
		if ok {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func midTurnGame() *game.EKGameState {
	stale := time.Now().Add(-time.Hour)
	return &game.EKGameState{
		Phase:     "playing",
		Turn:      "ana",
		TurnOrder: []string{"ana", "bo"},
		Players: map[string]*game.PlayerState{
			"ana": {Color: "#f00", Alive: true, Hand: []string{"df_001"}},
			"bo":  {Color: "#00f", Alive: true, Hand: []string{"df_002"}},
		},
		Deck:    []string{"sk_001", "sk_002", "sk_003"},
		Discard: []string{},
		// Written by the process that died: a deadline nothing is counting to.
		TurnEndsAt: &stale,
	}
}

// The turn timer is the clock that keeps a game moving. Losing it to a restart
// leaves the table waiting on a player forever.
func TestRehydratedGameRestartsItsTurnTimer(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "restarted")
	for _, name := range []string{"ana", "bo"} {
		postJoin(t, r, room.RoomKey, name)
	}

	next := restart(t, h, room.ID, midTurnGame(), 1)
	gs := rehydrate(t, next, room.RoomKey, room.ID)

	if gs.TurnTimer == nil {
		t.Fatal("no turn timer was armed, so nothing will ever move the turn on")
	}
	if gs.TurnEndsAt == nil || !gs.TurnEndsAt.After(time.Now()) {
		t.Fatalf("turn deadline = %v, want one in the future — the client counts down to it", gs.TurnEndsAt)
	}

	// The real proof: the clock runs out and draws for the player.
	if !waitFor(t, next, room.RoomKey, 3*time.Second, func() bool { return gs.Turn == "bo" }) {
		t.Fatalf("the turn never moved off ana; turn=%q hand=%v", gs.Turn, gs.Players["ana"].Hand)
	}
}

// A prompt nobody answered is the other way a restart can wedge a table: no
// action is allowed while one is pending.
func TestRehydratedGameRestartsThePendingPromptClock(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "mid prompt")
	for _, name := range []string{"ana", "bo"} {
		postJoin(t, r, room.RoomKey, name)
	}

	stored := midTurnGame()
	stored.PendingDefuse = &game.DefuseState{PlayerID: "ana", ExplosiveID: "ex_001"}

	next := restart(t, h, room.ID, stored, 0)
	gs := rehydrate(t, next, room.RoomKey, room.ID)

	if gs.PendingTimer == nil {
		t.Fatal("no clock on the unanswered prompt, so the table waits on ana forever")
	}
}

// A Nope window is a promise that the card underneath it resolves shortly.
func TestRehydratedGameRestartsTheNopeWindow(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "mid nope")
	for _, name := range []string{"ana", "bo"} {
		postJoin(t, r, room.RoomKey, name)
	}

	stored := midTurnGame()
	stored.NopeWindow = &game.NopeWindowState{PlayerID: "ana", CardID: "sk_001", Category: "skip"}

	next := restart(t, h, room.ID, stored, 0)
	gs := rehydrate(t, next, room.RoomKey, room.ID)

	if gs.NopeTimer == nil {
		t.Fatal("the nope window has no clock, so the card it covers never resolves")
	}
	if gs.NopeWindow.ExpiredAt == nil || !gs.NopeWindow.ExpiredAt.After(time.Now()) {
		t.Fatalf("nope deadline = %v, want one in the future", gs.NopeWindow.ExpiredAt)
	}
}

// Nothing is owed a clock once the game is over, and the stale deadline the
// snapshot carries must not be handed to a client as a live countdown.
func TestRehydratedFinishedGameGetsNoClocks(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "already over")
	postJoin(t, r, room.RoomKey, "ana")

	stored := midTurnGame()
	stored.Phase = "ended"
	winner := "ana"
	stored.Winner = &winner

	next := restart(t, h, room.ID, stored, 30)
	gs := rehydrate(t, next, room.RoomKey, room.ID)

	if gs.TurnTimer != nil || gs.PendingTimer != nil || gs.NopeTimer != nil {
		t.Fatal("a finished game was given clocks")
	}
	if gs.TurnEndsAt != nil {
		t.Fatalf("turn deadline = %v, want none: the game is over", gs.TurnEndsAt)
	}
}

// Only the load from storage arms anything. A game already in memory is being
// played, and restarting its turn timer on every read would hand the current
// player an endless turn.
func TestRehydratingDoesNotRearmTwiceOverAnInMemoryGame(t *testing.T) {
	h, r := newTestHandler(t)
	room := postCreate(t, h, r, "already loaded")
	postJoin(t, r, room.RoomKey, "ana")

	next := restart(t, h, room.ID, midTurnGame(), 30)
	first := rehydrate(t, next, room.RoomKey, room.ID)
	armed := first.TurnEndsAt

	second := rehydrate(t, next, room.RoomKey, room.ID)
	if second != first {
		t.Fatal("the second load built a second copy of the same game")
	}
	if second.TurnEndsAt != armed {
		t.Fatal("the clock was restarted for a game that was already in memory")
	}
}
