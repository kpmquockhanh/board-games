<template>
  <div class="table-zone" ref="tableZone">
    <div class="table-surface">
      <div
        class="draw-pile"
        :class="{ 'pile-shake': pileAnimating }"
        role="button"
        tabindex="0"
        @click="$emit('drawCard')"
        @keydown.enter="$emit('drawCard')"
        @keydown.space.prevent="$emit('drawCard')"
      >
        <div class="draw-card">
          <div class="draw-card-inner">
            <span class="draw-icon">🔥</span>
            <span class="draw-label">DRAW</span>
          </div>
        </div>
        <div class="deck-count">
          <span class="deck-count-num">{{ store.drawPileCount }}</span>
          <span class="deck-count-label">left</span>
        </div>
      </div>

      <div v-if="store.recentlyPlayed.length > 0" class="discard-zone">
        <div class="discard-label">DISCARD</div>
        <div class="discard-stack">
          <div
            v-for="(card, i) in store.recentlyPlayed.slice(0, 3).reverse()"
            :key="i"
            class="discard-card"
            :style="{ '--i': i }"
          >
            <img :src="`/cards/${card.file}`" :alt="card.name" loading="lazy">
          </div>
        </div>
      </div>

      <div v-if="store.attackStackCount > 0" class="attack-indicator" role="status" aria-live="polite">
        <Zap class="attack-icon" :size="14" />
        <span class="attack-count">{{ store.attackStackCount }} forced draw{{ store.attackStackCount > 1 ? 's' : '' }}</span>
      </div>
    </div>

    <div
      v-for="name in store.turnOrder"
      :key="name"
      class="player-seat"
      :class="{
        'active-turn': store.gameState?.turn === name,
        eliminated: !store.gameState?.players[name]?.alive,
        'is-me': store.me && name === store.me.name
      }"
      :style="seatStyle(name)"
    >
      <div class="avatar-wrapper">
        <div
          class="avatar"
          :style="{ background: store.gameState?.players[name]?.color }"
          :aria-label="`${name}${!store.gameState?.players[name]?.alive ? ' (eliminated)' : ''}`"
        >{{ name.trim().slice(0, 2).toUpperCase() }}</div>
        <div class="card-count-badge">
          {{ store.gameState?.players[name]?.hand?.length || 0 }}
        </div>
      </div>
      <div class="player-name">
        {{ name }}{{ store.me && name === store.me.name ? ' (you)' : '' }}
      </div>
      <div v-if="store.gameState?.turn === name" class="turn-indicator">Turn</div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Zap } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const tableZone = ref(null)
const pileAnimating = ref(false)

let lastDrawCount = 0
watch(
  () => store.drawPileCount,
  (newCount, oldCount) => {
    if (oldCount && newCount < oldCount) {
      pileAnimating.value = true
      setTimeout(() => { pileAnimating.value = false }, 400)
    }
    lastDrawCount = newCount
  }
)

function seatStyle(name) {
  if (!tableZone.value) return {}
  const zone = tableZone.value
  const names = store.turnOrder
  const idx = names.indexOf(name)
  const cx = zone.clientWidth / 2
  const cy = zone.clientHeight / 2
  const rx = zone.clientWidth * 0.38
  const ry = zone.clientHeight * 0.38
  const angle = (idx / names.length) * Math.PI * 2 - Math.PI / 2
  return {
    left: (cx + Math.cos(angle) * rx) + 'px',
    top: (cy + Math.sin(angle) * ry) + 'px',
  }
}

defineExpose({ tableZone })
defineEmits(['drawCard'])
</script>

<style scoped>
.table-zone {
  position: relative;
  flex: 1;
  min-height: 340px;
  overflow: visible;
}

