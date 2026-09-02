package storage

import (
	"encoding/json"

	"ping/models"
)

type Store interface {
	CreateRoom(game string, maxPlayers int, name string) (*models.Room, error)
	GetRoom(roomKey string) (*models.Room, error)
	GetRoomByID(roomID int64) (*models.Room, error)
	UpdateRoomStatus(roomID int64, status string) error
	DeleteRoom(roomKey string) error
	ListActiveRooms(game string) ([]models.RoomListItem, error)

	AddPlayer(roomID int64, playerName, color string) error
	RemovePlayer(roomID int64, playerName string) error
	RemovePlayerByRoomKey(roomKey string, playerName string) error
	RemoveAllPlayers(roomID int64) error
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
}
