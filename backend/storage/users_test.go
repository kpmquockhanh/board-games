package storage

import (
	"testing"
	"time"

	"ping/models"
)

func TestAGuestIsFoundByItsSessionOnly(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/u.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	u, err := s.CreateGuest("hash-a")
	if err != nil || u == nil || !u.Guest {
		t.Fatalf("create guest: %+v, %v", u, err)
	}
	if got, _ := s.UserBySession("hash-a"); got == nil || got.ID != u.ID {
		t.Fatalf("by session: %+v, want %s", got, u.ID)
	}
	for _, hash := range []string{"", "hash-b"} {
		if got, err := s.UserBySession(hash); got != nil || err != nil {
			t.Fatalf("session %q found %+v, %v", hash, got, err)
		}
	}
}

// Pruning a guest drops its sessions and unlinks its seats; the seats stay.
func TestPruningAGuestKeepsTheirSeats(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/u.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	old, _ := s.CreateGuest("old")
	recent, _ := s.CreateGuest("recent")
	s.db.Exec("UPDATE users SET last_seen_at = ? WHERE id = ?", sqliteTime(time.Now().Add(-48*time.Hour)), old.ID)
	room, _ := s.CreateRoom("ek", 6, "x")
	s.AddPlayer(room.ID, "ana", "red", "", old.ID)

	n, err := s.DeleteIdleGuests(time.Now().Add(-24 * time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("pruned %d, %v; want 1", n, err)
	}
	if u, _ := s.UserBySession("old"); u != nil {
		t.Fatal("the pruned guest's session still signs in")
	}
	if u, _ := s.UserBySession("recent"); u == nil || u.ID != recent.ID {
		t.Fatal("a recently seen guest was pruned")
	}
	players, _ := s.GetRoomPlayers(room.ID)
	if len(players) != 1 || players[0].UserID != nil {
		t.Fatalf("seat after pruning: %+v", players)
	}
}

// Pruning is for guests nobody will come back to. An account someone signed in
// to stays however long they are away.
func TestPruningNeverTakesASignedInAccount(t *testing.T) {
	s, err := NewSQLite(t.TempDir() + "/u.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	guest, _ := s.CreateGuest("old")
	u, err := s.SignIn("new", "old", models.Identity{Provider: "google", Subject: "1"})
	if err != nil || u.ID != guest.ID || u.Guest {
		t.Fatalf("sign in: %+v, %v", u, err)
	}
	s.db.Exec("UPDATE users SET last_seen_at = ?", sqliteTime(time.Now().Add(-365*24*time.Hour)))
	if n, err := s.DeleteIdleGuests(time.Now()); err != nil || n != 0 {
		t.Fatalf("pruned %d, %v", n, err)
	}
	if got, _ := s.UserBySession("new"); got == nil || got.ID != u.ID {
		t.Fatal("the signed-in account is gone")
	}
}
