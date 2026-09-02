<template>
  <div v-if="store.cardNameInputModal.show" class="modal" ref="modalRef">
    <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="cardname-modal-title">
      <h2 id="cardname-modal-title">{{ store.cardNameInputModal.title }}</h2>
      <p>{{ store.cardNameInputModal.desc }}</p>
      <input
        ref="inputRef"
        v-model="cardName"
        class="card-input"
        type="text"
        placeholder="Type card name..."
        @keyup.enter="submit"
        @keyup.escape="cancel"
      />
      <div class="modal-actions">
        <button class="btn" @click="submit" :disabled="!cardName.trim()">Confirm</button>
        <button class="btn btn-secondary" @click="cancel">Cancel</button>
      </div>
    </div>
  </div>
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

.card-input {
  width: 100%;
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.05);
  color: var(--steam-cream);
  font-size: 1rem;
  outline: none;
  transition: border-color var(--ease-standard);
  box-sizing: border-box;
}

.card-input:focus {
  border-color: var(--chili-orange);
}

.card-input::placeholder {
  color: var(--mild-cream);
  opacity: 0.5;
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
