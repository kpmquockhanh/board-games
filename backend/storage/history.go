package storage

import (
	"database/sql"
	"fmt"

	"ping/models"
)

// migrateHistory adds the record of finished games. It outlives the room: a
// room's rows are deleted a while after it ends (see the reaper), so a match
// keeps the room's key and name as text rather than a reference to it.
//
// A player is linked to the account that sat in their seat when the game
// ended. The link goes with the account: pruning a guest leaves the row and
// clears the link, and folding a guest into an account on login moves it.
func migrateHistory(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS matches (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			game       TEXT NOT NULL,
			room_key   TEXT NOT NULL,
			room_name  TEXT NOT NULL DEFAULT '',
			started_at DATETIME NOT NULL,
			ended_at   DATETIME NOT NULL,
			UNIQUE(room_key, started_at)
		);

		CREATE TABLE IF NOT EXISTS match_players (
			match_id    INTEGER NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
			player_name TEXT NOT NULL,
			color       TEXT NOT NULL DEFAULT '',
			won         INTEGER NOT NULL DEFAULT 0,
			user_id     TEXT REFERENCES users(id) ON DELETE SET NULL,
			PRIMARY KEY (match_id, player_name)
		);
		CREATE INDEX IF NOT EXISTS idx_match_players_user ON match_players(user_id) WHERE user_id IS NOT NULL;
	`); err != nil {
		return fmt.Errorf("create matches: %w", err)
	}
	return nil
}

// RecordMatch writes a finished game down once. A game is known by its room
// and the moment it started, so recording the same ending twice is harmless.
func (s *SQLiteStorage) RecordMatch(m models.MatchRecord) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		"INSERT OR IGNORE INTO matches (game, room_key, room_name, started_at, ended_at) VALUES (?, ?, ?, ?, ?)",
		m.Game, m.RoomKey, m.RoomName, sqliteTime(m.StartedAt), sqliteTime(m.EndedAt),
	)
	if err != nil {
		return fmt.Errorf("record match: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil || n == 0 {
		return err
	}
	matchID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	for _, p := range m.Players {
		// Whoever's seat it is now, including a player forfeited out of the
		// game: their seat row is kept, only marked as left.
		if _, err := tx.Exec(`
			INSERT INTO match_players (match_id, player_name, color, won, user_id)
			VALUES (?, ?, ?, ?, (SELECT user_id FROM room_players WHERE room_id = ? AND player_name = ?))`,
			matchID, p.Name, p.Color, p.Won, m.RoomID, p.Name,
		); err != nil {
			return fmt.Errorf("record match player: %w", err)
		}
	}
	return tx.Commit()
}

// UserMatches is an account's finished games, newest first.
func (s *SQLiteStorage) UserMatches(userID string, limit int) ([]models.Match, error) {
	rows, err := s.db.Query(`
		SELECT id, game, room_name, started_at, ended_at FROM matches
		WHERE id IN (SELECT match_id FROM match_players WHERE user_id = ?)
		ORDER BY ended_at DESC, id DESC LIMIT ?`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	matches := []models.Match{}
	byID := map[int64]int{}
	for rows.Next() {
		var m models.Match
		var started, ended scanTime
		if err := rows.Scan(&m.ID, &m.Game, &m.RoomName, &started, &ended); err != nil {
			rows.Close()
			return nil, err
		}
		m.StartedAt, m.EndedAt = started.Time, ended.Time
		m.Players = []models.MatchPlayer{}
		byID[m.ID] = len(matches)
		matches = append(matches, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(matches) == 0 {
		return matches, err
	}

	players, err := s.db.Query(`
		SELECT match_id, player_name, color, won, COALESCE(user_id = ?, 0) FROM match_players
		WHERE match_id IN (SELECT match_id FROM match_players WHERE user_id = ?)
		ORDER BY match_id, won DESC, player_name`, userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer players.Close()
	for players.Next() {
		var id int64
		var p models.MatchPlayer
		if err := players.Scan(&id, &p.Name, &p.Color, &p.Won, &p.You); err != nil {
			return nil, err
		}
		if i, ok := byID[id]; ok {
			matches[i].Players = append(matches[i].Players, p)
		}
	}
	return matches, players.Err()
}

// UserSeats is every seat an account is sitting in at a table still being
// played, newest first.
func (s *SQLiteStorage) UserSeats(userID string) ([]models.Seat, error) {
	rows, err := s.db.Query(`
		SELECT r.game, r.room_key, r.name, rp.player_name, rp.color, rp.disconnected_at IS NOT NULL
		FROM room_players rp JOIN rooms r ON r.id = rp.room_id
		WHERE rp.user_id = ? AND rp.left_at IS NULL AND r.status = 'active'
		ORDER BY rp.joined_at DESC, rp.id DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seats := []models.Seat{}
	for rows.Next() {
		var st models.Seat
		if err := rows.Scan(&st.Game, &st.RoomKey, &st.RoomName, &st.PlayerName, &st.Color, &st.Held); err != nil {
			return nil, err
		}
		seats = append(seats, st)
	}
	return seats, rows.Err()
}

// UserCompanions is who an account has finished games with, most recently
// played with first. A seat with no account behind it, such as one whose guest
// was pruned, cannot be told apart from anyone else of that name, so it is
// left out; so are the account's own other seats.
func (s *SQLiteStorage) UserCompanions(userID string, limit int) ([]models.Companion, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT other.user_id, other.player_name, other.color, m.id, m.ended_at
		FROM match_players me
		JOIN match_players other ON other.match_id = me.match_id
		JOIN matches m ON m.id = me.match_id
		WHERE me.user_id = ? AND other.user_id IS NOT NULL AND other.user_id != ?
		ORDER BY m.ended_at DESC, m.id DESC`, userID, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	companions := []models.Companion{}
	byUser := map[string]int{}
	counted := map[string]map[int64]bool{}
	for rows.Next() {
		var other, name, color string
		var matchID int64
		var ended scanTime
		if err := rows.Scan(&other, &name, &color, &matchID, &ended); err != nil {
			return nil, err
		}
		i, seen := byUser[other]
		if !seen {
			if len(companions) == limit {
				continue
			}
			// The first row is the newest game together, so its name is the
			// one they last played under.
			i = len(companions)
			byUser[other] = i
			counted[other] = map[int64]bool{}
			companions = append(companions, models.Companion{Name: name, Color: color, LastPlayedAt: ended.Time})
		}
		// Two seats of theirs in one game is still one game together.
		if !counted[other][matchID] {
			counted[other][matchID] = true
			companions[i].Games++
		}
	}
	return companions, rows.Err()
}
