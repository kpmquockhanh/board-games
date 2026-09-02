package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"ping/game"
	"ping/models"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLite(dbPath string) (*SQLiteStorage, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			game        TEXT NOT NULL,
			room_key    TEXT NOT NULL UNIQUE,
			name        TEXT NOT NULL DEFAULT '',
			max_players INTEGER DEFAULT 6,
			status      TEXT DEFAULT 'active',
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_rooms_game_key ON rooms(game, room_key);
		CREATE INDEX IF NOT EXISTS idx_rooms_status ON rooms(status);

		CREATE TABLE IF NOT EXISTS room_players (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id     INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			player_name TEXT NOT NULL,
			color       TEXT NOT NULL,
			ready       INTEGER DEFAULT 0,
			joined_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			left_at     DATETIME,
			UNIQUE(room_id, player_name)
		);
		CREATE INDEX IF NOT EXISTS idx_room_players_room ON room_players(room_id);

		CREATE TABLE IF NOT EXISTS timelines (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id    INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
			event_type TEXT NOT NULL,
			player     TEXT,
			payload    TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_timelines_room ON timelines(room_id, created_at);
	`)
	if err != nil {
		return err
	}

	// Migration: add ready column if missing (for existing DBs)
	_, _ = db.Exec("ALTER TABLE room_players ADD COLUMN ready INTEGER DEFAULT 0")

	// Migration: add name column to rooms if missing (for existing DBs)
	_, _ = db.Exec("ALTER TABLE rooms ADD COLUMN name TEXT NOT NULL DEFAULT ''")

	return nil
}

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func generateRoomKey() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func (s *SQLiteStorage) CreateRoom(game string, maxPlayers int, name string) (*models.Room, error) {
	for attempts := 0; attempts < 10; attempts++ {
		key := generateRoomKey()
		result, err := s.db.Exec(
			"INSERT INTO rooms (game, room_key, name, max_players, status) VALUES (?, ?, ?, ?, 'active')",
			game, key, name, maxPlayers,
		)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint") {
				continue
			}
			return nil, fmt.Errorf("create room: %w", err)
		}
		id, _ := result.LastInsertId()
		return &models.Room{
			ID:         id,
			Game:       game,
			RoomKey:    key,
			Name:       name,
			MaxPlayers: maxPlayers,
			Status:     "active",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("failed to generate unique room key after 10 attempts")
}

func (s *SQLiteStorage) GetRoom(roomKey string) (*models.Room, error) {
	var r models.Room
	err := s.db.QueryRow(
		"SELECT id, game, room_key, name, max_players, status, created_at, updated_at FROM rooms WHERE room_key = ?",
		roomKey,
	).Scan(&r.ID, &r.Game, &r.RoomKey, &r.Name, &r.MaxPlayers, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room: %w", err)
	}
	return &r, nil
}

func (s *SQLiteStorage) GetRoomByID(roomID int64) (*models.Room, error) {
	var r models.Room
	err := s.db.QueryRow(
		"SELECT id, game, room_key, name, max_players, status, created_at, updated_at FROM rooms WHERE id = ?",
		roomID,
	).Scan(&r.ID, &r.Game, &r.RoomKey, &r.Name, &r.MaxPlayers, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get room by id: %w", err)
	}
	return &r, nil
}

func (s *SQLiteStorage) UpdateRoomStatus(roomID int64, status string) error {
	_, err := s.db.Exec(
		"UPDATE rooms SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		status, roomID,
	)
	return err
}

func (s *SQLiteStorage) ListActiveRooms(game string) ([]models.RoomListItem, error) {
	rows, err := s.db.Query(`
		SELECT r.room_key, r.name, r.max_players, r.created_at,
		       (SELECT COUNT(*) FROM room_players rp WHERE rp.room_id = r.id AND rp.left_at IS NULL) AS player_count
		FROM rooms r
		WHERE r.game = ? AND r.status = 'active'
		ORDER BY r.created_at DESC
	`, game)
	if err != nil {
		return nil, fmt.Errorf("list active rooms: %w", err)
	}
	defer rows.Close()

	var rooms []models.RoomListItem
	for rows.Next() {
		var rm models.RoomListItem
		if err := rows.Scan(&rm.RoomKey, &rm.Name, &rm.MaxPlayers, &rm.CreatedAt, &rm.PlayerCount); err != nil {
			continue
		}
		rooms = append(rooms, rm)
	}
	return rooms, nil
}

func (s *SQLiteStorage) AddPlayer(roomID int64, playerName, color string) error {
	_, err := s.db.Exec(
		"INSERT INTO room_players (room_id, player_name, color) VALUES (?, ?, ?) ON CONFLICT(room_id, player_name) DO UPDATE SET color = excluded.color, ready = 0, left_at = NULL",
		roomID, playerName, color,
	)
	return err
}

func (s *SQLiteStorage) RemovePlayer(roomID int64, playerName string) error {
	_, err := s.db.Exec(
		"UPDATE room_players SET left_at = CURRENT_TIMESTAMP WHERE room_id = ? AND player_name = ? AND left_at IS NULL",
		roomID, playerName,
	)
	return err
}

func (s *SQLiteStorage) RemovePlayerByRoomKey(roomKey string, playerName string) error {
	_, err := s.db.Exec(
		"UPDATE room_players SET left_at = CURRENT_TIMESTAMP WHERE room_id = (SELECT id FROM rooms WHERE room_key = ?) AND player_name = ? AND left_at IS NULL",
		roomKey, playerName,
	)
	return err
}

func (s *SQLiteStorage) RemoveAllPlayers(roomID int64) error {
	_, err := s.db.Exec(
		"UPDATE room_players SET left_at = CURRENT_TIMESTAMP WHERE room_id = ? AND left_at IS NULL",
		roomID,
	)
	return err
}

func (s *SQLiteStorage) DeletePlayer(roomKey string, playerName string) error {
	_, err := s.db.Exec(
		"DELETE FROM room_players WHERE room_id = (SELECT id FROM rooms WHERE room_key = ?) AND player_name = ?",
		roomKey, playerName,
	)
	return err
}

func (s *SQLiteStorage) DeleteRoom(roomKey string) error {
	_, err := s.db.Exec("DELETE FROM rooms WHERE room_key = ?", roomKey)
	return err
}

func (s *SQLiteStorage) GetRoomPlayers(roomID int64) ([]models.RoomPlayer, error) {
	rows, err := s.db.Query(
		"SELECT id, room_id, player_name, color, ready, joined_at, left_at FROM room_players WHERE room_id = ? AND left_at IS NULL ORDER BY joined_at",
		roomID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []models.RoomPlayer
	for rows.Next() {
		var p models.RoomPlayer
		if err := rows.Scan(&p.ID, &p.RoomID, &p.PlayerName, &p.Color, &p.Ready, &p.JoinedAt, &p.LeftAt); err != nil {
			continue
		}
		players = append(players, p)
	}
	return players, nil
}

func (s *SQLiteStorage) IsPlayerInRoom(roomID int64, playerName string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM room_players WHERE room_id = ? AND player_name = ? AND left_at IS NULL",
		roomID, playerName,
	).Scan(&count)
	return count > 0, err
}

func (s *SQLiteStorage) GetActivePlayerCount(roomID int64) (int, error) {
	var count int
	err := s.db.QueryRow(
		"SELECT COUNT(*) FROM room_players WHERE room_id = ? AND left_at IS NULL",
		roomID,
	).Scan(&count)
	return count, err
}

func (s *SQLiteStorage) SetPlayerReady(roomID int64, playerName string, ready bool) error {
	readyInt := 0
	if ready {
		readyInt = 1
	}
	_, err := s.db.Exec(
		"UPDATE room_players SET ready = ? WHERE room_id = ? AND player_name = ? AND left_at IS NULL",
		readyInt, roomID, playerName,
	)
	return err
}

func (s *SQLiteStorage) AddTimelineEvent(roomID int64, eventType, player string, payload string) error {
	_, err := s.db.Exec(
		"INSERT INTO timelines (room_id, event_type, player, payload) VALUES (?, ?, ?, ?)",
		roomID, eventType, player, payload,
	)
	return err
}

func (s *SQLiteStorage) GetTimeline(roomID int64, limit int) ([]models.TimelineEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(
		"SELECT id, room_id, event_type, player, payload, created_at FROM timelines WHERE room_id = ? ORDER BY created_at DESC LIMIT ?",
		roomID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.TimelineEvent
	for rows.Next() {
		var e models.TimelineEvent
		if err := rows.Scan(&e.ID, &e.RoomID, &e.EventType, &e.Player, &e.Payload, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *SQLiteStorage) SaveSnapshot(roomID int64, player string, state string) error {
	_, err := s.db.Exec(
		"INSERT INTO timelines (room_id, event_type, player, payload) VALUES (?, 'snapshot', ?, ?)",
		roomID, player, state,
	)
	return err
}

func (s *SQLiteStorage) UpdateSnapshot(roomID int64, player string, state string) error {
	result, err := s.db.Exec(
		"UPDATE timelines SET payload = ?, player = ?, created_at = CURRENT_TIMESTAMP WHERE room_id = ? AND event_type = 'snapshot' AND id = (SELECT id FROM timelines WHERE room_id = ? AND event_type = 'snapshot' ORDER BY created_at DESC LIMIT 1)",
		state, player, roomID, roomID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return s.SaveSnapshot(roomID, player, state)
	}
	return nil
}

func (s *SQLiteStorage) MergeState(roomID int64, game string, action string, data json.RawMessage) (json.RawMessage, error) {
	latest, _, err := s.GetLatestSnapshot(roomID)
	if err != nil {
		return nil, err
	}

	switch game {
	case "hotpot":
		return mergeHotpotState(latest, data, action)
	case "ek":
		return mergeEKState(latest, data, action)
	default:
		return mergeGenericState(latest, data)
	}
}

func mergeHotpotState(latest string, delta json.RawMessage, action string) (json.RawMessage, error) {
	var current models.HotpotState
	if latest != "" {
		if err := json.Unmarshal([]byte(latest), &current); err != nil {
			current = models.HotpotState{}
		}
	}
	if current.Feed == nil {
		current.Feed = []models.HotpotFeedEntry{}
	}

	var d struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Text  string `json:"text"`
		Emoji string `json:"emoji"`
	}
	if err := json.Unmarshal(delta, &d); err != nil {
		return nil, fmt.Errorf("invalid delta: %w", err)
	}

	entry := models.HotpotFeedEntry{
		ID:    generateID(),
		Ts:    time.Now().UnixMilli(),
		Type:  action,
		Name:  d.Name,
		Color: d.Color,
		Text:  d.Text,
		Emoji: d.Emoji,
	}

	current.Feed = append(current.Feed, entry)

	const maxFeedEntries = 100
	if len(current.Feed) > maxFeedEntries {
		current.Feed = current.Feed[len(current.Feed)-maxFeedEntries:]
	}

	return json.Marshal(current)
}

func mergeEKState(latest string, delta json.RawMessage, action string) (json.RawMessage, error) {
	var current models.EKRoomState
	if latest != "" {
		if err := json.Unmarshal([]byte(latest), &current); err != nil {
			current = models.EKRoomState{}
		}
	}
	if current.RoomSettings.EnabledCategories == nil {
		current.RoomSettings.EnabledCategories = make(map[string]bool)
	}

	var d models.EKRoomState
	if err := json.Unmarshal(delta, &d); err != nil {
		return nil, fmt.Errorf("invalid delta: %w", err)
	}

	if d.GameState != nil {
		if current.GameState == nil {
			current.GameState = d.GameState
		} else {
			mergeEKGameState(current.GameState, d.GameState)
		}
	}

	if d.RoomSettings.MinPlayers != 0 {
		current.RoomSettings.MinPlayers = d.RoomSettings.MinPlayers
	}
	if d.RoomSettings.MaxPlayers != 0 {
		current.RoomSettings.MaxPlayers = d.RoomSettings.MaxPlayers
	}
	if d.RoomSettings.HandSize != 0 {
		current.RoomSettings.HandSize = d.RoomSettings.HandSize
	}
	if d.RoomSettings.StartingDefense != 0 {
		current.RoomSettings.StartingDefense = d.RoomSettings.StartingDefense
	}
	if d.RoomSettings.ExplosiveCount != 0 {
		current.RoomSettings.ExplosiveCount = d.RoomSettings.ExplosiveCount
	}
	if d.RoomSettings.DeckSizeMultiplier != 0 {
		current.RoomSettings.DeckSizeMultiplier = d.RoomSettings.DeckSizeMultiplier
	}
	if d.RoomSettings.TurnTimer != 0 {
		current.RoomSettings.TurnTimer = d.RoomSettings.TurnTimer
	}
	if d.RoomSettings.IsPublic {
		current.RoomSettings.IsPublic = d.RoomSettings.IsPublic
	}
	if d.RoomSettings.AllowSpectators {
		current.RoomSettings.AllowSpectators = d.RoomSettings.AllowSpectators
	}
	for k, v := range d.RoomSettings.EnabledCategories {
		current.RoomSettings.EnabledCategories[k] = v
	}

	return json.Marshal(current)
}

func mergeEKGameState(current, delta *game.EKGameState) {
	for k, v := range delta.Players {
		if v == nil {
			continue
		}
		if existing, ok := current.Players[k]; ok {
			existing.Color = v.Color
			existing.Hand = v.Hand
			existing.Alive = v.Alive
		} else {
			current.Players[k] = v
		}
	}

	if len(delta.Log) > 0 {
		current.Log = append(current.Log, delta.Log...)
		const maxLogEntries = 50
		if len(current.Log) > maxLogEntries {
			current.Log = current.Log[len(current.Log)-maxLogEntries:]
		}
	}

	if len(delta.Deck) > 0 {
		current.Deck = delta.Deck
	}
	if len(delta.Discard) > 0 {
		current.Discard = delta.Discard
	}
	if delta.Turn != "" {
		current.Turn = delta.Turn
	}
	if len(delta.TurnOrder) > 0 {
		current.TurnOrder = delta.TurnOrder
	}
	if delta.Phase != "" {
		current.Phase = delta.Phase
	}
	if delta.Winner != nil {
		current.Winner = delta.Winner
	}
	if delta.AttackStack > 0 {
		current.AttackStack = delta.AttackStack
	}
}

func mergeGenericState(latest string, delta json.RawMessage) (json.RawMessage, error) {
	var current map[string]interface{}
	if latest != "" {
		if err := json.Unmarshal([]byte(latest), &current); err != nil {
			current = make(map[string]interface{})
		}
	} else {
		current = make(map[string]interface{})
	}

	var d map[string]interface{}
	if err := json.Unmarshal(delta, &d); err != nil {
		return nil, fmt.Errorf("invalid delta: %w", err)
	}

	result := deepMerge(current, d)
	return json.Marshal(result)
}

func deepMerge(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		if baseVal, ok := result[k]; ok {
			if baseMap, ok := baseVal.(map[string]interface{}); ok {
				if overMap, ok := v.(map[string]interface{}); ok {
					result[k] = deepMerge(baseMap, overMap)
					continue
				}
			}
		}
		result[k] = v
	}
	return result
}

func generateID() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = "0123456789abcdef"[rand.Intn(16)]
	}
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), string(b))
}

func (s *SQLiteStorage) GetLatestSnapshot(roomID int64) (string, bool, error) {
	var payload string
	err := s.db.QueryRow(
		"SELECT payload FROM timelines WHERE room_id = ? AND event_type = 'snapshot' ORDER BY created_at DESC LIMIT 1",
		roomID,
	).Scan(&payload)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return payload, true, nil
}

func (s *SQLiteStorage) SavePlayers(roomID int64, players string) error {
	_, err := s.db.Exec(
		"INSERT INTO timelines (room_id, event_type, player, payload) VALUES (?, 'players', '', ?)",
		roomID, players,
	)
	return err
}

func (s *SQLiteStorage) GetLatestPlayers(roomID int64) (string, bool, error) {
	var payload string
	err := s.db.QueryRow(
		"SELECT payload FROM timelines WHERE room_id = ? AND event_type = 'players' ORDER BY created_at DESC LIMIT 1",
		roomID,
	).Scan(&payload)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return payload, true, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

func (s *SQLiteStorage) logError(context string, err error) {
	if err != nil {
		log.Printf("[storage] %s: %v", context, err)
	}
}
