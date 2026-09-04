<template>
  <div class="hand-zone">
    <!-- Spectating message -->
    <div v-if="store.amISpectating" class="spectating-banner">
      <span>Spectating — you don't have cards</span>
      <button class="leave-spectate-btn" @click="$emit('leave')">
        <LogOut :size="14" />
        Leave room
      </button>
    </div>

    <!-- Card hand with floating selection pill -->
    <div v-else class="hand-cards-wrap">
      <!-- Floating selection pill -->
      <Transition name="pill">
        <div v-if="store.selectedCardIndices.length > 0" class="selection-pill">
          <span class="pill-count">{{ store.selectedCardIndices.length }} selected</span>
          <span class="pill-sep"></span>
          <button class="pill-clear" @click="store.clearSelection()">Clear</button>
          <span v-if="store.selectedCombo" class="pill-combo">
            {{ store.selectedCombo.count }}x {{ store.selectedCombo.category }}
          </span>
          <button
            v-if="store.isMyTurn && !store.nopeWindow.active"
            class="pill-play"
            :disabled="store.selectedCardIndices.length === 0"
            @click="$emit('playSelected')"
          >
            <Play :size="12" />
            <span>Play{{ store.selectedCombo ? ` ${store.selectedCombo.count}x` : '' }}</span>
          </button>
        </div>
      </Transition>

      <div class="hand-cards">
        <div
          v-for="(cardId, i) in store.myHand"
          :key="i"
          class="hand-card"
          :class="{
            selected: store.selectedCardIndices.includes(i),
            disabled: !store.isMyTurn && !(store.nopeWindow.active && store.getCategoryName(cardId) === 'Nope'),
            [`cat-${getCategoryKey(cardId)}`]: true
          }"
          :style="{ '--card-index': i, '--total': store.myHand.length }"
          @click="$emit('cardClick', i)"
        >
          <div class="card-badge" :class="`badge-${getCategoryKey(cardId)}`">
            {{ store.getCategoryName(cardId) }}
          </div>
          <div class="card-icon-area">
            <img
              v-if="store.getCardData(cardId)"
              :src="`/cards/${store.getCardData(cardId).file}`"
              :alt="store.getCardData(cardId)?.name"
              class="card-image"
              loading="lazy"
            >
          </div>
          <div class="card-title">{{ getCardDisplayName(cardId) }}</div>
        </div>
      </div>
    </div>

    <!-- Nope hint -->
    <div v-if="store.nopeWindow.active && !store.isMyTurn && store.canINope" class="nope-hint" aria-live="polite">
      Select a Nope card to play!
    </div>

    <!-- Action buttons (only nope and leave remain) -->
    <div v-if="!store.amISpectating && (store.nopeWindow.active || false)" class="hand-actions">
      <button
        v-if="store.nopeWindow.active && !store.isMyTurn && store.canINope"
        class="action-btn nope-btn"
        :disabled="!hasNopeSelected"
        @click="$emit('playNope')"
      >
        <ShieldOff class="btn-icon" :size="16" />
        <span>Nope!</span>
        <span class="nope-countdown">{{ store.nopeTimeLeft }}s</span>
      </button>
    </div>

    <!-- Spectating leave -->
    <div v-if="store.amISpectating" class="hand-actions">
      <button class="action-btn leave-btn" @click="$emit('leave')">
        <LogOut class="btn-icon" :size="16" />
        <span>Leave room</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Play, ShieldOff, LogOut } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()

defineEmits(['cardClick', 'playSelected', 'playNope', 'leave'])

const hasNopeSelected = computed(() => {
  if (store.selectedCardIndices.length === 0) return false
  const hand = store.myHand
  return store.selectedCardIndices.some(i => {
    const cardId = hand[i]
    return store.getCardData(cardId) && store.getCategoryName(cardId) === 'Nope'
  })
})

function getCategoryKey(cardId) {
  const name = store.getCategoryName(cardId)
  const map = {
    'Attack': 'attack',
    'Defense': 'defense',
    'Skip': 'skip',
    'Super Skip': 'super-skip',
    'Reverse': 'reverse',
    'Future Vision': 'future',
    'Nope': 'nope',
    'Shuffle': 'shuffle',
    'Draw From Bottom': 'draw-bottom',
    'Swap Top & Bottom': 'swap',
    'Garbage Collection': 'garbage',
    'Catomic Bomb': 'catomic',
    'Mark': 'mark',
    'Bury': 'bury',
    'Dig Deeper': 'dig',
    'Favor': 'favor',
    'Clone': 'clone',
    'Explosive': 'explosive',
    'Group Effects': 'group',
    'Special Power': 'special',
    'Cat Cards': 'cat',
  }
  return map[name] || 'action'
}

