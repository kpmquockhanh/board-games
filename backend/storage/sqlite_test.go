package storage

import (
	"testing"
	"time"

	"ping/models"
)

func TestCascadeNowFires(t *testing.T) {
	dir := t.TempDir()
	s, err := NewSQLite(dir + "/v.db")
	if err != nil {
		t.Fatal(err)
	}

	var fk int
	if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	r, _ := s.CreateRoom("ek", 6, "x")
	s.AddPlayer(r.ID, "a", "red", "", "")
	s.AddTimelineEvent(r.ID, "join", "a", "")
	if err := s.DeleteRoom(r.RoomKey); err != nil {
		t.Fatal(err)
	}

	var np, nt int
	s.db.QueryRow("SELECT COUNT(*) FROM room_players").Scan(&np)
	s.db.QueryRow("SELECT COUNT(*) FROM timelines").Scan(&nt)
	if np != 0 || nt != 0 {
		t.Fatalf("orphans remain: room_players=%d timelines=%d", np, nt)
	}
	s.Close()
}

func TestMigrateSweepsPreExistingOrphans(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/legacy.db"

	// Simulate the old behaviour: cascade off, so a room delete orphans rows.
	s, err := NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	r, _ := s.CreateRoom("ek", 6, "x")
	s.AddPlayer(r.ID, "a", "red", "", "")
	s.AddTimelineEvent(r.ID, "join", "a", "")
	if _, err := s.db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		t.Fatal(err)
	}
	s.DeleteRoom(r.RoomKey)
	var np int
	s.db.QueryRow("SELECT COUNT(*) FROM room_players").Scan(&np)
	if np != 1 {
		t.Fatalf("setup failed: wanted 1 orphan, got %d", np)
	}
	s.Close()

	// Reopening runs migrate(), which should sweep them.
	s2, err := NewSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	var np2, nt2 int
	s2.db.QueryRow("SELECT COUNT(*) FROM room_players").Scan(&np2)
	s2.db.QueryRow("SELECT COUNT(*) FROM timelines").Scan(&nt2)
	if np2 != 0 || nt2 != 0 {
		t.Fatalf("migrate left orphans: room_players=%d timelines=%d", np2, nt2)
	}
}

