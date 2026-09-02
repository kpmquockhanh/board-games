<template>
  <div v-if="store.discardPickerModal.show" class="modal" ref="modalRef">
    <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="discard-picker-title">
      <h2 id="discard-picker-title">Pick a card from the discard pile</h2>
      <p>Select the card you want to pick up:</p>
      <div class="discard-grid">
        <CardView
          v-for="card in discardCards"
          :key="card.id"
          :cardData="card"
          size="small"
          @click="store.discardPickerModal.resolve(card.id)"
        />
      </div>
      <div v-if="discardCards.length === 0" class="empty-state">
        No cards in the discard pile
      </div>
      <div class="modal-actions">
        <button class="btn btn-secondary" @click="store.discardPickerModal.resolve(null)">Cancel</button>
      </div>
    </div>
  </div>
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
}

.modal-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.9rem;
  line-height: 1.5;
  margin-bottom: 18px;
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

.empty-state {
  color: var(--mild-cream);
  opacity: 0.5;
  font-size: 0.9rem;
  padding: 20px 0;
  margin-bottom: 16px;
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
