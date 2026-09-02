package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	NOPE_WINDOW_SECONDS = 8
	MAX_LOG_ENTRIES     = 50
	TURN_TIMER_INTERVAL = 1 * time.Second
)

type LogEntry struct {
	Name string    `json:"name"`
	Text string    `json:"text"`
	Ts   time.Time `json:"ts"`
}

type PlayerState struct {
	Color string   `json:"color"`
	Hand  []string `json:"hand"`
	Alive bool     `json:"alive"`
}

type EKGameState struct {
	Players          map[string]*PlayerState `json:"players"`
	Deck             []string                `json:"deck"`
	Discard          []string                `json:"discard"`
	Turn             string                  `json:"turn"`
	TurnOrder        []string                `json:"turnOrder"`
	Phase            string                  `json:"phase"`
	Winner           *string                 `json:"winner"`
	Log              []LogEntry              `json:"log"`
	AttackStack      int                     `json:"attackStack"`
	ReverseDirection bool                    `json:"reverseDirection"`

	NopeWindow     *NopeWindowState        `json:"NopeWindow,omitempty"`
	PendingDefuse  *DefuseState            `json:"PendingDefuse,omitempty"`
	PendingGarbage *GarbageCollectionState `json:"PendingGarbage,omitempty"`
	PendingFavor   *PendingFavorState      `json:"PendingFavor,omitempty"`
	TurnTimer      *time.Timer             `json:"-"`
	NopeTimer      *time.Timer             `json:"-"`
	CancelFunc     chan struct{}           `json:"-"`
}

type NopeWindowState struct {
	PlayerID       string     `json:"PlayerID"`
	CardID         string     `json:"CardID"`
	CardName       string     `json:"CardName"`
	Category       string     `json:"Category"`
	NopeCount      int        `json:"NopeCount"`
	LastNopePlayer string     `json:"LastNopePlayer"`
	ExpiredAt      *time.Time `json:"ExpiredAt,omitempty"`

	ComboCategory  string   `json:"ComboCategory,omitempty"`
	ComboCount     int      `json:"ComboCount,omitempty"`
	ComboCards     []string `json:"ComboCards,omitempty"`
	ComboTarget    string   `json:"ComboTarget,omitempty"`
	ComboCardID    string   `json:"ComboCardID,omitempty"`
	ComboDiscardID string   `json:"ComboDiscardID,omitempty"`

	FavorTargetID string `json:"FavorTargetID,omitempty"`
}

type DefuseState struct {
	PlayerID    string `json:"PlayerID"`
	ExplosiveID string `json:"ExplosiveID"`
}

type GarbageCollectionState struct {
	PlayerID  string            `json:"PlayerID"`
	Responded map[string]string `json:"Responded"`
}

type PendingFavorState struct {
	PlayerID string `json:"PlayerID"`
	TargetID string `json:"TargetID"`
}

type ActionResult struct {
	State      *EKGameState
	Messages   []WSMessage
	Prompt     *WSMessage
	NopeWindow bool
}

type WSMessage struct {
	Type    string      `json:"type"`
	Room    string      `json:"room"`
	Player  string      `json:"player"`
	Payload interface{} `json:"payload"`
}

type DefusePromptPayload struct {
	Player string `json:"player"`
	Card   string `json:"card"`
}

type GameAction struct {
	Action  string          `json:"action"`
	Data    json.RawMessage `json:"data"`
	Player  string          `json:"player"`
	RoomKey string          `json:"roomKey"`
}

func ProcessAction(gs *EKGameState, action GameAction) *ActionResult {
	switch action.Action {
	case "startGame":
		return handleStartGame(gs, action)
	case "drawCard":
		return handleDrawCard(gs, action)
	case "playCard":
		return handlePlayCard(gs, action)
	case "playNope":
		return handlePlayNope(gs, action)
	case "nopeResolved":
		return handleNopeResolved(gs, action)
	case "playCombo":
		return handlePlayCombo(gs, action)
	case "forceDraw":
		return handleForceDraw(gs, action)
	case "resolveDefuse":
		return handleResolveDefuse(gs, action)
	case "resolveGarbageCollection":
		return handleResolveGarbageCollection(gs, action)
	case "resolveFavor":
		return handleResolveFavor(gs, action)
	default:
		return &ActionResult{State: gs}
	}
}

