<template>
  <Transition name="modal">
    <div v-if="store.targetSelectModal.show" class="modal" data-testid="target-modal" ref="modalRef">
      <div class="modal-card" role="dialog" aria-modal="true" aria-labelledby="target-modal-title">
        <h2 id="target-modal-title">Choose a target</h2>
        <p>Select a player to steal from:</p>
        <div class="player-list">
          <button
            v-for="player in store.targetSelectModal.players"
            :key="player"
            class="player-btn"
            :data-testid="'target-player-' + player"
            @click="store.targetSelectModal.resolve(player)"
          >
            {{ player }}
          </button>
        </div>
        <div class="modal-actions">
          <button class="btn btn-secondary" data-testid="target-cancel" @click="store.targetSelectModal.resolve(null)">Cancel</button>
        </div>
      </div>
    </div>
  </Transition>
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

.player-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.player-btn {
  padding: 12px 16px;
  min-height: 52px;
  border-radius: var(--radius-md);
  border: var(--edge-w) solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  background: var(--surface);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1rem;
  cursor: pointer;
  transition: var(--transition-interactive);
}

.player-btn:hover {
  background: var(--chili);
}

.player-btn:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

.modal-actions {
  display: flex;
  gap: 10px;
  justify-content: center;
  margin-top: 16px;
}
</style>
