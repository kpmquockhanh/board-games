<template>
  <div v-if="store.targetSelectModal.show" class="modal" ref="modalRef">
    <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="target-modal-title">
      <h2 id="target-modal-title">Choose a target</h2>
      <p>Select a player to steal from:</p>
      <div class="player-list">
        <button
          v-for="player in store.targetSelectModal.players"
          :key="player"
          class="player-btn"
          @click="store.targetSelectModal.resolve(player)"
        >
          {{ player }}
        </button>
      </div>
      <div class="modal-actions">
        <button class="btn btn-secondary" @click="store.targetSelectModal.resolve(null)">Cancel</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const modalRef = ref(null)
const firstBtn = ref(null)

watch(() => store.targetSelectModal.show, async (show) => {
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

.player-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.player-btn {
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.05);
  color: var(--steam-cream);
  font-size: 1rem;
  cursor: pointer;
  transition: all var(--ease-standard);
}

.player-btn:hover {
  border-color: var(--chili-orange);
  background: rgba(226, 99, 44, 0.12);
  color: var(--chili-orange);
}

.player-btn:active {
  transform: scale(0.98);
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
