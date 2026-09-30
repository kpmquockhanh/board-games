package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"ping/handlers"
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
	hub.OnDisconnect(h.HandleDisconnect)

	// Retires rooms everyone walked away from, and eventually deletes them.
	// Nothing else frees a room: a mid-game disconnect keeps its player's
	// seat, so without this a table that dropped out stays full forever.
	// Tunable through REAP_* — see handlers.ReaperConfigFromEnv.
	reaperCfg := handlers.ReaperConfigFromEnv()
	reaperCtx, stopReaper := context.WithCancel(context.Background())
	defer stopReaper()
	go h.RunReaper(reaperCtx, reaperCfg)

	auth := handlers.AuthFromEnv()
	h.UseAuth(auth)

	api := router.Group("/api")
	api.Use(h.Identify)
	{
		h.RegisterAccount(api)
		h.RegisterAuth(api)
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
	if names := auth.Names(); len(names) > 0 {
		fmt.Printf("   Login: %s\n", strings.Join(names, ", "))
	} else {
		fmt.Printf("   Login: off (set AUTH_GOOGLE_* or AUTH_DISCORD_*)\n")
	}
	if reaperCfg.Enabled {
		fmt.Printf("   Reaper: every %s — forfeit dropped players after %s, abandon idle rooms after %s, delete after %s, keep %d events\n",
			reaperCfg.Interval, reaperCfg.DropPlayerAfter, reaperCfg.AbandonAfter,
			reaperCfg.DeleteAfter, reaperCfg.TimelineKeep)
	} else {
		fmt.Printf("   Reaper: disabled\n")
	}
	log.Fatal(router.Run(addr))
}
