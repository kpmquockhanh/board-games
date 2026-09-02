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
}

type RoomPlayer struct {
	ID         int64      `json:"id"`
	RoomID     int64      `json:"room_id"`
	PlayerName string     `json:"player_name"`
	Color      string     `json:"color"`
	Ready      bool       `json:"ready"`
	JoinedAt   time.Time  `json:"joined_at"`
	LeftAt     *time.Time `json:"left_at"`
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

type EKRoomState struct {
	RoomSettings EKRoomSettings     `json:"roomSettings"`
	GameState    *game.EKGameState  `json:"gameState,omitempty"`
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
	Players      map[string]*game.PlayerState `json:"players"`
	TurnOrder    []string                      `json:"turnOrder"`
	HandSize     int                           `json:"handSize"`
	DefenseCount int                           `json:"defenseCount"`
	Multiplier   float64                       `json:"multiplier"`
	EnabledCats  map[string]bool               `json:"enabledCats"`
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
	"ek": {
		"startGame":               true,
		"drawCard":                true,
		"playCard":                true,
		"playCombo":               true,
		"playNope":                true,
		"nopeResolved":            true,
		"resolveDefuse":           true,
		"resolveGarbageCollection": true,
		"resolveFavor":            true,
		"forceDraw":               true,
		"defuse":                  true,
		"eliminate":               true,
		"endGame":                 true,
		"update":                  true,
		"updateSettings":          true,
		"toggleReady":             true,
		"join":                    true,
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
