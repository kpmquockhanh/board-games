<template>
  <div v-if="store.defusePositionModal.show" class="modal" ref="modalRef">
    <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="position-modal-title">
      <h2 id="position-modal-title">Place the Explosive</h2>
      <p>Choose where to put the Explosive card back in the deck:</p>
      <div class="position-picker">
        <div class="position-labels">
          <span>Bottom of deck</span>
          <span>Top of deck</span>
        </div>
        <input
          type="range"
          min="0"
          :max="store.defusePositionModal.deckLength"
          v-model.number="defusePosition"
          class="position-slider"
          aria-label="Explosive card position in deck"
          ref="sliderRef"
        >
        <div class="position-value" aria-live="polite">
          Position: {{ defusePosition + 1 }} of {{ store.defusePositionModal.deckLength + 1 }}
        </div>
      </div>
      <button class="btn" ref="placeBtn" @click="store.defusePositionModal.resolve(defusePosition)">Place it!</button>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const defusePosition = ref(0)
const sliderRef = ref(null)
const placeBtn = ref(null)

watch(() => store.defusePositionModal.show, async (show) => {
  if (show) {
    defusePosition.value = 0
    await nextTick()
    sliderRef.value?.focus()
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

.position-picker {
  margin: 20px 0;
}

.position-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.7rem;
  color: var(--muted-cream);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-bottom: 8px;
}

.position-slider {
  width: 100%;
  height: 8px;
  border-radius: 4px;
  background: var(--line);
  outline: none;
  -webkit-appearance: none;
  appearance: none;
}

.position-slider:focus-visible {
  outline: 2px solid var(--chili-orange);
  outline-offset: 4px;
}

.position-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  cursor: pointer;
  border: 2px solid var(--steam-cream);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
  transition: transform var(--ease-standard);
}

.position-slider::-webkit-slider-thumb:hover {
  transform: scale(1.15);
}

.position-slider::-moz-range-thumb {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  cursor: pointer;
  border: 2px solid var(--steam-cream);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.4);
}

.position-value {
  margin-top: 12px;
  font-size: 0.85rem;
  color: var(--gold);
  font-weight: 600;
  text-align: center;
}
</style>
