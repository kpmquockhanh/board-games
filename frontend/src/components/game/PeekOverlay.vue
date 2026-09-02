<template>
  <div v-if="store.localTopCards" class="peek-overlay">
    <div class="peek-card" role="dialog" aria-modal="true" aria-labelledby="peek-title">
      <h2 id="peek-title">You peeked at the future!</h2>
      <p>Top {{ store.localTopCards.length }} cards of the deck:</p>
      <div class="peek-cards">
        <div v-for="(cardId, i) in store.localTopCards" :key="i" class="peek-card-item">
          <img
            v-if="store.getCardData(cardId)"
            :src="`/cards/${store.getCardData(cardId).file}`"
            :alt="store.getCardData(cardId)?.name"
            loading="lazy"
          >
          <span v-else class="peek-card-id">{{ cardId }}</span>
          <div class="peek-card-label">{{ i === 0 ? 'Top' : i === store.localTopCards.length - 1 ? 'Bottom' : 'Middle' }}</div>
        </div>
      </div>
      <button class="btn" @click="store.closeTopCards()">Got it!</button>
    </div>
  </div>
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
  background: rgba(21, 15, 12, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.peek-card {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 32px 28px;
  max-width: 480px;
  width: 100%;
  text-align: center;
}

.peek-card h2 {
  font-size: 1.4rem;
  margin-bottom: 8px;
  color: var(--gold);
}

.peek-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.9rem;
  margin-bottom: 20px;
}

.peek-cards {
  display: flex;
  gap: 12px;
  justify-content: center;
  margin-bottom: 24px;
}

.peek-card-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
}

.peek-card-item img {
  width: 100px;
  height: 140px;
  object-fit: cover;
  border-radius: 10px;
  border: 2px solid var(--gold);
  box-shadow: 0 4px 16px rgba(238, 194, 92, 0.3);
}

.peek-card-id {
  width: 100px;
  height: 140px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  border: 2px solid var(--line);
  background: var(--charcoal);
  color: var(--mild-cream);
  font-size: 0.75rem;
  opacity: 0.6;
}

.peek-card-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--gold);
  font-weight: 600;
}
</style>
