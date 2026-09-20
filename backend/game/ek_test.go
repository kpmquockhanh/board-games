package game

import (
	"bytes"
	"encoding/json"
	"testing"
)

// newTestGame deals a real game of two players, then replaces the hands so each
// test can set up the exact situation it cares about.
func newTestGame(t *testing.T, names ...string) *EKGameState {
	t.Helper()
	if len(names) == 0 {
		names = []string{"alice", "bob"}
	}
	players := map[string]*PlayerState{}
	for _, n := range names {
		players[n] = &PlayerState{Color: "#fff"}
	}
	cats := map[string]bool{}
	for c := range CardCategories {
		cats[c] = true
	}
	data, err := json.Marshal(map[string]interface{}{
		"players": players, "turnOrder": names,
		"handSize": 5, "defenseCount": 1, "multiplier": 1.0, "enabledCats": cats,
	})
	if err != nil {
		t.Fatal(err)
	}
	gs := &EKGameState{}
	res := ProcessAction(gs, GameAction{Action: "startGame", Data: data, Player: names[0]})
	if res.Error != "" {
		t.Fatalf("startGame rejected: %s", res.Error)
	}
	if gs.Phase != "playing" {
		t.Fatalf("phase = %q, want playing", gs.Phase)
	}
	return gs
}

func play(t *testing.T, gs *EKGameState, player string, cards ...string) *ActionResult {
	t.Helper()
	data, _ := json.Marshal(map[string]interface{}{"cards": cards})
	return ProcessAction(gs, GameAction{Action: "playCard", Data: data, Player: player})
}

// playThrough plays a card and then lets the Nope window run out, the way the
// countdown does at a real table when nobody answers it.
func playThrough(t *testing.T, gs *EKGameState, player string, cards ...string) *ActionResult {
	t.Helper()
	res := play(t, gs, player, cards...)
	if res.Error != "" || gs.NopeWindow == nil {
		return res
	}
	return ProcessAction(gs, GameAction{Action: "nopeResolved", Player: player})
}

func TestPlayCardRejectsCardNotInHand(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001"}

	res := play(t, gs, "alice", "at_003")

	if res.Error == "" {
		t.Fatal("playing a card the player does not hold was allowed")
	}
	if len(gs.Discard) != 0 {
		t.Errorf("discard = %v, want empty", gs.Discard)
	}
	if gs.AttackStack != 0 {
		t.Errorf("attackStack = %d, want 0 — the effect ran anyway", gs.AttackStack)
	}
	if gs.Turn != "alice" {
		t.Errorf("turn = %q, want alice", gs.Turn)
	}
}

func TestPlayCardRejectsUnknownCardWithoutPanicking(t *testing.T) {
	gs := newTestGame(t)

	res := play(t, gs, "alice", "zz_999")

	if res.Error == "" {
		t.Fatal("an unknown card id was accepted")
	}
}

func TestPlayCardRejectsOutOfTurn(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["bob"].Hand = []string{"sk_001"}

	if res := play(t, gs, "bob", "sk_001"); res.Error == "" {
		t.Fatal("bob played on alice's turn")
	}
}

func TestPlayComboRejectsCardsNotInHand(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{}
	gs.Players["bob"].Hand = []string{"sk_001"}
	data, _ := json.Marshal(map[string]interface{}{
		"cards": []string{"cc_001", "cc_001"}, "target": "bob",
	})

	res := ProcessAction(gs, GameAction{Action: "playCombo", Data: data, Player: "alice"})

	if res.Error == "" {
		t.Fatal("a combo of cards the player does not hold was allowed")
	}
	if len(gs.Players["bob"].Hand) != 1 {
		t.Errorf("bob was robbed anyway: %v", gs.Players["bob"].Hand)
	}
}

func TestDefusePutsKittenWhereTheSliderSays(t *testing.T) {
	cases := []struct {
		position int
		wantNext string // the card the next draw takes
	}{
		{position: 0, wantNext: "ex_001"}, // top: the next player draws it
		{position: 3, wantNext: "top_c"},  // bottom of a three-card deck
	}
	for _, tc := range cases {
		gs := newTestGame(t)
		gs.Deck = []string{"bottom_c", "mid_c", "top_c"}
		gs.Players["alice"].Hand = []string{"df_001"}
		gs.PendingDefuse = &DefuseState{PlayerID: "alice", ExplosiveID: "ex_001"}
		data, _ := json.Marshal(map[string]interface{}{"useDefuse": true, "position": tc.position})

		ProcessAction(gs, GameAction{Action: "resolveDefuse", Data: data, Player: "alice"})

		if got := gs.Deck[len(gs.Deck)-1]; got != tc.wantNext {
			t.Errorf("position %d: next draw = %q, want %q (deck %v)", tc.position, got, tc.wantNext, gs.Deck)
		}
	}
}

