package main

import (
	"fmt"
	"log"
	"os"

	"ping/handlers"
	"ping/models"
	"ping/storage"
	ws "ping/ws"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/ping.db"
	}

	store, err := storage.NewSQLite(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer store.Close()

	hub := ws.NewHub()
	hub.OnDisconnect(func(room, player string) {
		if err := store.RemovePlayerByRoomKey(room, player); err != nil {
			log.Printf("[ws] failed to remove player %s from room %s: %v", player, room, err)
			return
		}

		hub.BroadcastToRoom(room, models.WSMessage{
			Type:   "player_left",
			Room:   room,
			Player: player,
		})
		log.Printf("[ws] player disconnected: room=%s player=%s", room, player)
	})
	go hub.Run()

	router := gin.New()
	// router.Use(gin.Logger())
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	h := handlers.NewHandler(store, hub)

	api := router.Group("/api")
	{
		api.POST("/:game/create", h.CreateRoom)
		api.POST("/:game/join", h.JoinRoom)
		api.GET("/:game/rooms", h.ListRooms)
		api.GET("/:game/rooms/:roomKey", h.GetRoom)
		api.DELETE("/:game/rooms/:roomKey", h.DeleteRoom)
		api.POST("/:game/rooms/:roomKey/leave", h.LeaveRoom)
		api.GET("/:game/rooms/:roomKey/timeline", h.GetTimeline)
		api.GET("/:game/rooms/:roomKey/state", h.GetState)
		api.POST("/:game/rooms/:roomKey/state", h.SaveState)
		api.GET("/:game/rooms/:roomKey/players", h.GetPlayers)
	}

	router.GET("/ws", h.HandleWS)

	addr := ":8080"
	fmt.Printf("🏓 Ping running at http://localhost%s\n", addr)
	fmt.Printf("   WebSocket: ws://localhost%s/ws?room={room}&player={name}\n", addr)
	fmt.Printf("   Create: POST http://localhost%s/api/{game}/create\n", addr)
	fmt.Printf("   Join: POST http://localhost%s/api/{game}/join\n", addr)
	fmt.Printf("   Database: %s\n", dbPath)
	log.Fatal(router.Run(addr))
}
