<template>
  <div class="table-zone" ref="tableZone">
    <div class="draw-pile" :class="{ 'pile-shake': pileAnimating }">
      <div class="pile-cards">
        <div class="pile-card back-1"></div>
        <div class="pile-card back-2"></div>
        <div class="pile-card back-3"></div>
      </div>
      <div class="draw-count">{{ store.drawPileCount }}</div>
    </div>

    <div v-if="store.recentlyPlayed.length > 0" class="discard-zone">
      <div class="discard-stack">
        <div
          v-for="(card, i) in store.recentlyPlayed.slice(0, 5).reverse()"
          :key="i"
          class="discard-card"
          :style="{ '--i': i }"
        >
          <img :src="`/cards/${card.file}`" :alt="card.name" loading="lazy">
        </div>
      </div>
    </div>

    <div v-if="store.attackStackCount > 0" class="attack-indicator" role="status" aria-live="polite">
      <Zap class="attack-icon" :size="16" />
      <span class="attack-label">Forced draws</span>
      <span class="attack-count">{{ store.attackStackCount }}</span>
    </div>

    <div
      v-for="name in store.turnOrder"
      :key="name"
      class="player-seat"
      :class="{ 'active-turn': store.gameState?.turn === name }"
      :style="seatStyle(name)"
    >
      <div
        class="avatar"
        :class="{
          me: store.me && name === store.me.name,
          eliminated: !store.gameState?.players[name]?.alive
        }"
        :style="{ background: store.gameState?.players[name]?.color }"
        :aria-label="`${name}${!store.gameState?.players[name]?.alive ? ' (eliminated)' : ''}`"
      >{{ name.trim().slice(0, 2).toUpperCase() }}</div>
      <div class="pname">{{ name }}{{ store.me && name === store.me.name ? ' (you)' : '' }}</div>
      <div
        class="pstatus"
        :style="{ color: store.gameState?.players[name]?.alive ? 'var(--mild-cream)' : 'var(--broth-red)' }"
      >
        {{ store.gameState?.players[name]?.alive
          ? ((store.gameState?.players[name]?.hand?.length || 0) + ' cards')
          : 'eliminated'
        }}
      </div>
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
  const rx = zone.clientWidth * 0.4
  const ry = zone.clientHeight * 0.36
  const angle = (idx / names.length) * Math.PI * 2 - Math.PI / 2
  return {
    left: (cx + Math.cos(angle) * rx) + 'px',
    top: (cy + Math.sin(angle) * ry) + 'px',
  }
}

defineExpose({ tableZone })
</script>

<style scoped>
.table-zone {
  position: relative;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 24px;
  height: 360px;
  margin-bottom: 16px;
  overflow: hidden;
}

