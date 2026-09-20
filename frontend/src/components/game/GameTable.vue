<template>
  <div class="table-zone" ref="tableZone">
    <!-- Opponents sit on a wrapping rail above the table and the local player
         sits below it. The seats used to be absolutely positioned around an
         ellipse whose radius was derived from the zone width alone, which
         ignored the seat's own width — at phone widths the left and right
         avatars ended up underneath the centred table. Flow layout makes that
         overlap structurally impossible, so there are no magic offsets to
         keep in sync with the table size. -->
    <div class="seat-rail" v-if="opponents.length">
      <div
        v-for="name in opponents"
        :key="name"
        class="player-seat"
        :data-testid="'player-seat-' + name"
        :data-alive="store.gameState?.players[name]?.alive"
        :class="{
          'active-turn': store.gameState?.turn === name,
          eliminated: !store.gameState?.players[name]?.alive,
          offline: store.isOffline(name)
        }"
      >
        <div class="avatar-wrapper">
          <div
            class="avatar"
            :style="{
              background: store.gameState?.players[name]?.color,
              color: readableInk(store.gameState?.players[name]?.color),
            }"
            :aria-label="`${name}${!store.gameState?.players[name]?.alive ? ' (eliminated)' : ''}`"
          >{{ name.trim().slice(0, 2).toUpperCase() }}</div>
          <div class="card-count-badge" :data-testid="'hand-count-' + name">
            {{ store.gameState?.players[name]?.handCount || 0 }}
          </div>
        </div>
        <div class="player-name">
          {{ name }}
          <span v-if="store.isOffline(name)" class="offline-tag">offline</span>
        </div>
        <div v-if="store.gameState?.turn === name" class="turn-indicator" data-testid="turn-indicator">Turn</div>
      </div>
    </div>

    <div class="table-band">
      <div class="table-surface">
        <div
          class="draw-pile"
          data-testid="draw-pile"
          role="button"
          tabindex="0"
          @click="$emit('drawCard')"
          @keydown.enter="$emit('drawCard')"
          @keydown.space.prevent="$emit('drawCard')"
        >
          <div class="draw-card" :class="{ 'pile-shake': pileAnimating }" ref="drawCardEl">
            <div class="draw-card-inner">
              <Flame class="draw-icon" :size="30" aria-hidden="true" />
              <span class="draw-label">DRAW</span>
            </div>
          </div>
          <div class="deck-count">
            <span class="deck-count-num" data-testid="deck-count">{{ store.drawPileCount }}</span>
            <span class="deck-count-label">left</span>
          </div>
        </div>

        <div v-if="store.recentlyPlayed.length > 0" class="discard-zone">
          <div class="discard-label">DISCARD</div>
          <div class="discard-stack">
            <div
              v-for="(entry, i) in store.recentlyPlayed.slice(0, 3).reverse()"
              :key="entry.key"
              class="discard-card"
              :style="{ '--i': i }"
            >
              <img :src="`/cards/${entry.card.file}`" :alt="entry.card.name" loading="lazy">
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="store.turnTimeLeft > 0"
        class="turn-timer"
        :class="{ urgent: store.turnTimeLeft <= 5 }"
        role="timer"
        aria-live="off"
      >
        <Timer :size="13" />
        <span>{{ store.turnTimeLeft }}s</span>
      </div>

      <div v-if="store.attackStackCount > 0" class="attack-indicator" role="status" aria-live="polite">
        <Zap class="attack-icon" :size="14" />
        <span class="attack-count" data-testid="attack-count">{{ store.attackStackCount }} forced draw{{ store.attackStackCount > 1 ? 's' : '' }}</span>
      </div>
    </div>

    <!-- Iterated over a 0-or-1 array rather than written out under a v-if, so
         the local player's seat is the same markup as everyone else's. -->
    <div class="seat-rail mine" v-if="mySeat.length">
      <div
        v-for="name in mySeat"
        :key="name"
        class="player-seat is-me"
        :data-testid="'player-seat-' + name"
        :data-alive="store.gameState?.players[name]?.alive"
        :class="{
          'active-turn': store.gameState?.turn === name,
          eliminated: !store.gameState?.players[name]?.alive,
          offline: store.isOffline(name)
        }"
      >
        <div class="avatar-wrapper">
          <div
            class="avatar"
            :style="{
              background: store.gameState?.players[name]?.color,
              color: readableInk(store.gameState?.players[name]?.color),
            }"
            :aria-label="`${name}${!store.gameState?.players[name]?.alive ? ' (eliminated)' : ''}`"
          >{{ name.trim().slice(0, 2).toUpperCase() }}</div>
          <div class="card-count-badge" :data-testid="'hand-count-' + name">
            {{ store.gameState?.players[name]?.handCount || 0 }}
          </div>
        </div>
        <div class="player-name">
          {{ name }} (you)
          <span v-if="store.isOffline(name)" class="offline-tag">offline</span>
        </div>
        <div v-if="store.gameState?.turn === name" class="turn-indicator mine" data-testid="turn-indicator">Your turn</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Zap, Timer, Flame } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import { readableInk } from '../../utils/contrast'

