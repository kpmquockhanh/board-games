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
| `REAP_GUESTS_AFTER` | `2160h` (90 days) | How long a guest account is kept after its browser was last seen |
| `AUTH_BASE_URL` | from the request | Public address of the site, e.g. `https://ping.example.com`. Login's callback is `<AUTH_BASE_URL>/api/auth/<provider>/callback`; register exactly that with each provider |
| `AUTH_GOOGLE_CLIENT_ID` / `AUTH_GOOGLE_CLIENT_SECRET` | unset | Turn on "Sign in with Google" (needs both) |
| `AUTH_DISCORD_CLIENT_ID` / `AUTH_DISCORD_CLIENT_SECRET` | unset | Turn on "Sign in with Discord" (needs both) |

Durations use Go's format: `30s`, `10m`, `24h`, `1h30m`. Keep
`REAP_DROP_PLAYER_AFTER` shorter than `REAP_ABANDON_AFTER` — otherwise an idle
room is abandoned before a dropped player is ever forfeited out of it, and one
player going quiet costs everyone at the table their game.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/me` | The browser's account (made as a guest if it has none) |
| GET | `/api/me/seats` | Tables a signed-in account is sitting at, to rejoin from any device (empty for a guest) |
| GET | `/api/me/matches` | The account's last 20 finished games, newest first |
| GET | `/api/me/played-with` | People the account has finished games with (by account, under their latest name), most recent first |
| GET | `/api/auth/session` | Who the browser is signed in as, and the providers on offer (never makes a guest) |
| GET | `/api/auth/:provider/login?return=/path` | Start logging in; comes back to `return` |
| GET | `/api/auth/:provider/callback` | Where the provider sends the browser back |
| POST | `/api/auth/logout` | Sign this browser out |
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

### Identity

- **Seat token (per tab).** Joining returns a `seat_token`. Send it as the
  `X-Seat-Token` header on requests about that room, and as `&token=` on the
  WebSocket URL. It is what proves a seat is yours; the player name alone is
  not trusted.
- **Account (per browser).** Joining also sets an httpOnly `ping_session`
  cookie for a guest account, made on first join. All tabs of a browser share
  it, so one account can hold several seats. Nothing requires an account; it
  records who sat where.
- **Login (optional).** With a provider configured, the menu on the hub and
  Hotpot pages offers "Sign in with …" (OAuth 2 with PKCE). Signing in upgrades
  the browser's guest in place, so its seats come along; signing in on a second
  browser folds that browser's guest into the account. The session cookie is
  replaced on every login. Logging in with an identity the site has never
  seen upgrades a guest, but never joins an account already signed in on that
  browser: it gets an account of its own. Signing out ends only this browser's
  session and never takes anyone out of a game.
- **What an account gets you.** The join form is filled in with the name and
  colour you last played as (signed in only). The hub lists your tables, and
  each has a Rejoin button: signed in, you can take your seat to another
  device by joining under its name without the old tab's token, as long as
  the seat is yours and nobody is connected on it. The hub also shows your
  recent games. A match is recorded when an Exploding Kitchen game ends, and
  stays after its room is deleted.
- **Played with.** The hub lists the people you have finished games with, one
  row per account however many names they used, and an Invite button that
  copies the link to your newest table (signed in, while you are sitting at
  one). There are no friend requests or messages; you send the link yourself.

## WebSocket

Connect to `ws://localhost:8080/ws?room={room}&player={name}` for real-time game updates.