func TestNopeWindowOpensEvenWhenNobodyHoldsANope(t *testing.T) {
	// The window used to be skipped when no Nope was out there, which told the
	// whole table exactly that: no countdown meant no Nope in anyone's hand.
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"cc_001"} // no Nope anywhere

	res := play(t, gs, "alice", "sk_001")

	if !res.NopeWindow || gs.NopeWindow == nil {
		t.Fatal("the play resolved on the spot, leaking that nobody holds a Nope")
	}
	if gs.Turn != "alice" {
		t.Errorf("turn = %q, want alice — the Skip resolved early", gs.Turn)
	}
}

func TestWindowClosesOnceEveryResponderHasPassed(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"np_002"}
	gs.Players["carol"].Hand = []string{"cc_001"}
	play(t, gs, "alice", "sk_001")

	if res := ProcessAction(gs, GameAction{Action: "passNope", Player: "bob"}); res.Error != "" {
		t.Fatalf("bob's pass was rejected: %s", res.Error)
	}
	if gs.NopeWindow == nil {
		t.Fatal("the window closed while carol could still answer")
	}

	// Carol has no Nope, and passes all the same — which is what keeps an
	// early close from meaning anything about her hand.
	if res := ProcessAction(gs, GameAction{Action: "passNope", Player: "carol"}); res.Error != "" {
		t.Fatalf("carol's pass was rejected: %s", res.Error)
	}
	if gs.NopeWindow != nil {
		t.Fatal("window left open although everyone passed")
	}
	if gs.Turn != "bob" {
		t.Errorf("turn = %q, want bob — the Skip did not resolve", gs.Turn)
	}
}

func TestAPlayerCannotPassOnTheirOwnPlay(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"np_002"}
	play(t, gs, "alice", "sk_001")

	if res := ProcessAction(gs, GameAction{Action: "passNope", Player: "alice"}); res.Error == "" {
		t.Fatal("alice closed her own window by passing")
	}
	if gs.NopeWindow == nil {
		t.Fatal("the window closed on alice's own pass")
	}
}

func TestANopeClearsThePassesOfTheRoundBefore(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["alice"].Hand = []string{"sk_001", "np_003"}
	gs.Players["bob"].Hand = []string{"np_002"}
	gs.Players["carol"].Hand = []string{"np_004"}
	play(t, gs, "alice", "sk_001")
	ProcessAction(gs, GameAction{Action: "passNope", Player: "carol"})

	ProcessAction(gs, GameAction{Action: "playNope", Player: "bob"})

	if gs.NopeWindow == nil {
		t.Fatal("no window to counter bob's Nope")
	}
	if len(gs.NopeWindow.Passed) != 0 {
		t.Errorf("passes = %v, want them spent with the Nope", gs.NopeWindow.Passed)
	}
}

func TestNopeWindowOpensWhenAnOpponentHoldsANope(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"np_002"}

	res := play(t, gs, "alice", "sk_001")

	if !res.NopeWindow || gs.NopeWindow == nil {
		t.Fatal("no window opened although bob holds a Nope")
	}
	if gs.Turn != "alice" {
		t.Errorf("turn = %q, want alice — the Skip resolved early", gs.Turn)
	}
}

func TestNopeReopensTheWindowForACounter(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001", "np_003"}
	gs.Players["bob"].Hand = []string{"np_002"}
	play(t, gs, "alice", "sk_001")

	res := ProcessAction(gs, GameAction{Action: "playNope", Player: "bob"})

	if !res.NopeWindow {
		t.Fatal("window not re-armed: alice cannot counter bob's Nope")
	}
	if gs.NopeWindow == nil || gs.NopeWindow.NopeCount != 1 {
		t.Fatalf("nope count = %v, want 1", gs.NopeWindow)
	}
}

