package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	NOPE_WINDOW_SECONDS = 8
	MAX_LOG_ENTRIES     = 50

	// How long the table waits for a player to answer a prompt — a defuse, a
	// Favor, a Garbage Collection pick — before deciding for them. Nothing else
	// can happen while one is pending, so without this one closed tab is enough
	// to stall the game permanently.
	PENDING_TIMEOUT_SECONDS = 30
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
	Winners          []string                `json:"winners,omitempty"`
	Log              []LogEntry              `json:"log"`
	AttackStack      int                     `json:"attackStack"`
	ReverseDirection bool                    `json:"reverseDirection"`

	// When the current turn is forced to draw, for clients to count down to.
	TurnEndsAt *time.Time `json:"turnEndsAt,omitempty"`

	// When this game was dealt. With the room, it names the game in the
	// match history.
	StartedAt time.Time `json:"startedAt"`

	// The category of the last card whose effect actually happened. Clone used
	// to copy whatever sat on top of the discard pile, which is just as often
	// a spent Nope or a dead player's dumped hand.
	LastPlayed string `json:"lastPlayed,omitempty"`

	NopeWindow     *NopeWindowState        `json:"NopeWindow,omitempty"`
	PendingDefuse  *DefuseState            `json:"PendingDefuse,omitempty"`
	PendingGarbage *GarbageCollectionState `json:"PendingGarbage,omitempty"`
	PendingFavor   *PendingFavorState      `json:"PendingFavor,omitempty"`
	PendingChoice  *PendingChoiceState     `json:"PendingChoice,omitempty"`
	TurnTimer      *time.Timer             `json:"-"`
	NopeTimer      *time.Timer             `json:"-"`
	PendingTimer   *time.Timer             `json:"-"`
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

	// Who has already answered this window with a pass. Everyone who may
	// respond gets to pass, Nope in hand or not, so a window that closes
	// early says nothing about anybody's cards.
	Passed []string `json:"Passed,omitempty"`

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

// PendingChoiceState is a card waiting on a decision only one player can make:
// which of the cards Dig Deeper turned up to keep, or where to put the card
// Bury took off the top. Both used to decide at random on the player's behalf,
// which is the entire point of those two cards.
type PendingChoiceState struct {
	PlayerID string   `json:"PlayerID"`
	Kind     string   `json:"Kind"`
	Cards    []string `json:"Cards,omitempty"`
	DeckSize int      `json:"DeckSize"`
}

type ActionResult struct {
	State      *EKGameState
	Messages   []WSMessage
	Prompt     *WSMessage
	NopeWindow bool
	Error      string
}

// reject refuses an action without touching the state. The message is shown to
// the player who tried it, so it is phrased for them and not for a log.
func reject(gs *EKGameState, msg string) *ActionResult {
	return &ActionResult{State: gs, Error: msg}
}

// pendingReason names the resolution the table is waiting on, if any. Nothing
// else may happen until it clears, or effects resolve twice.
func (gs *EKGameState) pendingReason() string {
	switch {
	case gs.NopeWindow != nil:
		return "wait for the Nope window to close"
	case gs.PendingDefuse != nil:
		return "someone is defusing an Explosive card"
	case gs.PendingGarbage != nil:
		return "waiting for everyone to pick a card for the deck"
	case gs.PendingFavor != nil:
		return "waiting for a Favor to be given"
	case gs.PendingChoice != nil:
		return "waiting for a card to be placed"
	}
	return ""
}

// validateCards reports why cards cannot be played from this hand, or "" when
// every one of them is a real, non-explosive card the player actually holds.
func validateCards(player *PlayerState, cards []string) string {
	held := map[string]int{}
	for _, c := range player.Hand {
		held[c]++
	}
	for _, c := range cards {
		if GetCardData(c) == nil {
			return "that card doesn't exist"
		}
		if GetCardCategory(c) == "explosive" {
			return "you can't play an Explosive card"
		}
		if held[c] == 0 {
			return "that card isn't in your hand"
		}
		held[c]--
	}
	return ""
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
	case "passNope":
		return handlePassNope(gs, action)
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
	case "resolveChoice":
		return handleResolveChoice(gs, action)
	default:
		return &ActionResult{State: gs}
	}
}

