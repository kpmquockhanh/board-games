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

## Configuration

The backend reads its settings from the environment. Every variable is optional;
an unset or unusable value falls back to the default and is logged.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `./data/ping.db` | SQLite database file |
| `REAP_ENABLED` | `true` | Set false to leave rooms and seats alone entirely |
| `REAP_INTERVAL` | `5m` | How often the background sweep runs |
| `REAP_DROP_PLAYER_AFTER` | `5m` | How long a seat is held for a player whose socket dropped mid-game before they forfeit |
| `REAP_ABANDON_AFTER` | `10m` | How long an active room with nobody connected stays active |
| `REAP_DELETE_AFTER` | `24h` | How long an ended or abandoned room is kept before its rows are deleted |
| `REAP_TIMELINE_KEEP` | `200` | Non-snapshot timeline events retained per room |

Durations use Go's format: `30s`, `10m`, `24h`, `1h30m`. Keep
`REAP_DROP_PLAYER_AFTER` shorter than `REAP_ABANDON_AFTER` — otherwise an idle
room is abandoned before a dropped player is ever forfeited out of it, and one
player going quiet costs everyone at the table their game.

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
