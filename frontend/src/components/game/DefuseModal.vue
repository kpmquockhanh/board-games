<template>
  <Transition name="modal">
    <div v-if="store.modal.show" class="modal" data-testid="defuse-modal" ref="modalRef">
      <div class="modal-card" role="dialog" aria-modal="true" :aria-labelledby="titleId">
        <h2 :id="titleId">{{ store.modal.title }}</h2>
        <p>{{ store.modal.desc }}</p>
        <div class="modal-actions">
          <button class="btn" data-testid="defuse-yes" ref="firstBtn" @click="store.modal.resolve(true)">Yes, defuse!</button>
          <button class="btn btn-secondary" data-testid="defuse-no" @click="store.modal.resolve(false)">No, let me explode</button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const titleId = 'defuse-modal-title'
const modalRef = ref(null)
const firstBtn = ref(null)

watch(() => store.modal.show, async (show) => {
  if (show) {
    await nextTick()
    firstBtn.value?.focus()
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

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