const store = useEkStore()
const tableZone = ref(null)
const drawCardEl = ref(null)
const pileAnimating = ref(false)

const mySeat = computed(() => {
  const me = store.me?.name
  return me && store.turnOrder.includes(me) ? [me] : []
})

const opponents = computed(() =>
  store.turnOrder.filter((name) => name !== store.me?.name)
)

watch(
  () => store.drawPileCount,
  (newCount, oldCount) => {
    if (oldCount && newCount < oldCount) {
      pileAnimating.value = true
      setTimeout(() => { pileAnimating.value = false }, 400)
    }
  }
)

defineExpose({
  tableZone,
  getDeckRect: () => drawCardEl.value?.getBoundingClientRect() || null,
})
defineEmits(['drawCard'])
</script>

<style scoped>
.table-zone {
  position: relative;
  flex: 1;
  min-height: 340px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  padding: 10px 12px;
  overflow: visible;
}

.seat-rail {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: flex-start;
  gap: 8px 14px;
  flex-shrink: 0;
}

.table-band {
  position: relative;
  flex: 0 1 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 0;
}

.table-surface {
  width: 320px;
  max-width: 100%;
  min-height: 190px;
  background: var(--table);
  border-radius: var(--radius-lg);
  border: var(--edge-w) solid var(--edge);
  box-shadow: var(--lift-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 28px;
  padding: 14px;
}

.draw-pile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: transform var(--duration-fast) var(--ease-out);
}

.draw-pile:hover .draw-card {
  transform: translateY(-3px);
  box-shadow: 0 8px 0 var(--edge);
}

.draw-pile:active .draw-card {
  transform: translateY(3px);
  box-shadow: 0 1px 0 var(--edge);
}

.draw-pile:focus-visible {
  outline: 3px solid var(--chili);
  outline-offset: 4px;
}

.draw-card.pile-shake {
  animation: pileShake 0.4s ease;
}

@keyframes pileShake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-3px) rotate(-1.5deg); }
  40% { transform: translateX(3px) rotate(1.5deg); }
  60% { transform: translateX(-2px) rotate(-0.8deg); }
  80% { transform: translateX(2px) rotate(0.8deg); }
}

.draw-card {
  width: 80px;
  height: 112px;
  border-radius: var(--radius-sm);
  background: var(--chili);
  border: var(--edge-w) solid var(--edge);
  box-shadow: 0 5px 0 var(--edge);
  color: var(--ink);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  transition:
    transform var(--duration-fast) var(--ease-out),
    box-shadow var(--duration-fast) var(--ease-out);
}

/* A faint check keeps the flat fill from looking like an unloaded image. */
.draw-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: repeating-conic-gradient(rgba(27, 14, 6, 0.07) 0% 25%, transparent 0% 50%) 50%/14px 14px;
}

.draw-card-inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  position: relative;
  z-index: 1;
}

.draw-icon {
  flex-shrink: 0;
}

.draw-label {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 0.9rem;
  color: var(--ink);
  letter-spacing: 0.1em;
}

.deck-count {
  display: flex;
  align-items: baseline;
  gap: 4px;
  padding: 2px 10px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--surface);
  font-family: 'Baloo 2', sans-serif;
}

.deck-count-num {
  font-size: 1rem;
  font-weight: 800;
  color: var(--ink);
}

.deck-count-label {
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--muted-cream);
}

.discard-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.discard-label {
  padding: 2px 9px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--surface);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.6rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--ink);
  font-weight: 800;
}

.discard-stack {
  position: relative;
  width: 70px;
  height: 98px;
}

.discard-card {
  position: absolute;
  width: 70px;
  height: 98px;
  border-radius: 10px;
  border: 2.5px solid var(--edge);
  background: var(--surface);
  overflow: hidden;
  transition: transform var(--duration-normal) ease;
  transform: rotate(calc(var(--i) * -4deg)) translateY(calc(var(--i) * 6px));
}

.discard-card:first-child {
  box-shadow: 0 3px 0 var(--edge);
}

.discard-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.discard-card:hover {
  transform: rotate(0deg) translateY(-6px) scale(1.05);
  z-index: var(--z-cards);
}

.attack-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.82rem;
  font-weight: 800;
  padding: 5px 14px;
  border-radius: 100px;
  background: var(--broth);
  color: var(--surface);
  border: 2.5px solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  white-space: nowrap;
  animation: attack-pulse 2s ease-in-out infinite;
}

