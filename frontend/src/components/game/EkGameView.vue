<template>
  <div class="game">
    <div class="game-info">
      <div class="game-info-left">
        <span class="room-name-pill" v-if="store.roomName">{{ store.roomName }}</span>
        <span class="turn-pill" :class="{ myturn: store.isMyTurn, spectating: store.amISpectating }">
          {{ store.amISpectating ? 'Spectating' : (store.isMyTurn ? 'Your turn!' : (store.gameState?.turn || '') + "'s turn") }}
        </span>
        <span
          v-if="store.roomSettings.turnTimer > 0 && store.turnTimeLeft > 0"
          class="timer-pill"
          :class="{ urgent: store.turnTimeLeft <= 10 }"
          role="timer"
          aria-live="polite"
        >
          {{ store.turnTimeLeft }}s
        </span>
      </div>
      <div class="game-meta-actions">
        <button
          class="meta-btn"
          aria-label="Leave room"
          @click="$emit('leave')"
        >
          <LogOut :size="16" />
        </button>
        <button
          v-if="isHost"
          class="meta-btn meta-btn-danger"
          aria-label="Delete room"
          @click="$emit('delete')"
        >
          <Trash2 :size="16" />
        </button>
      </div>
    </div>

    <GameTable ref="gameTableRef" />

    <DefuseModal />
    <PositionPickerModal />
    <TargetSelectModal />
    <CardNameInputModal />
    <DiscardPickerModal />
    <GarbageCollectionModal />
    <FavorModal />
    <NopeOverlay />
    <PeekOverlay />

    <GameHand
      @cardClick="onCardClick"
      @drawCard="store.drawCard()"
      @playSelected="store.playSelected()"
      @playNope="onPlayNope"
      @leave="$emit('leave')"
    />

    <GameLog />
  </div>
</template>

<script setup>
import { useEkStore } from '../../stores/explodingKitchen'
import { LogOut, Trash2 } from '@lucide/vue'
import GameTable from './GameTable.vue'
import GameHand from './GameHand.vue'
import GameLog from './GameLog.vue'
import DefuseModal from './DefuseModal.vue'
import PositionPickerModal from './PositionPickerModal.vue'
import TargetSelectModal from './TargetSelectModal.vue'
import CardNameInputModal from './CardNameInputModal.vue'
import DiscardPickerModal from './DiscardPickerModal.vue'
import GarbageCollectionModal from './GarbageCollectionModal.vue'
import FavorModal from './FavorModal.vue'
import NopeOverlay from './NopeOverlay.vue'
import PeekOverlay from './PeekOverlay.vue'

const props = defineProps({
  isHost: { type: Boolean, default: false },
})

defineEmits(['leave', 'delete'])

const store = useEkStore()

function onCardClick(idx) {
  if (store.nopeWindow.active && !store.isMyTurn) {
    const cardId = store.myHand[idx]
    if (store.getCategoryName(cardId) === 'Nope') {
      store.selectCard(idx)
    }
    return
  }
  if (store.isMyTurn) {
    store.selectCard(idx)
  }
}

function onPlayNope() {
  store.clearSelection()
  store.playNope()
}
</script>

<style scoped>
.game {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
}

.game-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-wrap: wrap;
  gap: 10px;
}

.game-info-left {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.turn-pill {
  font-size: 0.78rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: rgba(238, 194, 92, 0.12);
  color: var(--gold);
  border: 1px solid rgba(238, 194, 92, 0.3);
}

.room-name-pill {
  font-size: 0.85rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: rgba(238, 194, 92, 0.12);
  color: var(--steam-cream);
  border: 1px solid rgba(238, 194, 92, 0.3);
  font-weight: 600;
}

.turn-pill.myturn {
  color: var(--chili-orange);
  border-color: rgba(226, 99, 44, 0.3);
  background: rgba(226, 99, 44, 0.12);
}

.turn-pill.spectating {
  color: var(--mild-cream);
  border-color: var(--line);
  background: rgba(255, 255, 255, 0.05);
}

.timer-pill {
  font-size: 0.78rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: rgba(127, 168, 118, 0.12);
  color: #7fa876;
  border: 1px solid rgba(127, 168, 118, 0.3);
  font-weight: 600;
}

.timer-pill.urgent {
  color: var(--broth-red);
  border-color: rgba(183, 41, 31, 0.3);
  background: rgba(183, 41, 31, 0.12);
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
}

/* ─── Meta Actions ─── */

.game-meta-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}

.meta-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  border: 1px solid var(--line);
  background: transparent;
  color: var(--mild-cream);
  cursor: pointer;
  transition: all var(--ease-standard);
}

.meta-btn:hover {
  border-color: var(--chili-orange);
  color: var(--steam-cream);
  background: rgba(226, 99, 44, 0.08);
}

.meta-btn:active {
  transform: scale(0.95);
}

.meta-btn:focus-visible {
  outline: 2px solid var(--chili-orange);
  outline-offset: 2px;
}

.meta-btn svg {
  width: 16px;
  height: 16px;
}

.meta-btn-danger {
  border-color: rgba(183, 41, 31, 0.3);
}

.meta-btn-danger:hover {
  border-color: var(--broth-red);
  color: var(--broth-red);
  background: rgba(183, 41, 31, 0.08);
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  .game {
    padding: 12px;
  }

  .game-info {
    gap: 8px;
  }
}
</style>
