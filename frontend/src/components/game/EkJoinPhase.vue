<template>
  <div class="choose-overlay">
    <div class="choose-card">
      <div class="choose-icon" aria-hidden="true"><Bomb :size="32" /></div>
      <h1>Join a kitchen</h1>
      <p>Enter the room key shared by the host, or pick an active room below.</p>
      <input
        type="text"
        :value="joinKey"
        @input="$emit('update:joinKey', $event.target.value)"
        placeholder="Room key"
        maxlength="6"
        class="room-key-input" data-testid="ek-join-key-input"
        @keydown.enter="$emit('submitKey')"
        aria-label="Room key"
      >
      <button class="btn" data-testid="ek-join-continue" :disabled="joinKey.length < 4" @click="$emit('submitKey')">Continue</button>

      <div v-if="loadingRooms" class="rooms-loading">Loading rooms...</div>
      <div v-else-if="rooms.length > 0" class="rooms-section">
        <div class="rooms-label">Active rooms</div>
        <div class="rooms-list">
          <div
            v-for="room in rooms"
            :key="room.room_key"
            class="room-item"
          >
            <button class="room-item-join" :data-testid="'ek-room-item-' + room.room_key" @click="$emit('pickRoom', room.room_key)">
              <span class="room-item-name">{{ room.name || room.room_key }}</span>
              <span class="room-item-count">{{ room.player_count }}/{{ room.max_players }} players</span>
            </button>
          </div>
        </div>
      </div>
      <div v-else class="rooms-empty">No active rooms yet — create one!</div>

      <button class="back-link" @click="$emit('back')"><ArrowLeft :size="14" /> Back</button>
    </div>
  </div>
</template>

<script setup>
import { Bomb, ArrowLeft } from '@lucide/vue'
defineProps({
  joinKey: { type: String, default: '' },
  rooms: { type: Array, default: () => [] },
  loadingRooms: { type: Boolean, default: false },
})

defineEmits(['update:joinKey', 'submitKey', 'pickRoom', 'back'])
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
    radial-gradient(circle at 50% 20%, rgba(62, 123, 250, 0.2), rgba(255, 232, 194, 0) 58%),
    repeating-linear-gradient(-38deg, #ffdfae 0 26px, var(--paper) 26px 52px);
  pointer-events: none;
}

.choose-card {
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
  background: var(--blueberry);
  border: var(--edge-w) solid var(--edge);
  border-radius: 22px;
  box-shadow: var(--lift);
  color: var(--surface);
  transform: rotate(-5deg);
}

.choose-card h1 {
  font-size: 1.9rem;
  line-height: 1.1;
  margin-bottom: 8px;
  color: var(--ink);
}

.choose-card p {
  color: var(--muted-cream);
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.5;
  margin-bottom: 20px;
}

.room-key-input {
  width: 100%;
  padding: 14px 16px;
  min-height: 52px;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  background: var(--sand);
  color: var(--ink);
  font-size: 1.2rem;
  text-align: center;
  letter-spacing: 0.12em;
  font-weight: 800;
  outline: none;
  margin-bottom: 14px;
  font-family: inherit;
  text-transform: uppercase;
}

.room-key-input::placeholder {
  color: var(--muted-cream);
  text-transform: none;
  letter-spacing: normal;
}

.room-key-input:focus-visible {
  outline: 3px solid var(--chili);
  outline-offset: 3px;
}

.rooms-loading {
  margin-top: 16px;
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--muted-cream);
  text-align: center;
}

.rooms-section {
  margin-top: 20px;
  text-align: left;
}

.rooms-label {
  display: inline-block;
  padding: 4px 14px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--gold);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--ink);
  margin: 0 auto 12px;
}

.rooms-section {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.rooms-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-height: 200px;
  overflow-y: auto;
  /* The hard shadows would be clipped flush against the scroll edge without
     a little room to sit in. */
  padding: 2px 2px 6px;
}

.room-item {
  display: flex;
  align-items: center;
  gap: 6px;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  background: var(--surface);
  box-shadow: 0 3px 0 var(--edge);
  color: var(--ink);
  transition: var(--transition-interactive);
  font-family: inherit;
}

.room-item:hover {
  background: var(--sand);
}

.room-item:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

.room-item-join {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 16px;
  min-height: 52px;
  background: none;
  border: none;
  color: var(--ink);
  cursor: pointer;
  font-family: inherit;
}

.room-item-key {
  font-family: 'Baloo 2', monospace;
  font-weight: 800;
  font-size: 1rem;
  letter-spacing: 0.1em;
}

.room-item-name {
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1rem;
  color: var(--ink);
  /* Room names are free text, so they have to be allowed to truncate rather
     than push the player count off the row. */
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.room-item-count {
  flex-shrink: 0;
  padding: 3px 10px;
  border-radius: 100px;
  background: var(--sand);
  border: 2px solid var(--edge);
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--mild-cream);
  white-space: nowrap;
}

.rooms-empty {
  margin-top: 16px;
  font-size: 0.88rem;
  font-weight: 700;
  color: var(--muted-cream);
}

.back-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin-top: 12px;
  background: none;
  border: none;
  font-size: 0.9rem;
  cursor: pointer;
  font-family: inherit;
}

@media (max-width: 640px) {
  .choose-card {
    padding: 26px 18px;
    max-width: 100%;
  }

  .choose-card h1 {
    font-size: 1.55rem;
  }

  .room-key-input {
    font-size: 1.05rem;
    padding: 12px 14px;
  }

  .rooms-list {
    max-height: 160px;
  }
}
</style>
