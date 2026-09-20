<template>
  <Transition name="modal">
    <div v-if="store.chooseCardModal.show" class="modal" data-testid="choose-card-modal">
      <div class="modal-card modal-card-wide" role="dialog" aria-modal="true" aria-labelledby="choose-card-title">
        <h2 id="choose-card-title"><Pickaxe :size="18" /> {{ store.chooseCardModal.title }}</h2>
        <p>{{ store.chooseCardModal.desc }}</p>
        <div class="choose-grid">
          <button
            v-for="(card, i) in cards"
            :key="i"
            class="choose-card"
            :data-testid="'choose-card-' + i"
            @click="store.chooseCardModal.resolve(i)"
          >
            <CardView :cardData="card" size="small" />
            <span class="choose-depth">{{ depthLabel(i) }}</span>
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { computed } from 'vue'
import { Pickaxe } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import CardView from '../CardView.vue'

const store = useEkStore()

const cards = computed(() =>
  (store.chooseCardModal.cards || []).map((id) => store.getCardData(id) || { id, name: id })
)

// The cards come off the top of the deck in order, and the ones left behind go
// back the same way — so where each one sits is worth knowing.
function depthLabel(i) {
  if (i === 0) return 'Top'
  return `${i + 1}${i === 1 ? 'nd' : 'rd'} down`
}
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

.choose-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  justify-content: center;
  padding: 4px;
}

.choose-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  font-family: inherit;
  transition: transform var(--duration-fast) var(--ease-out);
}

.choose-card:hover {
  transform: translateY(-5px);
}

.choose-card:active {
  transform: translateY(0);
}

/* Which slot in the deck this card came from — a chip, so it reads as a label
   attached to the card rather than a caption floating under it. */
.choose-depth {
  padding: 3px 10px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--gold);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.66rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
</style>
