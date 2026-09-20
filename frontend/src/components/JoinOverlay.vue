<template>
  <div :class="['join-overlay', { inline }]">
    <div class="join-card">
      <div class="join-icon" v-if="icon"><component :is="icon" :size="32" /></div>
      <h1>{{ title }}</h1>
      <p>{{ description }}</p>
      <div v-if="!inline && roomStatus" class="room-status">
        <div class="room-status-row">
          <span class="room-status-badge" :class="roomStatus.status">{{ roomStatus.status }}</span>
          <span class="room-player-count">{{ roomStatus.players.length }}/{{ roomStatus.max_players }} players</span>
        </div>
        <div v-if="roomStatus.players.length > 0" class="room-players">
          <span v-for="player in roomStatus.players" :key="player" class="room-player">
            <span class="room-player-dot" :style="{ background: roomStatus.player_colors[player] }"></span>
            {{ player }}
          </span>
        </div>
      </div>
      <div v-if="!inline && roomError" class="room-error">{{ roomError }}</div>
      <input
        type="text"
        data-testid="join-name-input"
        v-model="name"
        :placeholder="namePlaceholder"
        maxlength="16"
        autocomplete="off"
        @keydown.enter="join"
      >
      <div class="swatches" role="radiogroup" aria-label="Player color">
        <div
          v-for="c in colors"
          :key="c"
          class="swatch"
          :class="{ active: c === chosenColor }"
          :style="{ background: c }"
          role="radio"
          :aria-checked="c === chosenColor"
          :aria-label="`Color ${colorName(c)}`"
          tabindex="0"
          @click="chosenColor = c"
          @keydown.enter="chosenColor = c"
          @keydown.space.prevent="chosenColor = c"
        ></div>
      </div>
      <button class="btn" data-testid="join-submit" :disabled="!name.trim()" @click="join">{{ buttonText }}</button>
      <button v-if="!inline" class="back-link" @click="emit('back')"><ArrowLeft :size="14" /> Back</button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { ArrowLeft } from '@lucide/vue'
import { getRoom } from '../api'

const props = defineProps({
  icon: { type: Object, default: null },
  title: { type: String, default: 'Join the table' },
  description: { type: String, default: '' },
  namePlaceholder: { type: String, default: 'Your name' },
  buttonText: { type: String, default: 'Sit down' },
  inline: { type: Boolean, default: false },
  game: { type: String, default: '' },
  roomKey: { type: String, default: '' },
  initialName: { type: String, default: '' },
  initialColor: { type: String, default: '' },
})

const emit = defineEmits(['join', 'back'])

const COLORS = ['#e2632c','#eec25c','#b7291f','#7fa876','#5c8fa8','#c97ab0','#d9a441','#8a6fb0']
const colors = COLORS
const COLOR_NAMES = {
  '#e2632c': 'orange',
  '#eec25c': 'gold',
  '#b7291f': 'red',
  '#7fa876': 'green',
  '#5c8fa8': 'blue',
  '#c97ab0': 'pink',
  '#d9a441': 'amber',
  '#8a6fb0': 'purple',
}

function colorName(hex) {
  return COLOR_NAMES[hex] || hex
}

const name = ref(props.initialName)
const chosenColor = ref(props.initialColor || COLORS[Math.floor(Math.random() * COLORS.length)])
const roomStatus = ref(null)
const roomError = ref('')
let pollTimer = null

async function fetchRoomStatus() {
  if (!props.game || !props.roomKey) return
  const data = await getRoom(props.game, props.roomKey)
  if (!data || data.error) {
    roomError.value = 'Room not found or no longer active'
    roomStatus.value = null
  } else {
    roomStatus.value = data
    roomError.value = ''
  }
}

function join() {
  if (!name.value.trim()) return
  emit('join', { name: name.value.trim().slice(0, 16), color: chosenColor.value })
}

onMounted(() => {
  if (!props.inline && props.game && props.roomKey) {
    fetchRoomStatus()
    pollTimer = setInterval(fetchRoomStatus, 5000)
  }
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.join-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-overlay);
  background: rgba(21, 15, 12, 0.94);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.join-overlay.inline {
  position: static;
  background: none;
  z-index: auto;
  padding: 0;
}

.join-overlay.inline .join-card {
  max-width: 100%;
  background: none;
  border: none;
  padding: 0;
}

.join-overlay.inline .join-icon,
.join-overlay.inline .join-card h1,
.join-overlay.inline .join-card p {
  display: none;
}

.join-card {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 20px;
  padding: 36px 32px;
  max-width: 380px;
  width: 100%;
  text-align: center;
}

.join-icon {
  font-size: 2rem;
  display: block;
  margin-bottom: 12px;
}

.join-card h1 {
  font-size: 1.7rem;
  margin-bottom: 8px;
}

.join-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.92rem;
  line-height: 1.5;
  margin-bottom: 22px;
}

.join-card input {
  width: 100%;
  padding: 13px 16px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--steam-cream);
  font-size: 1rem;
  outline: none;
  margin-bottom: 14px;
}

.join-card input:focus {
  border-color: var(--chili-orange);
}

