<template>
  <div class="lobby">
    <div class="lobby-head">
      <h1 class="lobby-title">
        {{ store.roomName || 'Waiting for chefs...' }}
      </h1>
      <p class="lobby-sub">Share this link with friends to get everyone in.</p>
    </div>

    <!-- One column on phones, roster beside the room card from 900px up. The
         markup order is unchanged, so the phone reading order is untouched. -->
    <div class="lobby-grid">
      <section class="lobby-col roster">
        <h2 class="col-label">Chefs at the table</h2>
        <div class="lobby-players">
          <div
            v-for="(name, idx) in store.playerList"
            :key="name"
            class="lobby-player"
            :data-testid="'lobby-player-' + name"
            :class="{ you: store.me && name === store.me.name, host: idx === 0, ready: store.lobby.players[name]?.ready }"
          >
            <span class="dot" :style="{ background: store.lobby.players[name].color }" :aria-label="`Player color: ${store.lobby.players[name].color}`"></span>
            <span class="player-name">{{ name }}{{ store.me && name === store.me.name ? ' (you)' : '' }}</span>
            <span v-if="idx === 0" class="host-badge">host</span>
            <span v-if="store.lobby.players[name]?.ready" class="ready-badge">ready</span>
            <span v-else class="waiting-badge">waiting</span>
          </div>
        </div>
        <p class="lobby-empty" v-if="!store.canStart">
          Waiting for more players... ({{ store.playerList.length }}/{{ store.roomSettings.minPlayers }} min)
        </p>
        <button
          v-if="store.canStart"
          class="start-btn"
          data-testid="lobby-ready-btn"
          :class="{ 'ready-btn': store.lobby.players[store.me?.name]?.ready }"
          @click="store.toggleReady()"
        >{{ store.lobby.players[store.me?.name]?.ready ? 'Not ready' : 'I\'m ready' }}</button>
        <p class="lobby-status" v-if="store.canStart && !store.allReady">
          Waiting for {{ store.playerList.filter(n => !store.lobby.players[n]?.ready).length }} more player(s) to be ready...
        </p>
        <p class="lobby-status all-ready" v-if="store.allReady" aria-live="polite">
          All players ready! Starting...
        </p>
      </section>

      <section class="lobby-col side">
        <RoomSettings
          v-if="isHost"
          v-model="hostSettings"
          :editable="true"
        />
        <RoomSettings
          v-else
          :model-value="store.roomSettings"
          :editable="false"
        />
        <p class="lobby-link">Share this link: {{ currentUrl }}</p>
        <div class="lobby-actions">
          <button class="action-btn" data-testid="lobby-leave" @click="$emit('leave')">Leave</button>
          <button v-if="isHost" class="action-btn danger" data-testid="lobby-delete" @click="$emit('delete')">Delete room</button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useEkStore } from '../../stores/explodingKitchen'
import RoomSettings from '../RoomSettings.vue'

const store = useEkStore()

defineEmits(['leave', 'delete'])

const isHost = computed(() => {
  const names = store.playerList
  return names.length > 0 && store.me && names[0] === store.me.name
})

const hostSettings = ref(null)
let settingsTimer = null

watch(() => store.roomSettings, (val) => {
  if (!hostSettings.value) {
    hostSettings.value = JSON.parse(JSON.stringify(val))
  }
}, { immediate: true })

watch(hostSettings, (val) => {
  if (val && isHost.value) {
    if (settingsTimer) clearTimeout(settingsTimer)
    settingsTimer = setTimeout(() => {
      store.updateSettings(JSON.parse(JSON.stringify(val)))
    }, 600)
  }
}, { deep: true })

const currentUrl = computed(() => window.location.href)
</script>

<style scoped>
.lobby {
  max-width: 1100px;
  margin: 0 auto;
  padding: 28px 20px 40px;
  text-align: center;
}

.lobby-head {
  margin-bottom: 24px;
}

.lobby-title {
  font-size: 2.1rem;
  line-height: 1.05;
  margin-bottom: 6px;
  color: var(--ink);
}

.lobby-sub {
  color: var(--muted-cream);
  font-size: 0.95rem;
  font-weight: 700;
}

.lobby-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 18px;
  text-align: left;
}

.lobby-col {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.col-label {
  align-self: flex-start;
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
}

.lobby-players {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.lobby-player {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  min-height: 56px;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  box-shadow: 0 3px 0 var(--edge);
  font-size: 0.95rem;
  font-weight: 700;
  color: var(--ink);
}

/* Names are user-supplied, so the row has to be able to truncate rather than
   let the badges spill out of the card. */
.player-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lobby-player .dot {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  border: 2.5px solid var(--edge);
  flex-shrink: 0;
}

.lobby-player.you {
  background: #fff6dd;
  box-shadow: 0 3px 0 var(--edge), inset 0 0 0 3px var(--gold);
}

.lobby-player.ready {
  border-color: var(--edge);
}

.host-badge,
.ready-badge,
.waiting-badge {
  flex-shrink: 0;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.66rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 3px 9px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  color: var(--ink);
}

.host-badge {
  background: var(--gold);
}

.ready-badge {
  background: var(--mint);
}

.waiting-badge {
  background: var(--dim-fill);
  border-color: var(--dim-edge);
  color: var(--dim-text);
}

.lobby-status {
  color: var(--muted-cream);
  font-size: 0.88rem;
  font-weight: 700;
}

.lobby-status.all-ready {
  color: #157a53;
  font-weight: 800;
}

.lobby-empty {
  color: var(--muted-cream);
  font-weight: 700;
  font-size: 0.92rem;
}

.start-btn {
  padding: 14px 32px;
  min-height: 56px;
  border-radius: var(--radius-md);
  border: var(--edge-w) solid var(--edge);
  box-shadow: var(--lift);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 1.15rem;
  background: var(--chili);
  color: var(--ink);
  transition: var(--transition-interactive);
}

.start-btn:hover {
  transform: translateY(-2px);
  box-shadow: var(--lift-lg);
}

.start-btn:active {
  transform: translateY(4px);
  box-shadow: 0 1px 0 var(--edge);
}

/* Already-ready is a toggle-off, so it stops shouting: mint fill, same
   outline, no change in size. */
.ready-btn {
  background: var(--mint);
}

.lobby-link {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--muted-cream);
  word-break: break-all;
}

.lobby-actions {
  display: flex;
  gap: 10px;
}

.action-btn {
  flex: 1;
  padding: 10px 22px;
  min-height: 48px;
  border-radius: var(--radius-sm);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 0.95rem;
  border: var(--edge-w) solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  background: var(--surface);
  color: var(--ink);
  transition: var(--transition-interactive);
}

.action-btn:hover {
  background: var(--sand);
}

.action-btn:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

.action-btn.danger {
  background: var(--broth);
  color: var(--surface);
}

.action-btn.danger:hover {
  background: #b81f1f;
}

@media (min-width: 900px) {
  .lobby-grid {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: 24px;
    align-items: start;
  }
}

@media (max-width: 640px) {
  .lobby {
    padding: 20px 12px 32px;
  }

  .lobby-title {
    font-size: 1.6rem;
  }

  .lobby-actions {
    flex-direction: column;
  }
}
</style>
