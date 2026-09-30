package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// A seat token is the secret a player is handed when they take a seat, and
// what proves the seat is theirs from then on. The name used to be the whole
// identity: anyone who typed a player's name could act as them, and asking for
// their state or opening their socket showed their hand.
//
// Only a hash is stored, so the database alone cannot be used to sit in
// someone else's seat.
const seatTokenHeader = "X-Seat-Token"

// newSecret makes a bearer secret and the hash to store in its place: a seat
// token, or the session in a user's cookie.
func newSecret() (token, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, hashSecret(token), nil
}

func hashSecret(token string) string {
	if token == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// presentedSeatToken reads the token a request carries. REST calls send it as
// a header; a browser cannot set headers on a WebSocket, so sockets send it in
// the query string.
func presentedSeatToken(c *gin.Context) string {
	if t := c.GetHeader(seatTokenHeader); t != "" {
		return t
	}
	return c.Query("token")
}

// seatedPlayer is the player whose seat the request's token is for, provided
// they are still sitting in it; "" otherwise.
func (h *Handler) seatedPlayer(c *gin.Context, roomID int64) (string, error) {
	name, seated, err := h.store.SeatHolder(roomID, hashSecret(presentedSeatToken(c)))
	if err != nil || !seated {
		return "", err
	}
	return name, nil
}