function getCardDisplayName(cardId) {
  const data = store.getCardData(cardId)
  if (!data) return 'Unknown'
  const catName = store.getCategoryName(cardId)
  if (catName === 'Cat Cards') return data.name
  return catName
}
</script>

<style scoped>
.hand-zone {
  position: relative;
  padding: 16px 20px 12px;
  background: rgba(28, 21, 18, 0.8);
  backdrop-filter: blur(8px);
  border-top: 1px solid var(--line);
  z-index: 60;
  flex-shrink: 0;
}

/* ─── Spectating ─── */

.spectating-banner {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 12px;
  color: var(--muted-cream);
  font-size: 0.85rem;
}

.leave-spectate-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 8px;
  border: 1px solid var(--line);
  background: transparent;
  color: var(--mild-cream);
  font-size: 0.8rem;
  font-family: inherit;
  cursor: pointer;
  transition: all var(--ease-standard);
}

.leave-spectate-btn:hover {
  border-color: var(--chili-orange);
  color: var(--steam-cream);
}

/* ─── Card Hand Wrap ─── */

.hand-cards-wrap {
  position: relative;
  padding-top: 8px;
}

/* ─── Floating Selection Pill ─── */

.selection-pill {
  position: absolute;
  top: -6px;
  left: 50%;
  transform: translateX(-50%);
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 14px;
  border-radius: 100px;
  background: rgba(28, 21, 18, 0.92);
  border: 1px solid var(--line);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
  z-index: 30;
  white-space: nowrap;
  font-size: 0.75rem;
}

.pill-count {
  color: var(--mild-cream);
  font-weight: 500;
}

.pill-sep {
  width: 1px;
  height: 14px;
  background: var(--line);
}

.pill-clear {
  background: none;
  border: none;
  color: var(--muted-cream);
  font-size: 0.72rem;
  font-family: inherit;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  transition: all var(--ease-standard);
}

.pill-clear:hover {
  color: var(--broth-red);
  background: rgba(183, 41, 31, 0.12);
}

.pill-combo {
  font-size: 0.7rem;
  font-weight: 600;
  color: var(--gold);
  background: rgba(238, 194, 92, 0.12);
  padding: 2px 8px;
  border-radius: 100px;
  border: 1px solid rgba(238, 194, 92, 0.25);
}

.pill-play {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 12px;
  border-radius: 100px;
  border: none;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
  font-size: 0.72rem;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  transition: all var(--ease-standard);
  margin-left: 2px;
}

.pill-play:hover:not(:disabled) {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

.pill-play:disabled {
  opacity: 0.4;
  cursor: default;
}

.pill-play:active:not(:disabled) {
  transform: scale(0.96);
}

/* Pill transition */
.pill-enter-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.pill-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}
.pill-enter-from {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}
.pill-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(4px);
}

/* ─── Card Hand ─── */

.hand-cards {
  display: flex;
  justify-content: center;
  gap: 0;
  padding: 8px 0 4px;
  min-height: 140px;
  perspective: 800px;
}

