<template>
  <Transition name="modal">
    <div v-if="store.positionModal.show" class="modal" data-testid="position-modal" ref="modalRef">
      <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="position-modal-title">
        <h2 id="position-modal-title">{{ store.positionModal.title }}</h2>
        <p>{{ store.positionModal.desc }}</p>
        <div class="position-picker">
          <div class="position-labels">
            <span>Top (drawn next)</span>
            <span>Bottom</span>
          </div>
          <input
            type="range"
            min="0"
            :max="store.positionModal.deckLength"
            v-model.number="defusePosition"
            class="position-slider"
            data-testid="position-slider"
            aria-label="How many cards down from the top to place the card"
            ref="sliderRef"
          >
          <div class="position-value" aria-live="polite">{{ positionLabel }}</div>
        </div>
        <button class="btn" data-testid="position-place" ref="placeBtn" @click="store.positionModal.resolve(defusePosition)">Place it!</button>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()

// The server inserts the card `position` places down from the top of the deck,
// and the top is what gets drawn next — so 0 hands it straight to the next
// player. The slider runs in that same direction to keep the two in step.
const defusePosition = ref(0)

const positionLabel = computed(() => {
  const n = defusePosition.value
  const deckLength = store.positionModal.deckLength
  if (n === 0) return 'On top — the next player draws it'
  if (n >= deckLength) return 'At the very bottom of the deck'
  return `${n} card${n === 1 ? '' : 's'} down from the top`
})
const sliderRef = ref(null)
const placeBtn = ref(null)

watch(() => store.positionModal.show, async (show) => {
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

.position-picker {
  margin: 20px 0;
}

.position-labels {
  display: flex;
  justify-content: space-between;
  font-size: 0.68rem;
  font-weight: 800;
  color: var(--muted-cream);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-bottom: 10px;
}

/* The track is an outlined trough and the thumb a chili puck that sits in it,
   so the control matches the pressable depth used everywhere else. */
.position-slider {
  width: 100%;
  height: 16px;
  border-radius: 100px;
  border: 2.5px solid var(--edge);
  background: var(--sand);
  outline: none;
  -webkit-appearance: none;
  appearance: none;
}

.position-slider::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--chili);
  cursor: pointer;
  border: 2.5px solid var(--edge);
  transition: transform var(--duration-fast) var(--ease-out);
}

.position-slider::-webkit-slider-thumb:hover {
  transform: scale(1.12);
}

.position-slider::-moz-range-thumb {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--chili);
  cursor: pointer;
  border: 2.5px solid var(--edge);
}

.position-value {
  display: inline-block;
  margin-top: 14px;
  padding: 6px 14px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--gold);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.85rem;
  font-weight: 800;
}
</style>
