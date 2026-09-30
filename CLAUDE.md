# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

"Ping Games": real-time multiplayer board games. Go backend (Gin, gorilla/websocket, pure-Go SQLite via `modernc.org/sqlite`, module name `ping`) and a Vue 3 + Vite + Pinia frontend. Two games: **Hotpot** (`hotpot`, route `/hotpot`) and **Exploding Kitchen** (`ek`, route `/exploding-kitchen`, an Exploding Kittens clone).

## Commands

Full stack in Docker (frontend :3000, backend :8080 internal, sqlite-web :8081). The backend container hot-reloads with CompileDaemon:
```bash
./dev.sh                                             # docker compose -f docker-compose.dev.yml up --build
docker compose -f docker-compose.prod.yml up --build -d
```

Without Docker:
```bash
cd backend && go run .                   # :8080, DB at ./data/ping.db (override with DB_PATH)
cd frontend && npm install && npm run dev  # :3000, proxies /api and /ws to VITE_API_PROXY (default http://localhost:8080)
cd frontend && npm run build
```

Tests (backend only; the frontend has no test or lint setup):
```bash
cd backend && go test ./...
cd backend && go test ./handlers -run TestReloadIsNotRefusedWhileTheOldSocketLingers -v   # single test
```
`game/carddata_test.go` reads `../../frontend/src/data/cards.json`, so run it from inside `backend/`.

Backend configuration is env-only (`DB_PATH`, `REAP_*`); see the table in README.md and `handlers/reaper_config.go`.

## Architecture

### Two different state models
- **Hotpot is client-authoritative.** Clients POST partial state to `POST /api/:game/rooms/:roomKey/state`; the server deep-merges it into the latest snapshot (`storage.MergeState` → `mergeHotpotState`), validates (`models.ValidateState`), stores it, and clients sync over WS.
- **EK is server-authoritative.** The same endpoint routes to `handlers.handleEKAction`. Lobby actions (`toggleReady`, `updateSettings`, `rematch`, `join`) are handled there; everything else goes through `game.ProcessAction(gs, GameAction)` in `backend/game/ek.go`, a pure rules engine returning an `ActionResult` (new state, broadcast messages, an optional private prompt, or an `Error` string phrased for the player). `applyEKResult` then persists, arms timers, broadcasts, and sends the prompt.
- Allowed action names per game are whitelisted in `models/types.go` (`validActions`). A new action must be added there, or it is rejected before reaching the engine.

### EK hidden information (important)
`EKGameState` holds every hand and the deck order and must never be sent to a client. Always send `gs.ViewFor(player)` (`GameView`); `broadcastEKState` sends one view per connected player. The design goes further than hiding hands: Nope windows open for every nopeable play and last the same time regardless of who holds a Nope, so neither timing nor UI shortcuts reveal who holds what. Keep it that way.

### EK concurrency and timers
Each room has a mutex (`getEKMutex`) held for every action *and* by timer callbacks (Nope window, pending-prompt timeout, turn timer in `armEKTimers`). Live state is cached in `Handler.ekStates` and persisted as a JSON snapshot in the `timelines` table (`models.EKRoomState` = settings + game state). After a server restart, `loadEKState` restores from the snapshot and `rearmEKTimers` restarts the clocks.

### Storage
`storage.Store` interface, `storage/sqlite.go` implementation. Tables: `rooms`, `room_players` (with `ready`, `left_at`, `disconnected_at`, `seat_token`, `user_id`), `timelines` (events plus `snapshot` rows holding full state), and `users` / `user_sessions` / `user_identities` (`storage/users.go`), and `matches` / `match_players` (`storage/history.go`: finished games, written by `recordEKMatch` from `applyEKResult` when a game ends; idempotent on (room_key, started_at); no FK to `rooms`, so history outlives the room). Migrations are inline in `migrate()` / `migrateUsers()` / `migrateHistory()`.

