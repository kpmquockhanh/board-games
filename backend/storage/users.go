package storage

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"ping/models"
)

// migrateUsers adds accounts. An account starts as a guest, made the first
// time a browser joins a room and kept by a cookie. Logging in upgrades the
// guest row in place, so what it collected carries over.
//
// A session is the cookie's secret, stored hashed like a seat token. It is a
// table of its own rather than a column on users so that one account can be
// signed in from several browsers, and a session can be ended on its own.
func migrateUsers(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id           TEXT PRIMARY KEY,
			guest        INTEGER NOT NULL DEFAULT 1,
			display_name TEXT NOT NULL DEFAULT '',
			color        TEXT NOT NULL DEFAULT '',
			created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_users_guest_seen ON users(guest, last_seen_at);

		CREATE TABLE IF NOT EXISTS user_sessions (
			token_hash TEXT PRIMARY KEY,
			user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);

		CREATE TABLE IF NOT EXISTS user_identities (
			provider   TEXT NOT NULL,
			subject    TEXT NOT NULL,
			user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			email      TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (provider, subject)
		);
		CREATE INDEX IF NOT EXISTS idx_user_identities_user ON user_identities(user_id);
	`); err != nil {
		return fmt.Errorf("create users: %w", err)
	}

	// Migration: user_id records whose seat it is across rooms. Null for seats
	// taken before accounts existed, and for a guest since pruned.
	_, _ = db.Exec("ALTER TABLE room_players ADD COLUMN user_id TEXT REFERENCES users(id) ON DELETE SET NULL")
	if _, err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_room_players_user ON room_players(user_id) WHERE user_id IS NOT NULL",
	); err != nil {
		return fmt.Errorf("create seat user index: %w", err)
	}
	return migrateHistory(db)
}

func newUserID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// CreateGuest makes a guest account signed in by the given session.
func (s *SQLiteStorage) CreateGuest(sessionTokenHash string) (*models.User, error) {
	id, err := newUserID()
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("INSERT INTO users (id, guest) VALUES (?, 1)", id); err != nil {
		return nil, fmt.Errorf("create guest: %w", err)
	}
	if _, err := tx.Exec("INSERT INTO user_sessions (token_hash, user_id) VALUES (?, ?)", sessionTokenHash, id); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.userByID(id)
}

// UserBySession is the account a session belongs to, or nil for a session
// that does not exist (never did, or its guest was pruned). Finding one counts
// as seeing its user, written at most once a minute so that a page polling the
// API does not turn every read into a write.
func (s *SQLiteStorage) UserBySession(sessionTokenHash string) (*models.User, error) {
	if sessionTokenHash == "" {
		return nil, nil
	}
	var id string
	err := s.db.QueryRow("SELECT user_id FROM user_sessions WHERE token_hash = ?", sessionTokenHash).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.db.Exec(
		"UPDATE users SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ? AND last_seen_at < ?",
		id, sqliteTime(time.Now().Add(-time.Minute)),
	); err != nil {
		return nil, err
	}
	return s.userByID(id)
}

func (s *SQLiteStorage) userByID(id string) (*models.User, error) {
	var u models.User
	err := s.db.QueryRow(
		"SELECT id, guest, display_name, color, created_at, last_seen_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Guest, &u.DisplayName, &u.Color, &u.CreatedAt, &u.LastSeenAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// RememberPlayerProfile keeps the name and colour a user last sat down with,
// to offer them again next time.
func (s *SQLiteStorage) RememberPlayerProfile(userID, name, color string) error {
	_, err := s.db.Exec("UPDATE users SET display_name = ?, color = ? WHERE id = ?", name, color, userID)
	return err
}

// SetSeatUser records whose seat a player's is. Every join through the seat's
// own token or name-claim lands here, so a guest whose cookie was cleared
// takes their seat's history with them to the guest that replaced it.
func (s *SQLiteStorage) SetSeatUser(roomID int64, playerName, userID string) error {
	if userID == "" {
		return nil
	}
	_, err := s.db.Exec(
		"UPDATE room_players SET user_id = ? WHERE room_id = ? AND player_name = ?",
		userID, roomID, playerName,
	)
	return err
}

// DeleteIdleGuests removes guest accounts not seen since the cutoff, along
// with their sessions. Their seats keep the rows and lose the link. A guest
// who comes back after that is simply made again.
func (s *SQLiteStorage) DeleteIdleGuests(cutoff time.Time) (int64, error) {
	res, err := s.db.Exec("DELETE FROM users WHERE guest = 1 AND last_seen_at < ?", sqliteTime(cutoff))
	if err != nil {
		return 0, fmt.Errorf("delete idle guests: %w", err)
	}
	return res.RowsAffected()
}

// SignIn logs a browser in as the account behind a provider identity, and
// replaces the browser's session with a new one, so a session secret known
// before the login is worth nothing after it.
//
// Whose account it becomes:
//   - An identity seen before signs in to its account. A guest the browser
//     was using is folded into it: its seats and matches move over and the
//     guest goes.
//   - A new identity upgrades the browser's guest in place, so the guest
//     keeps what it collected.
//   - A new identity on a browser with no account, or with one already
//     signed in, makes an account of its own. Joining a signed-in account
//     would hand whoever logs in next on a shared browser that account.
//
// An account that has signed in is never folded into another.
func (s *SQLiteStorage) SignIn(sessionTokenHash, previousSessionHash string, id models.Identity) (*models.User, error) {
	if id.Provider == "" || id.Subject == "" {
		return nil, fmt.Errorf("sign in: identity has no provider or subject")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var current string
	var currentGuest bool
	if previousSessionHash != "" {
		err := tx.QueryRow(`
			SELECT u.id, u.guest FROM user_sessions s JOIN users u ON u.id = s.user_id
			WHERE s.token_hash = ?`, previousSessionHash,
		).Scan(&current, &currentGuest)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if _, err := tx.Exec("DELETE FROM user_sessions WHERE token_hash = ?", previousSessionHash); err != nil {
			return nil, err
		}
	}

	var owner string
	err = tx.QueryRow(
		"SELECT user_id FROM user_identities WHERE provider = ? AND subject = ?", id.Provider, id.Subject,
	).Scan(&owner)
	switch {
	case err == nil:
		if current != "" && current != owner && currentGuest {
			if _, err := tx.Exec("UPDATE room_players SET user_id = ? WHERE user_id = ?", owner, current); err != nil {
				return nil, fmt.Errorf("move guest seats: %w", err)
			}
			if _, err := tx.Exec("UPDATE match_players SET user_id = ? WHERE user_id = ?", owner, current); err != nil {
				return nil, fmt.Errorf("move guest matches: %w", err)
			}
			if _, err := tx.Exec("DELETE FROM users WHERE id = ? AND guest = 1", current); err != nil {
				return nil, fmt.Errorf("drop guest: %w", err)
			}
		}
	case err == sql.ErrNoRows:
		// A first login upgrades this browser's guest in place. An account
		// already signed in here is somebody's, maybe someone else's using the
		// same browser, so a login it has never seen gets an account of its own.
		if currentGuest {
			owner = current
		}
		if owner == "" {
			if owner, err = newUserID(); err != nil {
				return nil, err
			}
			if _, err := tx.Exec("INSERT INTO users (id) VALUES (?)", owner); err != nil {
				return nil, fmt.Errorf("create user: %w", err)
			}
		}
		if _, err := tx.Exec(
			"INSERT INTO user_identities (provider, subject, user_id, email) VALUES (?, ?, ?, ?)",
			id.Provider, id.Subject, owner, id.Email,
		); err != nil {
			return nil, fmt.Errorf("link identity: %w", err)
		}
	default:
		return nil, err
	}

	// A name chosen at the table wins over the provider's.
	if _, err := tx.Exec(`
		UPDATE users SET guest = 0, last_seen_at = CURRENT_TIMESTAMP,
			display_name = CASE WHEN display_name = '' THEN ? ELSE display_name END
		WHERE id = ?`, id.Name, owner,
	); err != nil {
		return nil, err
	}
	if _, err := tx.Exec("INSERT INTO user_sessions (token_hash, user_id) VALUES (?, ?)", sessionTokenHash, owner); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.userByID(owner)
}

// EndSession signs one browser out. The account and its other sessions stay.
func (s *SQLiteStorage) EndSession(sessionTokenHash string) error {
	_, err := s.db.Exec("DELETE FROM user_sessions WHERE token_hash = ?", sessionTokenHash)
	return err
}

// UserProviders names the providers an account can sign in with.
func (s *SQLiteStorage) UserProviders(userID string) ([]string, error) {
	rows, err := s.db.Query("SELECT provider FROM user_identities WHERE user_id = ? ORDER BY provider", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	providers := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, rows.Err()
}
