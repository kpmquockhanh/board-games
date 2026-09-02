<template>
  <div v-if="store.modal.show" class="modal" ref="modalRef">
    <div class="modal-card" role="dialog" aria-modal="true" :aria-labelledby="titleId">
      <h2 :id="titleId">{{ store.modal.title }}</h2>
      <p>{{ store.modal.desc }}</p>
      <div class="modal-actions">
        <button class="btn" ref="firstBtn" @click="store.modal.resolve(true)">Yes, defuse!</button>
        <button class="btn btn-secondary" @click="store.modal.resolve(false)">No, let me explode</button>
      </div>
    </div>
  </div>
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

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
