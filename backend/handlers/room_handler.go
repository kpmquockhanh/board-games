package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"ping/game"
	"ping/models"
	"ping/storage"
	"ping/ws"

	"github.com/gin-gonic/gin"
)

var validGames = map[string]bool{
	"hotpot": true,
	"ek":     true,
}

var gameMaxPlayers = map[string]int{
	"hotpot": 8,
	"ek":     10,
}

type Handler struct {
	store      storage.Store
	hub        *ws.Hub
	ekStates   map[string]*game.EKGameState
	ekMu       map[string]*sync.Mutex
	ekMuGlobal sync.Mutex
	// presence counts arrivals on each seat. See arrive.
	presence   map[string]uint64
	presenceMu sync.Mutex
}

func NewHandler(store storage.Store, hub *ws.Hub) *Handler {
	return &Handler{
		store:    store,
		hub:      hub,
		ekStates: make(map[string]*game.EKGameState),
		ekMu:     make(map[string]*sync.Mutex),
		presence: make(map[string]uint64),
	}
}

func presenceKey(roomKey, playerName string) string {
	return roomKey + "\x00" + playerName
}

// arrive records that a player has just turned up on a seat — through the join
// endpoint or by opening a socket — and returns the epoch of that arrival.
//
// A reload does both while the socket the old page left behind is still open,
// and that socket's teardown lands afterwards. Without an ordering, the server
// processes it as "this player is gone" and takes a player who is sitting
// right there out of the room. The epoch gives it one: a teardown carrying an
// epoch older than the seat's current one belongs to a page that has already
// been replaced, and says nothing about where the player is.
func (h *Handler) arrive(roomKey, playerName string) uint64 {
	if playerName == "" {
		return 0
	}
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	h.presence[presenceKey(roomKey, playerName)]++
	return h.presence[presenceKey(roomKey, playerName)]
}

// currentEpoch is the latest arrival recorded on a seat.
func (h *Handler) currentEpoch(roomKey, playerName string) uint64 {
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	return h.presence[presenceKey(roomKey, playerName)]
}

// supersededBy reports whether a newer page has taken over this seat since the
// arrival that epoch names.
func (h *Handler) supersededBy(roomKey, playerName string, epoch uint64) bool {
	return h.currentEpoch(roomKey, playerName) > epoch
}

// forgetPresence drops a torn-down room's arrival counters.
func (h *Handler) forgetPresence(roomKey string) {
	prefix := roomKey + "\x00"
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	for k := range h.presence {
		if strings.HasPrefix(k, prefix) {
			delete(h.presence, k)
		}
	}
}

// markBack clears a held seat's drop stamp and, when there was one, tells the
// room its player is back. Nothing else ever takes that mark off: the rest of
// the table hears "player_disconnected" when a socket drops, and used to keep
// showing the player as offline for the whole rest of the game.
func (h *Handler) markBack(roomID int64, roomKey, playerName string) {
	wasOffline, err := h.store.MarkPlayerConnected(roomID, playerName)
	if err != nil {
		log.Printf("[room] failed to clear the drop stamp for %s: %v", playerName, err)
		return
	}
	if !wasOffline {
		return
	}
	h.hub.BroadcastToRoom(roomKey, models.WSMessage{
		Type:   "player_reconnected",
		Room:   roomKey,
		Player: playerName,
	})
}

// touch records that a room just saw activity, which is what keeps the reaper
// away from it. A failure here only risks reaping a live room early, so it is
// logged rather than propagated to the player whose action triggered it.
func (h *Handler) touch(roomID int64) {
	if err := h.store.TouchRoom(roomID); err != nil {
		log.Printf("[room] touch %d: %v", roomID, err)
	}
}

func (h *Handler) getEKMutex(roomKey string) *sync.Mutex {
	h.ekMuGlobal.Lock()
	defer h.ekMuGlobal.Unlock()
	if h.ekMu[roomKey] == nil {
		h.ekMu[roomKey] = &sync.Mutex{}
	}
	return h.ekMu[roomKey]
}

func (h *Handler) getEKState(roomKey string) *game.EKGameState {
	h.ekMuGlobal.Lock()
	defer h.ekMuGlobal.Unlock()
	return h.ekStates[roomKey]
}

func (h *Handler) setEKState(roomKey string, gs *game.EKGameState) {
	h.ekMuGlobal.Lock()
	defer h.ekMuGlobal.Unlock()
	h.ekStates[roomKey] = gs
}

