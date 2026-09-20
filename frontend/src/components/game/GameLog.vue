<template>
  <div class="kitchen-log" data-testid="game-log" :class="{ expanded: showLog }">
    <div class="log-header">
      <h3>KITCHEN LOG</h3>
      <button class="log-menu-btn" data-testid="game-log-toggle" aria-label="Toggle log" @click="showLog = !showLog">
        <X v-if="showLog" :size="18" />
        <Scroll v-else :size="18" />
      </button>
    </div>
    <template v-if="showLog">
      <div class="log-entries">
        <div v-for="(e, i) in store.recentLog" :key="i" class="log-entry" :data-testid="'log-entry-' + i">
          <span class="log-icon" :class="`icon-${getIcon(e.text).tone}`" aria-hidden="true">
            <component :is="getIcon(e.text).icon" :size="13" />
          </span>
          <div class="log-text">
            <span v-if="e.name && e.name !== 'system'" class="log-player">{{ e.name }}</span>
            <span class="log-action">{{ stripTrailingEmoji(e.text) }}</span>
          </div>
        </div>
        <div v-if="store.recentLog.length === 0" class="log-empty">
          Waiting for chefs to arrive...
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import {
  Scroll, X, Swords, Shield, FastForward, Ban, Shuffle, Hand, Bomb, Layers,
  ChefHat, UserPlus, Skull, Eye, Trash2, Copy, Sparkles, ArrowDownToLine,
  ArrowUpDown, Pickaxe, Trophy, Circle,
} from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()
const showLog = ref(false)

// The log used to label every entry with an emoji, which renders differently
// on every platform and reads as decoration rather than as a legend. These are
// the same stroke icons the cards use, toned to the eight card families so a
// log line and the card that caused it look related.
const LOG_ICONS = [
  ['attack', Swords, 'attack'],
  ['defuse', Shield, 'defend'],
  ['defense', Shield, 'defend'],
  ['skip', FastForward, 'evade'],
  ['nope', Ban, 'nope'],
  ['shuffle', Shuffle, 'chaos'],
  ['favor', Hand, 'steal'],
  ['exploded', Bomb, 'attack'],
  ['explosive', Bomb, 'attack'],
  ['catomic', Bomb, 'attack'],
  ['drew', Layers, 'neutral'],
  ['draw', Layers, 'neutral'],
  ['started', ChefHat, 'neutral'],
  ['joined', UserPlus, 'neutral'],
  ['eliminated', Skull, 'attack'],
  ['future', Eye, 'peek'],
  ['peek', Eye, 'peek'],
  ['mark', Eye, 'peek'],
  ['garbage', Trash2, 'chaos'],
  ['clone', Copy, 'steal'],
  ['rainbow', Sparkles, 'cat'],
  ['bury', ArrowDownToLine, 'chaos'],
  ['swap', ArrowUpDown, 'chaos'],
  ['dig', Pickaxe, 'peek'],
  ['wins', Trophy, 'steal'],
  ['won', Trophy, 'steal'],
]

const DEFAULT_ICON = { icon: Circle, tone: 'neutral' }

function getIcon(text) {
  if (!text) return DEFAULT_ICON
  const t = text.toLowerCase()
  const hit = LOG_ICONS.find(([needle]) => t.includes(needle))
  return hit ? { icon: hit[1], tone: hit[2] } : DEFAULT_ICON
}

// Log text comes from the server and still carries trailing emoji, so they are
// stripped here rather than rendered next to the icon that replaced them.
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
  width: 244px;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  box-shadow: var(--lift);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  z-index: 50;
}

/* ─── Header ─── */

.log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding: 7px 8px 7px 12px;
  background: var(--mint);
  border-bottom: var(--edge-w) solid var(--edge);
}

.log-header h3 {
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: var(--ink);
  margin: 0;
}

.log-menu-btn {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border-radius: 10px;
  border: 2.5px solid var(--edge);
  background: var(--surface);
  color: var(--ink);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: var(--transition-interactive);
}

.log-menu-btn:hover {
  background: var(--sand);
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
  padding: 6px 0;
  font-size: 0.78rem;
  font-weight: 700;
  line-height: 1.4;
  color: var(--mild-cream);
  border-bottom: 2px solid var(--line);
}

.log-entry:last-child {
  border-bottom: none;
}

.log-icon {
  flex-shrink: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--edge);
  border-radius: 8px;
  margin-top: 1px;
  color: var(--surface);
}

.icon-attack { background: var(--family-attack); }
.icon-defend { background: var(--family-defend); color: var(--ink); }
.icon-evade { background: var(--family-evade); }
.icon-peek { background: var(--family-peek); }
.icon-steal { background: var(--family-steal); color: var(--ink); }
.icon-chaos { background: var(--family-chaos); color: var(--ink); }
.icon-nope { background: var(--family-nope); }
.icon-cat { background: var(--family-cat); }
.icon-neutral { background: var(--sand); color: var(--ink); }

.log-text {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  min-width: 0;
}

.log-player {
  font-weight: 800;
  color: var(--ink);
  white-space: nowrap;
}

.log-action {
  color: var(--muted-cream);
  word-break: break-word;
}

.log-empty {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--muted-cream);
  text-align: center;
  padding: 20px 0;
}

/* ─── Scrollbar ─── */

.log-entries::-webkit-scrollbar {
  width: 5px;
}

.log-entries::-webkit-scrollbar-track {
  background: transparent;
}

.log-entries::-webkit-scrollbar-thumb {
  background: var(--dim-edge);
  border-radius: 4px;
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  /* Collapsed, the log is just its toggle — which still has to be a 44px
     target, so the shell is sized to the button rather than to the label. */
  .kitchen-log {
    width: 44px;
    height: 44px;
    top: 8px;
    left: 8px;
    border-radius: 12px;
  }

  .kitchen-log:not(.expanded) .log-header {
    border-bottom: none;
    padding: 0;
    background: var(--surface);
    height: 100%;
    justify-content: center;
  }

  .kitchen-log:not(.expanded) .log-header h3 {
    display: none;
  }

  .kitchen-log:not(.expanded) .log-menu-btn {
    width: 100%;
    height: 100%;
    border: none;
    border-radius: 0;
  }

  .kitchen-log.expanded {
    width: calc(100vw - 16px);
    height: auto;
    max-height: 50vh;
    top: 8px;
    left: 8px;
    right: 8px;
    border-radius: var(--radius-sm);
  }

  .log-header {
    min-height: 44px;
  }

  .log-header h3 {
    font-size: 0.68rem;
  }

  .log-entries {
    max-height: 220px;
    min-height: 0;
    padding: 4px 8px;
    overflow-y: auto;
  }

  .log-entry {
    font-size: 0.68rem;
    padding: 4px 0;
    gap: 6px;
  }

  .log-icon {
    width: 20px;
    height: 20px;
    border-radius: 6px;
  }
}
</style>
