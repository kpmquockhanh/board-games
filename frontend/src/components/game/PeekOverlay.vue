<template>
  <Transition name="modal">
    <div v-if="store.localTopCards" class="peek-overlay" data-testid="peek-overlay">
      <div class="peek-card" role="dialog" aria-modal="true" aria-labelledby="peek-title">
        <h2 id="peek-title">You peeked at the future!</h2>
        <p>The next {{ store.localTopCards.length }} card{{ store.localTopCards.length === 1 ? '' : 's' }} to be drawn:</p>
        <div class="peek-cards">
          <div v-for="(cardId, i) in store.localTopCards" :key="i" class="peek-card-item">
            <img
              v-if="store.getCardData(cardId)"
              :src="`/cards/${store.getCardData(cardId).file}`"
              :alt="store.getCardData(cardId)?.name"
              loading="lazy"
            >
            <span v-else class="peek-card-id">{{ cardId }}</span>
            <div class="peek-card-label">{{ i === 0 ? 'Next' : `${i + 1}${i === 1 ? 'nd' : 'rd'}` }}</div>
          </div>
        </div>
        <button class="btn" data-testid="peek-close" @click="store.closeTopCards()">Got it!</button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
</script>

<style scoped>
.peek-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-peek);
  background: rgba(27, 14, 6, 0.58);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.peek-card {
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 30px 26px;
  max-width: 480px;
  width: 100%;
  text-align: center;
}

.peek-card h2 {
  font-size: 1.5rem;
  margin-bottom: 8px;
  color: var(--ink);
}

.peek-card p {
  color: var(--muted-cream);
  font-size: 0.92rem;
  font-weight: 700;
  margin-bottom: 20px;
}

.peek-cards {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: center;
  margin-bottom: 24px;
}

.peek-card-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.peek-card-item img {
  width: 100px;
  height: 140px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  box-shadow: 0 4px 0 var(--edge);
  background: var(--sand);
}

.peek-card-id {
  width: 100px;
  height: 140px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--dim-edge);
  background: var(--dim-fill);
  color: var(--dim-text);
  font-size: 0.75rem;
  font-weight: 800;
}

/* Position in the draw order, chipped onto the card it describes. */
.peek-card-label {
  padding: 3px 12px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--grape);
  color: #fffdf6;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.66rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
</style>
