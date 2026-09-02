<template>
  <div v-if="store.garbageCollectionModal.show" class="modal" ref="modalRef">
    <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="gc-title">
      <h2 id="gc-title"><Trash2 :size="18" /> Garbage Collection</h2>
      <p>Choose one card from your hand to put into the draw pile:</p>
      <div class="gc-grid">
        <CardView
          v-for="card in handCards"
          :key="card.id"
          :cardData="card"
          size="small"
          @click="store.garbageCollectionModal.resolve(card.id)"
        />
      </div>
      <div v-if="handCards.length === 0" class="empty-state">
        You have no cards to discard
      </div>
    </div>
  </div>
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

.empty-state {
  color: var(--mild-cream);
  opacity: 0.5;
  font-size: 0.9rem;
  padding: 20px 0;
  margin-bottom: 16px;
}
</style>