func handleStartGame(gs *EKGameState, action GameAction) *ActionResult {
	var data struct {
		Players      map[string]*PlayerState `json:"players"`
		TurnOrder    []string                `json:"turnOrder"`
		HandSize     int                     `json:"handSize"`
		DefenseCount int                     `json:"defenseCount"`
		Multiplier   float64                 `json:"multiplier"`
		EnabledCats  map[string]bool         `json:"enabledCats"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return &ActionResult{State: gs}
	}

	multiplier := data.Multiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	allCards := BuildDeck(len(data.TurnOrder), multiplier, data.EnabledCats)
	explosives := []string{}
	deck := []string{}
	for _, c := range allCards {
		if GetCardCategory(c) == "explosive" {
			explosives = append(explosives, c)
		} else {
			deck = append(deck, c)
		}
	}

	hands := make(map[string][]string)
	defenseCount := data.DefenseCount
	if defenseCount <= 0 {
		defenseCount = 1
	}
	otherCards := data.HandSize - defenseCount
	if otherCards < 0 {
		otherCards = 0
	}

	for _, name := range data.TurnOrder {
		hand := []string{}
		for i := 0; i < defenseCount; i++ {
			hand = append(hand, RandomCardFromCategory("defense"))
		}
		for i := 0; i < otherCards; i++ {
			if len(deck) > 0 {
				hand = append(hand, deck[len(deck)-1])
				deck = deck[:len(deck)-1]
			}
		}
		hands[name] = hand
	}

	for _, c := range explosives {
		deck = append(deck, c)
	}
	deck = Shuffle(deck)

	gs.Players = make(map[string]*PlayerState)
	for _, name := range data.TurnOrder {
		color := ""
		if p, ok := data.Players[name]; ok {
			color = p.Color
		}
		gs.Players[name] = &PlayerState{
			Color: color,
			Hand:  hands[name],
			Alive: true,
		}
	}
	if len(data.TurnOrder) == 0 {
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "error", Payload: "Cannot start game with no players"},
			},
		}
	}

	gs.Deck = deck
	gs.Discard = []string{}
	gs.TurnOrder = data.TurnOrder
	gs.Turn = data.TurnOrder[0]
	gs.Phase = "playing"
	gs.Winner = nil
	gs.Log = []LogEntry{}
	gs.AttackStack = 0

	gs.addLog("system", fmt.Sprintf("Game started! %d chefs in the kitchen.", len(data.TurnOrder)))

	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handleDrawCard(gs *EKGameState, action GameAction) *ActionResult {
	if gs.Phase != "playing" || gs.Turn != action.Player {
		return &ActionResult{State: gs}
	}

	if len(gs.Deck) == 0 {
		gs.endGame(nil)
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	card := gs.Deck[len(gs.Deck)-1]
	gs.Deck = gs.Deck[:len(gs.Deck)-1]
	cardCat := GetCardCategory(card)

	if cardCat == "explosive" {
		gs.addLog(action.Player, "drew a card… and it was an Explosive card! 💥")

		player := gs.Players[action.Player]
		if player == nil {
			gs.eliminatePlayer(action.Player)
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}

		hasDefuse := false
		for _, c := range player.Hand {
			if GetCardCategory(c) == "defense" {
				hasDefuse = true
				break
			}
		}

		if hasDefuse {
			gs.PendingDefuse = &DefuseState{
				PlayerID:    action.Player,
				ExplosiveID: card,
			}
			prompt := &WSMessage{
				Type: "prompt_defuse",
				Payload: DefusePromptPayload{
					Player: action.Player,
					Card:   card,
				},
			}
			return &ActionResult{
				State:    gs,
				Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
				Prompt:   prompt,
			}
		} else {
			gs.eliminatePlayer(action.Player)
			gs.advanceTurn()
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
	} else {
		player := gs.Players[action.Player]
		if player != nil {
			player.Hand = append(player.Hand, card)
		}
		gs.addLog(action.Player, "drew a card.")
		if gs.AttackStack > 0 {
			gs.AttackStack--
		}
		if gs.AttackStack > 0 {
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		gs.advanceTurn()
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}
}

func handlePlayCard(gs *EKGameState, action GameAction) *ActionResult {
	if gs.Phase != "playing" || gs.Turn != action.Player {
		return &ActionResult{State: gs}
	}

	var data struct {
		Cards []string `json:"cards"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil || len(data.Cards) == 0 {
		return &ActionResult{State: gs}
	}

	player := gs.Players[action.Player]
	if player == nil {
		return &ActionResult{State: gs}
	}

	for _, cardID := range data.Cards {
		if GetCardCategory(cardID) == "explosive" {
			gs.addLog(action.Player, "can't play an Explosive card!")
			return &ActionResult{State: gs}
		}
	}

	cardID := data.Cards[0]
	cat := GetCardCategory(cardID)
	cardInfo := GetCardData(cardID)
	cardName := cardInfo.Name

	for _, cardID := range data.Cards {
		gs.Discard = append(gs.Discard, cardID)
	}
	removeFromHand(player, data.Cards)

	if cat == "nope" {
		gs.addLog(action.Player, fmt.Sprintf("played %s. (no effect on own turn) 🤚", cardName))
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	if IsNopeable(cat) && cat != "favor" {
		gs.addLog(action.Player, fmt.Sprintf("played %s. ⏳ Waiting for responses…", cardName))
		gs.NopeWindow = &NopeWindowState{
			PlayerID: action.Player,
			CardID:   cardID,
			CardName: cardName,
			Category: cat,
		}
		return &ActionResult{
			State:      gs,
			Messages:   []WSMessage{{Type: "state_updated", Payload: gs}},
			NopeWindow: true,
		}
	}

	switch cat {
	case "cat_cards":
		gs.addLog(action.Player, fmt.Sprintf("played %s. 🐱", cardName))
	case "skip":
		gs.handleSkip(action.Player)
	case "super_skip":
		gs.handleSuperSkip(action.Player)
	case "reverse":
		gs.handleReverse(action.Player)
	case "attack":
		gs.handleAttack(action.Player)
	case "future_vision":
		peekCount := min(3, len(gs.Deck))
		if peekCount > 0 {
			topCards := make([]string, peekCount)
			for i := 0; i < peekCount; i++ {
				topCards[i] = gs.Deck[len(gs.Deck)-1-i]
			}
			gs.addLog(action.Player, fmt.Sprintf("played %s. 🔮", cardName))
			return &ActionResult{
				State:      gs,
				Messages:   []WSMessage{{Type: "state_updated", Payload: gs}},
				NopeWindow: false,
				Prompt: &WSMessage{
					Type: "peek_cards",
					Payload: map[string]interface{}{
						"cards": topCards,
					},
				},
			}
		}
		gs.addLog(action.Player, fmt.Sprintf("played %s. 🔮", cardName))
	case "shuffle":
		gs.Deck = Shuffle(gs.Deck)
		gs.addLog(action.Player, fmt.Sprintf("shuffled the deck. 🔄"))
	case "draw_from_bottom":
		if len(gs.Deck) > 0 {
			bottom := gs.Deck[0]
			gs.Deck = gs.Deck[1:]
			player.Hand = append(player.Hand, bottom)
			gs.addLog(action.Player, "drew a card from the bottom of the deck! 📥")
		}
	case "swap_top_and_bottom":
		if len(gs.Deck) >= 2 {
			top := len(gs.Deck) - 1
			gs.Deck[0], gs.Deck[top] = gs.Deck[top], gs.Deck[0]
			gs.addLog(action.Player, "swapped the top and bottom cards of the deck. 🔃")
		}
	case "garbage_collection":
		alive := []string{}
		for _, name := range gs.TurnOrder {
			if p := gs.Players[name]; p != nil && p.Alive && len(p.Hand) > 0 {
				alive = append(alive, name)
			}
		}
		if len(alive) == 0 {
			gs.addLog(action.Player, "played Garbage Collection, but no one had cards to discard. 🗑️")
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		gs.PendingGarbage = &GarbageCollectionState{
			PlayerID:  action.Player,
			Responded: make(map[string]string),
		}
		gs.addLog(action.Player, "played Garbage Collection! Everyone must choose a card to put into the deck. 🗑️")
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	case "catomic_bomb":
		removed := 0
		removedCards := []string{}
		newDeck := []string{}
		for _, c := range gs.Deck {
			if GetCardCategory(c) == "explosive" {
				removed++
				removedCards = append(removedCards, c)
			} else {
				newDeck = append(newDeck, c)
			}
		}
		gs.Deck = append(newDeck, removedCards...)
		gs.addLog(action.Player, fmt.Sprintf("removed %d explosive card(s) and placed them on top of the deck! 💣", removed))
	case "mark":
		if len(gs.Deck) >= 3 {
			gs.addLog(action.Player, "marked the deck — peeked at the top 3 cards. 📍")
		}
	case "bury":
		if len(player.Hand) > 0 && len(gs.Deck) > 0 {
			buryIdx := rand.Intn(len(player.Hand))
			buriedCard := player.Hand[buryIdx]
			player.Hand = append(player.Hand[:buryIdx], player.Hand[buryIdx+1:]...)
			pos := rand.Intn(len(gs.Deck) + 1)
			gs.Deck = append(gs.Deck[:pos], append([]string{buriedCard}, gs.Deck[pos:]...)...)
			gs.addLog(action.Player, "buried a card in the deck. ⚰️")
		}
	case "dig_deeper":
		drawCount := min(3, len(gs.Deck))
		if drawCount > 0 {
			drawn := make([]string, drawCount)
			for i := 0; i < drawCount; i++ {
				drawn[i] = gs.Deck[len(gs.Deck)-1]
				gs.Deck = gs.Deck[:len(gs.Deck)-1]
			}
			keepIdx := rand.Intn(len(drawn))
			kept := drawn[keepIdx]
			player.Hand = append(player.Hand, kept)
			for i := len(drawn) - 1; i >= 0; i-- {
				if i != keepIdx {
					gs.Deck = append(gs.Deck, drawn[i])
				}
			}
			gs.addLog(action.Player, fmt.Sprintf("dug deeper and kept 1 of %d cards! ⛏️", drawCount))
		}
	case "favor":
		return gs.handleFavor(action.Player, cardName)
	case "clone":
		turnAdvanced := gs.handleClone(action.Player, cardName)
		if turnAdvanced {
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
	default:
		gs.addLog(action.Player, fmt.Sprintf("played %s.", cardName))
	}

	gs.advanceTurn()
	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handlePlayCombo(gs *EKGameState, action GameAction) *ActionResult {
	if gs.Phase != "playing" || gs.Turn != action.Player {
		return &ActionResult{State: gs}
	}

	player := gs.Players[action.Player]
	if player == nil {
		return &ActionResult{State: gs}
	}

	var data struct {
		Cards         []string `json:"cards"`
		Target        string   `json:"target"`
		CardID        string   `json:"cardId"`
		DiscardCardID string   `json:"discardCardId"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil || len(data.Cards) < 2 {
		return &ActionResult{State: gs}
	}

	combo := detectCombo(data.Cards)
	if combo == nil {
		return &ActionResult{State: gs}
	}

	for _, cardID := range combo.Cards {
		gs.Discard = append(gs.Discard, cardID)
	}
	removeFromHand(player, combo.Cards)

	catName := categoryNameMap[combo.Category]
	if catName == "" {
		catName = combo.Category
	}

	gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo. ⏳ Waiting for responses…", combo.Count, catName))
	gs.NopeWindow = &NopeWindowState{
		PlayerID:       action.Player,
		CardName:       fmt.Sprintf("%dx %s", combo.Count, catName),
		Category:       "combo",
		ComboCategory:  combo.Category,
		ComboCount:     combo.Count,
		ComboCards:     combo.Cards,
		ComboTarget:    data.Target,
		ComboCardID:    data.CardID,
		ComboDiscardID: data.DiscardCardID,
	}
	return &ActionResult{
		State:      gs,
		Messages:   []WSMessage{{Type: "state_updated", Payload: gs}},
		NopeWindow: true,
	}
}

func handlePlayComboInternal(gs *EKGameState, action GameAction, combo *ComboResult, target string, cardID string, discardCardID string) *ActionResult {
	player := gs.Players[action.Player]
	if player == nil {
		return &ActionResult{State: gs}
	}

	catName := categoryNameMap[combo.Category]
	if catName == "" {
		catName = combo.Category
	}

	if combo.Category == "combo_5x" {
		if discardCardID == "" {
			gs.addLog(action.Player, fmt.Sprintf("played 5x Rainbow combo, but didn't specify a card! 🌈"))
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		foundIdx := -1
		for i, c := range gs.Discard {
			if c == discardCardID {
				foundIdx = i
				break
			}
		}
		if foundIdx >= 0 {
			pickedCard := gs.Discard[foundIdx]
			gs.Discard = append(gs.Discard[:foundIdx], gs.Discard[foundIdx+1:]...)
			player.Hand = append(player.Hand, pickedCard)
			pickedData := GetCardData(pickedCard)
			pickedName := discardCardID
			if pickedData != nil {
				pickedName = pickedData.Name
			}
			gs.addLog(action.Player, fmt.Sprintf("played 5x Rainbow combo and picked %s from the discard pile! 🌈🎁", pickedName))
		} else {
			gs.addLog(action.Player, fmt.Sprintf("played 5x Rainbow combo targeting %s, but it wasn't in the discard pile! 🌈", discardCardID))
		}
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	alive := []string{}
	for _, name := range gs.TurnOrder {
		if name != action.Player && gs.Players[name] != nil && gs.Players[name].Alive {
			alive = append(alive, name)
		}
	}

	if target == "" || gs.Players[target] == nil || !gs.Players[target].Alive || target == action.Player {
		if len(alive) > 0 {
			target = alive[rand.Intn(len(alive))]
		} else {
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo! 🐱", combo.Count, catName))
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
	}

	targetHand := gs.Players[target].Hand

	if combo.Count >= 3 && cardID != "" {
		stolenIdx := -1
		for i, c := range targetHand {
			if c == cardID {
				stolenIdx = i
				break
			}
		}
		if stolenIdx >= 0 {
			stolenCard := targetHand[stolenIdx]
			gs.Players[target].Hand = append(targetHand[:stolenIdx], targetHand[stolenIdx+1:]...)
			player.Hand = append(player.Hand, stolenCard)
			stolenData := GetCardData(stolenCard)
			stolenName := cardID
			if stolenData != nil {
				stolenName = stolenData.Name
			}
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo and stole %s from %s! 🐱🎁", combo.Count, catName, stolenName, target))
		} else {
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo targeting %s, but they didn't have it! 🐱", combo.Count, catName, target))
		}
	} else {
		if len(targetHand) > 0 {
			stolenIdx := rand.Intn(len(targetHand))
			stolenCard := targetHand[stolenIdx]
			targetHand = append(targetHand[:stolenIdx], targetHand[stolenIdx+1:]...)
			gs.Players[target].Hand = targetHand
			player.Hand = append(player.Hand, stolenCard)
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo and stole a card from %s! 🐱🎁", combo.Count, catName, target))
		} else {
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo, but %s had no cards! 🐱", combo.Count, catName, target))
		}
	}

	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handlePlayNope(gs *EKGameState, action GameAction) *ActionResult {
	if gs.NopeWindow == nil {
		return &ActionResult{State: gs}
	}

	player := gs.Players[action.Player]
	if player == nil {
		return &ActionResult{State: gs}
	}

	if gs.NopeWindow.LastNopePlayer == action.Player {
		return &ActionResult{State: gs}
	}

	nopeIdx := -1
	for i, c := range player.Hand {
		if GetCardCategory(c) == "nope" {
			nopeIdx = i
			break
		}
	}
	if nopeIdx < 0 {
		return &ActionResult{State: gs}
	}

	nopeCard := player.Hand[nopeIdx]
	player.Hand = append(player.Hand[:nopeIdx], player.Hand[nopeIdx+1:]...)
	gs.Discard = append(gs.Discard, nopeCard)

	nopeInfo := GetCardData(nopeCard)
	nopeName := "Nope"
	if nopeInfo != nil {
		nopeName = nopeInfo.Name
	}

	gs.NopeWindow.NopeCount++
	gs.NopeWindow.LastNopePlayer = action.Player
	gs.addLog(action.Player, fmt.Sprintf("played %s! 🚫", nopeName))

	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handleNopeResolved(gs *EKGameState, action GameAction) *ActionResult {
	if gs.NopeWindow == nil {
		return &ActionResult{State: gs}
	}

	cancelled := gs.NopeWindow.NopeCount%2 == 1
	playerID := gs.NopeWindow.PlayerID
	cardName := gs.NopeWindow.CardName
	cat := gs.NopeWindow.Category

	comboCategory := gs.NopeWindow.ComboCategory
	comboCount := gs.NopeWindow.ComboCount
	comboCards := gs.NopeWindow.ComboCards
	comboTarget := gs.NopeWindow.ComboTarget
	comboCardID := gs.NopeWindow.ComboCardID
	comboDiscardID := gs.NopeWindow.ComboDiscardID

	gs.NopeWindow = nil

	if cancelled {
		gs.addLog("system", fmt.Sprintf("🚫 %s's %s was cancelled by a Nope!", playerID, cardName))
		if cat == "favor" {
			gs.PendingFavor = nil
		}
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	gs.addLog("system", fmt.Sprintf("✅ %s's %s goes through!", playerID, cardName))

	player := gs.Players[playerID]
	if player == nil {
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	switch cat {
	case "skip":
		gs.handleSkip(playerID)
	case "super_skip":
		gs.handleSuperSkip(playerID)
	case "reverse":
		gs.handleReverse(playerID)
	case "attack":
		gs.handleAttack(playerID)
	case "future_vision":
		peekCount := min(3, len(gs.Deck))
		var peekCards []string
		if peekCount > 0 {
			peekCards = make([]string, peekCount)
			for i := 0; i < peekCount; i++ {
				peekCards[i] = gs.Deck[len(gs.Deck)-1-i]
			}
		}
		gs.addLog(playerID, "peeked at the top 3 cards. 🔮")
		result := &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
		if len(peekCards) > 0 {
			result.Prompt = &WSMessage{
				Type: "peek_cards",
				Payload: map[string]interface{}{
					"cards": peekCards,
				},
			}
		}
		return result
	case "shuffle":
		gs.Deck = Shuffle(gs.Deck)
		gs.addLog(playerID, "shuffled the deck. 🔄")
	case "draw_from_bottom":
		if len(gs.Deck) > 0 {
			bottom := gs.Deck[0]
			gs.Deck = gs.Deck[1:]
			player.Hand = append(player.Hand, bottom)
			gs.addLog(playerID, "drew a card from the bottom of the deck! 📥")
		}
	case "swap_top_and_bottom":
		if len(gs.Deck) >= 2 {
			top := len(gs.Deck) - 1
			gs.Deck[0], gs.Deck[top] = gs.Deck[top], gs.Deck[0]
			gs.addLog(playerID, "swapped the top and bottom cards of the deck. 🔃")
		}
	case "garbage_collection":
		alive := []string{}
		for _, name := range gs.TurnOrder {
			if p := gs.Players[name]; p != nil && p.Alive && len(p.Hand) > 0 {
				alive = append(alive, name)
			}
		}
		if len(alive) == 0 {
			gs.addLog(playerID, "played Garbage Collection, but no one had cards to discard. 🗑️")
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		gs.PendingGarbage = &GarbageCollectionState{
			PlayerID:  playerID,
			Responded: make(map[string]string),
		}
		gs.addLog(playerID, "played Garbage Collection! Everyone must choose a card to put into the deck. 🗑️")
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	case "catomic_bomb":
		removed := 0
		removedCards := []string{}
		newDeck := []string{}
		for _, c := range gs.Deck {
			if GetCardCategory(c) == "explosive" {
				removed++
				removedCards = append(removedCards, c)
			} else {
				newDeck = append(newDeck, c)
			}
		}
		gs.Deck = append(newDeck, removedCards...)
		gs.addLog(playerID, fmt.Sprintf("removed %d explosive card(s) and placed them on top of the deck! 💣", removed))
	case "mark":
		if len(gs.Deck) >= 3 {
			gs.addLog(playerID, "marked the deck — peeked at the top 3 cards. 📍")
		}
	case "bury":
		if len(player.Hand) > 0 && len(gs.Deck) > 0 {
			buryIdx := rand.Intn(len(player.Hand))
			buriedCard := player.Hand[buryIdx]
			player.Hand = append(player.Hand[:buryIdx], player.Hand[buryIdx+1:]...)
			pos := rand.Intn(len(gs.Deck) + 1)
			gs.Deck = append(gs.Deck[:pos], append([]string{buriedCard}, gs.Deck[pos:]...)...)
			gs.addLog(playerID, "buried a card in the deck. ⚰️")
		}
	case "dig_deeper":
		drawCount := min(3, len(gs.Deck))
		if drawCount > 0 {
			drawn := make([]string, drawCount)
			for i := 0; i < drawCount; i++ {
				drawn[i] = gs.Deck[len(gs.Deck)-1]
				gs.Deck = gs.Deck[:len(gs.Deck)-1]
			}
			keepIdx := rand.Intn(len(drawn))
			kept := drawn[keepIdx]
			player.Hand = append(player.Hand, kept)
			for i := len(drawn) - 1; i >= 0; i-- {
				if i != keepIdx {
					gs.Deck = append(gs.Deck, drawn[i])
				}
			}
			gs.addLog(playerID, fmt.Sprintf("dug deeper and kept 1 of %d cards! ⛏️", drawCount))
		}
	case "favor":
		if gs.PendingFavor != nil && gs.PendingFavor.TargetID != "" {
			gs.addLog("system", fmt.Sprintf("%s must give a card to %s. 🎁", gs.PendingFavor.TargetID, playerID))
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		return gs.handleFavor(playerID, cardName)
	case "clone":
		turnAdvanced := gs.handleClone(playerID, cardName)
		if turnAdvanced {
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
	case "combo":
		combo := &ComboResult{
			Category: comboCategory,
			Count:    comboCount,
			Cards:    comboCards,
		}
		return handlePlayComboInternal(gs, GameAction{Action: "playCombo", Player: playerID}, combo, comboTarget, comboCardID, comboDiscardID)
	default:
		break
	}

	if cat != "skip" && cat != "super_skip" && cat != "reverse" && cat != "attack" {
		gs.advanceTurn()
	}

	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handleResolveDefuse(gs *EKGameState, action GameAction) *ActionResult {
	if gs.PendingDefuse == nil || gs.PendingDefuse.PlayerID != action.Player {
		return &ActionResult{State: gs}
	}

	var data struct {
		UseDefuse bool `json:"useDefuse"`
		Position  int  `json:"position"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil {
		gs.PendingDefuse = nil
		gs.eliminatePlayer(action.Player)
		gs.advanceTurn()
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	explosiveID := gs.PendingDefuse.ExplosiveID
	gs.PendingDefuse = nil

	if !data.UseDefuse {
		gs.eliminatePlayer(action.Player)
		gs.advanceTurn()
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	player := gs.Players[action.Player]
	if player == nil {
		gs.eliminatePlayer(action.Player)
		gs.advanceTurn()
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	defuseIdx := -1
	for i, c := range player.Hand {
		if GetCardCategory(c) == "defense" {
			defuseIdx = i
			break
		}
	}
	if defuseIdx < 0 {
		gs.eliminatePlayer(action.Player)
		gs.advanceTurn()
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	player.Hand = append(player.Hand[:defuseIdx], player.Hand[defuseIdx+1:]...)

	pos := len(gs.Deck) - data.Position
	if pos < 0 {
		pos = 0
	}
	if pos > len(gs.Deck) {
		pos = len(gs.Deck)
	}
	gs.Deck = append(gs.Deck[:pos], append([]string{explosiveID}, gs.Deck[pos:]...)...)

	gs.addLog(action.Player, "used a Defense card to defuse it! 🛡️")

	if gs.AttackStack > 0 {
		gs.AttackStack--
		if gs.AttackStack == 0 {
			gs.advanceTurn()
		}
	} else {
		gs.advanceTurn()
	}

	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handleResolveGarbageCollection(gs *EKGameState, action GameAction) *ActionResult {
	if gs.PendingGarbage == nil {
		return &ActionResult{State: gs}
	}

	player := gs.Players[action.Player]
	if player == nil || !player.Alive {
		return &ActionResult{State: gs}
	}

	var data struct {
		CardID string `json:"cardId"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return &ActionResult{State: gs}
	}

	if data.CardID == "" {
		return &ActionResult{State: gs}
	}

	cardIdx := -1
	for i, c := range player.Hand {
		if c == data.CardID {
			cardIdx = i
			break
		}
	}
	if cardIdx < 0 {
		return &ActionResult{State: gs}
	}

	card := player.Hand[cardIdx]
	player.Hand = append(player.Hand[:cardIdx], player.Hand[cardIdx+1:]...)
	gs.PendingGarbage.Responded[action.Player] = card

	cardData := GetCardData(card)
	cardName := card
	if cardData != nil {
		cardName = cardData.Name
	}
	gs.addLog(action.Player, fmt.Sprintf("chose %s to put into the deck. 🗑️", cardName))

	allResponded := true
	for _, name := range gs.TurnOrder {
		if p := gs.Players[name]; p != nil && p.Alive && len(p.Hand) > 0 {
			if _, ok := gs.PendingGarbage.Responded[name]; !ok {
				allResponded = false
				break
			}
		}
	}

	if allResponded {
		for _, cardID := range gs.PendingGarbage.Responded {
			pos := rand.Intn(len(gs.Deck) + 1)
			gs.Deck = append(gs.Deck[:pos], append([]string{cardID}, gs.Deck[pos:]...)...)
		}
		gs.Deck = Shuffle(gs.Deck)
		gs.PendingGarbage = nil
		gs.addLog("system", "All players have chosen! The deck has been shuffled. 🗑️🔄")
	}

	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handleForceDraw(gs *EKGameState, action GameAction) *ActionResult {
	if gs.Phase != "playing" || gs.Turn != action.Player {
		return &ActionResult{State: gs}
	}

	gs.addLog(action.Player, "ran out of time! ⏰ Forced to draw.")
	return handleDrawCard(gs, GameAction{
		Action: "drawCard",
		Player: action.Player,
	})
}

type ComboResult struct {
	Category string
	Count    int
	Cards    []string
}

func detectCombo(cards []string) *ComboResult {
	if len(cards) < 2 {
		return nil
	}

	for _, c := range cards {
		if GetCardCategory(c) == "explosive" {
			return nil
		}
	}

	if len(cards) == 5 {
		allDifferent := true
		seenIDs := map[string]bool{}
		seenCats := map[string]bool{}
		for _, c := range cards {
			cat := GetCardCategory(c)
			if cat == "cat_cards" {
				data := GetCardData(c)
				key := ""
				if data != nil {
					key = data.ID
				} else {
					key = c
				}
				if seenIDs[key] {
					allDifferent = false
					break
				}
				seenIDs[key] = true
			} else {
				if seenCats[cat] {
					allDifferent = false
					break
				}
				seenCats[cat] = true
			}
		}
		if allDifferent {
			return &ComboResult{
				Category: "combo_5x",
				Count:    5,
				Cards:    cards,
			}
		}
	}

	catCounts := map[string][]string{}
	for _, c := range cards {
		cat := GetCardCategory(c)
		catCounts[cat] = append(catCounts[cat], c)
	}

	for cat, catCards := range catCounts {
		if len(catCards) < 2 {
			continue
		}

		if cat == "cat_cards" {
			idCounts := map[string][]string{}
			for _, c := range catCards {
				data := GetCardData(c)
				if data != nil {
					idCounts[data.ID] = append(idCounts[data.ID], c)
				}
			}
			for _, idCards := range idCounts {
				if len(idCards) >= 2 {
					return &ComboResult{
						Category: cat,
						Count:    len(idCards),
						Cards:    idCards,
					}
				}
			}
		} else {
			return &ComboResult{
				Category: cat,
				Count:    len(catCards),
				Cards:    catCards,
			}
		}
	}
	return nil
}

func (gs *EKGameState) handleSkip(playerID string) {
	if gs.AttackStack > 0 {
		gs.AttackStack--
		if gs.AttackStack == 0 {
			gs.advanceTurn()
		}
	} else {
		gs.advanceTurn()
	}
	gs.addLog(playerID, "skipped their turn. ⏭️")
}

func (gs *EKGameState) handleSuperSkip(playerID string) {
	gs.AttackStack = 0
	gs.advanceTurn()
	gs.addLog(playerID, "used Super Skip! All forced turns cancelled. ⏭️✨")
}

func (gs *EKGameState) handleReverse(playerID string) {
	gs.ReverseDirection = !gs.ReverseDirection
	if gs.AttackStack > 0 {
		gs.AttackStack--
	}
	gs.advanceTurn()
	direction := "forward"
	if gs.ReverseDirection {
		direction = "reverse"
	}
	gs.addLog(playerID, fmt.Sprintf("reversed the turn order! Now playing %s. 🔄", direction))
}

func (gs *EKGameState) handleAttack(playerID string) {
	gs.AttackStack += 2
	gs.advanceTurn()
	gs.addLog(playerID, "played Attack! ⚔️")
}

func (gs *EKGameState) advanceTurn() {
	if len(gs.TurnOrder) == 0 {
		return
	}
	idx := -1
	for i, name := range gs.TurnOrder {
		if name == gs.Turn {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}

	startIdx := idx
	for {
		if gs.ReverseDirection {
			idx = (idx - 1 + len(gs.TurnOrder)) % len(gs.TurnOrder)
		} else {
			idx = (idx + 1) % len(gs.TurnOrder)
		}
		if idx == startIdx {
			return
		}
		if gs.Players[gs.TurnOrder[idx]] != nil && gs.Players[gs.TurnOrder[idx]].Alive {
			gs.Turn = gs.TurnOrder[idx]
			return
		}
	}
}

func (gs *EKGameState) eliminatePlayer(name string) {
	player := gs.Players[name]
	if player == nil {
		return
	}
	player.Alive = false
	gs.Discard = append(gs.Discard, player.Hand...)
	player.Hand = []string{}
	gs.addLog(name, "was eliminated! 💀")

	alive := []string{}
	for _, n := range gs.TurnOrder {
		if gs.Players[n] != nil && gs.Players[n].Alive {
			alive = append(alive, n)
		}
	}
	if len(alive) <= 1 {
		var winner *string
		if len(alive) == 1 {
			winner = &alive[0]
		}
		gs.endGame(winner)
	}
}

func (gs *EKGameState) endGame(winner *string) {
	gs.Phase = "ended"
	gs.Winner = winner
	if winner != nil {
		gs.addLog("system", fmt.Sprintf("🎉 %s wins the game!", *winner))
	} else {
		gs.addLog("system", "💀 Everyone was eliminated! No winner.")
	}
}

func (gs *EKGameState) addLog(name string, text string) {
	gs.Log = append(gs.Log, LogEntry{Name: name, Text: text, Ts: time.Now()})
	if len(gs.Log) > MAX_LOG_ENTRIES {
		gs.Log = gs.Log[len(gs.Log)-MAX_LOG_ENTRIES:]
	}
}

func removeFromHand(player *PlayerState, cards []string) {
	cardSet := map[string]int{}
	for _, c := range cards {
		cardSet[c]++
	}
	newHand := []string{}
	for _, c := range player.Hand {
		if count, ok := cardSet[c]; ok && count > 0 {
			cardSet[c]--
			continue
		}
		newHand = append(newHand, c)
	}
	player.Hand = newHand
}

func StartNopeWindowTimer(mu *sync.Mutex, gs *EKGameState, processNopeResolved func(), roomKey string) {
	if gs.NopeWindow == nil {
		return
	}

	expiresAt := time.Now().Add(NOPE_WINDOW_SECONDS * time.Second)
	gs.NopeWindow.ExpiredAt = &expiresAt

	timer := time.AfterFunc(NOPE_WINDOW_SECONDS*time.Second, func() {
		mu.Lock()
		defer mu.Unlock()

		if gs.NopeWindow == nil {
			return
		}

		gs.NopeTimer = nil
		processNopeResolved()
	})
	gs.NopeTimer = timer
}

func StartTurnTimer(mu *sync.Mutex, gs *EKGameState, forceDraw func(), timerSeconds int) {
	if gs.TurnTimer != nil {
		gs.TurnTimer.Stop()
		gs.TurnTimer = nil
	}

	if timerSeconds <= 0 || gs.Phase != "playing" {
		return
	}

	gs.TurnTimer = time.AfterFunc(time.Duration(timerSeconds)*time.Second, func() {
		mu.Lock()
		defer mu.Unlock()

		if gs.Phase != "playing" || gs.NopeWindow != nil || gs.PendingDefuse != nil || gs.PendingFavor != nil {
			return
		}

		gs.TurnTimer = nil
		forceDraw()
	})
}

func CancelPendingAction(gs *EKGameState) {
	if gs.CancelFunc != nil {
		close(gs.CancelFunc)
		gs.CancelFunc = nil
	}
	if gs.NopeTimer != nil {
		gs.NopeTimer.Stop()
		gs.NopeTimer = nil
	}
	if gs.TurnTimer != nil {
		gs.TurnTimer.Stop()
		gs.TurnTimer = nil
	}
	gs.NopeWindow = nil
	gs.PendingDefuse = nil
	gs.PendingFavor = nil
}

func (gs *EKGameState) handleFavor(playerID string, cardName string) *ActionResult {
	alive := []string{}
	for _, name := range gs.TurnOrder {
		if name != playerID && gs.Players[name] != nil && gs.Players[name].Alive && len(gs.Players[name].Hand) > 0 {
			alive = append(alive, name)
		}
	}
	if len(alive) == 0 {
		gs.addLog(playerID, fmt.Sprintf("played %s, but no one had cards to give. 🎁", cardName))
		gs.advanceTurn()
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}
	gs.PendingFavor = &PendingFavorState{
		PlayerID: playerID,
	}
	gs.addLog(playerID, fmt.Sprintf("played %s! Choose a target. 🎁", cardName))
	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func handleResolveFavor(gs *EKGameState, action GameAction) *ActionResult {
	if gs.PendingFavor == nil {
		return &ActionResult{State: gs}
	}

	var data struct {
		TargetID string `json:"targetId"`
		CardID   string `json:"cardId"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return &ActionResult{State: gs}
	}

	// Phase 1: Favor player selects target
	if gs.PendingFavor.TargetID == "" {
		if gs.PendingFavor.PlayerID != action.Player {
			return &ActionResult{State: gs}
		}
		if data.TargetID == "" {
			return &ActionResult{State: gs}
		}
		target := gs.Players[data.TargetID]
		if target == nil || !target.Alive || len(target.Hand) == 0 {
			gs.addLog(action.Player, fmt.Sprintf("tried to Favor %s, but they have no cards!", data.TargetID))
			gs.PendingFavor = nil
			gs.advanceTurn()
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		gs.PendingFavor.TargetID = data.TargetID
		gs.addLog("system", fmt.Sprintf("%s chose %s as target. Others can Nope. ⏳", action.Player, data.TargetID))
		gs.NopeWindow = &NopeWindowState{
			PlayerID:      action.Player,
			CardID:        "",
			CardName:      "Favor",
			Category:      "favor",
			FavorTargetID: data.TargetID,
		}
		return &ActionResult{
			State:      gs,
			Messages:   []WSMessage{{Type: "state_updated", Payload: gs}},
			NopeWindow: true,
		}
	}

	// Phase 2: Target player selects a card to give
	if gs.PendingFavor.TargetID != action.Player {
		return &ActionResult{State: gs}
	}
	if gs.NopeWindow != nil {
		return &ActionResult{State: gs}
	}

	targetID := gs.PendingFavor.TargetID
	favorPlayerID := gs.PendingFavor.PlayerID
	target := gs.Players[targetID]
	favorPlayer := gs.Players[favorPlayerID]
	if target == nil || !target.Alive || favorPlayer == nil || !favorPlayer.Alive {
		gs.PendingFavor = nil
		return &ActionResult{
			State: gs,
			Messages: []WSMessage{
				{Type: "state_updated", Payload: gs},
			},
		}
	}

	if data.CardID == "" {
		return &ActionResult{State: gs}
	}

	found := false
	for _, c := range target.Hand {
		if c == data.CardID {
			found = true
			break
		}
	}
	if !found {
		return &ActionResult{State: gs}
	}

	removeFromHand(target, []string{data.CardID})
	favorPlayer.Hand = append(favorPlayer.Hand, data.CardID)
	gs.addLog(action.Player, fmt.Sprintf("gave a card to %s! 🎁", favorPlayerID))
	gs.PendingFavor = nil
	return &ActionResult{
		State: gs,
		Messages: []WSMessage{
			{Type: "state_updated", Payload: gs},
		},
	}
}

func findOriginalCard(gs *EKGameState, cloneIndex int) string {
	for i := cloneIndex - 1; i >= 0; i-- {
		cardID := gs.Discard[i]
		if GetCardCategory(cardID) != "clone" {
			return cardID
		}
	}
	return ""
}

func (gs *EKGameState) handleClone(playerID string, cardName string) bool {
	if len(gs.Discard) < 2 {
		gs.addLog(playerID, fmt.Sprintf("played %s, but there was nothing to clone. 👻", cardName))
		return false
	}
	originalCard := findOriginalCard(gs, len(gs.Discard)-1)
	if originalCard == "" {
		gs.addLog(playerID, fmt.Sprintf("played %s, but there was nothing to clone. 👻", cardName))
		return false
	}
	originalCat := GetCardCategory(originalCard)
	gs.addLog(playerID, fmt.Sprintf("played %s, copying %s! 👻", cardName, originalCat))
	return executeCardEffect(gs, playerID, originalCat)
}

func executeCardEffect(gs *EKGameState, playerID string, cat string) bool {
	player := gs.Players[playerID]
	if player == nil {
		return false
	}

	switch cat {
	case "skip":
		gs.handleSkip(playerID)
		return true
	case "super_skip":
		gs.handleSuperSkip(playerID)
		return true
	case "reverse":
		gs.handleReverse(playerID)
		return true
	case "attack":
		gs.handleAttack(playerID)
		return true
	case "future_vision":
		peekCount := min(3, len(gs.Deck))
		if peekCount > 0 {
			topCards := make([]string, peekCount)
			for i := 0; i < peekCount; i++ {
				topCards[i] = gs.Deck[len(gs.Deck)-1-i]
			}
			gs.addLog(playerID, "cloned See the Future. 🔮")
		}
	case "shuffle":
		gs.Deck = Shuffle(gs.Deck)
		gs.addLog(playerID, "cloned Shuffle. 🔄")
	case "draw_from_bottom":
		if len(gs.Deck) > 0 {
			bottom := gs.Deck[0]
			gs.Deck = gs.Deck[1:]
			player.Hand = append(player.Hand, bottom)
			gs.addLog(playerID, "cloned Draw From Bottom! 📥")
		}
	case "swap_top_and_bottom":
		if len(gs.Deck) >= 2 {
			top := len(gs.Deck) - 1
			gs.Deck[0], gs.Deck[top] = gs.Deck[top], gs.Deck[0]
			gs.addLog(playerID, "cloned Swap Top & Bottom. 🔃")
		}
	case "catomic_bomb":
		removed := 0
		removedCards := []string{}
		newDeck := []string{}
		for _, c := range gs.Deck {
			if GetCardCategory(c) == "explosive" {
				removed++
				removedCards = append(removedCards, c)
			} else {
				newDeck = append(newDeck, c)
			}
		}
		gs.Deck = append(newDeck, removedCards...)
		gs.addLog(playerID, fmt.Sprintf("cloned Catomic Bomb! Removed %d explosive(s). 💣", removed))
	case "bury":
		if len(player.Hand) > 0 && len(gs.Deck) > 0 {
			buryIdx := rand.Intn(len(player.Hand))
			buriedCard := player.Hand[buryIdx]
			player.Hand = append(player.Hand[:buryIdx], player.Hand[buryIdx+1:]...)
			pos := rand.Intn(len(gs.Deck) + 1)
			gs.Deck = append(gs.Deck[:pos], append([]string{buriedCard}, gs.Deck[pos:]...)...)
			gs.addLog(playerID, "cloned Bury. ⚰️")
		}
	case "dig_deeper":
		drawCount := min(3, len(gs.Deck))
		if drawCount > 0 {
			drawn := make([]string, drawCount)
			for i := 0; i < drawCount; i++ {
				drawn[i] = gs.Deck[len(gs.Deck)-1]
				gs.Deck = gs.Deck[:len(gs.Deck)-1]
			}
			keepIdx := rand.Intn(len(drawn))
			kept := drawn[keepIdx]
			player.Hand = append(player.Hand, kept)
			for i := len(drawn) - 1; i >= 0; i-- {
				if i != keepIdx {
					gs.Deck = append(gs.Deck, drawn[i])
				}
			}
			gs.addLog(playerID, fmt.Sprintf("cloned Dig Deeper and kept 1 of %d cards! ⛏️", drawCount))
		}
	case "garbage_collection":
		alive := []string{}
		for _, name := range gs.TurnOrder {
			if p := gs.Players[name]; p != nil && p.Alive && len(p.Hand) > 0 {
				alive = append(alive, name)
			}
		}
		if len(alive) == 0 {
			gs.addLog(playerID, "cloned Garbage Collection, but no one had cards. 🗑️")
			return false
		}
		gs.PendingGarbage = &GarbageCollectionState{
			PlayerID:  playerID,
			Responded: make(map[string]string),
		}
		gs.addLog(playerID, "cloned Garbage Collection! Everyone must discard. 🗑️")
	case "mark":
		if len(gs.Deck) >= 3 {
			gs.addLog(playerID, "cloned Mark. 📍")
		}
	case "favor":
		gs.addLog(playerID, "cloned Favor! 🎁")
	default:
		gs.addLog(playerID, fmt.Sprintf("played Clone, copying %s.", cat))
	}
	return false
}

func ResetTurnTimer(gs *EKGameState, timerSeconds int) {
	if gs.TurnTimer != nil {
		gs.TurnTimer.Stop()
		gs.TurnTimer = nil
	}

	if timerSeconds <= 0 || gs.Phase != "playing" {
		return
	}

	gs.TurnTimer = time.AfterFunc(time.Duration(timerSeconds)*time.Second, nil)
}
