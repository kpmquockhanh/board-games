<template>
  <Transition name="modal">
    <div v-if="store.favorModal.show" class="modal" data-testid="favor-modal" ref="modalRef">
      <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="favor-title">
        <template v-if="store.favorModal.step === 'target'">
          <h2 id="favor-title"><Gift :size="18" /> Favor</h2>
          <p>Choose a player to take a card from:</p>
          <div class="favor-targets">
            <button
              v-for="name in aliveTargets"
              :key="name"
              class="target-btn"
              :data-testid="'favor-target-' + name"
              @click="store.favorModal.resolve(name)"
            >
              <span class="target-avatar" :style="{ background: getPlayerColor(name), color: readableInk(getPlayerColor(name)) }">
                {{ name.charAt(0).toUpperCase() }}
              </span>
              <span class="target-name">{{ name }}</span>
              <span class="target-count">{{ getPlayerCardCount(name) }} cards</span>
            </button>
          </div>
        </template>

        <template v-else-if="store.favorModal.step === 'card'">
          <h2 id="favor-title"><Gift :size="18" /> Give a Card</h2>
          <p>Choose one card from your hand to give:</p>
          <div class="favor-grid">
            <CardView
              v-for="(card, i) in handCards"
              :key="i"
              :cardData="card"
              size="small"
              :data-testid="'favor-card-' + i"
              @click="store.favorModal.resolve(card.id)"
            />
          </div>
          <div v-if="handCards.length === 0" class="empty-state">
            You have no cards to give
          </div>
        </template>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { Gift } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import { readableInk } from '../../utils/contrast'
import CardView from '../CardView.vue'

const store = useEkStore()
const modalRef = ref(null)

const aliveTargets = computed(() => {
  return store.turnOrder.filter(
    name => name !== store.me?.name &&
      store.gameState?.players?.[name]?.alive &&
      (store.gameState?.players?.[name]?.handCount || 0) > 0
  )
})

const handCards = computed(() => {
  const hand = store.gameState?.players?.[store.me?.name]?.hand || []
  return hand.map(id => store.getCardData(id)).filter(Boolean)
})

function getPlayerColor(name) {
  return store.gameState?.players?.[name]?.color || '#888'
}

function getPlayerCardCount(name) {
  return store.gameState?.players?.[name]?.handCount || 0
}

watch(() => store.favorModal.show, async (show) => {
  if (show) {
    await nextTick()
  }
})
</script>

<style scoped>
.modal {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  /* Ink at 58% rather than a near-opaque black — on a light ground the game
     behind the dialog should stay legible, not be blacked out. */
  background: rgba(27, 14, 6, 0.58);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal-card {
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 30px 26px;
  max-width: 420px;
  width: 100%;
  text-align: center;
}

.modal-card h2 {
  font-size: 1.45rem;
  color: var(--ink);
  margin-bottom: 10px;
}

.modal-card p {
  color: var(--muted-cream);
  font-size: 0.92rem;
  font-weight: 700;
  line-height: 1.5;
  margin-bottom: 18px;
}

.modal-card h2 {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.modal-card-wide {
  max-width: 560px;
}

.favor-targets {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 340px;
  overflow-y: auto;
  padding: 4px;
  margin-bottom: 16px;
}

.target-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  min-height: 56px;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  box-shadow: 0 3px 0 var(--edge);
  color: var(--ink);
  cursor: pointer;
  transition: var(--transition-interactive);
  text-align: left;
}

.target-btn:hover {
  background: var(--sand);
}

.target-btn:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

/* The colour comes from the player, so the initial sits on an ink outline and
   ink text — the one pairing that holds up against every seat colour. */
.target-avatar {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  border: 2.5px solid var(--edge);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.2rem;
  color: var(--ink);
  flex-shrink: 0;
}

/* Names are user-supplied, so the row truncates instead of pushing the count
   out of the button. */
.target-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1rem;
}

.target-count {
  flex-shrink: 0;
  padding: 3px 10px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--sand);
  color: var(--ink);
  font-size: 0.72rem;
  font-weight: 800;
}

.favor-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  max-height: 340px;
  overflow-y: auto;
  padding: 4px;
  margin-bottom: 16px;
}

/* Tan-on-tan rather than a faded cream — opacity on a light ground just
   erases text instead of quieting it. */
.empty-state {
  color: var(--dim-text);
  background: var(--sand);
  border: 2px dashed var(--dim-edge);
  border-radius: var(--radius-md);
  font-size: 0.9rem;
  font-weight: 700;
  padding: 18px 12px;
  margin-bottom: 16px;
}
</style>