// TouchRoom is what keeps a live room away from the reaper, so it has to move
// a room back out of the idle set.
func TestTouchRoomClearsIdleStatus(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/touch.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	room, err := s.CreateRoom("ek", 6, "table")
	if err != nil {
		t.Fatal(err)
	}

	// Two hours of silence.
	stale := sqliteTime(time.Now().Add(-2 * time.Hour))
	if _, err := s.db.Exec("UPDATE rooms SET last_activity_at = ? WHERE id = ?", stale, room.ID); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().Add(-time.Hour)
	idle, err := s.ListIdleRooms([]string{"active"}, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(idle) != 1 {
		t.Fatalf("idle rooms = %d, want 1", len(idle))
	}
	if got := idle[0].LastActivityAt.UTC().Format(sqliteLayout); got != stale {
		t.Fatalf("LastActivityAt = %q, want %q", got, stale)
	}

	if err := s.TouchRoom(room.ID); err != nil {
		t.Fatal(err)
	}

	idle, err = s.ListIdleRooms([]string{"active"}, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(idle) != 0 {
		t.Fatalf("room still idle after a touch: %+v", idle)
	}
}

// A status the reaper does not ask for must never come back.
func TestListIdleRoomsFiltersByStatus(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/status.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	active, _ := s.CreateRoom("ek", 6, "active")
	ended, _ := s.CreateRoom("ek", 6, "ended")
	s.UpdateRoomStatus(ended.ID, "ended")

	cutoff := time.Now().Add(time.Hour)
	idle, err := s.ListIdleRooms([]string{"ended"}, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(idle) != 1 || idle[0].RoomKey != ended.RoomKey {
		t.Fatalf("got %+v, want only the ended room %s", idle, ended.RoomKey)
	}

	if got, _ := s.ListIdleRooms(nil, cutoff); len(got) != 0 {
		t.Fatalf("no statuses should match nothing, got %d", len(got))
	}
	_ = active
}

func TestDisconnectStampLifecycle(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/dc.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	room, _ := s.CreateRoom("ek", 6, "table")
	s.AddPlayer(room.ID, "ana", "red", "", "")

	seat := func() models.RoomPlayer {
		t.Helper()
		players, err := s.GetRoomPlayers(room.ID)
		if err != nil || len(players) != 1 {
			t.Fatalf("players = %v, err = %v", players, err)
		}
		return players[0]
	}

	if got := seat(); got.DisconnectedAt != nil {
		t.Fatalf("a freshly joined player is stamped as dropped: %v", got.DisconnectedAt)
	}

	if err := s.MarkPlayerDisconnected(room.ID, "ana"); err != nil {
		t.Fatal(err)
	}
	dropped := seat()
	if dropped.DisconnectedAt == nil {
		t.Fatal("the drop was not recorded")
	}
	// The seat is held, not released: that is the whole point of the stamp.
	if dropped.LeftAt != nil {
		t.Fatal("the seat was released instead of being held")
	}
	if n, _ := s.GetActivePlayerCount(room.ID); n != 1 {
		t.Fatalf("active players = %d, want 1 — the seat is still theirs", n)
	}

	cleared, err := s.MarkPlayerConnected(room.ID, "ana")
	if err != nil {
		t.Fatal(err)
	}
	if !cleared {
		t.Fatal("coming back was not reported, so the table is never told they are online again")
	}
	if got := seat(); got.DisconnectedAt != nil {
		t.Fatalf("the stamp survived a reconnect: %v", got.DisconnectedAt)
	}
}

// Rejoining through AddPlayer clears the stamp in SQL, which is the path a
// brand-new join takes.
func TestAddPlayerClearsDisconnectStamp(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/dc2.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	room, _ := s.CreateRoom("ek", 6, "table")
	s.AddPlayer(room.ID, "ana", "red", "", "")
	s.MarkPlayerDisconnected(room.ID, "ana")

	if err := s.AddPlayer(room.ID, "ana", "blue", "", ""); err != nil {
		t.Fatal(err)
	}
	players, _ := s.GetRoomPlayers(room.ID)
	if players[0].DisconnectedAt != nil {
		t.Fatalf("stamp survived a rejoin: %v", players[0].DisconnectedAt)
	}
}

func TestListStalePlayersOnlyReturnsHeldSeatsPastTheCutoff(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/stale.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	room, _ := s.CreateRoom("ek", 6, "table")
	for _, name := range []string{"ana", "bo", "cy", "di"} {
		s.AddPlayer(room.ID, name, "red", "", "")
	}

	// ana dropped two hours ago; bo dropped just now; cy never dropped;
	// di dropped but then left for good.
	stale := sqliteTime(time.Now().Add(-2 * time.Hour))
	s.MarkPlayerDisconnected(room.ID, "ana")
	if _, err := s.db.Exec(
		"UPDATE room_players SET disconnected_at = ? WHERE room_id = ? AND player_name = 'ana'",
		stale, room.ID,
	); err != nil {
		t.Fatal(err)
	}
	s.MarkPlayerDisconnected(room.ID, "bo")
	s.MarkPlayerDisconnected(room.ID, "di")
	if _, err := s.db.Exec(
		"UPDATE room_players SET disconnected_at = ?, left_at = CURRENT_TIMESTAMP WHERE room_id = ? AND player_name = 'di'",
		stale, room.ID,
	); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListStalePlayers(time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("stale players = %+v, want only ana", got)
	}
	if got[0].PlayerName != "ana" || got[0].RoomKey != room.RoomKey || got[0].Game != "ek" {
		t.Fatalf("got %+v, want ana in %s", got[0], room.RoomKey)
	}

	// A room that is no longer active has no seats worth reclaiming.
	s.UpdateRoomStatus(room.ID, "ended")
	if got, _ := s.ListStalePlayers(time.Now().Add(-time.Hour)); len(got) != 0 {
		t.Fatalf("returned seats from a room that is not active: %+v", got)
	}
}
