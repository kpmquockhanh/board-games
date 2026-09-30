<template>
  <div class="container">
    <p class="subtitle">Pick a game to play with friends. Everyone who opens the same link joins the same room.</p>

    <div class="games-grid">
      <router-link to="/hotpot" class="game-card">
        <div class="card-icon"><Soup :size="36" /></div>
        <div class="card-title">Hotpot Night</div>
        <div class="card-desc">Cook together at a virtual table. Drop in ingredients, cheer, and chat while the broth simmers.</div>
        <div class="card-badge"><span class="dot"></span> Multiplayer</div>
        <div class="card-btn">Sit down →</div>
      </router-link>

      <router-link to="/exploding-kitchen" class="game-card">
        <div class="card-icon"><Bomb :size="36" /></div>
        <div class="card-title">Exploding Kitchen</div>
        <div class="card-desc">A card game of cooking chaos. Draw cards, avoid the exploding kitchen, use tactics to be the last chef standing.</div>
        <div class="card-badge"><span class="dot"></span> 2–6 Players</div>
        <div class="card-btn">Start cooking →</div>
      </router-link>

      <div class="coming-soon">
        <div class="card-icon"><Target :size="36" /></div>
        <div class="card-title">Coming Soon</div>
        <div class="card-desc">More games are on the way. Stay tuned for new ways to hang out with friends.</div>
        <div class="card-btn secondary" disabled>Not yet available</div>
      </div>
    </div>

    <section v-if="seats.length" class="account-section" data-testid="hub-seats">
      <h2>Your tables</h2>
      <ul class="rows">
        <li v-for="seat in seats" :key="seat.game + seat.room_key" class="row">
          <span class="dot" :style="{ background: seat.color }"></span>
          <span class="row-main">
            <span class="row-title">{{ seat.room_name || seat.room_key }}</span>
            <span class="row-sub">{{ gameName(seat.game) }} · as {{ seat.player_name }}<template v-if="seat.held"> · waiting for you</template></span>
          </span>
          <button class="row-btn" @click="rejoin(seat)">Rejoin</button>
        </li>
      </ul>
    </section>

    <section v-if="playedWith.length" class="account-section" data-testid="hub-played-with">
      <h2>Played with</h2>
      <ul class="rows">
        <li v-for="p in playedWith" :key="p.name + p.last_played_at" class="row">
          <span class="dot" :style="{ background: p.color }"></span>
          <span class="row-main">
            <span class="row-title">{{ p.name }}</span>
            <span class="row-sub">{{ p.games }} {{ p.games === 1 ? 'game' : 'games' }} together · last {{ when(p.last_played_at) }}</span>
          </span>
          <button v-if="inviteSeat" class="row-btn secondary" @click="invite(p)">Invite</button>
        </li>
      </ul>
      <p v-if="!inviteSeat" class="section-note">
        {{ signedIn ? 'Sit down at a table to invite them to it.' : 'Sign in to invite them to your table from here, or send them its link from the game.' }}
      </p>
    </section>

    <section v-if="matches.length" class="account-section" data-testid="hub-matches">
      <h2>Recent games</h2>
      <ul class="rows">
        <li v-for="m in matches" :key="m.id" class="row">
          <span :class="['result', youWon(m) ? 'won' : 'lost']">{{ youWon(m) ? 'Won' : 'Lost' }}</span>
          <span class="row-main">
            <span class="row-title">{{ m.room_name || gameName(m.game) }}</span>
            <span class="row-sub">{{ gameName(m.game) }} · {{ when(m.ended_at) }} · {{ m.players.map((p) => p.name).join(', ') }}</span>
          </span>
        </li>
      </ul>
    </section>

    <footer>
      Built with warmth · <a href="https://github.com">Ping</a>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { toast } from 'vue-sonner'
import { useRouter } from 'vue-router'
import { Soup, Bomb, Target } from '@lucide/vue'
import { getMySeats, getMyMatches, getPlayedWith, getAuthSession } from '../api'
import { saveSavedPlayer } from '../savedPlayer'

const GAMES = {
  ek: { name: 'Exploding Kitchen', path: '/exploding-kitchen' },
  hotpot: { name: 'Hotpot Night', path: '/hotpot' },
}

const router = useRouter()
const seats = ref([])
const matches = ref([])
const playedWith = ref([])
const signedIn = ref(false)

// An invite is the link to the newest table the account is sitting at. There
// is no messaging here, so it goes to the clipboard to send however they talk.
const inviteSeat = computed(() => seats.value.find((s) => GAMES[s.game]))

function gameName(game) {
  return GAMES[game]?.name || game
}

function youWon(match) {
  return match.players.some((p) => p.you && p.won)
}

