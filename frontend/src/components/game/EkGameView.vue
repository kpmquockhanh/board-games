<template>
  <div class="game">
    <header class="game-header">
      <div class="header-left">
        <span class="game-title">🔥 Exploding Kitchen 2</span>
      </div>
      <div class="header-right">
        <button class="header-btn settings-btn" aria-label="Settings">
          <Settings :size="18" />
        </button>
        <button class="header-btn leave-btn" @click="$emit('leave')">
          <LogOut :size="16" class="leave-icon" />
          <span class="leave-text">Leave Game</span>
        </button>
      </div>
    </header>

    <div class="game-body">
      <GameLog />

      <GameTable ref="gameTableRef" @drawCard="store.drawCard()" />

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
        @playSelected="store.playSelected()"
        @playNope="onPlayNope"
        @leave="$emit('leave')"
      />
    </div>
  </div>
</template>

<script setup>
import { useEkStore } from '../../stores/explodingKitchen'
import { Settings, LogOut } from '@lucide/vue'
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
  width: 100%;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* ─── Header ─── */

.game-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background: rgba(28, 21, 18, 0.85);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--line);
  z-index: 100;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.game-title {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 700;
  font-size: 1.1rem;
  color: var(--steam-cream);
  white-space: nowrap;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: none;
  cursor: pointer;
  font-family: inherit;
  font-weight: 600;
  transition: all var(--ease-standard);
}

.settings-btn {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: transparent;
  color: var(--mild-cream);
  border: 1px solid var(--line);
}

.settings-btn:hover {
  border-color: var(--chili-orange);
  color: var(--steam-cream);
  background: rgba(226, 99, 44, 0.08);
}

.leave-btn {
  padding: 10px 20px;
  border-radius: 100px;
  background: var(--broth-red);
  color: var(--steam-cream);
  font-size: 0.85rem;
}

.leave-btn:hover {
  filter: brightness(1.15);
  transform: translateY(-1px);
}

.leave-btn:active {
  transform: scale(0.97);
}

/* ─── Body ─── */

.game-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  .game-header {
    padding: 6px 10px;
  }

  .game-title {
    font-size: 0.8rem;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 140px;
  }

  .header-right {
    gap: 4px;
  }

  .settings-btn {
    width: 32px;
    height: 32px;
    border-radius: 8px;
  }

  .leave-btn {
    padding: 0;
    width: 32px;
    height: 32px;
    border-radius: 8px;
    gap: 0;
    justify-content: center;
  }

  .leave-text {
    display: none;
  }

  .leave-icon {
    margin: 0;
    width: 14px;
    height: 14px;
  }

  .settings-btn svg {
    width: 14px;
    height: 14px;
  }
}
</style>