.hand-card {
  width: 90px;
  height: 130px;
  background: #faf5ed;
  border-radius: 10px;
  border: 2px solid rgba(0, 0, 0, 0.08);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0;
  cursor: pointer;
  flex-shrink: 0;
  margin-left: -40px;
  position: relative;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
  transform-origin: bottom center;
  transform: rotate(calc((var(--card-index) - var(--total) / 2) * 4deg));
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.hand-card:first-child {
  margin-left: 0;
}

.hand-card:hover {
  transform: rotate(0deg) translateY(-20px) scale(1.08);
  z-index: 20;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
}

.hand-card.disabled {
  filter: grayscale(0.7) brightness(0.85);
  cursor: default;
  pointer-events: none;
}

.hand-card.selected {
  transform: rotate(0deg) translateY(-28px) scale(1.1);
  z-index: 25;
  box-shadow: 0 0 0 2px var(--gold), 0 8px 24px rgba(238, 194, 92, 0.3);
}

/* ─── Category Badge ─── */

.card-badge {
  width: 100%;
  text-align: center;
  font-size: 0.55rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 4px 0;
  color: white;
}

.badge-attack {
  background: linear-gradient(135deg, #ef4444, #dc2626);
}

.badge-defense {
  background: linear-gradient(135deg, #14b8a6, #0d9488);
}

.badge-skip {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
}

.badge-super-skip {
  background: linear-gradient(135deg, #6366f1, #4f46e5);
}

.badge-reverse {
  background: linear-gradient(135deg, #06b6d4, #0891b2);
}

.badge-future {
  background: linear-gradient(135deg, #8b5cf6, #7c3aed);
}

.badge-nope {
  background: linear-gradient(135deg, #6b7280, #4b5563);
}

.badge-shuffle {
  background: linear-gradient(135deg, #a855f7, #9333ea);
}

.badge-draw-bottom {
  background: linear-gradient(135deg, #f59e0b, #d97706);
}

.badge-swap {
  background: linear-gradient(135deg, #eab308, #ca8a04);
}

.badge-garbage {
  background: linear-gradient(135deg, #f97316, #ea580c);
}

.badge-catomic {
  background: linear-gradient(135deg, #f43f5e, #e11d48);
}

.badge-mark {
  background: linear-gradient(135deg, #ec4899, #db2777);
}

.badge-bury {
  background: linear-gradient(135deg, #78716c, #57534e);
}

.badge-dig {
  background: linear-gradient(135deg, #84cc16, #65a30d);
}

.badge-favor {
  background: linear-gradient(135deg, #10b981, #059669);
}

.badge-clone {
  background: linear-gradient(135deg, #0ea5e9, #0284c7);
}

.badge-explosive {
  background: linear-gradient(135deg, #b91c1c, #991b1b);
}

.badge-group {
  background: linear-gradient(135deg, #d946ef, #c026d3);
}

.badge-special {
  background: linear-gradient(135deg, #e879f9, #d946ef);
}

.badge-cat {
  background: linear-gradient(135deg, #a16207, #854d0e);
}

/* ─── Card Icon ─── */

.card-icon-area {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
}

.card-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

/* ─── Card Title ─── */

.card-title {
  width: 100%;
  text-align: center;
  font-size: 0.62rem;
  font-weight: 600;
  color: #3d2e1f;
  padding: 6px 4px 8px;
  line-height: 1.2;
}

/* ─── Nope Hint ─── */

.nope-hint {
  text-align: center;
  font-size: 0.78rem;
  color: var(--gold);
  font-weight: 600;
  padding: 4px 0;
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

/* ─── Action Buttons ─── */

.hand-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
  padding-top: 8px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 22px;
  border-radius: 100px;
  font-weight: 600;
  font-size: 0.85rem;
  border: none;
  cursor: pointer;
  font-family: inherit;
  transition: all var(--ease-standard);
  white-space: nowrap;
}

.action-btn:active:not(:disabled) {
  transform: scale(0.97);
}

.action-btn:disabled {
  opacity: 0.35;
  cursor: default;
}

.action-btn:focus-visible {
  outline: 2px solid var(--chili-orange);
  outline-offset: 2px;
}

.btn-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

/* Nope Button */
.nope-btn {
  background: rgba(183, 41, 31, 0.3);
  color: var(--broth-red);
  border: 1px solid rgba(183, 41, 31, 0.5);
}

.nope-btn:hover:not(:disabled) {
  background: rgba(183, 41, 31, 0.5);
}

.nope-countdown {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 26px;
  height: 18px;
  padding: 0 5px;
  border-radius: 100px;
  background: var(--broth-red);
  color: var(--steam-cream);
  font-size: 0.65rem;
  font-weight: 700;
  line-height: 1;
}

/* Leave Button */
.leave-btn {
  background: transparent;
  border: 1px solid var(--line);
  color: var(--mild-cream);
}

.leave-btn:hover {
  border-color: var(--chili-orange);
  color: var(--steam-cream);
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  .hand-zone {
    padding: 10px 8px 6px;
  }

  .selection-pill {
    font-size: 0.65rem;
    padding: 3px 8px;
    gap: 5px;
  }

  .pill-play {
    font-size: 0.6rem;
    padding: 2px 8px;
  }

  .hand-cards {
    justify-content: center;
    padding: 6px 0 2px;
    min-height: 90px;
  }

  .hand-card {
    width: 54px;
    height: 80px;
    margin-left: -35px;
  }

  .hand-card:first-child {
    margin-left: 0;
  }

  .hand-card:hover {
    transform: rotate(0deg) translateY(-12px) scale(1.05);
  }

  .hand-card.selected {
    transform: rotate(0deg) translateY(-16px) scale(1.08);
  }

  .card-badge {
    font-size: 0.42rem;
    padding: 2px 0;
  }

  .card-title {
    font-size: 0.48rem;
    padding: 3px 2px 4px;
  }

  .hand-actions {
    padding-top: 4px;
  }

  .action-btn {
    padding: 8px 14px;
    font-size: 0.75rem;
  }
}
</style>
