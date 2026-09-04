<template>
  <div class="kitchen-log" :class="{ expanded: showLog }">
    <div class="log-header">
      <h3>KITCHEN LOG</h3>
      <button class="log-menu-btn" aria-label="Toggle log" @click="showLog = !showLog">
        <X v-if="showLog" :size="14" />
        <Menu v-else :size="14" />
      </button>
    </div>
    <template v-if="showLog">
      <div class="log-entries">
        <div v-for="(e, i) in store.recentLog" :key="i" class="log-entry">
          <span class="log-emoji">{{ getEmoji(e.text) }}</span>
          <div class="log-text">
            <span v-if="e.name && e.name !== 'system'" class="log-player">{{ e.name }}</span>
            <span class="log-action">{{ stripTrailingEmoji(e.text) }}</span>
          </div>
        </div>
        <div v-if="store.recentLog.length === 0" class="log-empty">
          Waiting for chefs to arrive...
        </div>
      </div>
      <div class="log-chat">
        <input
          type="text"
          class="chat-input"
          placeholder="Chat with chefs..."
          readonly
        />
        <button class="chat-send" aria-label="Send message">
          <Send :size="14" />
        </button>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Menu, Send, X } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const showLog = ref(false)

function getEmoji(text) {
  if (!text) return '•'
  const t = text.toLowerCase()
  if (t.includes('attack')) return '⚔️'
  if (t.includes('defuse') || t.includes('defense')) return '🛡️'
  if (t.includes('skip')) return '⏭️'
  if (t.includes('nope')) return '🚫'
  if (t.includes('shuffle')) return '🔀'
  if (t.includes('favor')) return '🎁'
  if (t.includes('exploded') || t.includes('explosive') || t.includes('catomic')) return '💥'
  if (t.includes('drew') || t.includes('draw')) return '🃏'
  if (t.includes('started')) return '🍳'
  if (t.includes('joined')) return '👋'
  if (t.includes('eliminated')) return '💀'
  if (t.includes('future') || t.includes('peek') || t.includes('mark')) return '🔮'
  if (t.includes('garbage')) return '🗑️'
  if (t.includes('clone') || t.includes('cloned')) return '👻'
  if (t.includes('rainbow')) return '🌈'
  if (t.includes('bury') || t.includes('buried')) return '⚰️'
  if (t.includes('swap')) return '🔃'
  if (t.includes('dig')) return '⛏️'
  if (t.includes('wins') || t.includes('won')) return '🏆'
  return '•'
}

function stripTrailingEmoji(text) {
  if (!text) return ''
  return text.replace(/\s*[\u{1F300}-\u{1F9FF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}\u{FE00}-\u{FE0F}\u{1F000}-\u{1FAFF}\u{200D}\u{20E3}\u{E0020}-\u{E007F}]+$/u, '').trim()
}
</script>

<style scoped>
.kitchen-log {
  position: absolute;
  top: 12px;
  left: 12px;
  width: 240px;
  background: rgba(28, 21, 18, 0.92);
  backdrop-filter: blur(12px);
  border: 1px solid var(--line);
  border-radius: 16px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  z-index: 50;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

/* ─── Header ─── */

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 8px;
  background: rgba(110, 180, 140, 0.12);
  border-bottom: 1px solid rgba(110, 180, 140, 0.2);
}

.log-header h3 {
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: #7ec99a;
  margin: 0;
}

.log-menu-btn {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: none;
  background: rgba(255, 255, 255, 0.06);
  color: var(--muted-cream);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--ease-standard);
}

.log-menu-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--steam-cream);
}

/* ─── Entries ─── */

.log-entries {
  flex: 1;
  overflow-y: auto;
  padding: 8px 12px;
  max-height: 240px;
  min-height: 120px;
}

.log-entry {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 5px 0;
  font-size: 0.78rem;
  line-height: 1.4;
  color: var(--mild-cream);
  border-bottom: 1px solid rgba(253, 243, 228, 0.06);
}

.log-entry:last-child {
  border-bottom: none;
}

.log-emoji {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.85rem;
  background: rgba(255, 255, 255, 0.06);
  border-radius: 6px;
  margin-top: 1px;
}

.log-text {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.log-player {
  font-weight: 700;
  color: var(--steam-cream);
  white-space: nowrap;
}

.log-action {
  color: var(--mild-cream);
  opacity: 0.85;
  word-break: break-word;
}

.log-empty {
  font-size: 0.78rem;
  color: var(--muted-cream);
  text-align: center;
  padding: 20px 0;
}

/* ─── Chat Input ─── */

.log-chat {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-top: 1px solid var(--line);
  background: rgba(0, 0, 0, 0.15);
}

.chat-input {
  flex: 1;
  height: 32px;
  padding: 0 10px;
  border-radius: 8px;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.04);
  color: var(--mild-cream);
  font-size: 0.78rem;
  font-family: inherit;
  outline: none;
  transition: border-color var(--ease-standard);
  min-width: 1px;
}

.chat-input::placeholder {
  color: var(--muted-cream);
  opacity: 0.7;
}

.chat-input:focus {
  border-color: rgba(110, 180, 140, 0.4);
}

.chat-send {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  border: none;
  background: rgba(110, 180, 140, 0.2);
  color: #7ec99a;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--ease-standard);
  flex-shrink: 0;
}

.chat-send:hover {
  background: rgba(110, 180, 140, 0.35);
}

/* ─── Scrollbar ─── */

.log-entries::-webkit-scrollbar {
  width: 4px;
}

.log-entries::-webkit-scrollbar-track {
  background: transparent;
}

.log-entries::-webkit-scrollbar-thumb {
  background: var(--line);
  border-radius: 4px;
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  .kitchen-log {
    width: 26px;
    height: 26px;
    top: 8px;
    left: 8px;
    border-radius: 6px;
    padding: 0;
  }

  .kitchen-log:not(.expanded) .log-header {
    border-bottom: none;
    padding: 0;
  }

  .kitchen-log:not(.expanded) .log-header h3 {
    display: none;
  }

  .kitchen-log.expanded {
    width: calc(100vw - 16px);
    height: auto;
    max-height: 50vh;
    top: 8px;
    left: 8px;
    right: 8px;
    border-radius: 10px;
  }

  .log-header {
    /* padding: 0; */
    min-height: 26px;
  }

  .log-header h3 {
    font-size: 0.6rem;
  }

  .log-menu-btn {
    width: 26px;
    height: 26px;
    border-radius: 6px;
  }

  .log-entries {
    max-height: 220px;
    min-height: 0;
    padding: 4px 8px;
    overflow-y: auto;
  }

  .log-entry {
    font-size: 0.6rem;
    padding: 2px 0;
    gap: 5px;
  }

  .log-emoji {
    width: 18px;
    height: 18px;
    font-size: 0.65rem;
    border-radius: 4px;
  }

  .log-chat {
    padding: 4px 6px;
  }

  .chat-input {
    font-size: 0.65rem;
    height: 28px;
  }

  .chat-send {
    width: 28px;
    height: 28px;
  }
}
</style>
