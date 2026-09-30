package storage

import (
	"encoding/json"
	"time"

	"ping/models"
)

type Store interface {
	CreateRoom(game string, maxPlayers int, name string) (*models.Room, error)
	GetRoom(roomKey string) (*models.Room, error)
	GetRoomByID(roomID int64) (*models.Room, error)
	UpdateRoomStatus(roomID int64, status string) error
	DeleteRoom(roomKey string) error
	ListActiveRooms(game string) ([]models.RoomListItem, error)
	TouchRoom(roomID int64) error
	ListIdleRooms(statuses []string, cutoff time.Time) ([]models.Room, error)
	PruneTimelines(keep int) (int64, error)

	AddPlayer(roomID int64, playerName, color, seatTokenHash, userID string) error
	SetSeatUser(roomID int64, playerName, userID string) error
	SeatHolder(roomID int64, seatTokenHash string) (name string, seated bool, err error)
	ClaimUntokenedSeat(roomID int64, playerName, seatTokenHash string) (bool, error)
	ReissueSeatToken(roomID int64, playerName, userID, seatTokenHash string) (bool, error)
	RemovePlayer(roomID int64, playerName string) error
	RemovePlayerByRoomKey(roomKey string, playerName string) error
	RemoveAllPlayers(roomID int64) error
	MarkPlayerDisconnected(roomID int64, playerName string) error
	MarkPlayerConnected(roomID int64, playerName string) (bool, error)
	ListStalePlayers(cutoff time.Time) ([]models.StalePlayer, error)
	GetRoomPlayers(roomID int64) ([]models.RoomPlayer, error)
	IsPlayerInRoom(roomID int64, playerName string) (bool, error)
	GetActivePlayerCount(roomID int64) (int, error)
	SetPlayerReady(roomID int64, playerName string, ready bool) error

	AddTimelineEvent(roomID int64, eventType, player string, payload string) error
	GetTimeline(roomID int64, limit int) ([]models.TimelineEvent, error)
	MergeState(roomID int64, game string, action string, data json.RawMessage) (json.RawMessage, error)
	SaveSnapshot(roomID int64, player string, state string) error
	UpdateSnapshot(roomID int64, player string, state string) error
	GetLatestSnapshot(roomID int64) (string, bool, error)

	CreateGuest(sessionTokenHash string) (*models.User, error)
	UserBySession(sessionTokenHash string) (*models.User, error)
	RememberPlayerProfile(userID, name, color string) error
	DeleteIdleGuests(cutoff time.Time) (int64, error)
	SignIn(sessionTokenHash, previousSessionHash string, id models.Identity) (*models.User, error)
	EndSession(sessionTokenHash string) error
	UserProviders(userID string) ([]string, error)
	UserSeats(userID string) ([]models.Seat, error)

	RecordMatch(m models.MatchRecord) error
	UserMatches(userID string, limit int) ([]models.Match, error)
	UserCompanions(userID string, limit int) ([]models.Companion, error)
}