func TestCannotNopeYourOwnCard(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001", "np_003"}
	gs.Players["bob"].Hand = []string{"np_002"}
	play(t, gs, "alice", "sk_001")

	if res := ProcessAction(gs, GameAction{Action: "playNope", Player: "alice"}); res.Error == "" {
		t.Fatal("alice Noped her own play")
	}
}

func TestNothingElseHappensWhileTheWindowIsOpen(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001", "sk_002"}
	gs.Players["bob"].Hand = []string{"np_002"}
	play(t, gs, "alice", "sk_001")
	turnBefore := gs.Turn

	if res := ProcessAction(gs, GameAction{Action: "drawCard", Player: "alice"}); res.Error == "" {
		t.Error("drew a card during the Nope window")
	}
	if res := play(t, gs, "alice", "sk_002"); res.Error == "" {
		t.Error("played a second card during the Nope window")
	}
	if gs.Turn != turnBefore {
		t.Errorf("turn advanced to %q during the window", gs.Turn)
	}
}

func TestDetectComboPrefersTheFirstSelectedCategory(t *testing.T) {
	// Two candidate pairs: the server must pick the same one every time, or it
	// discards different cards than the client previewed.
	cards := []string{"sk_001", "sk_002", "at_001", "at_002"}
	for i := 0; i < 50; i++ {
		combo := detectCombo(cards)
		if combo == nil || combo.Category != "skip" {
			t.Fatalf("run %d: combo = %+v, want the skip pair", i, combo)
		}
	}
}

func TestStartGameWillNotRestartARunningGame(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001"}
	data, _ := json.Marshal(map[string]interface{}{
		"players":   map[string]*PlayerState{"alice": {}, "bob": {}},
		"turnOrder": []string{"alice", "bob"}, "handSize": 5, "defenseCount": 1, "multiplier": 1.0,
	})

	res := ProcessAction(gs, GameAction{Action: "startGame", Data: data, Player: "bob"})

	if res.Error == "" {
		t.Fatal("a running game was restarted")
	}
	if got := gs.Players["alice"].Hand; len(got) != 1 {
		t.Errorf("alice's hand was redealt: %v", got)
	}
}

func TestViewHidesOtherHandsAndTheDeck(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"sk_001", "np_002"}
	gs.Players["bob"].Hand = []string{"at_001", "df_001", "cc_003"}
	gs.Deck = []string{"ex_001", "sk_002"}

	view := gs.ViewFor("alice")

	if got := view.Players["alice"].Hand; len(got) != 2 {
		t.Errorf("alice cannot see her own hand: %v", got)
	}
	if got := view.Players["bob"].Hand; len(got) != 0 {
		t.Errorf("bob's hand leaked to alice: %v", got)
	}
	if got := view.Players["bob"].HandCount; got != 3 {
		t.Errorf("bob's card count = %d, want 3", got)
	}
	if view.DeckCount != 2 {
		t.Errorf("deckCount = %d, want 2", view.DeckCount)
	}

	// The marshalled form is what actually goes over the wire.
	blob, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	for _, leaked := range []string{"at_001", "cc_003", "ex_001", `"deck"`} {
		if bytes.Contains(blob, []byte(leaked)) {
			t.Errorf("view JSON leaks %s: %s", leaked, blob)
		}
	}
}

func TestViewRevealsEveryHandOnceTheGameIsOver(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["bob"].Hand = []string{"at_001"}
	gs.Phase = "ended"

	if got := gs.ViewFor("alice").Players["bob"].Hand; len(got) != 1 {
		t.Errorf("final hands stayed hidden: %v", got)
	}
}

func TestViewKeepsTheDrawnKittenToTheOnePlayerDeciding(t *testing.T) {
	gs := newTestGame(t)
	gs.PendingDefuse = &DefuseState{PlayerID: "alice", ExplosiveID: "ex_007"}

	if got := gs.ViewFor("alice").PendingDefuse.ExplosiveID; got != "ex_007" {
		t.Errorf("alice's own defuse prompt = %q, want ex_007", got)
	}
	bobs := gs.ViewFor("bob").PendingDefuse
	if bobs == nil || bobs.PlayerID != "alice" {
		t.Fatalf("bob should see that alice is defusing, got %+v", bobs)
	}
	if bobs.ExplosiveID != "" {
		t.Errorf("bob learned which kitten was drawn: %q", bobs.ExplosiveID)
	}
}

