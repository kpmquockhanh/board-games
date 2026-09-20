<template>
  <div class="choose-overlay">
    <div class="create-card">
      <div class="choose-icon" aria-hidden="true"><Bomb :size="32" /></div>
      <h1>{{ store.roomName || 'Your kitchen is ready' }}</h1>
      <p>Share this room key or link with friends so they can join.</p>
      <div class="room-key-display">
        <span class="room-key" data-testid="ek-room-key">{{ roomKey }}</span>
        <button class="copy-btn" data-testid="ek-copy-key" @click="copyKey" :aria-label="`Copy room key ${roomKey}`">Copy</button>
      </div>
      <div class="share-url">
        <input type="text" :value="shareUrl" readonly @click="copyUrl" aria-label="Share URL (click to copy)">
        <small>Click to copy link</small>
      </div>
      <div class="seat-section">
        <div class="seat-label">Take your seat</div>
        <JoinOverlay
          inline
          icon=""
          title=""
          description=""
          name-placeholder="Your name"
          button-text="Join the kitchen"
          @join="$emit('join', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { toast } from 'vue-sonner'
import { Bomb } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'
import JoinOverlay from '../JoinOverlay.vue'

const store = useEkStore()

const props = defineProps({
  roomKey: { type: String, required: true },
})

defineEmits(['join'])

const shareUrl = computed(() => {
  const base = window.location.origin + window.location.pathname
  return `${base}?room=${props.roomKey}`
})

function copyKey() {
  navigator.clipboard.writeText(props.roomKey)
  toast.success('Room key copied!')
}

function copyUrl() {
  navigator.clipboard.writeText(shareUrl.value)
  toast.success('Link copied!')
}
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
  overflow-y: auto;
}

.choose-overlay::before {
  content: '';
  position: fixed;
  inset: 0;
  background:
    radial-gradient(circle at 50% 18%, rgba(255, 197, 61, 0.42), rgba(255, 232, 194, 0) 60%),
    repeating-linear-gradient(-38deg, #ffdfae 0 26px, var(--paper) 26px 52px);
  pointer-events: none;
}

.create-card {
  position: relative;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 32px 26px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.choose-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 68px;
  height: 68px;
  margin-bottom: 14px;
  background: var(--mint);
  border: var(--edge-w) solid var(--edge);
  border-radius: 22px;
  box-shadow: var(--lift);
  color: var(--ink);
  transform: rotate(6deg);
}

.create-card h1 {
  font-size: 1.9rem;
  line-height: 1.1;
  margin-bottom: 8px;
  color: var(--ink);
}

.create-card p {
  color: var(--muted-cream);
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.5;
  margin-bottom: 20px;
}

.room-key-display {
  display: flex;
  align-items: stretch;
  gap: 10px;
  justify-content: center;
  margin-bottom: 16px;
}

/* The key is the one thing on this screen someone has to read aloud, so it
   gets the loudest treatment: gold fill, ink outline, wide tracking. */
.room-key {
  display: flex;
  align-items: center;
  font-family: 'Baloo 2', monospace;
  font-size: 1.8rem;
  font-weight: 800;
  letter-spacing: 0.15em;
  color: var(--ink);
  background: var(--gold);
  padding: 10px 22px;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
}

.copy-btn {
  padding: 10px 18px;
  min-height: 44px;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  background: var(--surface);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 0.95rem;
  cursor: pointer;
  box-shadow: 0 3px 0 var(--edge);
  transition: var(--transition-interactive);
}

.copy-btn:hover {
  background: var(--sand);
}

.copy-btn:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

.share-url {
  margin-bottom: 18px;
}

.share-url input {
  width: 100%;
  padding: 12px 14px;
  min-height: 44px;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  background: var(--sand);
  color: var(--mild-cream);
  font-size: 0.82rem;
  font-weight: 700;
  text-align: center;
  cursor: pointer;
  outline: none;
  font-family: inherit;
}

.share-url input:focus-visible {
  outline: 3px solid var(--chili);
  outline-offset: 3px;
}

.share-url small {
  display: block;
  margin-top: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--muted-cream);
}

/* Sharing the key and sitting down are two different jobs on one card, so the
   seat picker gets a rule and a label of its own rather than running straight
   on from the share link. */
.seat-section {
  margin-top: 22px;
  padding-top: 22px;
  border-top: 3px dashed rgba(27, 14, 6, 0.3);
}

.seat-label {
  display: inline-block;
  padding: 4px 14px;
  margin-bottom: 14px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--mint);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--ink);
}

@media (max-width: 640px) {
  .create-card {
    padding: 26px 18px;
    max-width: 100%;
  }

  .create-card h1 {
    font-size: 1.55rem;
  }

  .room-key {
    font-size: 1.4rem;
    padding: 8px 16px;
    justify-content: center;
  }

  .room-key-display {
    flex-direction: column;
    gap: 8px;
  }
}
</style>
