package handlers

import (
	"context"
	"log"
	"time"

	"ping/game"
	"ping/models"
)

// Room statuses the reaper moves between. A room is born "active"; a finished
// game is "ended"; the reaper writes "abandoned" for a room everyone walked
// away from. Only "active" rooms are advertised in the lobby list, so both of
// the other two are already invisible — the reaper's job is to stop them
// holding seats and, eventually, to delete them.
const (
	StatusActive    = "active"
	StatusEnded     = "ended"
	StatusAbandoned = "abandoned"
)

// ReaperConfig tunes the sweep. Durations are measured against a room's
// last_activity_at, which every player-driven request bumps.
type ReaperConfig struct {
	// Enabled is false to leave rooms entirely alone, which is sometimes what
	// you want while working on one locally.
	Enabled bool
	// Interval between sweeps.
	Interval time.Duration
	// AbandonAfter is how long an active room with nobody connected may stay
	// active. Long enough to outlast a reload, a tunnel or a flaky phone.
	AbandonAfter time.Duration
	// DeleteAfter is how long an ended or abandoned room is kept before its
	// rows go. Players can still reach a finished game's result screen until
	// then, and it leaves a window for looking at what happened.
	DeleteAfter time.Duration
	// DropPlayerAfter is how long one player's seat is held for them after
	// their socket drops mid-game. Shorter than AbandonAfter on purpose: one
	// player going quiet should cost that player their game, not cost the
	// people still at the table theirs.
	DropPlayerAfter time.Duration
	// TimelineKeep is how many non-snapshot events each room retains.
	TimelineKeep int
	// GuestsAfter is how long a guest account is kept after its browser was
	// last seen. A guest cannot log back in, so once its cookie is gone the
	// row is unreachable; this is the bound on how many of those pile up.
	GuestsAfter time.Duration
}

func DefaultReaperConfig() ReaperConfig {
	return ReaperConfig{
		Enabled:         true,
		Interval:        5 * time.Minute,
		AbandonAfter:    10 * time.Minute,
		DeleteAfter:     24 * time.Hour,
		DropPlayerAfter: 5 * time.Minute,
		TimelineKeep:    200,
		GuestsAfter:     90 * 24 * time.Hour,
	}
}

// ReapStats is what one sweep did, for the log and for tests.
type ReapStats struct {
	Abandoned    int
	Deleted      int
	Forfeited    int
	EventsPruned int64
	GuestsPruned int64
	Skipped      int
}

// RunReaper sweeps on a ticker until ctx is cancelled. It blocks, so callers
// run it in a goroutine.
func (h *Handler) RunReaper(ctx context.Context, cfg ReaperConfig) {
	if !cfg.Enabled {
		log.Printf("[reaper] disabled by %s: rooms and seats will be held indefinitely", EnvReapEnabled)
		return
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Printf("[reaper] started: every %s, forfeit after %s, abandon after %s, delete after %s",
		cfg.Interval, cfg.DropPlayerAfter, cfg.AbandonAfter, cfg.DeleteAfter)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[reaper] stopped")
			return
		case <-ticker.C:
			h.ReapOnce(time.Now(), cfg)
		}
	}
}

// ReapOnce runs a single sweep and reports what it did. Taking `now` as an
// argument is what lets a test drive the clock.
func (h *Handler) ReapOnce(now time.Time, cfg ReaperConfig) ReapStats {
	var stats ReapStats

	// Seats first: reclaiming one can finish a game, which moves the room to
	// "ended" and changes what the phases below are looking at.
	stats.Forfeited = h.forfeitStalePlayers(now.Add(-cfg.DropPlayerAfter))
	stats.Abandoned, stats.Skipped = h.abandonIdleRooms(now.Add(-cfg.AbandonAfter))
	stats.Deleted = h.deleteExpiredRooms(now.Add(-cfg.DeleteAfter))

	if n, err := h.store.PruneTimelines(cfg.TimelineKeep); err != nil {
		log.Printf("[reaper] prune timelines: %v", err)
	} else {
		stats.EventsPruned = n
	}

	if n, err := h.store.DeleteIdleGuests(now.Add(-cfg.GuestsAfter)); err != nil {
		log.Printf("[reaper] prune guests: %v", err)
	} else {
		stats.GuestsPruned = n
	}

	if stats.Abandoned > 0 || stats.Deleted > 0 || stats.Forfeited > 0 || stats.EventsPruned > 0 || stats.GuestsPruned > 0 {
		log.Printf("[reaper] forfeited=%d abandoned=%d deleted=%d events_pruned=%d guests_pruned=%d",
			stats.Forfeited, stats.Abandoned, stats.Deleted, stats.EventsPruned, stats.GuestsPruned)
	}
	return stats
}

// forfeitStalePlayers reclaims seats held for players who dropped mid-game and
// have not come back. This is the per-player counterpart to abandoning a room:
// one player losing their connection should not cost everyone else their game,
// so the table plays on without them.
func (h *Handler) forfeitStalePlayers(cutoff time.Time) int {
	stale, err := h.store.ListStalePlayers(cutoff)
	if err != nil {
		log.Printf("[reaper] list stale players: %v", err)
		return 0
	}

	forfeited := 0
	for _, sp := range stale {
		// They may have reconnected between the query and now, or on a socket
		// that never cleared the stamp. The hub is the live truth.
		if h.hub.IsPlayerConnected(sp.RoomKey, sp.PlayerName) {
			h.markBack(sp.RoomID, sp.RoomKey, sp.PlayerName)
			continue
		}
		if h.forfeitPlayer(sp) {
			forfeited++
		}
	}
	return forfeited
}

