<template>
  <div class="choose-overlay">
    <div class="choose-card">
      <button class="close-btn" @click="$router.push('/')" aria-label="Back to hub"><X :size="16" /></button>
      <div class="choose-icon" aria-hidden="true"><Bomb :size="32" /></div>
      <h1>Exploding Kitchen</h1>
      <p>A card game of cooking chaos. Draw cards, avoid the exploding kitchen, and be the last chef standing.</p>
      <div class="choose-actions">
        <button class="choose-btn create" @click="$emit('create')">
          <span>Create Room</span>
          <small>Start a new game and invite friends</small>
        </button>
        <button class="choose-btn join" @click="$emit('join')">
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
  background: rgba(21, 15, 12, 0.94);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: none;
  border: 1px solid var(--line);
  color: var(--mild-cream);
  width: 30px;
  height: 30px;
  border-radius: 50%;
  font-size: 0.85rem;
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color var(--ease-standard), color var(--ease-standard);
}

.close-btn:hover {
  border-color: var(--broth-red);
  color: var(--broth-red);
}

.choose-card {
  position: relative;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 36px 32px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.choose-icon {
  font-size: 2rem;
  display: block;
  margin-bottom: 12px;
}

.choose-card h1 {
  font-size: 1.7rem;
  margin-bottom: 8px;
}

.choose-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.92rem;
  line-height: 1.5;
  margin-bottom: 22px;
}

.choose-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.choose-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 18px 20px;
  border-radius: 14px;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--steam-cream);
  cursor: pointer;
  transition: all var(--ease-standard);
  font-family: inherit;
}

.choose-btn:hover {
  border-color: var(--chili-orange);
  transform: translateY(-2px);
}

.choose-btn span {
  font-weight: 700;
  font-size: 1rem;
}

.choose-btn small {
  font-size: 0.78rem;
  color: var(--muted-cream);
}

@media (max-width: 640px) {
  .choose-card {
    padding: 28px 20px;
    max-width: 100%;
  }

  .choose-card h1 {
    font-size: 1.4rem;
  }

  .choose-btn {
    padding: 14px 16px;
  }
}
</style>