func TestAutoResolveBuriesTheKittenForAnAbsentPlayer(t *testing.T) {
	gs := newTestGame(t)
	gs.Deck = []string{"sk_002", "sk_003"}
	gs.Players["alice"].Hand = []string{"df_001"}
	gs.PendingDefuse = &DefuseState{PlayerID: "alice", ExplosiveID: "ex_001"}

	res := AutoResolvePending(gs)

	if res == nil {
		t.Fatal("nothing resolved")
	}
	if gs.PendingDefuse != nil {
		t.Error("the prompt is still blocking the table")
	}
	if gs.Players["alice"].Alive != true {
		t.Error("alice was eliminated although she held a Defuse")
	}
	if len(gs.Deck) != 3 {
		t.Errorf("deck = %v, want the kitten back in it", gs.Deck)
	}
	if gs.Turn != "bob" {
		t.Errorf("turn = %q, want bob", gs.Turn)
	}
}

func TestAutoResolveAnswersGarbageCollectionForEveryone(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"at_001"}
	gs.Players["carol"].Hand = []string{"cc_002"}
	gs.PendingGarbage = &GarbageCollectionState{PlayerID: "alice", Responded: map[string]string{}}

	AutoResolvePending(gs)

	if gs.PendingGarbage != nil {
		t.Fatal("Garbage Collection never finished")
	}
	for _, name := range gs.TurnOrder {
		if n := len(gs.Players[name].Hand); n != 0 {
			t.Errorf("%s still holds %d cards", name, n)
		}
	}
}

func TestAutoResolveGivesAFavorForAnAbsentTarget(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{}
	gs.Players["bob"].Hand = []string{"cc_004"}
	gs.PendingFavor = &PendingFavorState{PlayerID: "alice", TargetID: "bob"}

	AutoResolvePending(gs)

	if gs.PendingFavor != nil {
		t.Fatal("the Favor is still blocking the table")
	}
	if got := gs.Players["alice"].Hand; len(got) != 1 || got[0] != "cc_004" {
		t.Errorf("alice's hand = %v, want the favoured card", got)
	}
}

func TestDrawPileSurvivesABigTable(t *testing.T) {
	cats := map[string]bool{}
	for c := range CardCategories {
		cats[c] = true
	}
	const handSize = 8 // the default: one free Defuse plus seven off the deck

	for players := 2; players <= 6; players++ {
		deck := BuildDeck(players, 1, cats, -1)
		kittens, others := 0, 0
		for _, c := range deck {
			if GetCardCategory(c) == "explosive" {
				kittens++
			} else {
				others++
			}
		}
		drawPile := others - players*(handSize-1) + kittens
		if drawPile < players*5 {
			t.Errorf("%d players: draw pile of %d is too small to play with", players, drawPile)
		}
		if density := float64(kittens) / float64(drawPile); density > 0.2 {
			t.Errorf("%d players: %.0f%% of the draw pile is Exploding Kittens", players, density*100)
		}
	}
}

func TestBuildDeckKeepsKittensWhenSettingsAreIncomplete(t *testing.T) {
	// A settings blob that names no categories at all must not produce a deck
	// with nothing in it — least of all one with no Exploding Kittens.
	deck := BuildDeck(4, 1, map[string]bool{}, -1)

	kittens := 0
	for _, c := range deck {
		if GetCardCategory(c) == "explosive" {
			kittens++
		}
	}
	if len(deck) == 0 {
		t.Fatal("empty deck")
	}
	if kittens != 3 {
		t.Errorf("kittens = %d, want 3", kittens)
	}
}

func TestBuildDeckHonoursAnExplicitKittenCount(t *testing.T) {
	deck := BuildDeck(4, 1, nil, 7)

	kittens := 0
	for _, c := range deck {
		if GetCardCategory(c) == "explosive" {
			kittens++
		}
	}
	if kittens != 7 {
		t.Errorf("kittens = %d, want the 7 the room asked for", kittens)
	}
}