func handleStartGame(gs *EKGameState, action GameAction) *ActionResult {
	var data struct {
		Players        map[string]*PlayerState `json:"players"`
		TurnOrder      []string                `json:"turnOrder"`
		HandSize       int                     `json:"handSize"`
		DefenseCount   int                     `json:"defenseCount"`
		Multiplier     float64                 `json:"multiplier"`
		EnabledCats    map[string]bool         `json:"enabledCats"`
		ExplosiveCount int                     `json:"explosiveCount"`
	}
	data.ExplosiveCount = -1 // -1 means "one fewer than the number of players"
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return &ActionResult{State: gs}
	}

	if gs.Phase == "playing" {
		return reject(gs, "a game is already in progress")
	}

	multiplier := data.Multiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	allCards := BuildDeck(len(data.TurnOrder), multiplier, data.EnabledCats, data.ExplosiveCount)
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
	gs.StartedAt = time.Now()
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
	if gs.Phase != "playing" {
		return reject(gs, "the game isn't running")
	}
	if gs.Turn != action.Player {
		return reject(gs, "it's not your turn")
	}
	if busy := gs.pendingReason(); busy != "" {
		return reject(gs, busy)
	}
	if p := gs.Players[action.Player]; p == nil || !p.Alive {
		return reject(gs, "you're out of the game")
	}

	if len(gs.Deck) == 0 {
		// Nobody exploded — everyone who is still in it has survived the deck.
		gs.endGame(gs.alivePlayers())
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
				Type:   "prompt_defuse",
				Player: action.Player,
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
	if gs.Phase != "playing" {
		return reject(gs, "the game isn't running")
	}
	if gs.Turn != action.Player {
		return reject(gs, "it's not your turn")
	}
	if busy := gs.pendingReason(); busy != "" {
		return reject(gs, busy)
	}

	var data struct {
		Cards []string `json:"cards"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil || len(data.Cards) == 0 {
		return reject(gs, "no card was played")
	}
	if len(data.Cards) > 1 {
		return reject(gs, "those cards don't make a combo — play one card at a time")
	}

	player := gs.Players[action.Player]
	if player == nil || !player.Alive {
		return reject(gs, "you're out of the game")
	}

	if why := validateCards(player, data.Cards); why != "" {
		return reject(gs, why)
	}

	cardID := data.Cards[0]
	cat := GetCardCategory(cardID)
	cardName := CardLabel(cardID)

	if cat == "defense" {
		return reject(gs, "a Defuse card is only used when you draw an Exploding Kitten")
	}

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
		return gs.openNopeWindow(&NopeWindowState{
			PlayerID: action.Player,
			CardID:   cardID,
			CardName: cardName,
			Category: cat,
		}, fmt.Sprintf("played %s", cardName))
	}

	// Everything nopeable resolves through the window above, so only the cards
	// nobody can answer are left here.
	gs.LastPlayed = cat

	switch cat {
	case "cat_cards":
		// One cat card on its own does nothing — and does not end the turn.
		// Two of a kind is a combo.
		gs.addLog(action.Player, fmt.Sprintf("played %s. 🐱", cardName))
	case "favor":
		return gs.handleFavor(action.Player, cardName)
	default:
		gs.addLog(action.Player, fmt.Sprintf("played %s.", cardName))
	}

	return &ActionResult{
		State:    gs,
		Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
	}
}

func handlePlayCombo(gs *EKGameState, action GameAction) *ActionResult {
	if gs.Phase != "playing" {
		return reject(gs, "the game isn't running")
	}
	if gs.Turn != action.Player {
		return reject(gs, "it's not your turn")
	}
	if busy := gs.pendingReason(); busy != "" {
		return reject(gs, busy)
	}

	player := gs.Players[action.Player]
	if player == nil || !player.Alive {
		return reject(gs, "you're out of the game")
	}

	var data struct {
		Cards         []string `json:"cards"`
		Target        string   `json:"target"`
		CardID        string   `json:"cardId"`
		DiscardCardID string   `json:"discardCardId"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil || len(data.Cards) < 2 {
		return reject(gs, "a combo needs at least two cards")
	}

	if why := validateCards(player, data.Cards); why != "" {
		return reject(gs, why)
	}

	combo := detectCombo(data.Cards)
	if combo == nil {
		return reject(gs, "those cards don't make a combo")
	}

	for _, cardID := range combo.Cards {
		gs.Discard = append(gs.Discard, cardID)
	}
	removeFromHand(player, combo.Cards)

	catName := categoryNameMap[combo.Category]
	if catName == "" {
		catName = combo.Category
	}

	return gs.openNopeWindow(&NopeWindowState{
		PlayerID:       action.Player,
		CardName:       fmt.Sprintf("%dx %s", combo.Count, catName),
		Category:       "combo",
		ComboCategory:  combo.Category,
		ComboCount:     combo.Count,
		ComboCards:     combo.Cards,
		ComboTarget:    data.Target,
		ComboCardID:    data.CardID,
		ComboDiscardID: data.DiscardCardID,
	}, fmt.Sprintf("played a %dx %s combo", combo.Count, catName))
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
			pickedName := CardLabel(pickedCard)
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
			if cardsMatch(cardID, c) {
				stolenIdx = i
				break
			}
		}
		if stolenIdx >= 0 {
			stolenCard := targetHand[stolenIdx]
			gs.Players[target].Hand = append(targetHand[:stolenIdx], targetHand[stolenIdx+1:]...)
			player.Hand = append(player.Hand, stolenCard)
			stolenName := CardLabel(stolenCard)
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo and stole %s from %s! 🐱🎁", combo.Count, catName, stolenName, target))
		} else {
			gs.addLog(action.Player, fmt.Sprintf("played %dx %s combo asking %s for a %s, but they didn't have one! 🐱", combo.Count, catName, target, CardLabel(cardID)))
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

// nopeResponders lists the living players who may still answer the open
// window. It deliberately ignores what anyone is holding: the window is armed
// for everyone who could respond, so its mere presence — and its length — never
// betrays who has a Nope. The player who made the play counts only once they
// have been Noped, since you may counter a Nope but not Nope yourself.
func (gs *EKGameState) nopeResponders() []string {
	if gs.NopeWindow == nil {
		return nil
	}
	var out []string
	for name, p := range gs.Players {
		if p == nil || !p.Alive {
			continue
		}
		if name == gs.NopeWindow.LastNopePlayer {
			continue
		}
		if name == gs.NopeWindow.PlayerID && gs.NopeWindow.NopeCount == 0 {
			continue
		}
		out = append(out, name)
	}
	return out
}

// everyoneHasPassed reports whether every player who could answer the window
// has said they will not. That — not an empty-handed table — is what closes the
// window ahead of its countdown.
func (gs *EKGameState) everyoneHasPassed() bool {
	responders := gs.nopeResponders()
	if len(responders) == 0 {
		return true
	}
	for _, name := range responders {
		if !contains(gs.NopeWindow.Passed, name) {
			return false
		}
	}
	return true
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// openNopeWindow arms the window for a play. It runs for every nopeable play,
// even when nobody at the table is holding a Nope: a play that resolved
// instantly used to announce, to the whole table, that no Nope was out there.
// The countdown is cut short only by the players themselves passing.
func (gs *EKGameState) openNopeWindow(nw *NopeWindowState, played string) *ActionResult {
	gs.NopeWindow = nw
	if len(gs.nopeResponders()) == 0 {
		gs.addLog(nw.PlayerID, played+".")
		return resolveNopeWindow(gs, false)
	}
	gs.addLog(nw.PlayerID, played+". ⏳ Waiting for responses…")
	return &ActionResult{
		State:      gs,
		Messages:   []WSMessage{{Type: "state_updated", Payload: gs}},
		NopeWindow: true,
	}
}

func handlePlayNope(gs *EKGameState, action GameAction) *ActionResult {
	if gs.NopeWindow == nil {
		return reject(gs, "there's nothing to Nope right now")
	}

	player := gs.Players[action.Player]
	if player == nil || !player.Alive {
		return reject(gs, "you're out of the game")
	}

	if gs.NopeWindow.LastNopePlayer == action.Player {
		return reject(gs, "you can't Nope twice in a row")
	}
	if action.Player == gs.NopeWindow.PlayerID && gs.NopeWindow.NopeCount == 0 {
		return reject(gs, "you can't Nope your own card")
	}

	nopeIdx := -1
	for i, c := range player.Hand {
		if GetCardCategory(c) == "nope" {
			nopeIdx = i
			break
		}
	}
	if nopeIdx < 0 {
		return reject(gs, "you don't have a Nope card")
	}

	nopeCard := player.Hand[nopeIdx]
	player.Hand = append(player.Hand[:nopeIdx], player.Hand[nopeIdx+1:]...)
	gs.Discard = append(gs.Discard, nopeCard)

	nopeName := CardLabel(nopeCard)

	gs.NopeWindow.NopeCount++
	gs.NopeWindow.LastNopePlayer = action.Player
	gs.addLog(action.Player, fmt.Sprintf("played %s! 🚫", nopeName))

	// A Nope reopens the window: whoever it was played against gets a fair
	// chance to counter it, instead of racing the original countdown. Passes
	// from the previous round are spent with it.
	gs.NopeWindow.Passed = nil
	if len(gs.nopeResponders()) == 0 {
		return resolveNopeWindow(gs, true)
	}
	return &ActionResult{
		State:      gs,
		Messages:   []WSMessage{{Type: "state_updated", Payload: gs}},
		NopeWindow: true,
	}
}

// handlePassNope records that a player is letting the play through. Once
// everyone who could answer has passed, the window closes without waiting out
// the rest of the countdown.
func handlePassNope(gs *EKGameState, action GameAction) *ActionResult {
	if gs.NopeWindow == nil {
		return reject(gs, "there's nothing to pass on right now")
	}
	if !contains(gs.nopeResponders(), action.Player) {
		return reject(gs, "it's not your call on this one")
	}
	if contains(gs.NopeWindow.Passed, action.Player) {
		return &ActionResult{State: gs, NopeWindow: false}
	}

	gs.NopeWindow.Passed = append(gs.NopeWindow.Passed, action.Player)

	if gs.everyoneHasPassed() {
		return resolveNopeWindow(gs, true)
	}
	return &ActionResult{
		State:    gs,
		Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
	}
}

func handleNopeResolved(gs *EKGameState, action GameAction) *ActionResult {
	return resolveNopeWindow(gs, true)
}

// resolveNopeWindow applies (or cancels) the pending play. announce is false
// when the window was never really open, so the log does not narrate a
// countdown nobody saw.
func resolveNopeWindow(gs *EKGameState, announce bool) *ActionResult {
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

	if announce {
		gs.addLog("system", fmt.Sprintf("✅ %s's %s goes through!", playerID, cardName))
	}

	// What a later Clone will copy.
	if cat != "clone" && cat != "combo" {
		gs.LastPlayed = cat
	}

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
		gs.addLog(playerID, "peeked at the top 3 cards. 🔮")
		return &ActionResult{
			State:    gs,
			Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
			Prompt:   gs.peekTop(playerID, 3),
		}
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
		gs.addLog(playerID, "marked the deck — peeked at the top 3 cards. 📍")
		return &ActionResult{
			State:    gs,
			Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
			Prompt:   gs.peekTop(playerID, 3),
		}
	case "bury":
		return gs.startBury(playerID)
	case "dig_deeper":
		return gs.startDigDeeper(playerID)
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
		turnAdvanced, prompt := gs.handleClone(playerID, cardName)
		if turnAdvanced || prompt != nil {
			return &ActionResult{
				State:    gs,
				Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
				Prompt:   prompt,
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

	if turnEndingCategories[cat] {
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

	gs.insertIntoDeck(explosiveID, data.Position)

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

// startBury takes the top card off the deck without showing it to anybody, and
// waits for the player to say where it should go back in.
func (gs *EKGameState) startBury(playerID string) *ActionResult {
	if len(gs.Deck) == 0 {
		gs.addLog(playerID, "played Bury, but the deck was empty. ⚰️")
		gs.advanceTurn()
		return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
	}

	card := gs.Deck[len(gs.Deck)-1]
	gs.Deck = gs.Deck[:len(gs.Deck)-1]
	gs.PendingChoice = &PendingChoiceState{
		PlayerID: playerID,
		Kind:     "bury",
		Cards:    []string{card},
		DeckSize: len(gs.Deck),
	}
	gs.addLog(playerID, "played Bury — placing the top card back in the deck. ⚰️")
	return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
}

// startDigDeeper turns up the top few cards for the player to choose one from.
func (gs *EKGameState) startDigDeeper(playerID string) *ActionResult {
	drawCount := min(3, len(gs.Deck))
	if drawCount == 0 {
		gs.addLog(playerID, "played Dig Deeper, but the deck was empty. ⛏️")
		gs.advanceTurn()
		return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
	}

	drawn := make([]string, drawCount)
	for i := 0; i < drawCount; i++ {
		drawn[i] = gs.Deck[len(gs.Deck)-1]
		gs.Deck = gs.Deck[:len(gs.Deck)-1]
	}
	gs.PendingChoice = &PendingChoiceState{
		PlayerID: playerID,
		Kind:     "dig_deeper",
		Cards:    drawn,
		DeckSize: len(gs.Deck),
	}
	gs.addLog(playerID, fmt.Sprintf("played Dig Deeper — choosing 1 of %d cards. ⛏️", drawCount))
	return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
}

func handleResolveChoice(gs *EKGameState, action GameAction) *ActionResult {
	if gs.PendingChoice == nil {
		return reject(gs, "there's nothing to place right now")
	}
	if gs.PendingChoice.PlayerID != action.Player {
		return reject(gs, "that's not your card to place")
	}

	var data struct {
		Index    int `json:"index"`
		Position int `json:"position"`
	}
	if err := json.Unmarshal(action.Data, &data); err != nil {
		return reject(gs, "invalid choice")
	}

	choice := gs.PendingChoice
	player := gs.Players[choice.PlayerID]
	gs.PendingChoice = nil

	switch choice.Kind {
	case "bury":
		if len(choice.Cards) > 0 {
			gs.insertIntoDeck(choice.Cards[0], data.Position)
		}
		gs.addLog(action.Player, "buried the card in the deck. ⚰️")

	case "dig_deeper":
		if data.Index < 0 || data.Index >= len(choice.Cards) {
			data.Index = 0
		}
		if player != nil {
			player.Hand = append(player.Hand, choice.Cards[data.Index])
		}
		// The rest go back the way they came, keeping their order.
		for i := len(choice.Cards) - 1; i >= 0; i-- {
			if i != data.Index {
				gs.Deck = append(gs.Deck, choice.Cards[i])
			}
		}
		gs.addLog(action.Player, fmt.Sprintf("dug deeper and kept %s! ⛏️", CardLabel(choice.Cards[data.Index])))
	}

	gs.advanceTurn()
	return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
}

// insertIntoDeck puts a card back position cards down from the top, where the
// top is what gets drawn next. Both Defuse and Bury count positions this way.
func (gs *EKGameState) insertIntoDeck(card string, position int) {
	pos := len(gs.Deck) - position
	if pos < 0 {
		pos = 0
	}
	if pos > len(gs.Deck) {
		pos = len(gs.Deck)
	}
	gs.Deck = append(gs.Deck[:pos], append([]string{card}, gs.Deck[pos:]...)...)
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

	gs.addLog(action.Player, fmt.Sprintf("chose %s to put into the deck. 🗑️", CardLabel(card)))

	gs.finishGarbageCollectionIfDone()

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

// cardsMatch reports whether naming one card should catch the other. A player
// names a card by what it does — "an Attack" — but the deck holds five separate
// Attack ids, so matching on the id alone meant a three-of-a-kind steal nearly
// always came up empty. Cat cards are the exception: a Tacocat is not a
// Cattermelon, and telling them apart is the whole point of them.
func cardsMatch(named, held string) bool {
	if named == held {
		return true
	}
	cat := GetCardCategory(named)
	if cat == "unknown" || cat != GetCardCategory(held) {
		return false
	}
	return cat != "cat_cards"
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
	catOrder := []string{}
	for _, c := range cards {
		cat := GetCardCategory(c)
		if _, seen := catCounts[cat]; !seen {
			catOrder = append(catOrder, cat)
		}
		catCounts[cat] = append(catCounts[cat], c)
	}

	// Walk the categories in the order the player selected them, so the server
	// picks the same combo the client previewed.
	for _, cat := range catOrder {
		catCards := catCounts[cat]
		if len(catCards) < 2 {
			continue
		}

		if cat == "cat_cards" {
			idCounts := map[string][]string{}
			idOrder := []string{}
			for _, c := range catCards {
				data := GetCardData(c)
				if data == nil {
					continue
				}
				if _, seen := idCounts[data.ID]; !seen {
					idOrder = append(idOrder, data.ID)
				}
				idCounts[data.ID] = append(idCounts[data.ID], c)
			}
			for _, id := range idOrder {
				if idCards := idCounts[id]; len(idCards) >= 2 {
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

// turnEndingCategories are the cards that finish the turn of the player who
// played them. Skip, Super Skip, Reverse and Attack end it too, but they pass
// the turn themselves, in their own handlers.
//
// Everything absent from this list — Shuffle, See the Future, Swap Top &
// Bottom, Mark, a lone cat card — leaves the player to act again and, in the
// end, draw. Turn-ending used to be the default, which made a Shuffle a free
// Skip.
var turnEndingCategories = map[string]bool{
	"draw_from_bottom": true, // the card off the bottom is the turn's draw
	"dig_deeper":       true,
	"bury":             true,
	"catomic_bomb":     true,
}

// peekTop returns a prompt showing playerID the top n cards of the deck, or nil
// when there is nothing to see.
func (gs *EKGameState) peekTop(playerID string, n int) *WSMessage {
	count := min(n, len(gs.Deck))
	if count <= 0 {
		return nil
	}
	cards := make([]string, count)
	for i := 0; i < count; i++ {
		cards[i] = gs.Deck[len(gs.Deck)-1-i]
	}
	return &WSMessage{
		Type:    "peek_cards",
		Player:  playerID,
		Payload: map[string]interface{}{"cards": cards},
	}
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
	// Like Skip, this ends one turn — under an Attack that is one of the
	// forced turns, not all of them.
	if gs.AttackStack > 0 {
		gs.AttackStack--
		if gs.AttackStack == 0 {
			gs.advanceTurn()
		}
	} else {
		gs.advanceTurn()
	}
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

// alivePlayers lists the players still in the game, in turn order.
func (gs *EKGameState) alivePlayers() []string {
	alive := []string{}
	for _, name := range gs.TurnOrder {
		if p := gs.Players[name]; p != nil && p.Alive {
			alive = append(alive, name)
		}
	}
	return alive
}

// finishGarbageCollectionIfDone closes a Garbage Collection once everyone who
// owes a card has given one, and reports whether it did. It is checked after a
// player responds, and again after a player leaves: someone with no hand left
// owes nothing, so a departure can complete the set on its own.
func (gs *EKGameState) finishGarbageCollectionIfDone() bool {
	if gs.PendingGarbage == nil {
		return false
	}
	for _, name := range gs.TurnOrder {
		if p := gs.Players[name]; p != nil && p.Alive && len(p.Hand) > 0 {
			if _, responded := gs.PendingGarbage.Responded[name]; !responded {
				return false
			}
		}
	}

	for _, cardID := range gs.PendingGarbage.Responded {
		pos := rand.Intn(len(gs.Deck) + 1)
		gs.Deck = append(gs.Deck[:pos], append([]string{cardID}, gs.Deck[pos:]...)...)
	}
	gs.Deck = Shuffle(gs.Deck)
	gs.PendingGarbage = nil
	gs.addLog("system", "All players have chosen! The deck has been shuffled. 🗑️🔄")
	return true
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

	if alive := gs.alivePlayers(); len(alive) <= 1 {
		gs.endGame(alive)
	}
}

// endGame ends the game with the given survivors. More than one means the deck
// ran out before they did, and they share the win.
func (gs *EKGameState) endGame(winners []string) {
	gs.Phase = "ended"
	gs.Winners = winners
	gs.Winner = nil

	switch len(winners) {
	case 0:
		gs.addLog("system", "💀 Everyone was eliminated! No winner.")
	case 1:
		gs.Winner = &winners[0]
		gs.addLog("system", fmt.Sprintf("🎉 %s wins the game!", winners[0]))
	default:
		gs.addLog("system", fmt.Sprintf("🏁 The deck ran out — %s survive and share the win!", strings.Join(winners, ", ")))
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

	if gs.NopeTimer != nil {
		gs.NopeTimer.Stop()
		gs.NopeTimer = nil
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
	gs.TurnEndsAt = nil

	if timerSeconds <= 0 || gs.Phase != "playing" {
		return
	}

	endsAt := time.Now().Add(time.Duration(timerSeconds) * time.Second)
	gs.TurnEndsAt = &endsAt

	gs.TurnTimer = time.AfterFunc(time.Duration(timerSeconds)*time.Second, func() {
		mu.Lock()
		defer mu.Unlock()

		if gs.Phase != "playing" || gs.NopeWindow != nil || gs.PendingDefuse != nil || gs.PendingFavor != nil {
			return
		}

		gs.TurnTimer = nil
		gs.TurnEndsAt = nil
		forceDraw()
	})
}

func CancelPendingAction(gs *EKGameState) {
	if gs.CancelFunc != nil {
		close(gs.CancelFunc)
		gs.CancelFunc = nil
	}
	CancelTimers(gs)
	gs.NopeWindow = nil
	gs.PendingDefuse = nil
	gs.PendingFavor = nil
	gs.PendingChoice = nil
}

// CancelTimers stops every clock without touching the state itself.
func CancelTimers(gs *EKGameState) {
	gs.TurnEndsAt = nil
	for _, t := range []**time.Timer{&gs.NopeTimer, &gs.TurnTimer, &gs.PendingTimer} {
		if *t != nil {
			(*t).Stop()
			*t = nil
		}
	}
}

// StartPendingTimer puts a deadline on an unanswered prompt. onExpire runs with
// the room mutex held, exactly as the other timers do.
func StartPendingTimer(mu *sync.Mutex, gs *EKGameState, onExpire func()) {
	if gs.PendingTimer != nil {
		gs.PendingTimer.Stop()
		gs.PendingTimer = nil
	}
	if gs.pendingPrompt() == "" || gs.Phase != "playing" {
		return
	}

	gs.PendingTimer = time.AfterFunc(PENDING_TIMEOUT_SECONDS*time.Second, func() {
		mu.Lock()
		defer mu.Unlock()
		gs.PendingTimer = nil
		if gs.pendingPrompt() == "" || gs.Phase != "playing" {
			return
		}
		onExpire()
	})
}

// pendingPrompt names the prompt the table is waiting on, if any. A Nope window
// is not one: it has its own, shorter clock.
func (gs *EKGameState) pendingPrompt() string {
	switch {
	case gs.PendingDefuse != nil:
		return "defuse"
	case gs.PendingGarbage != nil:
		return "garbage"
	case gs.PendingFavor != nil:
		return "favor"
	case gs.PendingChoice != nil:
		return "choice"
	}
	return ""
}

// AutoResolvePending answers the open prompt on behalf of whoever left it
// hanging, choosing at random so it never advantages the absent player.
func AutoResolvePending(gs *EKGameState) *ActionResult {
	switch gs.pendingPrompt() {
	case "defuse":
		playerID := gs.PendingDefuse.PlayerID
		pos := 0
		if len(gs.Deck) > 0 {
			pos = rand.Intn(len(gs.Deck) + 1)
		}
		data, _ := json.Marshal(map[string]interface{}{"useDefuse": true, "position": pos})
		gs.addLog(playerID, "ran out of time — the Explosive card was buried for them. ⏰")
		return handleResolveDefuse(gs, GameAction{Action: "resolveDefuse", Data: data, Player: playerID})

	case "favor":
		if gs.PendingFavor.TargetID == "" {
			playerID := gs.PendingFavor.PlayerID
			targets := []string{}
			for _, name := range gs.TurnOrder {
				if p := gs.Players[name]; name != playerID && p != nil && p.Alive && len(p.Hand) > 0 {
					targets = append(targets, name)
				}
			}
			if len(targets) == 0 {
				gs.PendingFavor = nil
				return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
			}
			target := targets[rand.Intn(len(targets))]
			data, _ := json.Marshal(map[string]string{"targetId": target})
			gs.addLog(playerID, "ran out of time — a Favor target was picked for them. ⏰")
			return handleResolveFavor(gs, GameAction{Action: "resolveFavor", Data: data, Player: playerID})
		}

		targetID := gs.PendingFavor.TargetID
		target := gs.Players[targetID]
		if target == nil || len(target.Hand) == 0 {
			gs.PendingFavor = nil
			return &ActionResult{State: gs, Messages: []WSMessage{{Type: "state_updated", Payload: gs}}}
		}
		card := target.Hand[rand.Intn(len(target.Hand))]
		data, _ := json.Marshal(map[string]string{"cardId": card})
		gs.addLog(targetID, "ran out of time — a card was given for them. ⏰")
		return handleResolveFavor(gs, GameAction{Action: "resolveFavor", Data: data, Player: targetID})

	case "choice":
		choice := gs.PendingChoice
		data := map[string]int{}
		if choice.Kind == "bury" {
			data["position"] = rand.Intn(choice.DeckSize + 1)
		} else {
			data["index"] = rand.Intn(max(1, len(choice.Cards)))
		}
		blob, _ := json.Marshal(data)
		gs.addLog(choice.PlayerID, "ran out of time — the card was placed for them. ⏰")
		return handleResolveChoice(gs, GameAction{Action: "resolveChoice", Data: blob, Player: choice.PlayerID})

	case "garbage":
		var last *ActionResult
		for _, name := range gs.TurnOrder {
			if gs.PendingGarbage == nil {
				break
			}
			p := gs.Players[name]
			if p == nil || !p.Alive || len(p.Hand) == 0 {
				continue
			}
			if _, done := gs.PendingGarbage.Responded[name]; done {
				continue
			}
			card := p.Hand[rand.Intn(len(p.Hand))]
			data, _ := json.Marshal(map[string]string{"cardId": card})
			gs.addLog(name, "ran out of time — a card was picked for them. ⏰")
			last = handleResolveGarbageCollection(gs, GameAction{Action: "resolveGarbageCollection", Data: data, Player: name})
		}
		return last
	}
	return nil
}

func (gs *EKGameState) handleFavor(playerID string, cardName string) *ActionResult {
	alive := []string{}
	for _, name := range gs.TurnOrder {
		if name != playerID && gs.Players[name] != nil && gs.Players[name].Alive && len(gs.Players[name].Hand) > 0 {
			alive = append(alive, name)
		}
	}
	if len(alive) == 0 {
		// A Favor that finds nobody to ask is wasted, but it does not end the
		// turn — no more than a successful one does.
		gs.addLog(playerID, fmt.Sprintf("played %s, but no one had cards to give. 🎁", cardName))
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
			return &ActionResult{
				State: gs,
				Messages: []WSMessage{
					{Type: "state_updated", Payload: gs},
				},
			}
		}
		gs.PendingFavor.TargetID = data.TargetID
		return gs.openNopeWindow(&NopeWindowState{
			PlayerID:      action.Player,
			CardID:        "",
			CardName:      "Favor",
			Category:      "favor",
			FavorTargetID: data.TargetID,
		}, fmt.Sprintf("asked %s for a Favor", data.TargetID))
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

func (gs *EKGameState) handleClone(playerID string, cardName string) (bool, *WSMessage) {
	original := gs.LastPlayed
	if original == "" || original == "clone" {
		gs.addLog(playerID, fmt.Sprintf("played %s, but there was nothing to clone. 👻", cardName))
		return false, nil
	}
	label := categoryNameMap[original]
	if label == "" {
		label = original
	}
	gs.addLog(playerID, fmt.Sprintf("played %s, copying %s! 👻", cardName, label))
	return executeCardEffect(gs, playerID, original)
}

// executeCardEffect runs a category's effect for a Clone. It reports whether the
// turn was passed on, and any private prompt the copied card produces.
func executeCardEffect(gs *EKGameState, playerID string, cat string) (bool, *WSMessage) {
	player := gs.Players[playerID]
	if player == nil {
		return false, nil
	}

	switch cat {
	case "skip":
		gs.handleSkip(playerID)
		return true, nil
	case "super_skip":
		gs.handleSuperSkip(playerID)
		return true, nil
	case "reverse":
		gs.handleReverse(playerID)
		return true, nil
	case "attack":
		gs.handleAttack(playerID)
		return true, nil
	case "future_vision":
		gs.addLog(playerID, "cloned See the Future. 🔮")
		return false, gs.peekTop(playerID, 3)
	case "mark":
		gs.addLog(playerID, "cloned Mark. 📍")
		return false, gs.peekTop(playerID, 3)
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
		gs.addLog(playerID, "cloned Bury. ⚰️")
		gs.startBury(playerID)
	case "dig_deeper":
		gs.addLog(playerID, "cloned Dig Deeper. ⛏️")
		gs.startDigDeeper(playerID)
	case "garbage_collection":
		alive := []string{}
		for _, name := range gs.TurnOrder {
			if p := gs.Players[name]; p != nil && p.Alive && len(p.Hand) > 0 {
				alive = append(alive, name)
			}
		}
		if len(alive) == 0 {
			gs.addLog(playerID, "cloned Garbage Collection, but no one had cards. 🗑️")
			return false, nil
		}
		gs.PendingGarbage = &GarbageCollectionState{
			PlayerID:  playerID,
			Responded: make(map[string]string),
		}
		gs.addLog(playerID, "cloned Garbage Collection! Everyone must discard. 🗑️")
	case "favor":
		gs.addLog(playerID, "cloned Favor! 🎁")
	default:
		gs.addLog(playerID, fmt.Sprintf("played Clone, copying %s.", cat))
	}
	return false, nil
}

// --- Per-player views ---------------------------------------------------
//
// The full EKGameState holds every hand and the deck in draw order. Sending it
// to the room hands every player a map of where the Exploding Kittens are, so
// clients are never given it: each one receives the game as they are entitled
// to see it.

type PlayerView struct {
	Color     string   `json:"color"`
	Hand      []string `json:"hand"`
	HandCount int      `json:"handCount"`
	Alive     bool     `json:"alive"`
}

type GameView struct {
	Players          map[string]*PlayerView `json:"players"`
	DeckCount        int                    `json:"deckCount"`
	Discard          []string               `json:"discard"`
	Turn             string                 `json:"turn"`
	TurnOrder        []string               `json:"turnOrder"`
	Phase            string                 `json:"phase"`
	Winner           *string                `json:"winner"`
	Winners          []string               `json:"winners,omitempty"`
	TurnEndsAt       *time.Time             `json:"turnEndsAt,omitempty"`
	Log              []LogEntry             `json:"log"`
	AttackStack      int                    `json:"attackStack"`
	ReverseDirection bool                   `json:"reverseDirection"`

	NopeWindow     *NopeWindowState        `json:"NopeWindow,omitempty"`
	PendingDefuse  *DefuseState            `json:"PendingDefuse,omitempty"`
	PendingGarbage *GarbageCollectionState `json:"PendingGarbage,omitempty"`
	PendingFavor   *PendingFavorState      `json:"PendingFavor,omitempty"`
	PendingChoice  *PendingChoiceState     `json:"PendingChoice,omitempty"`
}

// ViewFor renders the game as one player may see it: their own hand in full,
// everyone else's as a count, and the deck as nothing but its size. Once the
// game is over every hand is revealed, so the table can see how it ended.
func (gs *EKGameState) ViewFor(viewer string) *GameView {
	view := &GameView{
		Players:          make(map[string]*PlayerView, len(gs.Players)),
		DeckCount:        len(gs.Deck),
		Discard:          gs.Discard,
		Turn:             gs.Turn,
		TurnOrder:        gs.TurnOrder,
		Phase:            gs.Phase,
		Winner:           gs.Winner,
		Winners:          gs.Winners,
		TurnEndsAt:       gs.TurnEndsAt,
		Log:              gs.Log,
		AttackStack:      gs.AttackStack,
		ReverseDirection: gs.ReverseDirection,
		NopeWindow:       gs.NopeWindow,
		PendingGarbage:   gs.PendingGarbage,
		PendingFavor:     gs.PendingFavor,
	}
	if view.Discard == nil {
		view.Discard = []string{}
	}
	if view.TurnOrder == nil {
		view.TurnOrder = []string{}
	}
	if view.Log == nil {
		view.Log = []LogEntry{}
	}

	// A pending choice carries cards only that player may see — and for Bury,
	// not even they may see the card they are placing.
	if gs.PendingChoice != nil {
		pc := *gs.PendingChoice
		if pc.PlayerID != viewer || pc.Kind == "bury" {
			pc.Cards = nil
		}
		view.PendingChoice = &pc
	}

	// A defuse prompt names the card that was drawn, so it stays with the one
	// player who is deciding what to do about it.
	if gs.PendingDefuse != nil && gs.PendingDefuse.PlayerID == viewer {
		view.PendingDefuse = gs.PendingDefuse
	} else if gs.PendingDefuse != nil {
		view.PendingDefuse = &DefuseState{PlayerID: gs.PendingDefuse.PlayerID}
	}

	revealAll := gs.Phase == "ended"
	for name, p := range gs.Players {
		if p == nil {
			continue
		}
		pv := &PlayerView{
			Color:     p.Color,
			Hand:      []string{},
			HandCount: len(p.Hand),
			Alive:     p.Alive,
		}
		if name == viewer || revealAll {
			pv.Hand = append([]string{}, p.Hand...)
		}
		view.Players[name] = pv
	}
	return view
}