.swatches {
  display: flex;
  gap: 8px;
  justify-content: center;
  margin-bottom: 22px;
}

.swatch {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  cursor: pointer;
  border: 2px solid transparent;
  transition: border-color 0.15s ease;
}

.swatch.active {
  border-color: var(--steam-cream);
}

.room-status {
  margin-bottom: 18px;
  padding: 12px 14px;
  border-radius: 12px;
  background: rgba(28, 21, 18, 0.6);
  border: 1px solid var(--line);
}

.room-status-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-bottom: 8px;
}

.room-status-badge {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: 100px;
}

.room-status-badge.active {
  color: #7fa876;
  background: rgba(127, 168, 118, 0.15);
}

.room-player-count {
  font-size: 0.82rem;
  color: var(--mild-cream);
  opacity: 0.8;
}

.room-players {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: center;
}

.room-player {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 0.78rem;
  color: var(--mild-cream);
  opacity: 0.85;
}

.room-player-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.room-error {
  color: var(--broth-red);
  font-size: 0.85rem;
  margin-bottom: 14px;
  text-align: center;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 14px;
  background: none;
  border: none;
  color: var(--mild-cream);
  opacity: 0.6;
  font-size: 0.85rem;
  cursor: pointer;
  font-family: inherit;
}

.back-link:hover {
  opacity: 1;
}

/* ─── Kitchen theme ───
   This component is shared with Hotpot, so the rules above stay as the dark
   default and the cartoon look is layered on only while EkView has
   `theme-kitchen` on <html>. */

/* The other Exploding Kitchen entry screens sit on striped paper, so the join
   form matches rather than landing on a flat scrim on its own. */
html.theme-kitchen .join-overlay {
  background: var(--paper);
}

html.theme-kitchen .join-overlay:not(.inline)::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 50% 22%, rgba(255, 90, 43, 0.22), rgba(255, 232, 194, 0) 58%),
    repeating-linear-gradient(-38deg, #ffdfae 0 26px, var(--paper) 26px 52px);
  pointer-events: none;
}

/* The themed scrim above out-specifies the shared `.inline` reset, so the
   inline variant has to opt out again here — otherwise the overlay's backdrop
   paints as a grey block inside whatever card is hosting the form. */
html.theme-kitchen .join-overlay.inline {
  background: none;
}

html.theme-kitchen .join-card {
  position: relative;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 32px 26px;
}

/* Inline, this renders inside another card, so it drops its own shell. */
html.theme-kitchen .join-overlay.inline .join-card {
  background: none;
  border: none;
  box-shadow: none;
  padding: 0;
}

html.theme-kitchen .join-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 68px;
  height: 68px;
  margin-bottom: 14px;
  background: var(--chili);
  border: var(--edge-w) solid var(--edge);
  border-radius: 22px;
  box-shadow: var(--lift);
  color: var(--ink);
  transform: rotate(-6deg);
}

html.theme-kitchen .join-card h1 {
  color: var(--ink);
}

html.theme-kitchen .join-card p {
  color: var(--muted-cream);
  opacity: 1;
  font-weight: 700;
}

html.theme-kitchen .join-card input {
  min-height: 52px;
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  background: var(--sand);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
}

html.theme-kitchen .join-card input::placeholder {
  color: var(--dim-text);
  font-weight: 700;
}

html.theme-kitchen .join-card input:focus-visible {
  outline: 3px solid var(--chili);
  outline-offset: 3px;
}

/* Eight swatches at a 44px target are wider than the card, so they lay out as
   two rows of four rather than shrinking below the minimum tap size. */
html.theme-kitchen .swatches {
  display: grid;
  grid-template-columns: repeat(4, 44px);
  justify-content: center;
  gap: 10px;
  margin-bottom: 20px;
}

html.theme-kitchen .swatch {
  width: 44px;
  height: 44px;
  border: 3px solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  transition: var(--transition-interactive);
}

/* Selected is an inset cream ring — a colour change would fight the swatch's
   own colour, which is the whole point of the control. */
html.theme-kitchen .swatch.active {
  box-shadow: 0 3px 0 var(--edge), inset 0 0 0 5px var(--surface);
}

html.theme-kitchen .room-status {
  background: var(--sand);
  border: 2.5px solid var(--edge);
  border-radius: var(--radius-md);
}

html.theme-kitchen .room-status-badge {
  border: 2px solid var(--edge);
  font-weight: 800;
  color: var(--ink);
  background: var(--dim-fill);
}

html.theme-kitchen .room-status-badge.active {
  color: var(--ink);
  background: var(--mint);
}

html.theme-kitchen .room-player-count,
html.theme-kitchen .room-player {
  color: var(--mild-cream);
  opacity: 1;
  font-weight: 700;
}

html.theme-kitchen .room-player-dot {
  width: 10px;
  height: 10px;
  border: 2px solid var(--edge);
}

html.theme-kitchen .room-error {
  color: var(--broth);
  font-weight: 800;
}

html.theme-kitchen .back-link {
  min-height: 44px;
  opacity: 1;
  color: var(--muted-cream);
  font-weight: 800;
}

html.theme-kitchen .back-link:hover {
  color: var(--ink);
}
</style>
