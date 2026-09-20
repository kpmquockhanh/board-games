package game

import "testing"

func TestForfeitPlayerIsANoOpWhenThereIsNothingToDo(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")

	if got := ForfeitPlayer(gs, "nobody"); got != nil {
		t.Error("forfeited a player who is not in the game")
	}
	if got := ForfeitPlayer(nil, "alice"); got != nil {
		t.Error("forfeited against a nil game")
	}

	gs.Players["carol"].Alive = false
	if got := ForfeitPlayer(gs, "carol"); got != nil {
		t.Error("forfeited a player who is already out")
	}

	gs.Phase = "lobby"
	if got := ForfeitPlayer(gs, "alice"); got != nil {
		t.Error("forfeited during a lobby, where nobody is playing yet")
	}
}

func TestForfeitPlayerDiscardsHandAndTakesThemOut(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["bob"].Hand = []string{"sk_001", "at_003"}
	discardBefore := len(gs.Discard)

	res := ForfeitPlayer(gs, "bob")
	if res == nil {
		t.Fatal("forfeit reported nothing to do")
	}

	if gs.Players["bob"].Alive {
		t.Error("bob is still alive")
	}
	if len(gs.Players["bob"].Hand) != 0 {
		t.Errorf("hand = %v, want empty", gs.Players["bob"].Hand)
	}
	if len(gs.Discard) != discardBefore+2 {
		t.Errorf("discard grew by %d, want 2", len(gs.Discard)-discardBefore)
	}
	// The others play on.
	if gs.Phase != "playing" {
		t.Errorf("phase = %q, want playing", gs.Phase)
	}
}

// The turn must never be left on a player who is out: the turn timer forces a
// draw, and a draw from a dead player is refused, so the table would hang.
func TestForfeitPlayerOnTheirTurnMovesTheTurnOn(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Turn = "bob"

	ForfeitPlayer(gs, "bob")

	if gs.Turn == "bob" {
		t.Fatal("the turn is still on the player who left")
	}
	if p := gs.Players[gs.Turn]; p == nil || !p.Alive {
		t.Fatalf("the turn moved to %q, who is not playing", gs.Turn)
	}
}

func TestForfeitPlayerLeavesSomeoneElsesTurnAlone(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Turn = "alice"

	ForfeitPlayer(gs, "carol")

	if gs.Turn != "alice" {
		t.Errorf("turn = %q, want alice — she was mid-turn", gs.Turn)
	}
}

// Down to one player, the game is over rather than left running.
func TestForfeitPlayerEndsTheGameWhenOneIsLeft(t *testing.T) {
	gs := newTestGame(t, "alice", "bob")
	gs.Turn = "bob"

	ForfeitPlayer(gs, "bob")

	if gs.Phase != "ended" {
		t.Fatalf("phase = %q, want ended", gs.Phase)
	}
	if gs.Winner == nil || *gs.Winner != "alice" {
		t.Fatalf("winner = %v, want alice", gs.Winner)
	}
}

// A prompt only the leaver could answer has to be handed back, or the table
// waits on someone who is never going to answer.
func TestForfeitPlayerResolvesADefuseTheyOwed(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Turn = "bob"
	gs.Players["bob"].Hand = []string{"df_001"}
	gs.PendingDefuse = &DefuseState{PlayerID: "bob", ExplosiveID: "ex_001"}

	ForfeitPlayer(gs, "bob")

	if gs.PendingDefuse != nil {
		t.Fatal("the table is still waiting on a defuse from a player who left")
	}
	if gs.Players["bob"].Alive {
		t.Error("bob is still in the game")
	}
	if gs.Turn == "bob" {
		t.Error("the turn is still on the player who left")
	}
}

func TestForfeitPlayerResolvesAChoiceTheyOwed(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Turn = "bob"
	gs.PendingChoice = &PendingChoiceState{
		PlayerID: "bob", Kind: "bury", DeckSize: len(gs.Deck),
	}

	ForfeitPlayer(gs, "bob")

	if gs.PendingChoice != nil {
		t.Fatal("the table is still waiting on a choice from a player who left")
	}
}

// A Favor owed *to* someone else still has to clear when the debtor leaves.
func TestForfeitPlayerResolvesAFavorTheyOwed(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Turn = "alice"
	gs.Players["bob"].Hand = []string{"sk_001"}
	gs.PendingFavor = &PendingFavorState{PlayerID: "alice", TargetID: "bob"}

	ForfeitPlayer(gs, "bob")

	if gs.PendingFavor != nil {
		t.Fatal("the table is still waiting on a favor from a player who left")
	}
	// Alice asked, so she should have been given the card before bob went.
	if len(gs.Players["alice"].Hand) == 0 {
		t.Error("alice never received the favor she was owed")
	}
}

// Garbage Collection waits on everybody, so the players still at the table
// must keep choosing for themselves rather than have a card picked for them.
func TestForfeitPlayerDoesNotAnswerGarbageCollectionForOthers(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"at_003"}
	gs.Players["carol"].Hand = []string{"sh_001"}
	gs.PendingGarbage = &GarbageCollectionState{
		PlayerID: "alice", Responded: map[string]string{},
	}

	ForfeitPlayer(gs, "bob")

	if gs.PendingGarbage == nil {
		t.Fatal("the collection was closed while alice and carol still owed cards")
	}
	if _, picked := gs.PendingGarbage.Responded["alice"]; picked {
		t.Error("a card was picked for alice, who is still at the table")
	}
	if len(gs.Players["alice"].Hand) != 1 || len(gs.Players["carol"].Hand) != 1 {
		t.Error("someone else's hand was touched by another player leaving")
	}
}

// ...but a leaver who was the last holdout does complete it, and nothing else
// would re-run that check.
func TestForfeitPlayerCompletesGarbageCollectionWhenTheyWereTheLastHoldout(t *testing.T) {
	gs := newTestGame(t, "alice", "bob", "carol")
	gs.Players["alice"].Hand = []string{"sk_001"}
	gs.Players["bob"].Hand = []string{"at_003"}
	gs.Players["carol"].Hand = []string{}
	gs.PendingGarbage = &GarbageCollectionState{
		PlayerID:  "alice",
		Responded: map[string]string{"alice": "sk_002"},
	}

	ForfeitPlayer(gs, "bob")

	if gs.PendingGarbage != nil {
		t.Fatal("the collection is still open with nobody left to answer it")
	}
}
