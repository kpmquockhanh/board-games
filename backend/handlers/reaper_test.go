package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"ping/game"
	"ping/models"
	"ping/ws"
)

func testCfg() ReaperConfig {
	return ReaperConfig{
		Enabled:      true,
		Interval:     time.Minute,
		AbandonAfter: 10 * time.Minute,
		DeleteAfter:  24 * time.Hour,
		// Explicit, so the "too soon" cases rest on the configured window
		// rather than on SQLite truncating timestamps to whole seconds.
		DropPlayerAfter: 5 * time.Minute,
		TimelineKeep:    3,
	}
}

// later is how these tests make time pass: ReapOnce takes the clock as an
// argument, so a sweep can be run from the future against real timestamps.
func later(d time.Duration) time.Time { return time.Now().Add(d) }

// connect registers a client on the hub the way a real socket would, so
// RoomCount sees the room as occupied. Nothing writes to the nil conn: the
// reaper only ever counts.
func connect(t *testing.T, h *Handler, roomKey, player string) {
	t.Helper()
	h.hub.Register(&ws.Client{
		Room:   roomKey,
		Player: player,
		Send:   make(chan []byte, 256),
		Hub:    h.hub,
	})
	deadline := time.Now().Add(2 * time.Second)
	for h.hub.RoomCount(roomKey) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("client never registered on the hub")
		}
		time.Sleep(time.Millisecond)
	}
}

