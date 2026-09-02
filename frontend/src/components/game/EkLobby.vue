<template>
  <div class="lobby">
    <h1 class="lobby-title">
      {{ store.roomName || 'Waiting for chefs...' }}
    </h1>
    <p class="lobby-sub">Share this link with friends to get everyone in.</p>
    <div class="lobby-players">
      <div
        v-for="(name, idx) in store.playerList"
        :key="name"
        class="lobby-player"
        :class="{ you: store.me && name === store.me.name, host: idx === 0, ready: store.lobby.players[name]?.ready }"
      >
        <span class="dot" :style="{ background: store.lobby.players[name].color }" :aria-label="`Player color: ${store.lobby.players[name].color}`"></span>
        {{ name }}{{ store.me && name === store.me.name ? ' (you)' : '' }}
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
      :class="{ 'ready-btn': store.lobby.players[store.me?.name]?.ready }"
      @click="store.toggleReady()"
    >{{ store.lobby.players[store.me?.name]?.ready ? 'Not ready' : 'I\'m ready' }}</button>
    <p class="lobby-status" v-if="store.canStart && !store.allReady">
      Waiting for {{ store.playerList.filter(n => !store.lobby.players[n]?.ready).length }} more player(s) to be ready...
    </p>
    <p class="lobby-status all-ready" v-if="store.allReady" aria-live="polite">
      All players ready! Starting...
    </p>
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
      <button class="action-btn" @click="$emit('leave')">Leave</button>
      <button class="action-btn danger" @click="$emit('delete')">Delete room</button>
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
  max-width: 700px;
  margin: 0 auto;
  padding: 32px 20px;
  text-align: center;
}

.lobby-title {
  font-size: 1.8rem;
  margin-bottom: 4px;
}

.lobby-sub {
  color: var(--muted-cream);
  font-size: 0.92rem;
  margin-bottom: 20px;
}

.lobby-players {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  margin-bottom: 20px;
}

.lobby-player {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 100px;
  font-size: 0.9rem;
}

.lobby-player .dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.lobby-player.you {
  border-color: var(--gold);
}

.host-badge {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 2px 6px;
  border-radius: 100px;
  background: rgba(238, 194, 92, 0.15);
  color: var(--gold);
  margin-left: 4px;
}

.ready-badge {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 2px 6px;
  border-radius: 100px;
  background: rgba(72, 199, 142, 0.15);
  color: #48c78e;
  margin-left: 4px;
}

.waiting-badge {
  font-size: 0.65rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 2px 6px;
  border-radius: 100px;
  background: rgba(255, 255, 255, 0.08);
  color: var(--muted-cream);
  margin-left: 4px;
}

.lobby-player.ready {
  border-color: #48c78e;
}

.ready-btn {
  background: linear-gradient(135deg, #48c78e, #2ea86e) !important;
}

.lobby-status {
  color: var(--muted-cream);
  font-size: 0.85rem;
  margin-top: 12px;
  margin-bottom: 12px;
  font-style: italic;
}

.lobby-status.all-ready {
  color: #48c78e;
  font-weight: 600;
}

.lobby-empty {
  color: var(--muted-cream);
  font-style: italic;
  margin-bottom: 24px;
  font-size: 0.9rem;
}

.start-btn {
  padding: 14px 36px;
  border-radius: 100px;
  font-weight: 700;
  font-size: 1rem;
  border: none;
  background: linear-gradient(135deg, var(--chili-orange), var(--broth-red));
  color: var(--steam-cream);
  transition: transform var(--ease-standard);
}

.start-btn:hover {
  transform: translateY(-2px);
}

.lobby-link {
  margin-top: 20px;
  font-size: 0.78rem;
  color: var(--muted-cream);
  word-break: break-all;
}

.lobby-actions {
  display: flex;
  gap: 10px;
  margin-top: 16px;
  justify-content: center;
}

.action-btn {
  padding: 10px 22px;
  border-radius: 100px;
  font-weight: 600;
  font-size: 0.88rem;
  border: 1px solid var(--line);
  background: var(--charcoal);
  color: var(--steam-cream);
  transition: all var(--ease-standard);
}

.action-btn:hover {
  border-color: var(--chili-orange);
  transform: translateY(-1px);
}

.action-btn.danger {
  border-color: var(--broth-red);
  color: var(--broth-red);
}

.action-btn.danger:hover {
  background: var(--broth-red);
  color: var(--steam-cream);
}

@media (max-width: 640px) {
  .lobby {
    padding: 20px 12px;
  }

  .lobby-title {
    font-size: 1.4rem;
  }

  .lobby-players {
    flex-direction: column;
    gap: 8px;
  }

  .lobby-player {
    width: 100%;
    justify-content: center;
  }

  .lobby-actions {
    flex-direction: column;
    gap: 8px;
  }

  .lobby-actions .action-btn {
    width: 100%;
  }
}
</style>