.draw-pile {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 60%;
  max-width: 280px;
  aspect-ratio: 1;
  border-radius: 50%;
  background: radial-gradient(circle at 35% 30%, #3a2a1e, #1c1512 75%);
  border: 1px solid var(--line);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 4px;
  transition: transform 0.1s ease;
}

.draw-pile.pile-shake {
  animation: pileShake 0.4s ease;
}

@keyframes pileShake {
  0%, 100% { transform: translate(-50%, -50%) rotate(0deg); }
  20% { transform: translate(-50%, -50%) rotate(-2deg); }
  40% { transform: translate(-50%, -50%) rotate(2deg); }
  60% { transform: translate(-50%, -50%) rotate(-1deg); }
  80% { transform: translate(-50%, -50%) rotate(1deg); }
}

.pile-cards {
  position: absolute;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.pile-card {
  position: absolute;
  width: 50px;
  height: 70px;
  border-radius: 6px;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  background: repeating-conic-gradient(#2a1d16 0% 25%, #1c1512 0% 50%) 50%/12px 12px;
  border: 1px solid var(--line);
}

.pile-card.back-1 {
  transform: translate(-50%, -50%) rotate(-3deg) translateY(-2px);
}

.pile-card.back-2 {
  transform: translate(-50%, -50%) rotate(0deg) translateY(-4px);
}

.pile-card.back-3 {
  transform: translate(-50%, -50%) rotate(3deg) translateY(-6px);
}

.draw-count {
  font-family: 'Baloo 2', sans-serif;
  font-size: 1.6rem;
  font-weight: 800;
  color: var(--gold);
  position: relative;
  z-index: 1;
}

.discard-zone {
  position: absolute;
  right: 100px;
  top: 50%;
  transform: translateY(-50%);
}

.discard-stack {
  position: relative;
  width: 80px;
  height: 112px;
}

.discard-card {
  position: absolute;
  width: 80px;
  height: 112px;
  border-radius: 10px;
  border: 2px solid var(--line);
  background: var(--charcoal);
  overflow: hidden;
  transition: transform var(--duration-normal) ease;
  transform: rotate(calc(var(--i) * -5deg)) translateY(calc(var(--i) * 8px));
}

.discard-card:first-child {
  border-color: var(--gold);
  box-shadow: 0 0 10px rgba(238, 194, 92, 0.2);
}

.discard-card img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.discard-card:hover {
  transform: rotate(0deg) translateY(-8px) scale(1.05);
  z-index: var(--z-cards);
}

.player-seat {
  position: absolute;
  text-align: center;
  transform: translate(-50%, -50%);
  transition: all 0.3s ease;
}

.player-seat .avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  margin: 0 auto 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 700;
  font-size: 1rem;
  color: #1c1512;
  box-shadow: 0 4px 14px -4px rgba(0, 0, 0, 0.6);
  border: 2px solid rgba(253, 243, 228, 0.3);
  transition: box-shadow 0.2s ease;
}

.player-seat .avatar.me {
  box-shadow: 0 0 0 3px var(--gold), 0 4px 14px -4px rgba(0, 0, 0, 0.6);
}

.player-seat .avatar.eliminated {
  opacity: 0.3;
  filter: grayscale(1);
}

.player-seat .pname {
  font-size: 0.7rem;
  color: var(--mild-cream);
  opacity: 0.85;
}

.player-seat .pstatus {
  font-size: 0.65rem;
  margin-top: 2px;
}

.player-seat.active-turn .avatar {
  box-shadow: 0 0 0 3px var(--chili-orange), 0 0 20px rgba(226, 99, 44, 0.4);
}

.attack-indicator {
  position: absolute;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  padding: 7px 14px 7px 10px;
  border-radius: 100px;
  background: linear-gradient(135deg, rgba(183, 41, 31, 0.25), rgba(226, 99, 44, 0.15));
  color: var(--broth-red);
  border: 1px solid rgba(183, 41, 31, 0.45);
  font-weight: 600;
  z-index: var(--z-table-internal);
  box-shadow: 0 0 14px rgba(183, 41, 31, 0.35), inset 0 0 10px rgba(183, 41, 31, 0.1);
  animation: attack-pulse 2s ease-in-out infinite;
}

.attack-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  filter: drop-shadow(0 0 4px rgba(183, 41, 31, 0.5));
}

.attack-label {
  white-space: nowrap;
}

.attack-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  border-radius: 100px;
  background: var(--broth-red);
  color: var(--steam-cream);
  font-size: 0.72rem;
  font-weight: 700;
  line-height: 1;
  box-shadow: 0 0 8px rgba(183, 41, 31, 0.6);
  animation: count-pop 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes attack-pulse {
  0%, 100% {
    opacity: 1;
    box-shadow: 0 0 14px rgba(183, 41, 31, 0.35), inset 0 0 10px rgba(183, 41, 31, 0.1);
  }
  50% {
    opacity: 0.85;
    box-shadow: 0 0 20px rgba(183, 41, 31, 0.5), inset 0 0 12px rgba(183, 41, 31, 0.15);
  }
}

@keyframes count-pop {
  0% { transform: scale(0.6); }
  100% { transform: scale(1); }
}

@media (max-width: 640px) {
  .table-zone {
    height: 50vw;
    min-height: 280px;
    border-radius: 16px;
    margin-bottom: 12px;
  }

  .draw-pile {
    width: 50%;
  }

  .pile-card {
    width: 40px;
    height: 56px;
  }

  .draw-count {
    font-size: 1.2rem;
  }

  .discard-zone {
    right: 20px;
  }

  .discard-card {
    width: 56px;
    height: 78px;
  }

  .discard-stack {
    width: 56px;
    height: 78px;
  }

  .player-seat .avatar {
    width: 36px;
    height: 36px;
    font-size: 0.8rem;
  }

  .player-seat .pname {
    font-size: 0.6rem;
  }

  .player-seat .pstatus {
    font-size: 0.55rem;
  }

  .attack-indicator {
    padding: 5px 10px 5px 8px;
    font-size: 0.75rem;
  }
}
</style>
