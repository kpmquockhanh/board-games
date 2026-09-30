package handlers

import (
	"log"
	"net/http"
	"time"

	"ping/models"

	"github.com/gin-gonic/gin"
)

// A browser's account is kept by a cookie holding a session secret. The
// cookie is shared by every tab, so it says which person is playing, not which
// seat: two tabs of one browser are one account sitting in two seats, and the
// per-tab seat token still decides which seat a request speaks for.
//
// Nothing is refused for want of an account yet. It is recorded alongside
// the seat so that a later login can bring a guest's history with it.
const (
	sessionCookie = "ping_session"
	// sessionMaxAge is as long as browsers keep a cookie. It is renewed on
	// every join, so only a browser that stops playing lets it run out.
	sessionMaxAge = 400 * 24 * time.Hour
	userKey       = "user"
)

// Identify finds the account behind a request's cookie, if any, for the
// handlers after it. It never makes one: a page that only lists rooms does
// not need an account, and making one on every read would race concurrent
// first requests into several.
func (h *Handler) Identify(c *gin.Context) {
	if raw, err := c.Cookie(sessionCookie); err == nil && raw != "" {
		u, err := h.store.UserBySession(hashSecret(raw))
		if err != nil {
			log.Printf("[auth] look up session: %v", err)
		} else if u != nil {
			c.Set(userKey, u)
		}
	}
	c.Next()
}

func currentUser(c *gin.Context) *models.User {
	if v, ok := c.Get(userKey); ok {
		return v.(*models.User)
	}
	return nil
}

// ensureUser returns the request's account, making a guest for a browser
// that has none, and renews the cookie either way. An account is a record
// here, not a gate: if one cannot be made the player still plays, so a
// failure is logged and yields "".
func (h *Handler) ensureUser(c *gin.Context) string {
	if u := currentUser(c); u != nil {
		if raw, err := c.Cookie(sessionCookie); err == nil {
			setSessionCookie(c, raw)
		}
		return u.ID
	}
	token, hash, err := newSecret()
	if err != nil {
		log.Printf("[auth] make session: %v", err)
		return ""
	}
	u, err := h.store.CreateGuest(hash)
	if err != nil {
		log.Printf("[auth] make guest: %v", err)
		return ""
	}
	setSessionCookie(c, token)
	c.Set(userKey, u)
	return u.ID
}

func setSessionCookie(c *gin.Context, token string) {
	// Secure whenever the browser reached us over TLS, directly or through
	// the proxy in front. Plain http on localhost would drop a Secure cookie.
	secure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionMaxAge / time.Second),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// rememberProfile keeps the name and colour a user just sat down with.
func (h *Handler) rememberProfile(userID, name, color string) {
	if userID == "" {
		return
	}
	if err := h.store.RememberPlayerProfile(userID, name, color); err != nil {
		log.Printf("[auth] remember profile: %v", err)
	}
}

// Me is the account behind this browser, made now if it has none.
func (h *Handler) Me(c *gin.Context) {
	if h.ensureUser(c) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not make an account"})
		return
	}
	c.JSON(http.StatusOK, currentUser(c))
}
