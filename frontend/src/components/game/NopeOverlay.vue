<template>
  <div v-if="store.nopeWindow.active" class="nope-overlay" data-testid="nope-overlay" role="alert" aria-live="assertive">
    <div class="nope-banner">
      <div class="nope-banner-text">
        <span v-if="store.nopeWindow.playerId === store.me?.name">
          {{ waitingText }} {{ store.nopeTimeLeft }}s
        </span>
        <span v-else>
          {{ store.nopeWindow.playerId }} played {{ store.nopeWindow.cardName }}!
          {{ store.canINope ? 'Play a Nope to cancel, or let it through.' : waitingText }}
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
import { computed } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()

// A head count of who has yet to answer — never a hint about what they hold.
const waitingText = computed(() => {
  const n = store.nopeWaitingCount
  if (n <= 0) return 'Closing the window…'
  return `Waiting for ${n} player${n === 1 ? '' : 's'}…`
})
</script>

<style scoped>
.nope-overlay {
  position: absolute;
  top: 50px;
  left: 50%;
  transform: translateX(-50%);
  z-index: var(--z-nope);
  width: 340px;
  max-width: 92%;
}

/* This banner interrupts play for a few seconds, so it gets the loudest fill
   in the theme. Broth red is the one bright fill that carries light text. */
.nope-banner {
  background: var(--broth);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  box-shadow: var(--lift);
  padding: 12px 16px;
  text-align: center;
}

.nope-banner-text {
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.95rem;
  font-weight: 800;
  color: var(--surface);
  margin-bottom: 10px;
}

.nope-timer-bar {
  height: 10px;
  border-radius: 100px;
  border: 2.5px solid var(--edge);
  background: rgba(27, 14, 6, 0.35);
  overflow: hidden;
}

.nope-timer-fill {
  height: 100%;
  background: var(--gold);
  transition: width 1s linear;
}
</style>
