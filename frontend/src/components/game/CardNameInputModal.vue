<template>
  <Transition name="modal">
    <div v-if="store.cardNameInputModal.show" class="modal" data-testid="cardname-modal" ref="modalRef">
      <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="cardname-modal-title">
        <h2 id="cardname-modal-title">{{ store.cardNameInputModal.title }}</h2>
        <p>{{ store.cardNameInputModal.desc }}</p>
        <input
          ref="inputRef"
          v-model="cardName"
          class="card-input"
          data-testid="cardname-input"
          type="text"
          placeholder="Type card name..."
          @keyup.enter="submit"
          @keyup.escape="cancel"
        />
        <div class="modal-actions">
          <button class="btn" data-testid="cardname-confirm" @click="submit" :disabled="!cardName.trim()">Confirm</button>
          <button class="btn btn-secondary" @click="cancel">Cancel</button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const inputRef = ref(null)
const cardName = ref('')

watch(() => store.cardNameInputModal.show, async (show) => {
  if (show) {
    cardName.value = ''
    await nextTick()
    inputRef.value?.focus()
  }
})

function submit() {
  const val = cardName.value.trim()
  if (val) {
    store.cardNameInputModal.resolve(val)
  }
}

function cancel() {
  store.cardNameInputModal.resolve(null)
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

.card-input {
  width: 100%;
  padding: 12px 16px;
  min-height: 52px;
  border-radius: var(--radius-md);
  border: var(--edge-w) solid var(--edge);
  background: var(--sand);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.05rem;
  outline: none;
  box-sizing: border-box;
}

.card-input::placeholder {
  color: var(--dim-text);
  font-weight: 700;
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