// clearEKState drops a finished game but keeps the room's lock, so a caller
// already holding it stays mutually excluded with the next request.
func (h *Handler) clearEKState(roomKey string) {
	h.ekMuGlobal.Lock()
	defer h.ekMuGlobal.Unlock()
	if gs, ok := h.ekStates[roomKey]; ok {
		game.CancelPendingAction(gs)
	}
	delete(h.ekStates, roomKey)
}

func (h *Handler) removeEKState(roomKey string) {
	h.ekMuGlobal.Lock()
	defer h.ekMuGlobal.Unlock()
	if gs, ok := h.ekStates[roomKey]; ok {
		game.CancelPendingAction(gs)
	}
	delete(h.ekStates, roomKey)
	delete(h.ekMu, roomKey)
}

func (h *Handler) loadEKState(roomKey string, roomID int64) *game.EKGameState {
	if gs := h.getEKState(roomKey); gs != nil {
		return gs
	}

	stateJSON, ok, err := h.store.GetLatestSnapshot(roomID)
	if err != nil || !ok || stateJSON == "" {
		return nil
	}

	var raw struct {
		GameState *game.EKGameState `json:"gameState"`
	}
	if err := json.Unmarshal([]byte(stateJSON), &raw); err != nil {
		return nil
	}
	if raw.GameState == nil {
		return nil
	}

	gs := raw.GameState
	gs.CancelFunc = make(chan struct{})
	h.setEKState(roomKey, gs)

	// The clocks live in this process, not in the snapshot. A game rehydrated
	// here — after a restart, or after the room fell out of memory — carries
	// deadlines with nothing running behind them: the turn timer that would
	// force a draw is gone, and so is the one that answers a prompt its player
	// walked away from. The table would wait forever.
	//
	// Each window starts over rather than resuming where it was cut off. The
	// persisted deadline has usually passed by now, and honouring it would
	// force a draw for a player whose client has not finished reconnecting;
	// the server coming back is the first moment anyone could act, so it is
	// the fair moment to count from.
	h.rearmEKTimers(roomKey, roomID, gs)
	return gs
}

// rearmEKTimers puts the clocks back for a game that was loaded from storage
// rather than played into memory. Callers of loadEKState hold the room lock,
// which is the same lock the timer callbacks take.
func (h *Handler) rearmEKTimers(roomKey string, roomID int64, gs *game.EKGameState) {
	if gs.Phase == "playing" {
		log.Printf("[ek] rehydrated room=%s (turn=%s, prompt=%t), restarting its clocks",
			roomKey, gs.Turn, gs.NopeWindow != nil || gs.PendingDefuse != nil ||
				gs.PendingFavor != nil || gs.PendingChoice != nil || gs.PendingGarbage != nil)
	}
	h.armEKTimers(h.getEKMutex(roomKey), roomKey, roomID, gs, gs.NopeWindow != nil)
}

func (h *Handler) saveEKState(roomKey string, gs *game.EKGameState) error {
	room, err := h.store.GetRoom(roomKey)
	if err != nil || room == nil {
		return err
	}

	stateWrapper := h.buildEKRoomState(room.ID, gs)
	stateJSON, err := json.Marshal(stateWrapper)
	if err != nil {
		return err
	}

	return h.store.UpdateSnapshot(room.ID, "", string(stateJSON))
}

func defaultEKSettings() models.EKRoomSettings {
	return models.EKRoomSettings{
		MinPlayers:         2,
		MaxPlayers:         6,
		HandSize:           8,
		StartingDefense:    1,
		ExplosiveCount:     -1,
		DeckSizeMultiplier: 1,
		TurnTimer:          0,
		IsPublic:           true,
		AllowSpectators:    false,
		EnabledCategories: map[string]bool{
			"explosive": true, "attack": true, "skip": true, "super_skip": true, "reverse": true, "future_vision": true,
			"nope": true, "shuffle": true, "draw_from_bottom": true,
			"swap_top_and_bottom": true, "garbage_collection": true,
			"catomic_bomb": true, "mark": true, "bury": true,
			"dig_deeper": true, "favor": true, "clone": true,
			"group_effects": false, "special_power": false, "cat_cards": true,
		},
	}
}