// forfeitPlayer takes one player out of their game and releases their seat.
func (h *Handler) forfeitPlayer(sp models.StalePlayer) bool {
	// Release the seat whatever happens below, so a player who cannot be
	// forfeited from a game — because there is no game — still stops being
	// counted against the room's capacity.
	defer func() {
		if err := h.store.RemovePlayer(sp.RoomID, sp.PlayerName); err != nil {
			log.Printf("[reaper] release seat of %s in %s: %v", sp.PlayerName, sp.RoomKey, err)
		}
	}()

	if sp.Game != "ek" {
		return false
	}

	mu := h.getEKMutex(sp.RoomKey)
	mu.Lock()
	defer mu.Unlock()

	gs := h.loadEKState(sp.RoomKey, sp.RoomID)
	result := game.ForfeitPlayer(gs, sp.PlayerName)
	if result == nil {
		// No game running, or they were already out of it. The deferred seat
		// release is the whole of the work.
		return false
	}

	h.setEKState(sp.RoomKey, result.State)

	gameOver := result.State.Phase == "ended"
	if gameOver {
		game.CancelPendingAction(result.State)
		h.store.UpdateRoomStatus(sp.RoomID, StatusEnded)
	}

	h.hub.BroadcastToRoom(sp.RoomKey, models.WSMessage{
		Type:   "player_left",
		Room:   sp.RoomKey,
		Player: sp.PlayerName,
	})

	// applyEKResult saves the state, re-arms whichever clocks the new state
	// calls for — the turn may have moved to someone else — and publishes it.
	h.applyEKResult(mu, sp.RoomKey, sp.RoomID, result, "", gameOver)

	log.Printf("[reaper] forfeited player=%s room=%s (gone since %s)",
		sp.PlayerName, sp.RoomKey, sp.DisconnectedAt.Format(time.RFC3339))
	return true
}

// abandonIdleRooms retires active rooms that have gone quiet and have nobody
// connected. This is what unwedges a game everyone dropped out of: the seats a
// mid-game disconnect deliberately kept (see HandleDisconnect) are only held
// for as long as someone might plausibly come back for them.
func (h *Handler) abandonIdleRooms(cutoff time.Time) (abandoned, skipped int) {
	rooms, err := h.store.ListIdleRooms([]string{StatusActive}, cutoff)
	if err != nil {
		log.Printf("[reaper] list idle rooms: %v", err)
		return 0, 0
	}

	for _, room := range rooms {
		// The hub is the live truth: a room can be quiet in the database
		// while players sit in the lobby with a socket open.
		if h.hub.RoomCount(room.RoomKey) > 0 {
			skipped++
			continue
		}
		h.abandonRoom(room)
		abandoned++
	}
	return abandoned, skipped
}

func (h *Handler) abandonRoom(room models.Room) {
	// Under the room lock, so a timer callback already in flight finishes
	// first. Those callbacks re-arm the clocks, and one landing after the
	// state was dropped would put the room straight back in the map.
	mu := h.getEKMutex(room.RoomKey)
	mu.Lock()
	if gs := h.getEKState(room.RoomKey); gs != nil {
		game.CancelPendingAction(gs)
	}
	// removeEKState drops this very mutex from the map while it is held.
	// Unlocking it afterwards is still correct, and the next caller simply
	// gets a fresh one — which is what a torn-down room should give them.
	h.removeEKState(room.RoomKey)
	mu.Unlock()
	h.forgetPresence(room.RoomKey)

	if err := h.store.RemoveAllPlayers(room.ID); err != nil {
		log.Printf("[reaper] release seats in %s: %v", room.RoomKey, err)
	}
	if err := h.store.UpdateRoomStatus(room.ID, StatusAbandoned); err != nil {
		log.Printf("[reaper] mark %s abandoned: %v", room.RoomKey, err)
		return
	}
	log.Printf("[reaper] abandoned room=%s (%s) idle since %s",
		room.RoomKey, room.Game, room.LastActivityAt.Format(time.RFC3339))
}

// deleteExpiredRooms removes rooms that have been finished or abandoned long
// enough that nobody is coming back. The cascade does the child rows.
func (h *Handler) deleteExpiredRooms(cutoff time.Time) int {
	rooms, err := h.store.ListIdleRooms([]string{StatusEnded, StatusAbandoned}, cutoff)
	if err != nil {
		log.Printf("[reaper] list expired rooms: %v", err)
		return 0
	}

	deleted := 0
	for _, room := range rooms {
		// A rematch puts an ended room back to active, so anyone connected to
		// one at this point is on a result screen. Leave them their game.
		if h.hub.RoomCount(room.RoomKey) > 0 {
			continue
		}
		h.removeEKState(room.RoomKey)
		h.forgetPresence(room.RoomKey)
		if err := h.store.DeleteRoom(room.RoomKey); err != nil {
			log.Printf("[reaper] delete %s: %v", room.RoomKey, err)
			continue
		}
		log.Printf("[reaper] deleted room=%s (%s, %s)", room.RoomKey, room.Game, room.Status)
		deleted++
	}
	return deleted
}