func TestWhichCardsEndTheTurn(t *testing.T) {
	cases := []struct {
		name     string
		card     string
		wantTurn string
	}{
		{"Shuffle leaves you to draw", "dm_001", "alice"},
		{"Swap Top & Bottom leaves you to draw", "dm_012", "alice"},
		{"Mark leaves you to draw", "dm_015", "alice"},
		{"a lone cat card leaves you to draw", "cc_001", "alice"},
		{"See the Future leaves you to draw", "fv_001", "alice"},
		{"Skip ends the turn", "sk_001", "bob"},
		{"Attack ends the turn", "at_001", "bob"},
		{"Draw From Bottom ends the turn", "dm_011", "bob"},
		// Bury and Dig Deeper end the turn too, but only once the player has
		// placed their card — see TestBuryLetsThePlayerPlaceTheCard.
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs := newTestGame(t)
			gs.Deck = []string{"sk_002", "sk_003", "sk_004", "sk_005"}
			gs.Players["alice"].Hand = []string{tc.card}
			gs.Players["bob"].Hand = []string{"cc_002"}

			if res := playThrough(t, gs, "alice", tc.card); res.Error != "" {
				t.Fatalf("rejected: %s", res.Error)
			}
			if gs.Turn != tc.wantTurn {
				t.Errorf("turn = %q, want %q", gs.Turn, tc.wantTurn)
			}
		})
	}
}

func TestDefuseCardCannotBePlayedFromHand(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"df_001"}

	if res := play(t, gs, "alice", "df_001"); res.Error == "" {
		t.Fatal("a Defuse was played as a normal card")
	}
	if len(gs.Players["alice"].Hand) != 1 {
		t.Error("the Defuse was consumed anyway")
	}
}

func TestComboStealMatchesTheKindOfCardNamed(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"cc_001", "cc_001", "cc_001"}
	gs.Players["bob"].Hand = []string{"at_003"} // a different Attack id than the one named
	data, _ := json.Marshal(map[string]interface{}{
		"cards": []string{"cc_001", "cc_001", "cc_001"}, "target": "bob", "cardId": "at_001",
	})

	ProcessAction(gs, GameAction{Action: "playCombo", Data: data, Player: "alice"})
	ProcessAction(gs, GameAction{Action: "nopeResolved", Player: "alice"})

	if got := gs.Players["alice"].Hand; len(got) != 1 || got[0] != "at_003" {
		t.Errorf("alice's hand = %v, want the stolen Attack", got)
	}
	if got := gs.Players["bob"].Hand; len(got) != 0 {
		t.Errorf("bob kept the card: %v", got)
	}
}

func TestComboStealStillTellsCatCardsApart(t *testing.T) {
	gs := newTestGame(t)
	gs.Players["alice"].Hand = []string{"at_001", "at_002", "at_003"}
	gs.Players["bob"].Hand = []string{"cc_002"}
	data, _ := json.Marshal(map[string]interface{}{
		"cards": []string{"at_001", "at_002", "at_003"}, "target": "bob", "cardId": "cc_001",
	})

	ProcessAction(gs, GameAction{Action: "playCombo", Data: data, Player: "alice"})
	ProcessAction(gs, GameAction{Action: "nopeResolved", Player: "alice"})

	if got := gs.Players["bob"].Hand; len(got) != 1 {
		t.Errorf("naming a Tacocat took a different cat card: %v", got)
	}
}

func TestCatCardsAreDealtInMatchingPairs(t *testing.T) {
	// A cat-card combo needs two of the same cat. The deck used to hold one of
	// each, so the combo could never be made at all.
	deck := BuildDeck(4, 1, nil, -1)

	counts := map[string]int{}
	for _, c := range deck {
		if GetCardCategory(c) == "cat_cards" {
			counts[c]++
		}
	}
	if len(counts) == 0 {
		t.Fatal("no cat cards in the deck")
	}
	pairs := 0
	for _, n := range counts {
		pairs += n / 2
	}
	if pairs < 2 {
		t.Errorf("only %d cat pairs exist in the whole deck: %v", pairs, counts)
	}
}

func TestReverseEndsOneForcedTurnLikeSkip(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["alice"].Hand = []string{"sk_010"}
	gs.Players["bob"].Hand = []string{}
	gs.Players["carol"].Hand = []string{}
	gs.AttackStack = 2 // alice owes two turns

	playThrough(t, gs, "alice", "sk_010")

	if gs.AttackStack != 1 {
		t.Errorf("attackStack = %d, want 1 — Reverse ends one forced turn", gs.AttackStack)
	}
	if gs.Turn != "alice" {
		t.Errorf("turn = %q, want alice: she still owes a turn", gs.Turn)
	}
}

