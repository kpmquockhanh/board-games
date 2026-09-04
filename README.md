# Ping Games

A real-time multiplayer board game platform built with Go and Vue.js.

## Tech Stack

**Backend:** Go, Gin, WebSocket (gorilla/websocket), SQLite  
**Frontend:** Vue 3, Vite, Pinia, Vue Router  
**Infrastructure:** Docker, Docker Compose

## Games

- **Hotpot** - A multiplayer game
- **Ek** - A card-based game

## Getting Started

### Development

```bash
chmod +x dev.sh
./dev.sh
```

Services:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- SQLite Web UI: http://localhost:8081

### Production

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/:game/create` | Create a new room |
| POST | `/api/:game/join` | Join a room |
| GET | `/api/:game/rooms` | List all rooms |
| GET | `/api/:game/rooms/:roomKey` | Get room details |
| DELETE | `/api/:game/rooms/:roomKey` | Delete a room |
| POST | `/api/:game/rooms/:roomKey/leave` | Leave a room |
| GET | `/api/:game/rooms/:roomKey/timeline` | Get room timeline |
| GET | `/api/:game/rooms/:roomKey/state` | Get game state |
| POST | `/api/:game/rooms/:roomKey/state` | Save game state |
| GET | `/api/:game/rooms/:roomKey/players` | Get room players |

## WebSocket

Connect to `ws://localhost:8080/ws?room={room}&player={name}` for real-time game updates.