.attack-icon {
  flex-shrink: 0;
}

@keyframes attack-pulse {
  0%, 100% { transform: rotate(-1.5deg); }
  50% { transform: rotate(1.5deg); }
}

.turn-timer {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 12px;
  border-radius: 100px;
  border: 2.5px solid var(--edge);
  background: var(--surface);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.8rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.turn-timer.urgent {
  background: var(--broth);
  color: var(--surface);
}

/* ─── Seats ─── */

.player-seat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  width: 92px;
  text-align: center;
  transition: opacity var(--duration-slow) ease;
}

.avatar-wrapper {
  position: relative;
  display: inline-block;
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.2rem;
  color: var(--ink);
  border: 3px solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  transition:
    box-shadow var(--duration-normal) ease,
    filter var(--duration-slow) ease;
}

.player-seat.is-me .avatar {
  box-shadow: 0 0 0 4px var(--gold), 0 3px 0 var(--edge);
}

/* Greying out by opacity is unreadable on a light ground, so an eliminated
   chef desaturates onto the tan instead. */
.player-seat.eliminated .avatar {
  filter: grayscale(1) brightness(1.08);
}

.player-seat.eliminated .player-name {
  color: var(--dim-text);
  text-decoration: line-through;
}

.card-count-badge {
  position: absolute;
  bottom: -4px;
  right: -6px;
  min-width: 24px;
  height: 24px;
  padding: 0 5px;
  border-radius: 100px;
  background: var(--surface);
  border: 2.5px solid var(--edge);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.72rem;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.player-seat.active-turn .avatar {
  box-shadow: 0 0 0 4px var(--chili), 0 3px 0 var(--edge);
}

.player-name {
  font-size: 0.74rem;
  font-weight: 800;
  color: var(--mild-cream);
  white-space: nowrap;
  max-width: 92px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.turn-indicator {
  display: inline-block;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.6rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--ink);
  background: var(--chili);
  padding: 2px 9px;
  border-radius: 100px;
  border: 2px solid var(--edge);
}

.turn-indicator.mine {
  background: var(--gold);
}

.player-seat.offline .avatar {
  filter: grayscale(0.7);
}

.offline-tag {
  margin-left: 3px;
  font-size: 0.6rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted-cream);
}

@media (max-width: 640px) {
  .table-zone {
    min-height: 280px;
    padding: 8px;
    gap: 8px;
  }

  .seat-rail {
    gap: 6px 10px;
  }

  .table-surface {
    width: 100%;
    max-width: 300px;
    min-height: 150px;
    gap: 18px;
    padding: 12px;
  }

  .draw-card {
    width: 62px;
    height: 87px;
  }

  .draw-label {
    font-size: 0.72rem;
  }

  .discard-card,
  .discard-stack {
    width: 54px;
    height: 76px;
  }

  .player-seat {
    width: 72px;
  }

  .avatar {
    width: 40px;
    height: 40px;
    font-size: 0.82rem;
  }

  .card-count-badge {
    min-width: 20px;
    height: 20px;
    font-size: 0.62rem;
    border-width: 2px;
  }

  .player-name {
    font-size: 0.62rem;
    max-width: 72px;
  }

  .turn-indicator {
    font-size: 0.52rem;
    padding: 1px 7px;
  }
}

/* ─── Desktop ───
   Everything above was sized for a phone and then centred on bigger screens,
   which left the table and seats looking like a postage stamp. These tiers
   scale the whole cluster with the viewport so it fills the room it has. */

@media (min-width: 900px) {
  .table-zone {
    gap: 20px;
    padding: 20px 24px;
  }

  .seat-rail {
    gap: 14px 28px;
  }

  .table-surface {
    width: 420px;
    min-height: 240px;
    gap: 36px;
    padding: 20px;
  }

  .draw-card {
    width: 104px;
    height: 146px;
  }

  .draw-label {
    font-size: 1.05rem;
  }

  .discard-card,
  .discard-stack {
    width: 94px;
    height: 132px;
  }

  .player-seat {
    width: 120px;
  }

  .avatar {
    width: 60px;
    height: 60px;
    font-size: 1.5rem;
  }

  .player-name {
    font-size: 0.85rem;
    max-width: 120px;
  }
}

@media (min-width: 1280px) {
  .table-surface {
    width: 480px;
    min-height: 280px;
    gap: 44px;
  }

  .draw-card {
    width: 120px;
    height: 168px;
  }

  .discard-card,
  .discard-stack {
    width: 108px;
    height: 152px;
  }

  .avatar {
    width: 68px;
    height: 68px;
    font-size: 1.7rem;
  }
}
</style>