### Identity: seat tokens and accounts
- **Seat token** (`handlers/seat.go`, per tab, `frontend/src/seatToken.js`): issued on join, stored hashed in `room_players.seat_token`. It is the only proof of a seat; REST sends `X-Seat-Token`, the WS sends `&token=`. Never trust a player name from the client, and derive the acting player from the token (`seatedPlayer`).
- **Account** (`handlers/user.go`, per browser): httpOnly `ping_session` cookie → `user_sessions` → `users`. `Identify` middleware only reads it; `ensureUser` makes a guest, and only join and `GET /api/me` call it. It is recorded on the seat (`room_players.user_id`) and on history rows. The cookie is shared by all tabs, so it must never be used to decide *which seat* a request is for. The one exception is narrow: a *signed-in* (non-guest) account joining under a seat's name, without its token, may take that seat to this device (`takeSeatToThisDevice` → `ReissueSeatToken`, conditional on `user_id`), and only while nobody is connected on it. A guest never can, since two people may share a browser's guest. `GET /api/me/seats` / `/api/me/matches` / `/api/me/played-with` (`handlers/history.go`) feed the hub's "Your tables" / "Recent games" / "Played with". Played-with groups by `match_players.user_id`, never by name, and never returns other users' ids.
- **Login** (`handlers/auth.go`, `storage.SignIn`): optional OAuth 2 + PKCE with Google/Discord, on when `AUTH_*_CLIENT_ID`/`_SECRET` are set. `user_identities` maps (provider, subject) to a user. Signing in upgrades the current guest in place, or folds it into an existing account (seats and matches move, guest is deleted), and always rotates the session. A new identity never joins an account that is already signed in; it gets its own, otherwise the next person to log in on a shared browser is handed that account. The flow's state and verifier ride in a short-lived `ping_oauth` cookie. The callback URL comes from `AUTH_BASE_URL`, else `X-Forwarded-Host` (the Vite proxy sets `xfwd`, nginx sets the header), else Host. Tests use a fake provider (`auth_test.go`).

### WebSocket, presence, and reconnects
`ws/hub.go` keeps clients per room; `/ws?room=&player=&session=`. Much of the recent work is about reload and reconnect correctness, and the code comments explain why. In short:
- `session` is a per-tab id (`frontend/src/session.js`, sessionStorage). It lets a reloading tab evict its own stale socket (`Hub.EvictSession`) instead of looking like a second player using the same name.
- Each arrival on a seat gets an increasing **epoch** (`Handler.arrive`). A disconnect whose epoch is older than the seat's current one comes from a replaced page and is ignored (`HandleDisconnect` / `stillHere`). So `JoinRoom` must only `arrive` once the join is certain to succeed: a refused join that bumped the epoch would make the real holder's later drop look stale, and their seat would never be held.
- `JoinRoom` evicts every socket with the caller's session, since it cannot tell a reload's ghost from the tab's live socket. So a client must rejoin over REST *before* reopening its socket (`ws.beforeReconnect`), never after it opens. Rejoining after open closed the new socket, whose reconnect rejoined, and so on forever.
- Dropping mid-game does not remove the player. It sets `disconnected_at` and holds the seat. `markBack` clears it and broadcasts `player_reconnected`. In a lobby, a drop means the player left.
- `handlers/reaper.go` (background, `REAP_*`) forfeits players held longer than `REAP_DROP_PLAYER_AFTER` (`game.ForfeitPlayer`), abandons idle rooms, deletes old ones, and prunes timelines. Rooms stay alive through `touch()` / `last_activity_at`.
- Frontend: `savedPlayer.js` stores identity per room in sessionStorage (authoritative, per tab) with a localStorage fallback. This is so two players in two tabs of one browser don't overwrite each other.

A reload must put the player back on exactly the same screen (lobby ready state, in-game, pending prompts) with no extra clicks.

### Frontend layout
`views/` has one view per game plus `HubView`. Game logic lives in the Pinia stores (`stores/explodingKitchen.js`, `stores/hotpot.js`), with the shared socket in `stores/ws.js` and REST calls in `api.js`. EK UI is in `components/game/` (`EkView` switches between Create/Join/Choose/Lobby phases and `EkGameView`). Card art is `public/cards/*.jpg`, and card metadata is `src/data/cards.json`.

### Card data lives in two places
`backend/game/deck.go` (`CardCategories`) and `frontend/src/data/cards.json` must describe the same cards (ids, names, categories). `TestCardTableAgreesWithTheClientCardData` enforces this, so update both together.

## Verifying changes
For bugs involving reloads, reconnects, or two players, Go tests alone aren't enough: some causes are purely client-side. Run the backend and the Vite dev server, reproduce the scenario in a real browser with two players in two tabs of the same browser, and confirm the symptom is gone.
