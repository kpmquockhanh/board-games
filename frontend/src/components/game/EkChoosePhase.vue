<template>
  <div class="choose-overlay">
    <div class="choose-card">
      <button class="close-btn" @click="$router.push('/')" aria-label="Back to hub"><X :size="16" /></button>
      <div class="choose-icon" aria-hidden="true"><Bomb :size="32" /></div>
      <h1>Exploding Kitchen</h1>
      <p>A card game of cooking chaos. Draw cards, avoid the exploding kitchen, and be the last chef standing.</p>
      <div class="choose-actions">
        <button class="choose-btn create" data-testid="ek-choose-create" @click="$emit('create')">
          <span>Create Room</span>
          <small>Start a new game and invite friends</small>
        </button>
        <button class="choose-btn join" data-testid="ek-choose-join" @click="$emit('join')">
          <span>Join Room</span>
          <small>Enter a room key or pick an active room</small>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { X, Bomb } from '@lucide/vue'
defineEmits(['create', 'join'])
</script>

<style scoped>
.choose-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-overlay);
  background: var(--paper);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

/* The speed-line wedge and warm glow are what stop a flat light ground from
   reading as an empty page. Decorative only, so it sits behind everything. */
.choose-overlay::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 50% 22%, rgba(255, 90, 43, 0.22), rgba(255, 232, 194, 0) 58%),
    repeating-linear-gradient(-38deg, #ffdfae 0 26px, var(--paper) 26px 52px);
  pointer-events: none;
}

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  color: var(--ink);
  width: 44px;
  height: 44px;
  border-radius: 50%;
  box-shadow: 0 3px 0 var(--edge);
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--transition-interactive);
}

.close-btn:hover {
  background: var(--broth);
  color: var(--surface);
}

.close-btn:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

.choose-card {
  position: relative;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 34px 28px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.choose-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 72px;
  height: 72px;
  margin-bottom: 14px;
  background: var(--chili);
  border: var(--edge-w) solid var(--edge);
  border-radius: 24px;
  box-shadow: var(--lift);
  color: var(--ink);
  transform: rotate(-6deg);
}

.choose-card h1 {
  font-size: 2.1rem;
  line-height: 1.05;
  margin-bottom: 8px;
  color: var(--ink);
}

.choose-card p {
  color: var(--muted-cream);
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.5;
  margin-bottom: 22px;
}

.choose-actions {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.choose-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 16px 20px;
  min-height: 72px;
  border-radius: var(--radius-md);
  border: var(--edge-w) solid var(--edge);
  box-shadow: var(--lift);
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
  transition: var(--transition-interactive);
  font-family: inherit;
}

/* Create is the primary path, so it takes the one saturated fill on the
   screen; join stays on paper white. */
.choose-btn.create {
  background: var(--gold);
}

.choose-btn:hover {
  transform: translateY(-2px);
  box-shadow: var(--lift-lg);
}

.choose-btn:active {
  transform: translateY(4px);
  box-shadow: 0 1px 0 var(--edge);
}

.choose-btn span {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.15rem;
}

.choose-btn small {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--muted-cream);
}

.choose-btn.create small {
  color: #6b4a00;
}

@media (max-width: 640px) {
  .choose-card {
    padding: 28px 20px;
    max-width: 100%;
  }

  .choose-card h1 {
    font-size: 1.7rem;
  }

  .choose-btn {
    padding: 14px 16px;
  }
}
</style>
