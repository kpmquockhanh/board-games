<template>
  <div>
    <EkChoosePhase
      v-if="phase === 'choose' && !showNameInput"
      @create="showNameInput = true"
      @join="goToJoin"
    />

    <div v-if="phase === 'choose' && showNameInput" class="choose-overlay">
      <div class="create-card">
        <div class="choose-icon" aria-hidden="true"><Bomb :size="32" /></div>
        <h1>Name your room</h1>
        <p>Give your room a name so others can identify it.</p>
        <input
          type="text"
          v-model="createName"
          placeholder="Room name"
          maxlength="30"
          class="room-name-input"
          @keydown.enter="submitCreate"
          autofocus
        >
        <button class="btn" :disabled="!createName.trim()" @click="submitCreate">Create Room</button>
        <button class="back-link" @click="showNameInput = false; createName = ''"><ArrowLeft :size="14" /> Back</button>
      </div>
    </div>

    <EkCreatePhase
      v-if="phase === 'create'"
      :room-key="createdKey"
      @join="onCreateJoin"
    />

    <EkJoinPhase
      v-if="phase === 'join'"
      v-model:join-key="joinKey"
      :rooms="rooms"
      :loading-rooms="loadingRooms"
      @submit-key="submitJoinKey"
      @pick-room="pickRoom"
      @delete-room="onDeleteFromList"
      @back="phase = 'choose'"
    />

    <JoinOverlay
      v-if="phase === 'joinOverlay'"
      :icon="Bomb"
      title="Join the kitchen"
      description="A card game of cooking chaos. Draw cards, avoid the exploding kitchen, and be the last chef standing."
      name-placeholder="Your name"
      button-text="Join the kitchen"
      game="ek"
      :room-key="joinKey"
      :initial-name="savedName"
      :initial-color="savedColor"
      @join="onJoin"
      @back="phase = 'join'"
    />

    <EkLobby
      v-if="phase === 'lobby' && store.phase === 'lobby'"
      @leave="onLeave"
      @delete="onDelete"
    />

    <EkGameView
      v-if="phase === 'lobby' && store.phase === 'game'"
      :is-host="isHost"
      @leave="onLeave"
      @delete="onDelete"
    />

    <EkResultOverlay
      v-if="phase === 'lobby' && store.phase === 'game' && store.gameState?.phase === 'ended'"
      :is-host="isHost"
      @delete="onDelete"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Bomb, ArrowLeft } from '@lucide/vue'
import { useEkStore } from '../stores/explodingKitchen'
import { listRooms, deleteRoom } from '../api'
import JoinOverlay from '../components/JoinOverlay.vue'
import EkChoosePhase from '../components/game/EkChoosePhase.vue'
import EkCreatePhase from '../components/game/EkCreatePhase.vue'
import EkJoinPhase from '../components/game/EkJoinPhase.vue'
import EkLobby from '../components/game/EkLobby.vue'
import EkGameView from '../components/game/EkGameView.vue'
import EkResultOverlay from '../components/game/EkResultOverlay.vue'

const store = useEkStore()

const phase = ref('choose')
const createdKey = ref('')
const joinKey = ref('')
const rooms = ref([])
const loadingRooms = ref(false)
const savedName = ref('')
const savedColor = ref('')
const showNameInput = ref(false)
const createName = ref('')

const isHost = computed(() => {
  const names = store.playerList
  return names.length > 0 && store.me && names[0] === store.me.name
})

async function fetchRooms() {
  loadingRooms.value = true
  try {
    const res = await listRooms('ek')
    rooms.value = res?.rooms || []
  } catch {
    rooms.value = []
  } finally {
    loadingRooms.value = false
  }
}

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const room = params.get('room')
  if (room) {
    const saved = localStorage.getItem(`ek-player-${room}`)
    if (saved) {
      try {
        const player = JSON.parse(saved)
        const result = await store.joinLobby(player, room)
        if (result?.error) {
          savedName.value = player.name || ''
          savedColor.value = player.color || ''
          joinKey.value = room
          phase.value = 'joinOverlay'
          return
        }
        phase.value = 'lobby'
        return
      } catch {}
    }
    joinKey.value = room
    phase.value = 'joinOverlay'
  }
})

watch(
  () => store.roomKey,
  (val) => {
    if (!val && (phase.value === 'lobby')) {
      phase.value = 'choose'
      window.history.replaceState(null, '', window.location.pathname)
    }
  }
)

function goToJoin() {
  phase.value = 'join'
  fetchRooms()
}

function pickRoom(roomKey) {
  joinKey.value = roomKey
  phase.value = 'joinOverlay'
}

async function submitCreate() {
  const name = createName.value.trim()
  if (!name) return
  showNameInput.value = false
  const roomKey = await store.create(name)
  if (!roomKey) return
  createdKey.value = roomKey
  createName.value = ''
  phase.value = 'create'
  window.history.replaceState(null, '', `${window.location.pathname}?room=${roomKey}`)
}

function submitJoinKey() {
  if (joinKey.value.length >= 4) {
    phase.value = 'joinOverlay'
  }
}

async function onCreateJoin(player) {
  localStorage.setItem(`ek-player-${createdKey.value}`, JSON.stringify(player))
  const result = await store.joinLobby(player, createdKey.value)
  if (result?.error) {
    toast.error(result.error.charAt(0).toUpperCase() + result.error.slice(1))
    return
  }
  phase.value = 'lobby'
}

async function onJoin(player) {
  localStorage.setItem(`ek-player-${joinKey.value}`, JSON.stringify(player))
  window.history.replaceState(null, '', `${window.location.pathname}?room=${joinKey.value}`)
  savedName.value = ''
  savedColor.value = ''
  const result = await store.joinLobby(player, joinKey.value)
  if (result?.error) {
    toast.error(result.error.charAt(0).toUpperCase() + result.error.slice(1))
    return
  }
  phase.value = 'lobby'
}

async function onDelete() {
  if (!confirm('Delete this room? Everyone will be kicked out.')) return
  if (store.roomKey) {
    localStorage.removeItem(`ek-player-${store.roomKey}`)
  }
  await store.deleteRoom()
  phase.value = 'choose'
  window.history.replaceState(null, '', window.location.pathname)
}

async function onLeave() {
  if (!confirm('Leave this room?')) return
  if (store.roomKey) {
    localStorage.removeItem(`ek-player-${store.roomKey}`)
  }
  await store.leaveRoom()
  phase.value = 'choose'
  window.history.replaceState(null, '', window.location.pathname)
}

async function onDeleteFromList(roomKey) {
  if (!confirm('Delete this room?')) return
  await deleteRoom('ek', roomKey)
  await fetchRooms()
}
</script>

<style scoped>
.choose-overlay {
  position: fixed;
  inset: 0;
  z-index: 100;
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

.room-name-input {
  width: 100%;
  padding: 14px 16px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--steam-cream);
  font-size: 1.1rem;
  text-align: center;
  font-weight: 600;
  outline: none;
  margin-bottom: 14px;
  font-family: inherit;
}

.room-name-input:focus {
  border-color: var(--chili-orange);
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
  font-family: inherit;
}

.btn:hover {
  transform: translateY(-2px);
}

.btn:disabled {
  opacity: 0.5;
  cursor: default;
  transform: none;
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
