<template>
  <div class="result-overlay" data-testid="result-overlay">
    <div class="result-card" role="dialog" aria-modal="true" aria-labelledby="result-title">
      <h1 id="result-title" data-testid="result-title">{{ store.didIWin ? 'You win!' : 'Game Over' }}</h1>
      <p>{{ outcome }}</p>
      <div v-if="store.recentLog.length > 0" class="result-log">
        <h3>Game log</h3>
        <div class="result-log-entries">
          <div v-for="(e, i) in store.recentLog" :key="i" class="log-entry">
            <b>{{ e.name }}</b> {{ e.text }}
          </div>
        </div>
      </div>
      <div class="result-actions">
        <button class="btn" data-testid="result-rematch" @click="store.requestRematch()">Play again</button>
        <router-link to="/" class="btn btn-secondary">Back to hub</router-link>
        <button v-if="isHost" class="btn-link" @click="$emit('delete')">Delete room</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'

const props = defineProps({
  isHost: { type: Boolean, default: false },
})

defineEmits(['delete'])

const store = useEkStore()

const outcome = computed(() => {
  const winners = store.winners
  if (winners.length > 1) {
    const others = winners.filter((n) => n !== store.me?.name)
    return store.didIWin
      ? `The deck ran out — you and ${others.join(', ')} share the win.`
      : `The deck ran out — ${winners.join(', ')} share the win.`
  }
  if (winners.length === 1) {
    return store.didIWin
      ? 'Congratulations! You are the last chef standing.'
      : `${winners[0]} won the game.`
  }
  return 'No one survived.'
})
</script>

<style scoped>
.result-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-result);
  background: rgba(27, 14, 6, 0.58);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.result-card {
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 36px 30px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.result-card h1 {
  font-size: 2.2rem;
  line-height: 1.05;
  color: var(--ink);
  margin-bottom: 10px;
}

.result-card > p {
  color: var(--muted-cream);
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.5;
  margin-bottom: 24px;
}

/* The buttons are the global kitchen .btn / .btn-secondary / .btn-link — this
   only stacks them, so the three stay in step with every other screen. */
.result-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: stretch;
}

.result-log {
  margin-bottom: 20px;
  text-align: left;
}

.result-log h3 {
  display: inline-block;
  padding: 4px 12px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--gold);
  color: var(--ink);
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin-bottom: 10px;
}

.result-log-entries {
  max-height: 140px;
  overflow-y: auto;
  background: var(--sand);
  border: 2.5px solid var(--edge);
  border-radius: var(--radius-md);
  padding: 10px 14px;
}

.result-log-entries .log-entry {
  font-size: 0.78rem;
  font-weight: 700;
  padding: 4px 0;
  color: var(--mild-cream);
  border-bottom: 2px solid rgba(27, 14, 6, 0.12);
  line-height: 1.4;
}

.result-log-entries .log-entry:last-child {
  border-bottom: none;
}

.result-log-entries .log-entry b {
  color: var(--ink);
  font-weight: 800;
}
</style>