function when(iso) {
  const d = new Date(iso)
  const mins = Math.round((Date.now() - d) / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins} min ago`
  if (mins < 60 * 24) return `${Math.round(mins / 60)} h ago`
  return d.toLocaleDateString()
}

// Remember the seat for this tab, then open the room: the game's page
// rejoins a remembered seat by itself, and for a signed-in account the
// server hands this device the seat's token.
function rejoin(seat) {
  const game = GAMES[seat.game]
  if (!game) return
  saveSavedPlayer(`${seat.game}-player-${seat.room_key}`, { name: seat.player_name, color: seat.color })
  router.push({ path: game.path, query: { room: seat.room_key } })
}

async function invite(person) {
  const seat = inviteSeat.value
  if (!seat) return
  const url = `${location.origin}${GAMES[seat.game].path}?room=${seat.room_key}`
  try {
    await navigator.clipboard.writeText(url)
    toast.success(`Link to ${seat.room_name || seat.room_key} copied. Send it to ${person.name}.`)
  } catch {
    toast.error(`Couldn't copy the link: ${url}`)
  }
}

onMounted(async () => {
  const [s, m, p, auth] = await Promise.all([getMySeats(), getMyMatches(), getPlayedWith(), getAuthSession()])
  signedIn.value = !!(auth?.user && !auth.user.guest)
  seats.value = s
  matches.value = m.slice(0, 5)
  playedWith.value = p
})
</script>

<style scoped>
.container {
  max-width: 900px;
  margin: 0 auto;
  padding: 40px 24px 80px;
}

.subtitle {
  color: var(--mild-cream);
  opacity: 0.7;
  font-size: 1.05rem;
  line-height: 1.6;
  max-width: 480px;
  margin: 0 auto 40px;
  text-align: center;
}

.games-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 24px;
  margin-bottom: 60px;
}

.game-card {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 32px 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: transform 0.2s ease, border-color 0.2s ease;
  position: relative;
  overflow: hidden;
}

.game-card:hover {
  transform: translateY(-4px);
  border-color: var(--chili-orange);
}

.card-icon {
  font-size: 2.6rem;
  line-height: 1;
  display: flex;
  align-items: center;
  color: var(--chili-orange);
}

.card-title {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 700;
  font-size: 1.35rem;
}

.card-desc {
  color: var(--mild-cream);
  opacity: 0.75;
  font-size: 0.92rem;
  line-height: 1.55;
  flex: 1;
}

.card-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.76rem;
  padding: 5px 12px;
  border-radius: 100px;
  width: fit-content;
  background: rgba(238, 194, 92, 0.1);
  color: var(--gold);
  border: 1px solid rgba(238, 194, 92, 0.2);
}

.card-badge .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--gold);
}

.card-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 24px;
  border-radius: 100px;
  font-weight: 600;
  font-size: 0.9rem;
  border: none;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
  transition: transform 0.15s ease;
}

.card-btn:hover {
  transform: translateY(-2px);
}

.card-btn.secondary {
  background: transparent;
  border: 1px solid var(--line);
  color: var(--mild-cream);
}

.coming-soon {
  background: var(--panel);
  border: 1px dashed rgba(253, 243, 228, 0.1);
  border-radius: 20px;
  padding: 32px 28px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  opacity: 0.4;
}

.coming-soon .card-icon {
  filter: grayscale(1);
}

.account-section {
  margin-bottom: 40px;
}

.account-section h2 {
  font-family: 'Baloo 2', sans-serif;
  font-size: 1.1rem;
  margin: 0 0 12px;
}

.rows {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 12px 16px;
}

.row .dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex: none;
}

.row-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.row-title {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-sub {
  color: var(--mild-cream);
  opacity: 0.65;
  font-size: 0.82rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-btn {
  flex: none;
  padding: 8px 18px;
  border-radius: 100px;
  border: none;
  font-weight: 600;
  cursor: pointer;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
}

.row-btn.secondary {
  background: transparent;
  border: 1px solid rgba(253, 243, 228, 0.2);
  color: var(--steam-cream);
}

.section-note {
  margin: 10px 4px 0;
  font-size: 0.85rem;
  color: var(--mild-cream);
  opacity: 0.6;
}

.result {
  flex: none;
  font-size: 0.75rem;
  font-weight: 700;
  padding: 3px 10px;
  border-radius: 100px;
  min-width: 44px;
  text-align: center;
}

.result.won {
  background: rgba(238, 194, 92, 0.15);
  color: var(--gold);
}

.result.lost {
  background: rgba(253, 243, 228, 0.06);
  color: var(--mild-cream);
  opacity: 0.7;
}

footer {
  text-align: center;
  color: var(--mild-cream);
  opacity: 0.4;
  font-size: 0.8rem;
  border-top: 1px solid var(--line);
  padding-top: 28px;
}

footer a {
  color: var(--gold);
  text-decoration: none;
}

@media (max-width: 600px) {
  .container {
    padding: 40px 16px 60px;
  }
  h1 {
    font-size: 1.8rem;
  }
  .games-grid {
    grid-template-columns: 1fr;
  }
}
</style>
