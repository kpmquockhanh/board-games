<template>
  <Transition name="modal">
    <div v-if="store.discardPickerModal.show" class="modal" data-testid="discard-picker-modal" ref="modalRef">
      <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="discard-picker-title">
        <h2 id="discard-picker-title">Pick a card from the discard pile</h2>
        <p>Select the card you want to pick up:</p>
        <div class="discard-grid">
          <CardView
            v-for="(card, i) in discardCards"
            :key="i"
            :cardData="card"
            size="small"
            :data-testid="'discard-card-' + i"
            @click="store.discardPickerModal.resolve(card.id)"
          />
        </div>
        <div v-if="discardCards.length === 0" class="empty-state">
          No cards in the discard pile
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" data-testid="discard-cancel" @click="store.discardPickerModal.resolve(null)">Cancel</button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'
import CardView from '../CardView.vue'

const store = useEkStore()
const modalRef = ref(null)

const discardCards = computed(() => {
  const discard = store.gameState?.discard || []
  return discard.map(id => store.getCardData(id)).filter(Boolean)
})

watch(() => store.discardPickerModal.show, async (show) => {
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

.modal-card-wide {
  max-width: 560px;
}

.discard-grid {
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

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
