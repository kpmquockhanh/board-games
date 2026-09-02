<template>
  <div class="result-overlay">
    <div class="result-card" role="dialog" aria-modal="true" aria-labelledby="result-title">
      <h1 id="result-title">{{ store.winner === store.me?.name ? 'You win!' : 'Game Over' }}</h1>
      <p>
        {{ store.winner === store.me?.name
          ? 'Congratulations! You are the last chef standing.'
          : (store.winner ? store.winner + ' won the game.' : 'No one survived.')
        }}
      </p>
      <div v-if="store.recentLog.length > 0" class="result-log">
        <h3>Game log</h3>
        <div class="result-log-entries">
          <div v-for="(e, i) in store.recentLog" :key="i" class="log-entry">
            <b>{{ e.name }}</b> {{ e.text }}
          </div>
        </div>
      </div>
      <div class="result-actions">
        <router-link to="/" class="btn btn-secondary">Back to hub</router-link>
        <button v-if="isHost" class="btn-link" @click="$emit('delete')">Delete room</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useEkStore } from '../../stores/explodingKitchen'

const props = defineProps({
  isHost: { type: Boolean, default: false },
})

defineEmits(['delete'])

const store = useEkStore()
</script>

<style scoped>
.result-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-result);
  background: rgba(21, 15, 12, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.result-card {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 40px 36px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.result-card h1 {
  font-size: 2rem;
  margin-bottom: 10px;
}

.result-card > p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.95rem;
  line-height: 1.5;
  margin-bottom: 24px;
}

.result-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: center;
}

.result-log {
  margin-bottom: 20px;
  text-align: left;
}

.result-log h3 {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--gold);
  margin-bottom: 8px;
  text-align: center;
}

.result-log-entries {
  max-height: 140px;
  overflow-y: auto;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 10px;
  padding: 10px 14px;
}

.result-log-entries .log-entry {
  font-size: 0.78rem;
  padding: 3px 0;
  color: var(--mild-cream);
  border-bottom: 1px solid var(--line);
  line-height: 1.4;
}

.result-log-entries .log-entry:last-child {
  border-bottom: none;
}

.result-log-entries .log-entry b {
  color: var(--steam-cream);
}

.btn-link {
  background: transparent;
  border: none;
  color: var(--broth-red);
  padding: 8px 16px;
  font-size: 0.85rem;
  text-decoration: underline;
  cursor: pointer;
  font-family: inherit;
  transition: opacity var(--ease-standard);
}

.btn-link:hover {
  opacity: 0.7;
}
</style>