// countEvents splits a room's timeline into ordinary events and snapshots.
func countEvents(t *testing.T, h *Handler, roomID int64) (events, snapshots int) {
	t.Helper()
	rows, err := h.store.GetTimeline(roomID, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range rows {
		if e.EventType == "snapshot" {
			snapshots++
		} else {
			events++
		}
	}
	return events, snapshots
}

func TestReapAbandonsIdleRoomAndReleasesSeats(t *testing.T) {
	h, r := newTestHandler(t)

	room := postCreate(t, h, r, "quiet table")
	for _, name := range []string{"ana", "bo"} {
		if w := postJoin(t, r, room.RoomKey, name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}

	// Still fresh: the sweep must leave it alone.
	if got := h.ReapOnce(time.Now(), testCfg()); got.Abandoned != 0 {
		t.Fatalf("reaped a fresh room: %+v", got)
	}

	stats := h.ReapOnce(later(30*time.Minute), testCfg())
	if stats.Abandoned != 1 {
		t.Fatalf("abandoned = %d, want 1 (%+v)", stats.Abandoned, stats)
	}

	after, _ := h.store.GetRoom(room.RoomKey)
	if after.Status != StatusAbandoned {
		t.Fatalf("status = %q, want %q", after.Status, StatusAbandoned)
	}
	// The seats a mid-game disconnect keeps must be given back.
	if n, _ := h.store.GetActivePlayerCount(room.ID); n != 0 {
		t.Fatalf("active players = %d, want 0", n)
	}
	// And it must drop out of the lobby list.
	listed, _ := h.store.ListActiveRooms("ek")
	for _, item := range listed {
		if item.RoomKey == room.RoomKey {
			t.Fatal("abandoned room still advertised in the lobby list")
		}
	}
}

func TestReapSkipsRoomWithConnectedPlayers(t *testing.T) {
	h, r := newTestHandler(t)

	room := postCreate(t, h, r, "lobby table")
	postJoin(t, r, room.RoomKey, "ana")

	// A table idling in the lobby sends no actions but holds a socket open.
	connect(t, h, room.RoomKey, "ana")

	stats := h.ReapOnce(later(30*time.Minute), testCfg())
	if stats.Abandoned != 0 || stats.Skipped != 1 {
		t.Fatalf("reaped a room with a live socket: %+v", stats)
	}
	after, _ := h.store.GetRoom(room.RoomKey)
	if after.Status != StatusActive {
		t.Fatalf("status = %q, want %q", after.Status, StatusActive)
	}
}

func TestReapDeletesExpiredRoomsOnly(t *testing.T) {
	h, r := newTestHandler(t)

	recent := postCreate(t, h, r, "just finished")
	h.store.UpdateRoomStatus(recent.ID, StatusEnded)

	old := postCreate(t, h, r, "long over")
	h.store.AddPlayer(old.ID, "ana", "red")
	h.store.AddTimelineEvent(old.ID, "join", "ana", "")
	h.store.UpdateRoomStatus(old.ID, StatusEnded)

	// An hour on, neither is old enough to delete.
	if got := h.ReapOnce(later(time.Hour), testCfg()); got.Deleted != 0 {
		t.Fatalf("deleted a recently ended room: %+v", got)
	}

	// Two days on, both are — this asserts the grace period exists at all.
	stats := h.ReapOnce(later(48*time.Hour), testCfg())
	if stats.Deleted != 2 {
		t.Fatalf("deleted = %d, want 2 (%+v)", stats.Deleted, stats)
	}
	if got, _ := h.store.GetRoom(old.RoomKey); got != nil {
		t.Fatal("expired room survived")
	}
	// The cascade should have taken the child rows with it.
	players, _ := h.store.GetRoomPlayers(old.ID)
	events, snapshots := countEvents(t, h, old.ID)
	if len(players) != 0 || events != 0 || snapshots != 0 {
		t.Fatalf("orphans left behind: players=%d events=%d snapshots=%d",
			len(players), events, snapshots)
	}
}

// A room reaped as abandoned should later be deleted like an ended one.
func TestReapDeletesAbandonedRoomsAfterGracePeriod(t *testing.T) {
	h, r := newTestHandler(t)

	room := postCreate(t, h, r, "gone")
	if got := h.ReapOnce(later(30*time.Minute), testCfg()); got.Abandoned != 1 {
		t.Fatalf("setup: %+v", got)
	}
	if got := h.ReapOnce(later(48*time.Hour), testCfg()); got.Deleted != 1 {
		t.Fatalf("deleted = %d, want 1", got.Deleted)
	}
	if got, _ := h.store.GetRoom(room.RoomKey); got != nil {
		t.Fatal("abandoned room was never deleted")
	}
}

func TestReapPrunesTimelineButKeepsSnapshot(t *testing.T) {
	h, r := newTestHandler(t)

	room := postCreate(t, h, r, "chatty")
	for i := 0; i < 10; i++ {
		h.store.AddTimelineEvent(room.ID, "drawCard", "ana", "")
	}

	stats := h.ReapOnce(time.Now(), testCfg())
	if stats.EventsPruned != 7 {
		t.Fatalf("pruned = %d, want 7 (%+v)", stats.EventsPruned, stats)
	}

	events, snapshots := countEvents(t, h, room.ID)
	if events != 3 {
		t.Fatalf("kept %d events, want 3", events)
	}
	// CreateRoom writes the settings snapshot; losing it would lose the room.
	if snapshots != 1 {
		t.Fatalf("snapshots = %d, want 1 — the game state must never be pruned", snapshots)
	}
	if _, ok, _ := h.store.GetLatestSnapshot(room.ID); !ok {
		t.Fatal("room state is no longer readable after a prune")
	}
}

// The reaper's whole purpose: a table that dropped out mid-game must free up.
func TestReapUnwedgesAbandonedGame(t *testing.T) {
	h, r := newTestHandler(t)

	room := postCreate(t, h, r, "wedged")
	settings := defaultEKSettings()
	settings.MaxPlayers = 2
	blob, _ := json.Marshal(models.EKRoomState{RoomSettings: settings})
	h.store.UpdateSnapshot(room.ID, "", string(blob))

	for _, name := range []string{"ana", "bo"} {
		if w := postJoin(t, r, room.RoomKey, name); w.Code != http.StatusOK {
			t.Fatalf("seating %s: %d", name, w.Code)
		}
	}
	// Both dropped mid-game, so HandleDisconnect kept their seats and the
	// live game state stayed in memory.
	h.setEKState(room.RoomKey, &game.EKGameState{
		Phase:      "playing",
		Turn:       "ana",
		TurnOrder:  []string{"ana", "bo"},
		CancelFunc: make(chan struct{}),
	})

	// Until the sweep, the room is full and closed to newcomers.
	if w := postJoin(t, r, room.RoomKey, "cy"); w.Code == http.StatusOK {
		t.Fatal("a stranger joined a full room")
	}

	if got := h.ReapOnce(later(30*time.Minute), testCfg()); got.Abandoned != 1 {
		t.Fatalf("abandoned = %d, want 1", got.Abandoned)
	}

	if n, _ := h.store.GetActivePlayerCount(room.ID); n != 0 {
		t.Fatalf("seats still held: %d", n)
	}
	if h.getEKState(room.RoomKey) != nil {
		t.Fatal("in-memory game state was not released")
	}
}

// End-to-end proof that the activity bumps are wired up: a room someone acted
// in must survive a sweep that would otherwise have reaped it.
//
// CURRENT_TIMESTAMP truncates to whole seconds, and so does the cutoff it is
// compared against, so the create and the join have to land in different
// seconds for the difference between them to be visible at all. The test
// aligns to a second boundary first and then spaces them two seconds apart.
func TestReapSparesRoomAfterRealActivity(t *testing.T) {
	h, r := newTestHandler(t)

	time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second + 50*time.Millisecond)))
	created := time.Now()

	room := postCreate(t, h, r, "revived")
	time.Sleep(2 * time.Second)
	if w := postJoin(t, r, room.RoomKey, "ana"); w.Code != http.StatusOK {
		t.Fatalf("join: %d %s", w.Code, w.Body)
	}

	// A cutoff one second after the room was created: past the creation,
	// before the join. Only the join's touch can save the room here.
	cfg := testCfg()
	now := created.Add(time.Second).Add(cfg.AbandonAfter)

	if got := h.ReapOnce(now, cfg); got.Abandoned != 0 {
		t.Fatalf("reaped a room that was just joined: %+v", got)
	}
	after, _ := h.store.GetRoom(room.RoomKey)
	if after.Status != StatusActive {
		t.Fatalf("status = %q, want %q", after.Status, StatusActive)
	}

	// The same sweep from far enough ahead still reaps it, which shows the
	// assertion above held because of the touch and not because the cutoff
	// was never going to match.
	if got := h.ReapOnce(later(30*time.Minute), cfg); got.Abandoned != 1 {
		t.Fatalf("abandoned = %d, want 1", got.Abandoned)
	}
}