.table-surface {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 320px;
  height: 200px;
  background: linear-gradient(145deg, #5c3d2e, #3e2518 60%, #2e1a10);
  border-radius: 24px;
  border: 2px solid rgba(139, 90, 43, 0.4);
  box-shadow:
    0 8px 32px rgba(0, 0, 0, 0.5),
    inset 0 1px 0 rgba(255, 255, 255, 0.06),
    inset 0 -2px 8px rgba(0, 0, 0, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 40px;
}

.draw-pile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: transform 0.1s ease;
}

.draw-pile:hover {
  transform: scale(1.04);
}

.draw-pile:hover .draw-card {
  box-shadow: 0 4px 24px rgba(226, 99, 44, 0.5);
  border-color: rgba(255, 255, 255, 0.3);
}

.draw-pile:focus-visible {
  outline: 2px solid var(--chili-orange);
  outline-offset: 4px;
  border-radius: 10px;
}

.draw-pile.pile-shake {
  animation: pileShake 0.4s ease;
}

@keyframes pileShake {
  0%, 100% { transform: translateX(0); }
  20% { transform: translateX(-3px) rotate(-1deg); }
  40% { transform: translateX(3px) rotate(1deg); }
  60% { transform: translateX(-2px) rotate(-0.5deg); }
  80% { transform: translateX(2px) rotate(0.5deg); }
}

.draw-card {
  width: 80px;
  height: 112px;
  border-radius: 10px;
  background: linear-gradient(145deg, #e2632c, #c44d1c);
  border: 2px solid rgba(255, 255, 255, 0.15);
  box-shadow: 0 4px 16px rgba(226, 99, 44, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.draw-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: repeating-conic-gradient(rgba(255,255,255,0.04) 0% 25%, transparent 0% 50%) 50%/14px 14px;
  border-radius: 8px;
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
  font-size: 1.8rem;
  filter: drop-shadow(0 2px 4px rgba(0,0,0,0.3));
}

.draw-label {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 0.9rem;
  color: white;
  letter-spacing: 0.1em;
  text-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}

.deck-count {
  display: flex;
  align-items: baseline;
  gap: 4px;
  font-family: 'Baloo 2', sans-serif;
}

.deck-count-num {
  font-size: 1rem;
  font-weight: 700;
  color: var(--steam-cream);
}

.deck-count-label {
  font-size: 0.7rem;
  color: var(--muted-cream);
}

.discard-zone {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.discard-label {
  font-size: 0.6rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--muted-cream);
  font-weight: 600;
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
  border-radius: 8px;
  border: 2px solid var(--line);
  background: var(--charcoal);
  overflow: hidden;
  transition: transform var(--duration-normal) ease;
  transform: rotate(calc(var(--i) * -4deg)) translateY(calc(var(--i) * 6px));
}

.discard-card:first-child {
  border-color: rgba(253, 243, 228, 0.25);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
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
  position: absolute;
  bottom: -36px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: linear-gradient(135deg, rgba(183, 41, 31, 0.3), rgba(226, 99, 44, 0.15));
  color: var(--broth-red);
  border: 1px solid rgba(183, 41, 31, 0.45);
  font-weight: 600;
  white-space: nowrap;
  animation: attack-pulse 2s ease-in-out infinite;
}

.attack-icon {
  flex-shrink: 0;
}

@keyframes attack-pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.8; }
}

.player-seat {
  position: absolute;
  text-align: center;
  transform: translate(-50%, -50%);
  transition: all 0.3s ease;
  z-index: 10;
}

.avatar-wrapper {
  position: relative;
  display: inline-block;
  margin-bottom: 4px;
}

.avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 700;
  font-size: 1rem;
  color: #1c1512;
  box-shadow: 0 4px 14px -4px rgba(0, 0, 0, 0.6);
  border: 2.5px solid rgba(253, 243, 228, 0.35);
  transition: box-shadow 0.2s ease;
}

.player-seat.is-me .avatar {
  border-color: var(--gold);
  box-shadow: 0 0 0 2px var(--gold), 0 4px 14px -4px rgba(0, 0, 0, 0.6);
}

.player-seat.eliminated .avatar {
  opacity: 0.3;
  filter: grayscale(1);
}

.card-count-badge {
  position: absolute;
  bottom: -4px;
  right: -4px;
  min-width: 22px;
  height: 22px;
  padding: 0 5px;
  border-radius: 100px;
  background: var(--charcoal);
  border: 2px solid var(--line);
  color: var(--steam-cream);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.7rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

.player-seat.active-turn .avatar {
  border-color: var(--chili-orange);
  box-shadow: 0 0 0 2px var(--chili-orange), 0 0 20px rgba(226, 99, 44, 0.4);
}

.player-name {
  font-size: 0.72rem;
  color: var(--mild-cream);
  opacity: 0.9;
  white-space: nowrap;
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.turn-indicator {
  display: inline-block;
  margin-top: 2px;
  font-size: 0.6rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--chili-orange);
  background: rgba(226, 99, 44, 0.15);
  padding: 2px 8px;
  border-radius: 100px;
  border: 1px solid rgba(226, 99, 44, 0.3);
}

@media (max-width: 640px) {
  .table-zone {
    min-height: 280px;
  }

  .table-surface {
    width: 240px;
    height: 160px;
    border-radius: 18px;
    gap: 24px;
  }

  .draw-card {
    width: 60px;
    height: 84px;
  }

  .draw-icon {
    font-size: 1.4rem;
  }

  .draw-label {
    font-size: 0.7rem;
  }

  .discard-card {
    width: 52px;
    height: 73px;
  }

  .discard-stack {
    width: 52px;
    height: 73px;
  }

  .avatar {
    width: 38px;
    height: 38px;
    font-size: 0.8rem;
  }

  .card-count-badge {
    min-width: 18px;
    height: 18px;
    font-size: 0.6rem;
    border-width: 1.5px;
  }

  .player-name {
    font-size: 0.6rem;
  }

  .turn-indicator {
    font-size: 0.5rem;
    padding: 1px 6px;
  }
}
</style>
