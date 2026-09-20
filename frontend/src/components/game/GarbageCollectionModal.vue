<template>
  <Transition name="modal">
    <div v-if="store.garbageCollectionModal.show" class="modal" data-testid="garbage-modal" ref="modalRef">
      <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="gc-title">
        <h2 id="gc-title"><Trash2 :size="18" /> Garbage Collection</h2>
        <p>Choose one card from your hand to put into the draw pile:</p>
        <div class="gc-grid">
          <CardView
            v-for="(card, i) in handCards"
            :key="i"
            :cardData="card"
            size="small"
            :data-testid="'garbage-card-' + i"
            @click="store.garbageCollectionModal.resolve(card.id)"
          />
        </div>
        <div v-if="handCards.length === 0" class="empty-state">
          You have no cards to discard
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { Trash2 } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import CardView from '../CardView.vue'

const store = useEkStore()
const modalRef = ref(null)

const handCards = computed(() => {
  const hand = store.gameState?.players?.[store.me?.name]?.hand || []
  return hand.map(id => store.getCardData(id)).filter(Boolean)
})

watch(() => store.garbageCollectionModal.show, async (show) => {
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

.gc-grid {
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
