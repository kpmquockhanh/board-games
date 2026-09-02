<template>
  <div class="game-log">
    <h3>Game log</h3>
    <div v-for="(e, i) in store.recentLog" :key="i" class="log-entry">
      <b>{{ e.name }}</b> {{ e.text }}
      <span class="log-time">{{ formatTime(e.ts) }}</span>
    </div>
  </div>
</template>

<script setup>
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()

function formatTime(ts) {
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.game-log {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 14px;
  padding: 14px 18px;
  max-height: 160px;
  overflow-y: auto;
  margin-top: 16px;
}

.game-log h3 {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--gold);
  margin-bottom: 8px;
}

.log-entry {
  font-size: 0.82rem;
  color: var(--mild-cream);
  padding: 4px 0;
  border-bottom: 1px solid var(--line);
  line-height: 1.4;
}

.log-entry:last-child {
  border-bottom: none;
}

.log-entry b {
  color: var(--steam-cream);
}

.log-time {
  font-size: 0.68rem;
  color: var(--muted-cream);
  margin-left: 6px;
}

@media (max-width: 640px) {
  .game-log {
    padding: 10px 12px;
    max-height: 120px;
    margin-top: 12px;
  }
}
</style>
