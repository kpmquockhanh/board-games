package models

import (
	"encoding/json"
	"fmt"
	"time"

	"ping/game"
)

type WSMessage struct {
	Type    string          `json:"type"`
	Room    string          `json:"room"`
	Player  string          `json:"player"`
	Payload json.RawMessage `json:"payload"`
}

type Room struct {
	ID         int64     `json:"id"`
	Game       string    `json:"game"`
	RoomKey    string    `json:"room_key"`
	Name       string    `json:"name"`
	MaxPlayers int       `json:"max_players"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// LastActivityAt is the room's own clock, bumped by anything a player
	// does in it. The reaper measures silence by this, not by UpdatedAt.
	LastActivityAt time.Time `json:"last_activity_at"`
}

type RoomPlayer struct {
	ID         int64      `json:"id"`
	RoomID     int64      `json:"room_id"`
	PlayerName string     `json:"player_name"`
	Color      string     `json:"color"`
	Ready      bool       `json:"ready"`
	JoinedAt   time.Time  `json:"joined_at"`
	LeftAt     *time.Time `json:"left_at"`
	// DisconnectedAt is set while a player's seat is being held for them
	// after their socket dropped. Null means they are connected, or gone for
	// good — LeftAt tells those two apart.
	DisconnectedAt *time.Time `json:"disconnected_at"`
	// UserID is the account in the seat. Never sent: the roster goes to
	// every player, and an account id is not theirs to see.
	UserID *string `json:"-"`
}

// StalePlayer is a held seat whose player has been gone long enough to
// reclaim it.
type StalePlayer struct {
	RoomID         int64     `json:"room_id"`
	RoomKey        string    `json:"room_key"`
	Game           string    `json:"game"`
	PlayerName     string    `json:"player_name"`
	DisconnectedAt time.Time `json:"disconnected_at"`
}

type TimelineEvent struct {
	ID        int64     `json:"id"`
	RoomID    int64     `json:"room_id"`
	EventType string    `json:"event_type"`
	Player    string    `json:"player"`
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type RoomListItem struct {
	RoomKey     string    `json:"room_key"`
	Name        string    `json:"name"`
	PlayerCount int       `json:"player_count"`
	MaxPlayers  int       `json:"max_players"`
	CreatedAt   time.Time `json:"created_at"`
}

type SaveStateRequest struct {
	Action string          `json:"action"`
	Player string          `json:"player"`
	Data   json.RawMessage `json:"data"`
}

// --- Room state (stored as JSON in timelines table) ---

// EKRoomState is the server's own record, stored as a snapshot. It holds every
// hand and the deck, so it never goes to a client — see EKRoomStateView.
type EKRoomState struct {
	RoomSettings EKRoomSettings    `json:"roomSettings"`
	GameState    *game.EKGameState `json:"gameState,omitempty"`
}

// EKRoomStateView is the same room as one player may see it.
type EKRoomStateView struct {
	RoomSettings EKRoomSettings `json:"roomSettings"`
	GameState    *game.GameView `json:"gameState,omitempty"`
}

type EKRoomSettings struct {
	MinPlayers         int             `json:"minPlayers"`
	MaxPlayers         int             `json:"maxPlayers"`
	HandSize           int             `json:"handSize"`
	StartingDefense    int             `json:"startingDefense"`
	ExplosiveCount     int             `json:"explosiveCount"`
	DeckSizeMultiplier float64         `json:"deckSizeMultiplier"`
	TurnTimer          int             `json:"turnTimer"`
	IsPublic           bool            `json:"isPublic"`
	AllowSpectators    bool            `json:"allowSpectators"`
	EnabledCategories  map[string]bool `json:"enabledCategories"`
}

type GameStartData struct {
	Players        map[string]*game.PlayerState `json:"players"`
	TurnOrder      []string                     `json:"turnOrder"`
	HandSize       int                          `json:"handSize"`
	DefenseCount   int                          `json:"defenseCount"`
	Multiplier     float64                      `json:"multiplier"`
	EnabledCats    map[string]bool              `json:"enabledCats"`
	ExplosiveCount int                          `json:"explosiveCount"`
}

// --- Hotpot game state ---

type HotpotFeedEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji string `json:"emoji,omitempty"`
	Ts    int64  `json:"ts"`
}

type HotpotState struct {
	Feed []HotpotFeedEntry `json:"feed"`
}

// --- Exploding Kitchen game state ---

type EKPlayer struct {
	Color string   `json:"color"`
	Hand  []string `json:"hand"`
	Alive bool     `json:"alive"`
}

type EKLogEntry struct {
	Name string `json:"name"`
	Text string `json:"text"`
	Ts   int64  `json:"ts"`
}

type EKGameState struct {
	Players     map[string]EKPlayer `json:"players"`
	Deck        []string            `json:"deck"`
	Discard     []string            `json:"discard"`
	Turn        string              `json:"turn"`
	TurnOrder   []string            `json:"turnOrder"`
	Phase       string              `json:"phase"`
	Winner      *string             `json:"winner"`
	Log         []json.RawMessage   `json:"log"`
	AttackStack int                 `json:"attackStack"`
	TopCards    []string            `json:"topCards"`
}

// --- Action validation ---

var validActions = map[string]map[string]bool{
	"hotpot": {
		"join":  true,
		"leave": true,
		"drop":  true,
		"cheer": true,
		"chat":  true,
	},
	// Only actions a client is allowed to send. startGame, nopeResolved and
	// forceDraw are driven by the server itself (ready-up, the Nope timer and
	// the turn timer) and would let a player restart a game, cut the Nope
	// window short or draw for someone else.
	"ek": {
		"drawCard":                 true,
		"playCard":                 true,
		"playCombo":                true,
		"playNope":                 true,
		"passNope":                 true,
		"resolveDefuse":            true,
		"resolveGarbageCollection": true,
		"resolveFavor":             true,
		"resolveChoice":            true,
		"updateSettings":           true,
		"toggleReady":              true,
		"rematch":                  true,
		"join":                     true,
	},
}

func IsValidAction(game, action string) bool {
	actions, ok := validActions[game]
	if !ok {
		return false
	}
	return actions[action]
}

func ValidateState(game string, state json.RawMessage) error {
	switch game {
	case "hotpot":
		return validateHotpotState(state)
	case "ek":
		return validateEKState(state)
	}
	return nil
}

func validateHotpotState(state json.RawMessage) error {
	var s HotpotState
	if err := json.Unmarshal(state, &s); err != nil {
		return fmt.Errorf("invalid hotpot state: %w", err)
	}
	if s.Feed == nil {
		s.Feed = []HotpotFeedEntry{}
	}
	return nil
}

func validateEKState(state json.RawMessage) error {
	var s EKGameState
	if err := json.Unmarshal(state, &s); err != nil {
		return fmt.Errorf("invalid ek state: %w", err)
	}
	if s.Players == nil {
		s.Players = make(map[string]EKPlayer)
	}
	if s.TurnOrder == nil {
		s.TurnOrder = []string{}
	}
	if s.Deck == nil {
		s.Deck = []string{}
	}
	if s.Discard == nil {
		s.Discard = []string{}
	}
	if s.Log == nil {
		s.Log = []json.RawMessage{}
	}
	if s.Phase == "" {
		s.Phase = "lobby"
	}
	if s.AttackStack < 0 {
		s.AttackStack = 0
	}
	return nil
}

// User is the person behind a browser: a guest made silently on their first
// join, kept by a cookie. It says who someone is across rooms and visits; the
// seat token still says which seat in a room a tab is sitting in.
type User struct {
	ID          string    `json:"id"`
	Guest       bool      `json:"guest"`
	DisplayName string    `json:"display_name"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// Identity is who a login provider says someone is. Subject is the
// provider's stable id for them; the name and email are only what the
// provider showed at the time.
type Identity struct {
	Provider string
	Subject  string
	Name     string
	Email    string
}

// Seat is a room an account is sitting in, for the list of games it can
// go back to. Held means the seat is waiting for its player to come back.
type Seat struct {
	Game       string `json:"game"`
	RoomKey    string `json:"room_key"`
	RoomName   string `json:"room_name"`
	PlayerName string `json:"player_name"`
	Color      string `json:"color"`
	Held       bool   `json:"held"`
}

// MatchRecord is a finished game as it is written down: who played it, who
// won, and which seats were whose at the end.
type MatchRecord struct {
	Game      string
	RoomID    int64
	RoomKey   string
	RoomName  string
	StartedAt time.Time
	EndedAt   time.Time
	Players   []MatchPlayer
}

type MatchPlayer struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Won   bool   `json:"won"`
	// You marks the seats the asking account played in.
	You bool `json:"you"`
}

// Match is a finished game in an account's history.
type Match struct {
	ID        int64         `json:"id"`
	Game      string        `json:"game"`
	RoomName  string        `json:"room_name"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   time.Time     `json:"ended_at"`
	Players   []MatchPlayer `json:"players"`
}

// Companion is someone an account has finished games with. They are known by
// the account behind their seat, so a player who changes their name between
// games is still one person, and are shown under the name and colour they
// last played with.
type Companion struct {
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	Games        int       `json:"games"`
	LastPlayedAt time.Time `json:"last_played_at"`
}
