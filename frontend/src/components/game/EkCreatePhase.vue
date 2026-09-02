<template>
  <div class="choose-overlay">
    <div class="create-card">
      <div class="choose-icon" aria-hidden="true"><Bomb :size="32" /></div>
      <h1>{{ store.roomName || 'Your kitchen is ready' }}</h1>
      <p>Share this room key or link with friends so they can join.</p>
      <div class="room-key-display">
        <span class="room-key">{{ roomKey }}</span>
        <button class="copy-btn" @click="copyKey" :aria-label="`Copy room key ${roomKey}`">Copy</button>
      </div>
      <div class="share-url">
        <input type="text" :value="shareUrl" readonly @click="copyUrl" aria-label="Share URL (click to copy)">
        <small>Click to copy link</small>
      </div>
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
  background: rgba(21, 15, 12, 0.94);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.create-card {
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

.create-card h1 {
  font-size: 1.7rem;
  margin-bottom: 8px;
}

.create-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.92rem;
  line-height: 1.5;
  margin-bottom: 22px;
}

.room-key-display {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: center;
  margin-bottom: 16px;
}

.room-key {
  font-family: 'Baloo 2', monospace;
  font-size: 1.8rem;
  font-weight: 800;
  letter-spacing: 0.15em;
  color: var(--gold);
  background: rgba(238, 194, 92, 0.1);
  padding: 10px 22px;
  border-radius: 12px;
  border: 1px solid rgba(238, 194, 92, 0.3);
}

.copy-btn {
  padding: 10px 18px;
  border-radius: 10px;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--steam-cream);
  font-weight: 600;
  font-size: 0.85rem;
  cursor: pointer;
  font-family: inherit;
  transition: border-color var(--ease-standard);
}

.copy-btn:hover {
  border-color: var(--chili-orange);
}

.share-url {
  margin-bottom: 18px;
}

.share-url input {
  width: 100%;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--mild-cream);
  font-size: 0.82rem;
  text-align: center;
  cursor: pointer;
  outline: none;
  font-family: inherit;
}

.share-url input:focus {
  border-color: var(--chili-orange);
}

.share-url small {
  display: block;
  margin-top: 6px;
  font-size: 0.72rem;
  color: var(--muted-cream);
}

@media (max-width: 640px) {
  .create-card {
    padding: 28px 20px;
    max-width: 100%;
  }

  .create-card h1 {
    font-size: 1.4rem;
  }

  .room-key {
    font-size: 1.4rem;
    padding: 8px 16px;
  }

  .room-key-display {
    flex-direction: column;
    gap: 8px;
  }
}
</style>
