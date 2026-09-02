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
      <button class="btn" :disabled="!name.trim()" @click="join">{{ buttonText }}</button>
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
  background: #1c1512;
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

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 13px 28px;
  border-radius: 100px;
  font-weight: 600;
  font-size: 0.95rem;
  border: none;
  cursor: pointer;
  width: 100%;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
  transition: transform 0.15s ease;
}

.btn:hover {
  transform: translateY(-2px);
}

.btn:disabled {
  opacity: 0.5;
  cursor: default;
  transform: none;
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
</style>
