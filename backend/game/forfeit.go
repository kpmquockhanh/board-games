package game

// Forfeiting is what happens to a player who disconnects mid-game and does not
// come back. The seat is held for a while — a reload or a tunnel should not
// cost anyone their game — but not forever, because every other player at the
// table is waiting on someone who is gone.

// blockedOnlyOn reports whether the table is waiting on this one player, and
// so would stay stuck if they simply vanished.
//
// Garbage Collection is deliberately excluded: it waits on everybody at once,
// and the players who are still here should pick their own cards rather than
// have one chosen for them because somebody else left. A departure is instead
// handled by the completion re-check in ForfeitPlayer.
//
// A Nope window is excluded too. It is not owned by anyone — it is an
// invitation to the whole table — and it closes on its own within seconds.
func (gs *EKGameState) blockedOnlyOn(name string) bool {
	switch gs.pendingPrompt() {
	case "defuse":
		return gs.PendingDefuse.PlayerID == name
	case "choice":
		return gs.PendingChoice.PlayerID == name
	case "favor":
		// Before a target is picked the asker is holding things up; after, it
		// is the target who owes a card.
		if gs.PendingFavor.TargetID == "" {
			return gs.PendingFavor.PlayerID == name
		}
		return gs.PendingFavor.TargetID == name
	}
	return false
}

// ForfeitPlayer takes a player out of a game in progress. It returns nil when
// there is nothing to do — no such player, already out, or no game running —
// so the caller can tell a real forfeit from a no-op.
func ForfeitPlayer(gs *EKGameState, name string) *ActionResult {
	if gs == nil || gs.Phase != "playing" {
		return nil
	}
	player := gs.Players[name]
	if player == nil || !player.Alive {
		return nil
	}

	wasTheirTurn := gs.Turn == name

	// Give the table back whatever it was waiting on this player alone for,
	// using the same fallbacks a prompt timeout uses. This runs first, while
	// they are still alive and still holding a hand, because those helpers
	// need both.
	if gs.blockedOnlyOn(name) {
		AutoResolvePending(gs)
	}

	// That resolution can end the game, and can kill this player outright —
	// a defuse resolved by someone with no Defuse card does exactly that.
	if gs.Phase != "playing" {
		return forfeitResult(gs)
	}

	if player.Alive {
		gs.addLog(name, "left the game. 👋")
		gs.eliminatePlayer(name)
	}
	if gs.Phase != "playing" {
		return forfeitResult(gs)
	}

	// Their hand is gone, so they owe no card to a Garbage Collection. If they
	// were the last one it was waiting on, it is finished now and nothing else
	// would notice.
	gs.finishGarbageCollectionIfDone()

	// The turn must never rest on a player who is out: the turn timer would
	// force a draw for them, and the rules refuse a draw from someone who is
	// not alive, so the table would sit there until everyone gave up. The one
	// thing allowed to settle first is a Nope window, which is resolving the
	// card whose turn this still is and has its own clock.
	if wasTheirTurn && gs.NopeWindow == nil {
		gs.advanceTurn()
	}

	return forfeitResult(gs)
}

func forfeitResult(gs *EKGameState) *ActionResult {
	return &ActionResult{
		State:    gs,
		Messages: []WSMessage{{Type: "state_updated", Payload: gs}},
	}
}
