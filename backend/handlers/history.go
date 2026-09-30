package handlers

import (
	"log"
	"net/http"
	"time"

	"ping/game"
	"ping/models"

	"github.com/gin-gonic/gin"
)

// matchHistoryLimit is how many finished games an account is shown.
const matchHistoryLimit = 20

// companionLimit is how many people an account is shown it has played with.
const companionLimit = 20

// RegisterAccount routes what a browser can ask about its own account.
func (h *Handler) RegisterAccount(api gin.IRoutes) {
	api.GET("/me", h.Me)
	api.GET("/me/seats", h.MySeats)
	api.GET("/me/matches", h.MyMatches)
	api.GET("/me/played-with", h.MyCompanions)
}

// MySeats lists the tables a signed-in account is sitting at, so it can go
// back to one from any device. A guest gets nothing: without a login, the
// account is this browser, whose tabs already remember their own seats, and
// the seat could not be taken to another device anyway (see JoinRoom).
func (h *Handler) MySeats(c *gin.Context) {
	u := currentUser(c)
	if u == nil || u.Guest {
		c.JSON(http.StatusOK, gin.H{"seats": []models.Seat{}})
		return
	}
	seats, err := h.store.UserSeats(u.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"seats": seats})
}

// MyMatches is the account's finished games. A guest has a history too; it
// comes along when the guest signs in.
func (h *Handler) MyMatches(c *gin.Context) {
	u := currentUser(c)
	if u == nil {
		c.JSON(http.StatusOK, gin.H{"matches": []models.Match{}})
		return
	}
	matches, err := h.store.UserMatches(u.ID, matchHistoryLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

// MyCompanions is who the account has finished games with, for the hub's
// list of people to invite back. Like the history it comes from, a guest has
// one too.
func (h *Handler) MyCompanions(c *gin.Context) {
	u := currentUser(c)
	if u == nil {
		c.JSON(http.StatusOK, gin.H{"played_with": []models.Companion{}})
		return
	}
	companions, err := h.store.UserCompanions(u.ID, companionLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"played_with": companions})
}

// recordEKMatch writes a finished game into the history. It is only a
// record: failing to write it is logged and the game goes on regardless.
func (h *Handler) recordEKMatch(roomID int64, roomKey string, gs *game.EKGameState) {
	if gs == nil || gs.Phase != "ended" {
		return
	}
	m := models.MatchRecord{
		Game:      "ek",
		RoomID:    roomID,
		RoomKey:   roomKey,
		StartedAt: gs.StartedAt,
		EndedAt:   time.Now(),
	}
	// A game dealt before games were stamped. Its ending is only reached
	// once, so the time it ended names it well enough.
	if m.StartedAt.IsZero() {
		m.StartedAt = m.EndedAt
	}
	if room, err := h.store.GetRoomByID(roomID); err == nil && room != nil {
		m.RoomName = room.Name
	}
	won := map[string]bool{}
	for _, name := range gs.Winners {
		won[name] = true
	}
	for _, name := range gs.TurnOrder {
		p := gs.Players[name]
		if p == nil {
			continue
		}
		m.Players = append(m.Players, models.MatchPlayer{Name: name, Color: p.Color, Won: won[name]})
	}
	if err := h.store.RecordMatch(m); err != nil {
		log.Printf("[history] record match in %s: %v", roomKey, err)
	}
}