func TestCloneCopiesTheLastCardPlayedNotTheDiscardPile(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Deck = []string{"sk_002", "sk_003", "sk_004"}
	gs.Players["alice"].Hand = []string{"sf_011"}
	gs.Players["bob"].Hand = []string{}
	gs.Players["carol"].Hand = []string{}
	// Bob exploded earlier and his hand was dumped on the discard pile.
	gs.Discard = []string{"at_001", "dm_001", "cc_001"}
	gs.LastPlayed = "attack"

	playThrough(t, gs, "alice", "sf_011")

	if gs.AttackStack != 2 {
		t.Errorf("attackStack = %d, want 2 — Clone should have copied the Attack", gs.AttackStack)
	}
}

func TestCloneOfSeeTheFutureActuallyShowsTheCards(t *testing.T) {
	gs := newTestGame(t)
	gs.Deck = []string{"sk_002", "sk_003", "sk_004"}
	gs.Players["alice"].Hand = []string{"sf_011"}
	gs.Players["bob"].Hand = []string{}
	gs.LastPlayed = "future_vision"

	res := playThrough(t, gs, "alice", "sf_011")

	if res.Prompt == nil || res.Prompt.Type != "peek_cards" {
		t.Fatalf("no peek prompt: %+v", res.Prompt)
	}
	if res.Prompt.Player != "alice" {
		t.Errorf("prompt went to %q", res.Prompt.Player)
	}
}

func TestMarkShowsTheTopOfTheDeck(t *testing.T) {
	gs := newTestGame(t)
	gs.Deck = []string{"sk_002", "sk_003", "sk_004"}
	gs.Players["alice"].Hand = []string{"dm_015"}
	gs.Players["bob"].Hand = []string{}

	res := playThrough(t, gs, "alice", "dm_015")

	if res.Prompt == nil || res.Prompt.Type != "peek_cards" {
		t.Fatal("Mark still shows nothing")
	}
}

func TestAnEmptyDeckIsSharedBetweenTheSurvivors(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["carol"].Alive = false
	gs.Deck = []string{}

	ProcessAction(gs, GameAction{Action: "drawCard", Player: "alice"})

	if gs.Phase != "ended" {
		t.Fatalf("phase = %q, want ended", gs.Phase)
	}
	if len(gs.Winners) != 2 {
		t.Errorf("winners = %v, want both survivors", gs.Winners)
	}
	last := gs.Log[len(gs.Log)-1].Text
	if last == "💀 Everyone was eliminated! No winner." {
		t.Error("two survivors were reported as nobody surviving")
	}
}

func TestDigDeeperLetsThePlayerChoose(t *testing.T) {
	gs := newTestGame(t)
	gs.Deck = []string{"sk_005", "cc_001", "at_002", "df_003"} // last is the top
	gs.Players["alice"].Hand = []string{"dm_018"}
	gs.Players["bob"].Hand = []string{}

	playThrough(t, gs, "alice", "dm_018")

	if gs.PendingChoice == nil || gs.PendingChoice.Kind != "dig_deeper" {
		t.Fatalf("no choice offered: %+v", gs.PendingChoice)
	}
	if got := gs.PendingChoice.Cards; len(got) != 3 || got[0] != "df_003" {
		t.Fatalf("cards offered = %v, want the top three", got)
	}
	if gs.Turn != "alice" {
		t.Error("the turn passed before alice had chosen")
	}

	data, _ := json.Marshal(map[string]int{"index": 2}) // the deepest of the three
	ProcessAction(gs, GameAction{Action: "resolveChoice", Data: data, Player: "alice"})

	if got := gs.Players["alice"].Hand; len(got) != 1 || got[0] != "cc_001" {
		t.Errorf("alice kept %v, want the card she picked", got)
	}
	if got := gs.Deck[len(gs.Deck)-1]; got != "df_003" {
		t.Errorf("top of deck = %q, want the untouched cards back in order", got)
	}
	if gs.Turn != "bob" {
		t.Errorf("turn = %q, want bob: Dig Deeper ends the turn", gs.Turn)
	}
}

