<template>
  <div class="game">
    <header class="game-header">
      <div class="header-left">
        <span class="title-mark" aria-hidden="true"><Flame :size="16" /></span>
        <span class="game-title">Exploding Kitchen 2</span>
      </div>
      <div class="header-right">
        <button class="header-btn leave-btn" data-testid="game-leave" aria-label="Leave game" @click="$emit('leave')">
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
      <ChooseCardModal />
      <GarbageCollectionModal />
      <FavorModal />
      <NopeOverlay />
      <PeekOverlay />

      <GameHand
        ref="gameHandRef"
        @cardClick="onCardClick"
        @playSelected="store.playSelected()"
        @playNope="onPlayNope"
        @passNope="store.passNope()"
        @leave="$emit('leave')"
      />

      <DrawFlight ref="drawFlightRef" />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'
import { LogOut, Flame } from '@lucide/vue'
import DrawFlight from './DrawFlight.vue'
import GameTable from './GameTable.vue'
import GameHand from './GameHand.vue'
import GameLog from './GameLog.vue'
import DefuseModal from './DefuseModal.vue'
import PositionPickerModal from './PositionPickerModal.vue'
import TargetSelectModal from './TargetSelectModal.vue'
import CardNameInputModal from './CardNameInputModal.vue'
import DiscardPickerModal from './DiscardPickerModal.vue'
import ChooseCardModal from './ChooseCardModal.vue'
import GarbageCollectionModal from './GarbageCollectionModal.vue'
import FavorModal from './FavorModal.vue'
import NopeOverlay from './NopeOverlay.vue'
import PeekOverlay from './PeekOverlay.vue'

defineProps({
  isHost: { type: Boolean, default: false },
})

defineEmits(['leave', 'delete'])

const store = useEkStore()

const gameTableRef = ref(null)
const gameHandRef = ref(null)
const drawFlightRef = ref(null)

// A card drawn from the deck is already in the hand by the time we hear about
// it, so the flight is a ghost copy travelling to where the real card sits.
// The card stays hidden until the ghost lands on it.
watch(
  () => store.drawFlight,
  async (drawn) => {
    if (!drawn) return
    // Wait for the fan to lay out, or the card has no box to fly to yet.
    await nextTick()
    const deckRect = gameTableRef.value?.getDeckRect()
    const cardRect = gameHandRef.value?.getCardRect(drawn.uid)
    try {
      await drawFlightRef.value?.run(drawn.cardId, deckRect, cardRect)
    } finally {
      // Reveals the card whether or not the flight actually ran — a missing
      // rect or reduced motion must not leave it invisible.
      store.clearDrawFlight()
    }
  }
)

function onCardClick(idx) {
  // During a window the only card that can be picked is a Nope — including by
  // the player who is being Noped and may counter.
  if (store.nopeWindow.active) {
    const cardId = store.myHand[idx]
    if (store.canINope && store.getCategoryName(cardId) === 'Nope') {
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
  background: var(--paper);
}

/* ─── Header ─── */

.game-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 20px;
  background: var(--surface);
  border-bottom: var(--edge-w) solid var(--edge);
  z-index: 100;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.title-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  border-radius: 10px;
  border: 2.5px solid var(--edge);
  background: var(--chili);
  color: var(--ink);
  transform: rotate(-6deg);
}

.game-title {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.15rem;
  color: var(--ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.header-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  cursor: pointer;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  border: 2.5px solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  transition: var(--transition-interactive);
}

.leave-btn {
  padding: 8px 18px;
  min-height: 44px;
  border-radius: var(--radius-sm);
  background: var(--broth);
  color: var(--surface);
  font-size: 0.9rem;
}

.leave-btn:hover {
  background: #b81f1f;
}

.leave-btn:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
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

  .title-mark {
    width: 26px;
    height: 26px;
    border-radius: 8px;
  }

  .game-title {
    font-size: 0.9rem;
    max-width: 150px;
  }

  .header-right {
    gap: 4px;
  }

  /* Icon-only at this width, but the tap target stays at 44px — the label
     moves to aria-label rather than disappearing outright. */
  .leave-btn {
    padding: 0;
    width: 44px;
    height: 44px;
    border-radius: 12px;
    gap: 0;
  }

  .leave-text {
    display: none;
  }

  .leave-icon {
    margin: 0;
    width: 18px;
    height: 18px;
  }
}
</style>
