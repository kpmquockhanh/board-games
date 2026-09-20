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
          class="room-name-input" data-testid="ek-create-name-input"
          @keydown.enter="submitCreate"
          autofocus
        >
        <button class="btn" data-testid="ek-create-submit" :disabled="!createName.trim()" @click="submitCreate">Create Room</button>
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

    <!-- Wrapped here rather than inside the component so the overlay gets a
         leave as well as an enter — the v-if that removes it lives out here. -->
    <Transition name="modal" appear>
      <EkResultOverlay
        v-if="phase === 'lobby' && store.phase === 'game' && store.gameState?.phase === 'ended'"
        :is-host="isHost"
        @delete="onDelete"
      />
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { toast } from 'vue-sonner'
import { Bomb, ArrowLeft } from '@lucide/vue'
import { useEkStore } from '../stores/explodingKitchen'
import { listRooms } from '../api'
import JoinOverlay from '../components/JoinOverlay.vue'
import { loadSavedPlayer, saveSavedPlayer, forgetSavedPlayer } from '../savedPlayer'
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

// The kitchen palette lives on <html> rather than on this view's root so it
// also reaches nodes teleported to <body> — RoomSettings' modal and the
// draw-flight animation — which render outside this subtree. Hub and Hotpot
// keep the original dark theme.
onMounted(() => {
  document.documentElement.classList.add('theme-kitchen')
})

onUnmounted(() => {
  document.documentElement.classList.remove('theme-kitchen')
})

onMounted(async () => {
  const params = new URLSearchParams(window.location.search)
  const room = params.get('room')
  if (room) {
    const player = loadSavedPlayer(`ek-player-${room}`)
    if (player) {
      try {
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
  saveSavedPlayer(`ek-player-${createdKey.value}`, player)
  const result = await store.joinLobby(player, createdKey.value)
  if (result?.error) {
    toast.error(result.error.charAt(0).toUpperCase() + result.error.slice(1))
    return
  }
  phase.value = 'lobby'
}

async function onJoin(player) {
  saveSavedPlayer(`ek-player-${joinKey.value}`, player)
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
    forgetSavedPlayer(`ek-player-${store.roomKey}`)
  }
  await store.deleteRoom()
  phase.value = 'choose'
  window.history.replaceState(null, '', window.location.pathname)
}

async function onLeave() {
  if (!confirm('Leave this room?')) return
  if (store.roomKey) {
    forgetSavedPlayer(`ek-player-${store.roomKey}`)
  }
  await store.leaveRoom()
  phase.value = 'choose'
  window.history.replaceState(null, '', window.location.pathname)
}
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
}

/* Same striped paper the other entry phases use, so naming a room does not
   drop the flow onto a flat scrim for one screen. */
.choose-overlay::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 50% 22%, rgba(255, 90, 43, 0.22), rgba(255, 232, 194, 0) 58%),
    repeating-linear-gradient(-38deg, #ffdfae 0 26px, var(--paper) 26px 52px);
  pointer-events: none;
}

.create-card {
  position: relative;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 32px 28px;
  max-width: 400px;
  width: 100%;
  text-align: center;
}

.choose-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 64px;
  margin-bottom: 14px;
  background: var(--chili);
  border: var(--edge-w) solid var(--edge);
  border-radius: 22px;
  box-shadow: var(--lift);
  color: var(--ink);
  transform: rotate(-6deg);
}

.create-card h1 {
  font-size: 1.9rem;
  margin-bottom: 8px;
  color: var(--ink);
}

.create-card p {
  color: var(--muted-cream);
  font-size: 0.95rem;
  font-weight: 700;
  line-height: 1.5;
  margin-bottom: 22px;
}

.room-name-input {
  width: 100%;
  padding: 14px 16px;
  border-radius: var(--radius-sm);
  border: var(--edge-w) solid var(--edge);
  background: var(--sand);
  color: var(--ink);
  font-size: 1.1rem;
  text-align: center;
  font-weight: 800;
  outline: none;
  margin-bottom: 16px;
  font-family: inherit;
}

.room-name-input::placeholder {
  color: var(--muted-cream);
}

.room-name-input:focus-visible {
  outline: 3px solid var(--chili);
  outline-offset: 3px;
}

/* The shared .btn and .back-link rules in main.css already carry the kitchen
   treatment; only the layout bits that were local stay here. */
.btn {
  width: 100%;
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
</style>