func (h *Handler) CreateRoom(c *gin.Context) {
	game := c.Param("game")
	if !validGames[game] {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown game"})
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room name is required"})
		return
	}

	room, err := h.store.CreateRoom(game, gameMaxPlayers[game], body.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create room"})
		return
	}

	initState := models.EKRoomState{RoomSettings: defaultEKSettings()}
	stateJSON, _ := json.Marshal(initState)
	h.store.SaveSnapshot(room.ID, "", string(stateJSON))

	c.JSON(http.StatusOK, gin.H{
		"room_key":    room.RoomKey,
		"room_name":   room.Name,
		"max_players": room.MaxPlayers,
		"status":      room.Status,
	})
}

func (h *Handler) JoinRoom(c *gin.Context) {
	game := c.Param("game")
	if !validGames[game] {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown game"})
		return
	}

	var body struct {
		RoomKey    string `json:"room_key"`
		PlayerName string `json:"player_name"`
		Color      string `json:"color"`
		// Session is the browser tab asking. It is what lets a reload be told
		// apart from a second person typing the same name.
		Session string `json:"session"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	if body.RoomKey == "" || body.PlayerName == "" || body.Color == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing room_key, player_name, or color"})
		return
	}

	room, err := h.store.GetRoom(body.RoomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}
	if room.Status != "active" {
		c.JSON(http.StatusConflict, gin.H{"error": "room is not active"})
		return
	}

	maxPlayers := room.MaxPlayers
	if game == "ek" {
		stateJSON, _, snapErr := h.store.GetLatestSnapshot(room.ID)
		if snapErr == nil && stateJSON != "" {
			var snap models.EKRoomState
			if json.Unmarshal([]byte(stateJSON), &snap) == nil {
				if snap.GameState != nil {
					if snap.GameState.Phase == "playing" || snap.GameState.Phase == "ended" {
						if _, isInGame := snap.GameState.Players[body.PlayerName]; !isInGame {
							c.JSON(http.StatusConflict, gin.H{"error": "game is already in progress"})
							return
						}
					}
				}
				// The room's own limit, which the lobby lets the host set.
				if n := snap.RoomSettings.MaxPlayers; n > 0 && n < maxPlayers {
					maxPlayers = n
				}
			}
		}
	}

	// Before the capacity check: a player who is still seated is rejoining,
	// not taking a new seat. A mid-game disconnect keeps their row (see
	// HandleDisconnect), so at a full table this used to count them against
	// the limit and refuse them their own game with "room is full".
	// Claim the seat before reading it. A reload's request can overtake the
	// teardown of the socket its previous page left open, and this is what
	// tells that teardown, when it lands, that it has been superseded.
	h.arrive(body.RoomKey, body.PlayerName)
	h.hub.EvictSession(body.RoomKey, body.PlayerName, body.Session)

	exists, err := h.store.IsPlayerInRoom(room.ID, body.PlayerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		// Only a live socket from some other page means the name is taken. The
		// one a reloading tab left behind is its own, and refusing it was what
		// threw a player back to the join screen for reloading mid-game.
		if h.hub.IsPlayerConnectedFromElsewhere(body.RoomKey, body.PlayerName, body.Session) {
			c.JSON(http.StatusConflict, gin.H{"error": "player is already connected"})
			return
		}
		h.touch(room.ID)
		h.markBack(room.ID, body.RoomKey, body.PlayerName)
		players, _ := h.store.GetRoomPlayers(room.ID)
		playerNames := make([]string, len(players))
		for i, p := range players {
			playerNames[i] = p.PlayerName
		}
		c.JSON(http.StatusOK, gin.H{
			"room_key":    room.RoomKey,
			"room_name":   room.Name,
			"status":      room.Status,
			"max_players": room.MaxPlayers,
			"players":     playerNames,
		})
		return
	}

	count, err := h.store.GetActivePlayerCount(room.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if count >= maxPlayers {
		c.JSON(http.StatusConflict, gin.H{"error": "room is full"})
		return
	}

	if err := h.store.AddPlayer(room.ID, body.PlayerName, body.Color); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add player"})
		return
	}

	h.store.AddTimelineEvent(room.ID, "join", body.PlayerName, "")
	h.touch(room.ID)

	players, _ := h.store.GetRoomPlayers(room.ID)
	playerNames := make([]string, len(players))
	playerColors := make(map[string]string)
	for i, p := range players {
		playerNames[i] = p.PlayerName
		playerColors[p.PlayerName] = p.Color
	}

	joinedPayload, _ := json.Marshal(gin.H{"player": body.PlayerName, "color": body.Color})
	h.hub.BroadcastToRoom(body.RoomKey, models.WSMessage{
		Type:    "player_joined",
		Room:    body.RoomKey,
		Player:  body.PlayerName,
		Payload: joinedPayload,
	})

	c.JSON(http.StatusOK, gin.H{
		"room_key":    room.RoomKey,
		"room_name":   room.Name,
		"status":      room.Status,
		"max_players": room.MaxPlayers,
		"players":     playerNames,
	})
}

func (h *Handler) ListRooms(c *gin.Context) {
	game := c.Param("game")
	if !validGames[game] {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown game"})
		return
	}

	rooms, err := h.store.ListActiveRooms(game)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	// A room set to Private is reachable by its key, but is not advertised.
	// The setting had no effect at all before.
	public := []models.RoomListItem{}
	for _, item := range rooms {
		if game == "ek" && !h.roomIsPublic(item.RoomKey) {
			continue
		}
		public = append(public, item)
	}
	c.JSON(http.StatusOK, gin.H{"rooms": public})
}

// roomIsPublic reports whether a room should appear in the room list. Rooms
// whose settings cannot be read are treated as public, as they were before.
func (h *Handler) roomIsPublic(roomKey string) bool {
	room, err := h.store.GetRoom(roomKey)
	if err != nil || room == nil {
		return true
	}
	stateJSON, ok, err := h.store.GetLatestSnapshot(room.ID)
	if err != nil || !ok || stateJSON == "" {
		return true
	}
	var snap models.EKRoomState
	if json.Unmarshal([]byte(stateJSON), &snap) != nil {
		return true
	}
	return snap.RoomSettings.IsPublic
}

func (h *Handler) GetRoom(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	players, _ := h.store.GetRoomPlayers(room.ID)
	playerNames := make([]string, len(players))
	playerColors := make(map[string]string)
	for i, p := range players {
		playerNames[i] = p.PlayerName
		playerColors[p.PlayerName] = p.Color
	}

	c.JSON(http.StatusOK, gin.H{
		"room_key":      room.RoomKey,
		"room_name":     room.Name,
		"game":          room.Game,
		"status":        room.Status,
		"max_players":   room.MaxPlayers,
		"players":       playerNames,
		"player_colors": playerColors,
	})
}

func (h *Handler) GetTimeline(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	events, err := h.store.GetTimeline(room.ID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

func (h *Handler) GetState(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	if game == "ek" {
		mu := h.getEKMutex(roomKey)
		mu.Lock()
		defer mu.Unlock()
		// Never fall through to the raw snapshot for ek: it holds every hand
		// and the deck in draw order.
		// c.Query("player") is untrusted, but a view built for a name shows
		// only that player's own hand, which they already hold.
		roomState := models.EKRoomStateView{RoomSettings: defaultEKSettings()}
		if gs := h.loadEKState(roomKey, room.ID); gs != nil {
			roomState.GameState = gs.ViewFor(c.Query("player"))
		}
		stateJSON, _, snapErr := h.store.GetLatestSnapshot(room.ID)
		if snapErr == nil && stateJSON != "" {
			var snap models.EKRoomState
			if json.Unmarshal([]byte(stateJSON), &snap) == nil {
				roomState.RoomSettings = snap.RoomSettings
			}
		}
		payload, _ := json.Marshal(roomState)
		c.JSON(http.StatusOK, gin.H{"state": json.RawMessage(payload)})
		return
	}

	state, ok, err := h.store.GetLatestSnapshot(room.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !ok {
		state = string([]byte("{}"))
	}

	c.JSON(http.StatusOK, gin.H{"state": json.RawMessage(state)})
}

func (h *Handler) SaveState(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	var body models.SaveStateRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	if body.Action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing action"})
		return
	}
	if !models.IsValidAction(game, body.Action) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action for this game"})
		return
	}
	if len(body.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing data"})
		return
	}
	if body.Player != "" {
		inRoom, err := h.store.IsPlayerInRoom(room.ID, body.Player)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if !inRoom {
			c.JSON(http.StatusBadRequest, gin.H{"error": "player not in room"})
			return
		}
	}

	// One bump for every validated action, before the game-specific split, so
	// the reaper sees a room that is being played in as alive.
	h.touch(room.ID)

	if game == "ek" {
		h.handleEKAction(c, room, roomKey, body)
		return
	}

	merged, err := h.store.MergeState(room.ID, game, body.Action, body.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge state"})
		return
	}

	if err := models.ValidateState(game, merged); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateSnapshot(room.ID, body.Player, string(merged)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save state"})
		return
	}

	h.store.AddTimelineEvent(room.ID, body.Action, body.Player, "")

	if body.Action == "endGame" {
		h.store.RemoveAllPlayers(room.ID)
		h.store.UpdateRoomStatus(room.ID, "ended")
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type: "game_ended",
			Room: roomKey,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "state": json.RawMessage(merged)})
}

func (h *Handler) handleEKAction(c *gin.Context, room *models.Room, roomKey string, body models.SaveStateRequest) {
	mu := h.getEKMutex(roomKey)
	mu.Lock()
	defer mu.Unlock()

	switch body.Action {
	case "toggleReady":
		players, err := h.store.GetRoomPlayers(room.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to get players"})
			return
		}

		var currentPlayer *models.RoomPlayer
		for i := range players {
			if players[i].PlayerName == body.Player {
				currentPlayer = &players[i]
				break
			}
		}
		if currentPlayer == nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "player not found in room"})
			return
		}

		newReady := !currentPlayer.Ready
		if err := h.store.SetPlayerReady(room.ID, body.Player, newReady); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to update ready status"})
			return
		}

		readyPayload, _ := json.Marshal(gin.H{"player": body.Player, "ready": newReady})
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type:    "player_ready",
			Room:    roomKey,
			Player:  body.Player,
			Payload: readyPayload,
		})

		// Refresh players list after toggle
		players, _ = h.store.GetRoomPlayers(room.ID)

		stateJSON, _, snapErr := h.store.GetLatestSnapshot(room.ID)
		if snapErr != nil || stateJSON == "" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "no room state found"})
			return
		}
		var raw models.EKRoomState
		if json.Unmarshal([]byte(stateJSON), &raw) != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "failed to parse room state"})
			return
		}

		allReady := len(players) >= 2
		for _, pl := range players {
			if !pl.Ready {
				allReady = false
				break
			}
		}

		if allReady {
			minPlayers := 2
			if raw.RoomSettings.MinPlayers > 0 {
				minPlayers = raw.RoomSettings.MinPlayers
			}
			if len(players) >= minPlayers {
				turnOrder := make([]string, 0, len(players))
				playersData := make(map[string]*game.PlayerState)
				for _, pl := range players {
					turnOrder = append(turnOrder, pl.PlayerName)
					playersData[pl.PlayerName] = &game.PlayerState{Color: pl.Color}
				}

				handSize := raw.RoomSettings.HandSize
				if handSize <= 0 {
					handSize = 8
				}
				defenseCount := raw.RoomSettings.StartingDefense
				if defenseCount <= 0 {
					defenseCount = 1
				}
				multiplier := raw.RoomSettings.DeckSizeMultiplier
				if multiplier <= 0 {
					multiplier = 1
				}
				enabledCats := raw.RoomSettings.EnabledCategories
				if enabledCats == nil {
					enabledCats = map[string]bool{}
				}

				explosiveCount := raw.RoomSettings.ExplosiveCount
				if explosiveCount == 0 {
					explosiveCount = -1
				}

				startData, _ := json.Marshal(models.GameStartData{
					Players:        playersData,
					TurnOrder:      turnOrder,
					HandSize:       handSize,
					DefenseCount:   defenseCount,
					Multiplier:     multiplier,
					EnabledCats:    enabledCats,
					ExplosiveCount: explosiveCount,
				})

				gs := &game.EKGameState{}
				gs.CancelFunc = make(chan struct{})
				h.setEKState(roomKey, gs)

				action := game.GameAction{
					Action:  "startGame",
					Data:    startData,
					Player:  body.Player,
					RoomKey: roomKey,
				}

				result := game.ProcessAction(gs, action)
				if result != nil {
					gs = result.State
					h.setEKState(roomKey, gs)
					h.saveEKState(roomKey, gs)
					h.broadcastEKMessages(roomKey, room.ID, result.Messages)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
		return

	case "updateSettings":
		var data struct {
			RoomSettings models.EKRoomSettings `json:"roomSettings"`
		}
		if json.Unmarshal(body.Data, &data) == nil {
			stateJSON, _, err := h.store.GetLatestSnapshot(room.ID)
			if err == nil && stateJSON != "" {
				var raw models.EKRoomState
				if json.Unmarshal([]byte(stateJSON), &raw) == nil {
					raw.RoomSettings = data.RoomSettings
					updated, _ := json.Marshal(raw)
					h.store.UpdateSnapshot(room.ID, body.Player, string(updated))

					settingsPayload, _ := json.Marshal(raw.RoomSettings)
					h.hub.BroadcastToRoom(roomKey, models.WSMessage{
						Type:    "room_settings_updated",
						Room:    roomKey,
						Payload: settingsPayload,
					})

					c.JSON(http.StatusOK, gin.H{"ok": true, "state": raw.RoomSettings})
					return
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "state": nil})
		return

	case "rematch":
		// Put a finished room back in the lobby, keeping its settings and its
		// players, so a table can play again without making a new room.
		gs := h.loadEKState(roomKey, room.ID)
		if gs == nil || gs.Phase != "ended" {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "there's no finished game to replay"})
			return
		}

		h.clearEKState(roomKey)

		stateJSON, _, snapErr := h.store.GetLatestSnapshot(room.ID)
		snap := models.EKRoomState{RoomSettings: defaultEKSettings()}
		if snapErr == nil && stateJSON != "" {
			_ = json.Unmarshal([]byte(stateJSON), &snap)
		}
		snap.GameState = nil
		if blob, err := json.Marshal(snap); err == nil {
			h.store.UpdateSnapshot(room.ID, body.Player, string(blob))
		}

		h.store.UpdateRoomStatus(room.ID, "active")

		players, _ := h.store.GetRoomPlayers(room.ID)
		for _, pl := range players {
			h.store.SetPlayerReady(room.ID, pl.PlayerName, false)
		}

		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type:   "rematch",
			Room:   roomKey,
			Player: body.Player,
		})

		c.JSON(http.StatusOK, gin.H{"ok": true})
		return

	case "join":
		joinedPayload, _ := json.Marshal(gin.H{"player": body.Player})
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type:    "player_joined",
			Room:    roomKey,
			Player:  body.Player,
			Payload: joinedPayload,
		})

		c.JSON(http.StatusOK, gin.H{"ok": true})
		return
	}

	gs := h.loadEKState(roomKey, room.ID)
	if gs == nil {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "game state not found"})
		return
	}

	result := game.ProcessAction(gs, game.GameAction{
		Action:  body.Action,
		Data:    body.Data,
		Player:  body.Player,
		RoomKey: roomKey,
	})
	if result == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "action processing failed"})
		return
	}

	// A rejected action leaves the state untouched, so there is nothing to save
	// or broadcast — just tell the player why.
	if result.Error != "" {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": result.Error})
		return
	}

	gs = result.State
	h.setEKState(roomKey, gs)

	gameOver := gs.Phase == "ended"
	if gameOver {
		// Keep the state and the players: the result screen reads them, and a
		// player who is still "in the room" can leave or delete it afterwards.
		// Only the clocks stop here.
		game.CancelPendingAction(gs)
		h.store.UpdateRoomStatus(room.ID, "ended")
	}

	h.applyEKResult(mu, roomKey, room.ID, result, body.Player, gameOver)

	payload, _ := json.Marshal(result.State.ViewFor(body.Player))
	c.JSON(http.StatusOK, gin.H{"ok": true, "state": json.RawMessage(payload)})
}

// applyEKResult persists a processed action, publishes it to the room, and
// restarts whichever clocks the new state calls for. The room mutex is held by
// every caller, including the timer callbacks.
func (h *Handler) applyEKResult(mu *sync.Mutex, roomKey string, roomID int64, result *game.ActionResult, requester string, gameOver bool) {
	gs := result.State
	if err := h.saveEKState(roomKey, gs); err != nil {
		log.Printf("[ek] save state error: %v", err)
	}

	// Before the broadcast: the clocks are what stamp NopeWindow.ExpiredAt and
	// TurnEndsAt onto the state, and a client that receives a deadline-less
	// window has no countdown to run, so it ignores the window altogether.
	h.armEKTimers(mu, roomKey, roomID, gs, result.NopeWindow)

	h.broadcastEKMessages(roomKey, roomID, result.Messages)

	// After the final state, so clients read the winner off a state they have.
	if gameOver {
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type: "game_ended",
			Room: roomKey,
		})
	}

	h.sendEKPrompt(roomKey, requester, result.Prompt)
}

// armEKTimers sets the clocks for the state as it now stands: the Nope
// countdown, the deadline on a prompt nobody has answered, and the turn timer.
func (h *Handler) armEKTimers(mu *sync.Mutex, roomKey string, roomID int64, gs *game.EKGameState, nopeWindow bool) {
	if gs.Phase != "playing" {
		game.CancelTimers(gs)
		return
	}

	if nopeWindow {
		game.StartNopeWindowTimer(mu, gs, func() {
			log.Printf("[ek] nope window expired for room=%s", roomKey)
			h.runEKTimeout(mu, roomKey, roomID, gs, game.ProcessAction(gs, game.GameAction{
				Action:  "nopeResolved",
				RoomKey: roomKey,
			}))
		}, roomKey)
	}

	// A player who closes their tab at a defuse prompt used to stop the game
	// for everyone, since nothing else may happen while one is pending.
	game.StartPendingTimer(mu, gs, func() {
		log.Printf("[ek] pending prompt timed out for room=%s", roomKey)
		h.runEKTimeout(mu, roomKey, roomID, gs, game.AutoResolvePending(gs))
	})

	game.StartTurnTimer(mu, gs, func() {
		log.Printf("[ek] turn timeout for room=%s player=%s", roomKey, gs.Turn)
		h.runEKTimeout(mu, roomKey, roomID, gs, game.ProcessAction(gs, game.GameAction{
			Action:  "forceDraw",
			Player:  gs.Turn,
			RoomKey: roomKey,
		}))
	}, h.ekTurnTimerSeconds(roomID))
}

// runEKTimeout publishes whatever a timer decided on the table's behalf.
func (h *Handler) runEKTimeout(mu *sync.Mutex, roomKey string, roomID int64, gs *game.EKGameState, result *game.ActionResult) {
	if result == nil || result.Error != "" {
		return
	}
	h.setEKState(roomKey, result.State)

	gameOver := result.State.Phase == "ended"
	if gameOver {
		game.CancelPendingAction(result.State)
		h.store.UpdateRoomStatus(roomID, "ended")
	}

	h.applyEKResult(mu, roomKey, roomID, result, "", gameOver)
}

// ekTurnTimerSeconds reads the turn limit from the room settings. It used to
// come from the body of whichever action was being taken, so a client could
// name any limit it liked and half the actions carried none at all.
func (h *Handler) ekTurnTimerSeconds(roomID int64) int {
	stateJSON, ok, err := h.store.GetLatestSnapshot(roomID)
	if err != nil || !ok || stateJSON == "" {
		return 0
	}
	var snap models.EKRoomState
	if json.Unmarshal([]byte(stateJSON), &snap) != nil {
		return 0
	}
	return snap.RoomSettings.TurnTimer
}

func (h *Handler) buildEKRoomState(roomID int64, gs *game.EKGameState) *models.EKRoomState {
	roomState := &models.EKRoomState{GameState: gs}
	stateJSON, _, err := h.store.GetLatestSnapshot(roomID)
	if err == nil && stateJSON != "" {
		var snap models.EKRoomState
		if json.Unmarshal([]byte(stateJSON), &snap) == nil {
			roomState.RoomSettings = snap.RoomSettings
		}
	}
	return roomState
}

// sendEKPrompt delivers a private prompt (a defuse choice, a peek at the deck)
// to the player it names. That is not always whoever sent the request: a Nope
// window resolves for the player who played the card.
func (h *Handler) sendEKPrompt(roomKey string, fallbackPlayer string, prompt *game.WSMessage) {
	if prompt == nil {
		return
	}
	target := prompt.Player
	if target == "" {
		target = fallbackPlayer
	}
	promptPayload, _ := json.Marshal(prompt.Payload)
	h.hub.SendToPlayer(roomKey, target, models.WSMessage{
		Type:    prompt.Type,
		Room:    roomKey,
		Player:  target,
		Payload: promptPayload,
	})
}

func (h *Handler) broadcastEKMessages(roomKey string, roomID int64, messages []game.WSMessage) {
	for _, msg := range messages {
		if gs, ok := msg.Payload.(*game.EKGameState); ok && msg.Type == "state_updated" {
			h.broadcastEKState(roomKey, gs)
			continue
		}
		marshaled, _ := json.Marshal(msg.Payload)
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type:    msg.Type,
			Room:    roomKey,
			Payload: marshaled,
		})
	}
}

// broadcastEKState sends every connected player the game as only they may see
// it. Never broadcast the raw state: it carries every hand and the deck order.
func (h *Handler) broadcastEKState(roomKey string, gs *game.EKGameState) {
	for _, player := range h.hub.ConnectedPlayers(roomKey) {
		payload, _ := json.Marshal(gs.ViewFor(player))
		h.hub.SendToPlayer(roomKey, player, models.WSMessage{
			Type:    "state_updated",
			Room:    roomKey,
			Player:  player,
			Payload: payload,
		})
	}
}

// HandleDisconnect decides what a dropped socket means. In a lobby it means the
// player left. Mid-game it does not: they are still in the game, still in the
// turn order, and taking them out of the room would refuse every action they
// tried after reconnecting.
func (h *Handler) HandleDisconnect(roomKey, playerName string, epoch uint64) {
	if h.stillHere(roomKey, playerName, epoch) {
		return
	}

	room, err := h.store.GetRoom(roomKey)
	if err != nil || room == nil {
		return
	}

	inGame := false
	if room.Game == "ek" {
		mu := h.getEKMutex(roomKey)
		mu.Lock()
		if gs := h.loadEKState(roomKey, room.ID); gs != nil && gs.Phase == "playing" {
			if p, ok := gs.Players[playerName]; ok && p != nil && p.Alive {
				inGame = true
			}
		}
		mu.Unlock()
	}

	// Checked again: the reads above take long enough for a reloading page to
	// finish joining in the middle of them, and acting on a stale teardown
	// after that point is what used to take a player out of their own room.
	if h.stillHere(roomKey, playerName, epoch) {
		return
	}

	if inGame {
		// Stamp the seat rather than releasing it. The seat is still theirs,
		// but now it has a clock on it: the reaper reclaims it if they stay
		// away, instead of the table waiting on them indefinitely.
		if err := h.store.MarkPlayerDisconnected(room.ID, playerName); err != nil {
			log.Printf("[ws] failed to stamp %s in room %s: %v", playerName, roomKey, err)
		}
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type:   "player_disconnected",
			Room:   roomKey,
			Player: playerName,
		})
		log.Printf("[ws] player dropped mid-game, holding their seat: room=%s player=%s", roomKey, playerName)
		return
	}

	if err := h.store.RemovePlayerByRoomKey(roomKey, playerName); err != nil {
		log.Printf("[ws] failed to remove player %s from room %s: %v", playerName, roomKey, err)
		return
	}
	h.hub.BroadcastToRoom(roomKey, models.WSMessage{
		Type:   "player_left",
		Room:   roomKey,
		Player: playerName,
	})
	log.Printf("[ws] player disconnected: room=%s player=%s", roomKey, playerName)
}

// stillHere reports whether a dropped socket's player is in fact present: a
// newer page has claimed the seat, or another one of theirs is still open.
func (h *Handler) stillHere(roomKey, playerName string, epoch uint64) bool {
	if h.supersededBy(roomKey, playerName, epoch) {
		return true
	}
	return h.hub.IsPlayerConnected(roomKey, playerName)
}

func (h *Handler) GetPlayers(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	players, err := h.store.GetRoomPlayers(room.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	marshalPlayers, _ := json.Marshal(players)

	c.JSON(http.StatusOK, gin.H{"players": json.RawMessage(marshalPlayers)})
}

func (h *Handler) DeleteRoom(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	if !validGames[game] {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown game"})
		return
	}

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	playerName := c.Query("player")
	if playerName == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "only a player in the room can delete it"})
		return
	}
	inRoom, err := h.store.IsPlayerInRoom(room.ID, playerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !inRoom {
		c.JSON(http.StatusForbidden, gin.H{"error": "player not in room"})
		return
	}

	h.hub.CloseRoom(roomKey)
	h.removeEKState(roomKey)
	h.forgetPresence(roomKey)

	if err := h.store.DeleteRoom(roomKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete room"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *Handler) LeaveRoom(c *gin.Context) {
	game := c.Param("game")
	roomKey := c.Param("roomKey")

	if !validGames[game] {
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown game"})
		return
	}

	room, err := h.store.GetRoom(roomKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	if room.Game != game {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room does not belong to this game"})
		return
	}

	var body struct {
		PlayerName string `json:"player_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PlayerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "player_name required"})
		return
	}

	inRoom, err := h.store.IsPlayerInRoom(room.ID, body.PlayerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !inRoom {
		c.JSON(http.StatusConflict, gin.H{"error": "player not in room"})
		return
	}

	if err := h.store.RemovePlayer(room.ID, body.PlayerName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove player"})
		return
	}

	h.store.AddTimelineEvent(room.ID, "leave", body.PlayerName, "")

	h.hub.BroadcastToRoom(roomKey, models.WSMessage{
		Type:   "player_left",
		Room:   roomKey,
		Player: body.PlayerName,
	})

	players, _ := h.store.GetRoomPlayers(room.ID)
	playerNames := make([]string, len(players))
	for i, p := range players {
		playerNames[i] = p.PlayerName
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "players": playerNames})
}
