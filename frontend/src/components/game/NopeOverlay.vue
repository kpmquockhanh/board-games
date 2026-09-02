<template>
  <div v-if="store.nopeWindow.active" class="nope-overlay" role="alert" aria-live="assertive">
    <div class="nope-banner">
      <div class="nope-banner-text">
        <span v-if="store.isMyTurn">
          Waiting for responses… {{ store.nopeTimeLeft }}s
        </span>
        <span v-else>
          {{ store.nopeWindow.playerId }} played {{ store.nopeWindow.cardName }}!
          {{ store.canINope ? 'Play a Nope to cancel!' : 'Waiting for responses…' }}
          {{ store.nopeTimeLeft }}s
        </span>
      </div>
      <div class="nope-timer-bar">
        <div class="nope-timer-fill" :style="{ width: (store.nopeTimeLeft / 8 * 100) + '%' }"></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
</script>

<style scoped>
.nope-overlay {
  position: absolute;
  top: 50px;
  left: 50%;
  transform: translateX(-50%);
  z-index: var(--z-nope);
  width: 320px;
  max-width: 90%;
}

.nope-banner {
  background: rgba(30, 20, 15, 0.95);
  border: 2px solid rgba(238, 194, 92, 0.6);
  border-radius: 12px;
  padding: 12px 16px;
  text-align: center;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
}

.nope-banner-text {
  font-size: 0.9rem;
  color: var(--mild-cream);
  margin-bottom: 8px;
  font-weight: 600;
}

.nope-timer-bar {
  height: 4px;
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
  overflow: hidden;
}

.nope-timer-fill {
  height: 100%;
  background: var(--gold);
  border-radius: 2px;
  transition: width 1s linear;
}
</style>
