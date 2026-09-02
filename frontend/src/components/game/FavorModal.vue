<template>
  <div v-if="store.favorModal.show" class="modal" ref="modalRef">
    <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="favor-title">
      <template v-if="store.favorModal.step === 'target'">
        <h2 id="favor-title"><Gift :size="18" /> Favor</h2>
        <p>Choose a player to take a card from:</p>
        <div class="favor-targets">
          <button
            v-for="name in aliveTargets"
            :key="name"
            class="target-btn"
            @click="store.favorModal.resolve(name)"
          >
            <span class="target-avatar" :style="{ background: getPlayerColor(name) }">
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
            v-for="card in handCards"
            :key="card.id"
            :cardData="card"
            size="small"
            @click="store.favorModal.resolve(card.id)"
          />
        </div>
        <div v-if="handCards.length === 0" class="empty-state">
          You have no cards to give
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { Gift } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import CardView from '../CardView.vue'

const store = useEkStore()
const modalRef = ref(null)

const aliveTargets = computed(() => {
  return store.turnOrder.filter(
    name => name !== store.me?.name &&
      store.gameState?.players?.[name]?.alive &&
      (store.gameState?.players?.[name]?.hand?.length || 0) > 0
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
  return store.gameState?.players?.[name]?.hand?.length || 0
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
  background: rgba(21, 15, 12, 0.88);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.modal-card {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 32px 28px;
  max-width: 420px;
  width: 100%;
  text-align: center;
}

.modal-card-wide {
  max-width: 560px;
}

.modal-card h2 {
  font-size: 1.3rem;
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.modal-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.9rem;
  line-height: 1.5;
  margin-bottom: 18px;
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
  padding: 12px 16px;
  background: var(--panel-hover, rgba(255,255,255,0.05));
  border: 1px solid var(--line);
  border-radius: 12px;
  color: var(--cream);
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
}

.target-btn:hover {
  background: var(--panel-active, rgba(255,255,255,0.1));
  border-color: var(--accent, #f59e0b);
}

.target-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 0.9rem;
  color: white;
  flex-shrink: 0;
}

.target-name {
  font-weight: 600;
  font-size: 0.95rem;
}

.target-count {
  margin-left: auto;
  font-size: 0.8rem;
  opacity: 0.6;
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

.empty-state {
  color: var(--mild-cream);
  opacity: 0.5;
  font-size: 0.9rem;
  padding: 20px 0;
  margin-bottom: 16px;
}
</style>
