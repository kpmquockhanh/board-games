<template>
  <div class="hand-zone">
    <div class="hand-label">
      {{ store.amISpectating ? 'Your cards (spectating)' : 'Your hand' }}
      <span v-if="store.nopeWindow.active && !store.isMyTurn && store.canINope" class="nope-hint" aria-live="polite">
        Select a Nope card to play!
      </span>
    </div>
    <div class="hand-cards">
      <template v-if="store.amISpectating">
        <div class="empty-hand">Spectators don't have cards</div>
      </template>
      <template v-else>
        <CardView
          v-for="(cardId, i) in store.myHand"
          :key="i"
          :card-data="store.getCardData(cardId)"
          :category-name="store.getCategoryName(cardId)"
          :selected="store.selectedCardIndices.includes(i)"
          :disabled="!store.isMyTurn && !(store.nopeWindow.active && store.getCardData(cardId)?.id && store.getCategoryName(cardId) === 'Nope')"
          @click="$emit('cardClick', i)"
        />
        <div v-if="store.myHand.length === 0" class="empty-hand">No cards in hand</div>
      </template>
    </div>
    <div v-if="store.selectedCardIndices.length > 0" class="selection-info">
      <span class="selection-count">{{ store.selectedCardIndices.length }} card(s) selected</span>
      <button class="clear-btn" @click="store.clearSelection()">Clear</button>
      <span v-if="store.selectedCombo" class="combo-badge">
        {{ store.selectedCombo.count }}x {{ store.selectedCombo.category }} combo!
      </span>
    </div>

    <!-- PRIMARY GAME ACTIONS -->
    <div v-if="!store.amISpectating" class="hand-actions">
      <button
        class="hand-action-btn draw-btn"
        :class="{ 'is-my-turn': store.isMyTurn && !store.nopeWindow.active }"
        :disabled="!store.isMyTurn || store.nopeWindow.active"
        @click="$emit('drawCard')"
      >
        <Layers class="btn-icon" :size="18" />
        <span>Draw a card</span>
      </button>

      <button
        v-if="store.isMyTurn && !store.nopeWindow.active"
        class="hand-action-btn play-btn"
        :disabled="store.selectedCardIndices.length === 0"
        @click="$emit('playSelected')"
      >
        <Play class="btn-icon" :size="18" />
        <span>{{ store.selectedCombo ? `Play ${store.selectedCombo.count}x Combo` : 'Play selected' }}</span>
      </button>

      <button
        v-if="store.nopeWindow.active && !store.isMyTurn && store.canINope"
        class="hand-action-btn nope-btn"
        :disabled="!hasNopeSelected"
        @click="$emit('playNope')"
      >
        <ShieldOff class="btn-icon" :size="18" />
        <span>Nope!</span>
        <span class="nope-countdown">{{ store.nopeTimeLeft }}s</span>
      </button>
    </div>

    <!-- SPECTATING ACTIONS -->
    <div v-else class="hand-actions spectating-actions">
      <button class="hand-action-btn leave-btn" @click="$emit('leave')">
        <LogOut class="btn-icon" :size="18" />
        <span>Leave room</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Layers, Play, ShieldOff, LogOut } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import CardView from '../CardView.vue'

const store = useEkStore()

defineEmits(['cardClick', 'drawCard', 'playSelected', 'playNope', 'leave'])

const hasNopeSelected = computed(() => {
  if (store.selectedCardIndices.length === 0) return false
  const hand = store.myHand
  return store.selectedCardIndices.some(i => {
    const cardId = hand[i]
    return store.getCardData(cardId) && store.getCategoryName(cardId) === 'Nope'
  })
})
</script>

<style scoped>
.hand-zone {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 18px;
  padding: 16px 20px;
  margin-bottom: 16px;
}

.hand-label {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--gold);
  margin-bottom: 10px;
}

.hand-cards {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  min-height: 132px;
  align-items: flex-end;
}

.empty-hand {
  opacity: 0.5;
  font-size: 0.85rem;
}

.nope-hint {
  display: inline-block;
  margin-left: 8px;
  font-size: 0.75rem;
  color: var(--gold);
  font-weight: 600;
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

/* ─── Selection Info ─── */

.selection-info {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--line);
}

.selection-count {
  font-size: 0.75rem;
  color: var(--muted-cream);
}

.clear-btn {
  font-size: 0.7rem;
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid var(--line);
  background: transparent;
  color: var(--mild-cream);
  cursor: pointer;
  font-family: inherit;
  transition: all var(--ease-standard);
}

.clear-btn:hover {
  border-color: var(--broth-red);
  color: var(--broth-red);
}

.combo-badge {
  font-size: 0.75rem;
  padding: 4px 10px;
  border-radius: 6px;
  background: rgba(238, 194, 92, 0.15);
  color: var(--gold);
  border: 1px solid rgba(238, 194, 92, 0.3);
  font-weight: 600;
}

/* ─── Game Actions ─── */

.hand-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--line);
}

.hand-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 12px 24px;
  border-radius: 100px;
  font-weight: 600;
  font-size: 0.9rem;
  border: none;
  cursor: pointer;
  font-family: inherit;
  transition: all var(--ease-standard);
  white-space: nowrap;
}

.hand-action-btn:active:not(:disabled) {
  transform: scale(0.97);
}

.hand-action-btn:disabled {
  opacity: 0.35;
  cursor: default;
}

.hand-action-btn:focus-visible {
  outline: 2px solid var(--chili-orange);
  outline-offset: 2px;
}

.btn-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

/* ─── Draw Button ─── */

.draw-btn {
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
}

.draw-btn:hover:not(:disabled) {
  filter: brightness(1.1);
  transform: translateY(-1px);
}

.draw-btn.is-my-turn {
  animation: draw-pulse 2s ease-in-out infinite;
}

@keyframes draw-pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(226, 99, 44, 0); }
  50% { box-shadow: 0 0 0 6px rgba(226, 99, 44, 0.2); }
}

/* ─── Play Button ─── */

.play-btn {
  background: var(--charcoal);
  border: 1px solid var(--line);
  color: var(--steam-cream);
}

.play-btn:hover:not(:disabled) {
  border-color: var(--chili-orange);
  transform: translateY(-1px);
}

/* ─── Nope Button ─── */

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
  min-width: 28px;
  height: 20px;
  padding: 0 6px;
  border-radius: 100px;
  background: var(--broth-red);
  color: var(--steam-cream);
  font-size: 0.7rem;
  font-weight: 700;
  line-height: 1;
}

/* ─── Leave Button (Spectating) ─── */

.leave-btn {
  background: transparent;
  border: 1px solid var(--line);
  color: var(--mild-cream);
}

.leave-btn:hover {
  border-color: var(--chili-orange);
  color: var(--steam-cream);
}

.spectating-actions {
  justify-content: center;
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  .hand-zone {
    padding: 12px;
    border-radius: 14px;
    margin-bottom: 12px;
  }

  .hand-cards {
    flex-wrap: nowrap;
    overflow-x: auto;
    scroll-snap-type: x mandatory;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 4px;
    min-height: 100px;
  }

  .hand-cards > :deep(.card-view) {
    scroll-snap-align: center;
  }

  .selection-info {
    flex-wrap: wrap;
    gap: 6px;
  }

  .hand-actions {
    flex-direction: column;
    gap: 8px;
    margin-top: 12px;
    padding-top: 12px;
  }

  .hand-action-btn {
    width: 100%;
    justify-content: center;
    padding: 11px 20px;
    font-size: 0.85rem;
  }
}
</style>
