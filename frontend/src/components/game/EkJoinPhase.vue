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
        class="room-key-input"
        @keydown.enter="$emit('submitKey')"
        aria-label="Room key"
      >
      <button class="btn" :disabled="joinKey.length < 4" @click="$emit('submitKey')">Continue</button>

      <div v-if="loadingRooms" class="rooms-loading">Loading rooms...</div>
      <div v-else-if="rooms.length > 0" class="rooms-section">
        <div class="rooms-label">Active rooms</div>
        <div class="rooms-list">
          <div
            v-for="room in rooms"
            :key="room.room_key"
            class="room-item"
          >
            <button class="room-item-join" @click="$emit('pickRoom', room.room_key)">
              <span class="room-item-name">{{ room.name || room.room_key }}</span>
              <span class="room-item-count">{{ room.player_count }}/{{ room.max_players }} players</span>
            </button>
            <button class="room-item-delete" @click.stop="$emit('deleteRoom', room.room_key)" aria-label="Delete room"><X :size="14" /></button>
          </div>
        </div>
      </div>
      <div v-else class="rooms-empty">No active rooms yet — create one!</div>

      <button class="back-link" @click="$emit('back')"><ArrowLeft :size="14" /> Back</button>
    </div>
  </div>
</template>

<script setup>
import { X, Bomb, ArrowLeft } from '@lucide/vue'
defineProps({
  joinKey: { type: String, default: '' },
  rooms: { type: Array, default: () => [] },
  loadingRooms: { type: Boolean, default: false },
})

defineEmits(['update:joinKey', 'submitKey', 'pickRoom', 'deleteRoom', 'back'])
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

.room-key-input {
  width: 100%;
  padding: 14px 16px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--steam-cream);
  font-size: 1.1rem;
  text-align: center;
  letter-spacing: 0.12em;
  font-weight: 600;
  outline: none;
  margin-bottom: 14px;
  font-family: inherit;
}

.room-key-input:focus {
  border-color: var(--chili-orange);
}

.rooms-loading {
  margin-top: 16px;
  font-size: 0.85rem;
  color: var(--muted-cream);
  text-align: center;
}

.rooms-section {
  margin-top: 18px;
  text-align: left;
}

.rooms-label {
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--gold);
  margin-bottom: 10px;
  text-align: center;
}

.rooms-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 180px;
  overflow-y: auto;
}

.room-item {
  display: flex;
  align-items: center;
  gap: 6px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--steam-cream);
  transition: all var(--ease-standard);
  font-family: inherit;
}

.room-item:hover {
  border-color: var(--chili-orange);
}

.room-item-join {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: none;
  border: none;
  color: var(--steam-cream);
  cursor: pointer;
  font-family: inherit;
}

.room-item-delete {
  padding: 8px 12px;
  background: none;
  border: none;
  color: var(--mild-cream);
  opacity: 0.4;
  cursor: pointer;
  font-size: 0.9rem;
  font-family: inherit;
  transition: all var(--ease-standard);
}

.room-item-delete:hover {
  opacity: 1;
  color: var(--broth-red);
}

.room-item-key {
  font-family: 'Baloo 2', monospace;
  font-weight: 700;
  font-size: 1rem;
  letter-spacing: 0.1em;
}

.room-item-name {
  font-weight: 600;
  font-size: 0.92rem;
  color: var(--steam-cream);
}

.room-item-count {
  font-size: 0.78rem;
  color: var(--muted-cream);
}

.rooms-empty {
  margin-top: 16px;
  font-size: 0.85rem;
  color: var(--muted-cream);
  font-style: italic;
}

@media (max-width: 640px) {
  .choose-card {
    padding: 28px 20px;
    max-width: 100%;
  }

  .choose-card h1 {
    font-size: 1.4rem;
  }

  .room-key-input {
    font-size: 1rem;
    padding: 12px 14px;
  }

  .rooms-list {
    max-height: 140px;
  }
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