func TestBuryLetsThePlayerPlaceTheCard(t *testing.T) {
	gs := newTestGame(t)
	gs.Deck = []string{"sk_005", "cc_001", "at_002"} // at_002 is on top
	gs.Players["alice"].Hand = []string{"dm_016"}
	gs.Players["bob"].Hand = []string{}

	playThrough(t, gs, "alice", "dm_016")

	if gs.PendingChoice == nil || gs.PendingChoice.Kind != "bury" {
		t.Fatalf("no choice offered: %+v", gs.PendingChoice)
	}
	if gs.PendingChoice.DeckSize != 2 {
		t.Errorf("deck size offered = %d, want 2", gs.PendingChoice.DeckSize)
	}

	data, _ := json.Marshal(map[string]int{"position": 2}) // the very bottom
	ProcessAction(gs, GameAction{Action: "resolveChoice", Data: data, Player: "alice"})

	if got := gs.Deck[0]; got != "at_002" {
		t.Errorf("bottom of deck = %q, want the buried card", got)
	}
	if len(gs.Deck) != 3 {
		t.Errorf("deck = %v, want the card back in it", gs.Deck)
	}
	if gs.Turn != "bob" {
		t.Errorf("turn = %q, want bob", gs.Turn)
	}
}

func TestNobodySeesTheCardBeingBuried(t *testing.T) {
	gs := newTestGame(t)
	gs.PendingChoice = &PendingChoiceState{PlayerID: "alice", Kind: "bury", Cards: []string{"ex_001"}, DeckSize: 4}

	for _, viewer := range []string{"alice", "bob"} {
		if got := gs.ViewFor(viewer).PendingChoice.Cards; len(got) != 0 {
			t.Errorf("%s can see the buried card: %v", viewer, got)
		}
	}
}

func TestOnlyTheDiggerSeesTheDugCards(t *testing.T) {
	gs := newTestGame(t)
	gs.PendingChoice = &PendingChoiceState{PlayerID: "alice", Kind: "dig_deeper", Cards: []string{"ex_001", "sk_001"}, DeckSize: 4}

	if got := gs.ViewFor("alice").PendingChoice.Cards; len(got) != 2 {
		t.Errorf("alice cannot see her own cards: %v", got)
	}
	if got := gs.ViewFor("bob").PendingChoice.Cards; len(got) != 0 {
		t.Errorf("bob can see the dug cards: %v", got)
	}
}

func TestAnAbandonedChoiceResolvesItself(t *testing.T) {
	gs := newTestGame(t)
	gs.Deck = []string{"sk_005"}
	gs.PendingChoice = &PendingChoiceState{PlayerID: "alice", Kind: "dig_deeper", Cards: []string{"at_001", "cc_002"}, DeckSize: 1}

	AutoResolvePending(gs)

	if gs.PendingChoice != nil {
		t.Fatal("the choice is still blocking the table")
	}
	if len(gs.Players["alice"].Hand) == 0 {
		t.Error("alice was given nothing")
	}
	if gs.Turn != "bob" {
		t.Errorf("turn = %q, want bob", gs.Turn)
	}
}

func TestFavorNeverEndsYourTurn(t *testing.T) {
	// Whether the Favor finds a target or not, the player still owes a draw.
	t.Run("with nobody to ask", func(t *testing.T) {
		gs := newTestGame(t)
		gs.Players["alice"].Hand = []string{"sf_004"}
		gs.Players["bob"].Hand = []string{}

		play(t, gs, "alice", "sf_004")

		if gs.Turn != "alice" {
			t.Errorf("turn = %q, want alice", gs.Turn)
		}
	})

	t.Run("with a target who turns out to have nothing", func(t *testing.T) {
		gs := newTestGame(t)
		gs.Players["alice"].Hand = []string{}
		gs.Players["bob"].Hand = []string{}
		gs.PendingFavor = &PendingFavorState{PlayerID: "alice"}
		data, _ := json.Marshal(map[string]string{"targetId": "bob"})

		ProcessAction(gs, GameAction{Action: "resolveFavor", Data: data, Player: "alice"})

		if gs.Turn != "alice" {
			t.Errorf("turn = %q, want alice", gs.Turn)
		}
	})

	t.Run("when it succeeds", func(t *testing.T) {
		gs := newTestGame(t)
		gs.Players["alice"].Hand = []string{}
		gs.Players["bob"].Hand = []string{"cc_004"}
		gs.PendingFavor = &PendingFavorState{PlayerID: "alice", TargetID: "bob"}
		data, _ := json.Marshal(map[string]string{"cardId": "cc_004"})

		ProcessAction(gs, GameAction{Action: "resolveFavor", Data: data, Player: "bob"})

		if gs.Turn != "alice" {
			t.Errorf("turn = %q, want alice", gs.Turn)
		}
	})
}
