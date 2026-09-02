package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
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
}

func NewHandler(store storage.Store, hub *ws.Hub) *Handler {
	return &Handler{
		store:    store,
		hub:      hub,
		ekStates: make(map[string]*game.EKGameState),
		ekMu:     make(map[string]*sync.Mutex),
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
	return gs
}

func (h *Handler) saveEKState(roomKey string, gs *game.EKGameState) error {
	room, err := h.store.GetRoom(roomKey)
	if err != nil || room == nil {
		return err
	}

	stateWrapper := models.EKRoomState{
		GameState: gs,
	}
	stateJSON, err := json.Marshal(stateWrapper)
	if err != nil {
		return err
	}

	return h.store.UpdateSnapshot(room.ID, "", string(stateJSON))
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

	initState := models.EKRoomState{
		RoomSettings: models.EKRoomSettings{
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
		},
	}
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

	if game == "ek" {
		stateJSON, _, snapErr := h.store.GetLatestSnapshot(room.ID)
		if snapErr == nil && stateJSON != "" {
			var snap models.EKRoomState
			if json.Unmarshal([]byte(stateJSON), &snap) == nil && snap.GameState != nil {
				if snap.GameState.Phase == "playing" || snap.GameState.Phase == "ended" {
					if _, isInGame := snap.GameState.Players[body.PlayerName]; !isInGame {
						c.JSON(http.StatusConflict, gin.H{"error": "game is already in progress"})
						return
					}
				}
			}
		}
	}

	count, err := h.store.GetActivePlayerCount(room.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if count >= room.MaxPlayers {
		c.JSON(http.StatusConflict, gin.H{"error": "room is full"})
		return
	}

	exists, err := h.store.IsPlayerInRoom(room.ID, body.PlayerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		if h.hub.IsPlayerConnected(body.RoomKey, body.PlayerName) {
			c.JSON(http.StatusConflict, gin.H{"error": "player is already connected"})
			return
		}
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

	if err := h.store.AddPlayer(room.ID, body.PlayerName, body.Color); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add player"})
		return
	}

	h.store.AddTimelineEvent(room.ID, "join", body.PlayerName, "")

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
	if rooms == nil {
		rooms = []models.RoomListItem{}
	}
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
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
		gs := h.getEKState(roomKey)
		if gs != nil {
			roomState := models.EKRoomState{GameState: gs}
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

				startData, _ := json.Marshal(models.GameStartData{
					Players:      playersData,
					TurnOrder:    turnOrder,
					HandSize:     handSize,
					DefenseCount: defenseCount,
					Multiplier:   multiplier,
					EnabledCats:  enabledCats,
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

	gs = result.State
	h.setEKState(roomKey, gs)

	if body.Action == "endGame" || (gs.Phase == "ended") {
		h.store.RemoveAllPlayers(room.ID)
		h.store.UpdateRoomStatus(room.ID, "ended")
		h.removeEKState(roomKey)
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type: "game_ended",
			Room: roomKey,
		})
	}

	if result.NopeWindow {
		game.StartNopeWindowTimer(mu, gs, func() {
			log.Printf("[ek] nope window expired for room=%s", roomKey)
			resolveAction := game.GameAction{
				Action:  "nopeResolved",
				Player:  body.Player,
				RoomKey: roomKey,
			}
			result := game.ProcessAction(gs, resolveAction)
			if result != nil {
				h.setEKState(roomKey, result.State)
				h.saveEKState(roomKey, result.State)
				h.broadcastEKMessages(roomKey, room.ID, result.Messages)
				if result.Prompt != nil {
					promptPayload, _ := json.Marshal(result.Prompt.Payload)
					h.hub.SendToPlayer(roomKey, body.Player, models.WSMessage{
						Type:    result.Prompt.Type,
						Room:    roomKey,
						Player:  body.Player,
						Payload: promptPayload,
					})
				}
			}
		}, roomKey)
	}

	if err := h.saveEKState(roomKey, gs); err != nil {
		log.Printf("[ek] save state error: %v", err)
	}

	h.broadcastEKMessages(roomKey, room.ID, result.Messages)

	if result.Prompt != nil {
		result.Prompt.Room = roomKey
		promptPayload, _ := json.Marshal(result.Prompt.Payload)
		h.hub.SendToPlayer(roomKey, body.Player, models.WSMessage{
			Type:    result.Prompt.Type,
			Room:    roomKey,
			Player:  body.Player,
			Payload: promptPayload,
		})
	}

	if result.State.Phase == "playing" && result.State.PendingDefuse == nil && result.State.PendingGarbage == nil && result.State.NopeWindow == nil {
		timerSeconds := 0
		if body.Data != nil {
			var data struct {
				TurnTimer int `json:"turnTimer"`
			}
			if json.Unmarshal(body.Data, &data) == nil && data.TurnTimer > 0 {
				timerSeconds = data.TurnTimer
			}
		}
		if timerSeconds > 0 {
			go func() {
				game.StartTurnTimer(mu, gs, func() {
					log.Printf("[ek] turn timeout for room=%s player=%s", roomKey, gs.Turn)
					forceAction := game.GameAction{
						Action:  "forceDraw",
						Player:  gs.Turn,
						RoomKey: roomKey,
					}
					result := game.ProcessAction(gs, forceAction)
					if result != nil {
						h.setEKState(roomKey, result.State)
						h.saveEKState(roomKey, result.State)
						h.broadcastEKMessages(roomKey, room.ID, result.Messages)
					}
				}, timerSeconds)
			}()
		}
	}

	payload, _ := json.Marshal(result.State)
	c.JSON(http.StatusOK, gin.H{"ok": true, "state": json.RawMessage(payload)})
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

func (h *Handler) broadcastEKMessages(roomKey string, roomID int64, messages []game.WSMessage) {
	for _, msg := range messages {
		marshaled, _ := json.Marshal(msg.Payload)
		h.hub.BroadcastToRoom(roomKey, models.WSMessage{
			Type:    msg.Type,
			Room:    roomKey,
			Payload: marshaled,
		})
	}
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
	if playerName != "" {
		inRoom, err := h.store.IsPlayerInRoom(room.ID, playerName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
		if !inRoom {
			c.JSON(http.StatusForbidden, gin.H{"error": "player not in room"})
			return
		}
	}

	h.hub.CloseRoom(roomKey)
	h.removeEKState(roomKey)

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
