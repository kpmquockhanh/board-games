<template>
  <div>
    <!-- CHOOSE PHASE -->
    <div v-if="phase === 'choose' && !showNameInput" class="choose-overlay">
      <div class="choose-card">
        <button class="close-btn" @click="$router.push('/')"><X :size="16" /></button>
        <div class="choose-icon"><Soup :size="32" /></div>
        <h1>Hotpot Night</h1>
        <p>Cook together at a virtual table. Drop in ingredients, cheer, and chat while the broth simmers.</p>
        <div class="choose-actions">
          <button class="choose-btn create" @click="showNameInput = true">
            <span>Create Room</span>
            <small>Start a new table and invite friends</small>
          </button>
          <button class="choose-btn join" @click="goToJoin">
            <span>Join Room</span>
            <small>Enter a room key or pick an active room</small>
          </button>
        </div>
      </div>
    </div>

    <!-- NAME INPUT PHASE -->
    <div v-if="phase === 'choose' && showNameInput" class="choose-overlay">
      <div class="create-card">
        <div class="choose-icon"><Soup :size="32" /></div>
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

    <!-- CREATE PHASE -->
    <div v-if="phase === 'create'" class="choose-overlay">
      <div class="create-card">
        <div class="choose-icon"><Soup :size="32" /></div>
        <h1>{{ store.roomName || 'Your table is ready' }}</h1>
        <p>Share this room key or link with friends so they can join.</p>
        <div class="room-key-display">
          <span class="room-key">{{ createdKey }}</span>
          <button class="copy-btn" @click="copyKey">Copy</button>
        </div>
        <div class="share-url">
          <input type="text" :value="shareUrl" readonly @click="copyUrl">
          <small>Click to copy link</small>
        </div>
        <JoinOverlay
          inline
          icon=""
          title=""
          description=""
          name-placeholder="Your name"
          button-text="Sit down"
          @join="onCreateJoin"
        />
      </div>
    </div>

    <!-- JOIN PHASE (manual) -->
    <div v-if="phase === 'join'" class="choose-overlay">
      <div class="choose-card">
        <div class="choose-icon"><Soup :size="32" /></div>
        <h1>Join a table</h1>
        <p>Enter the room key shared by the host, or pick an active room below.</p>
        <input
          type="text"
          v-model="joinKey"
          placeholder="Room key"
          maxlength="6"
          class="room-key-input"
          @keydown.enter="submitJoinKey"
        >
        <button class="btn" :disabled="joinKey.length < 4" @click="submitJoinKey">Continue</button>

        <div v-if="loadingRooms" class="rooms-loading">Loading rooms…</div>
        <div v-else-if="rooms.length > 0" class="rooms-section">
          <div class="rooms-label">Active rooms</div>
          <div class="rooms-list">
            <div
              v-for="room in rooms"
              :key="room.room_key"
              class="room-item"
            >
              <button class="room-item-join" @click="pickRoom(room.room_key)">
                <span class="room-item-name">{{ room.name || room.room_key }}</span>
                <span class="room-item-count">{{ room.player_count }}/{{ room.max_players }} players</span>
              </button>
              <button class="room-item-delete" @click.stop="onDeleteFromList(room.room_key)" title="Delete room"><X :size="14" /></button>
            </div>
          </div>
        </div>
        <div v-else class="rooms-empty">No active rooms yet — create one!</div>

        <button class="back-link" @click="phase = 'choose'"><ArrowLeft :size="14" /> Back</button>
      </div>
    </div>

    <!-- JOIN PHASE (with key — from URL or after entering) -->
    <JoinOverlay
      v-if="phase === 'joinOverlay'"
      :icon="Soup"
      title="Join the pot"
      description="Everyone who opens this link lands at the same virtual table. Drop in ingredients, cheers, and chat — your name and messages here are visible to anyone else who joins."
      name-placeholder="Your name"
      button-text="Sit down"
      game="hotpot"
      :room-key="joinKey"
      :initial-name="savedName"
      :initial-color="savedColor"
      @join="onJoin"
      @back="phase = 'join'"
    />

    <div class="app" v-if="phase === 'playing'">
      <div class="game-info">
        <span class="room-name-pill" v-if="store.roomName">{{ store.roomName }}</span>
        <span class="room-key-pill" @click="copyRoomKey" title="Copy room key">Room: {{ store.roomKey }} <ClipboardCopy :size="12" /></span>
        <span class="status-pill">{{ store.playerCount }} {{ store.playerCount === 1 ? 'person' : 'people' }} at the table</span>
        <button class="leave-btn" @click="onLeave">Leave table</button>
        <button class="delete-btn" @click="onDelete">Delete room</button>
      </div>

      <div class="table-zone" ref="tableZone">
        <div class="table-oval"></div>
        <svg class="pot-svg" viewBox="0 0 200 200">
          <g stroke="var(--steam-cream)" stroke-width="4" fill="none" stroke-linecap="round">
            <path class="steam s1" d="M85 70 q-6 -12 0 -24 q6 -12 0 -24"/>
            <path class="steam s2" d="M105 68 q-6 -12 0 -24 q6 -12 0 -24"/>
            <path class="steam s3" d="M120 74 q6 -12 0 -24 q-6 -12 0 -24"/>
          </g>
          <ellipse cx="100" cy="104" rx="72" ry="14" fill="#2b1c14"/>
          <path d="M32 104 C32 155 60 182 100 182 C140 182 168 155 168 104 Z" fill="#1c1512" stroke="#3a2a1e" stroke-width="2"/>
          <clipPath id="pc"><path d="M34 106 C34 154 62 178 100 178 C138 178 166 154 166 106 Z"/></clipPath>
          <g clip-path="url(#pc)">
            <rect x="32" y="106" width="68" height="90" fill="#8a1f16"/>
            <rect x="100" y="106" width="68" height="90" fill="#5c4a2e"/>
            <path d="M100 106 C100 128 114 128 100 146 C86 164 100 164 100 182" stroke="#1c1512" stroke-width="4" fill="none"/>
          </g>
          <circle class="bubble b1" cx="62" cy="130" r="3" fill="var(--chili-orange)"/>
          <circle class="bubble b2" cx="80" cy="140" r="2.4" fill="var(--gold)"/>
          <circle class="bubble b3" cx="128" cy="132" r="2.6" fill="#f2e8d5"/>
        </svg>

        <div
          v-for="player in store.players"
          :key="player.player_name"
          class="seat"
          :style="seatStyle(player.player_name)"
        >
          <div class="bubble-msg" :class="{ show: showBubble(player.player_name) }">{{ bubbleText(player.player_name) }}</div>
          <div class="avatar" :class="{ me: store.me && player.player_name === store.me.name }" :style="{ background: player.color }">
            {{ player.player_name.trim().slice(0, 2).toUpperCase() }}
          </div>
          <div class="name">{{ player.player_name }}{{ store.me && player.player_name === store.me.name ? ' (you)' : '' }}</div>
        </div>

        <div class="empty-seats" v-if="store.playerCount === 0">Table is empty — share this link to fill the seats.</div>
      </div>

      <div class="panels">
        <div class="panel">
          <h3>Drop something in</h3>
          <div class="picker-grid">
            <div
              v-for="item in ITEMS"
              :key="item.name"
              class="pick-chip"
              @click="onDropItem(item)"
            >{{ item.emoji }} {{ item.name }}</div>
          </div>
          <h3 style="margin-top: 6px;">Say cheers</h3>
          <div class="cheers-row">
            <button class="cheers-btn" v-for="emoji in ['🥂','🔥','😋','👏']" :key="emoji" @click="onCheer(emoji)">{{ emoji }}</button>
          </div>
          <form class="chat-form" @submit.prevent="onSendChat">
            <input v-model="chatText" type="text" maxlength="80" placeholder="Say something to the table…" autocomplete="off">
            <button type="submit">Send</button>
          </form>
        </div>
        <div class="panel">
          <h3>What's happening</h3>
          <div class="feed">
            <div v-if="store.feed.length === 0" class="feed-empty">Nothing yet — be the first to drop something in.</div>
            <div v-for="f in store.recentFeed" :key="f.id" class="feed-item">
              <div class="feed-avatar" :style="{ background: f.color || '#888' }"></div>
              <div class="feed-text">
                <span v-if="f.type === 'join'"><b>{{ f.name }}</b> sat down at the table</span>
                <span v-else-if="f.type === 'leave'"><b>{{ f.name }}</b> left the table</span>
                <span v-else-if="f.type === 'drop'"><b>{{ f.name }}</b> dropped in {{ f.emoji || '' }} {{ f.text }}</span>
                <span v-else-if="f.type === 'cheer'"><b>{{ f.name }}</b> {{ f.text }}</span>
                <span v-else><b>{{ f.name }}</b>: {{ f.text }}</span>
                <span class="feed-time">{{ formatTime(f.ts) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { toast } from 'vue-sonner'
import { X, Soup, ArrowLeft, ClipboardCopy } from '@lucide/vue'
import { useHotpotStore } from '../stores/hotpot'
import { listRooms, deleteRoom } from '../api'
import JoinOverlay from '../components/JoinOverlay.vue'

const store = useHotpotStore()
const chatText = ref('')
const tableZone = ref(null)

const phase = ref('choose')
const createdKey = ref('')
const joinKey = ref('')
const rooms = ref([])
const loadingRooms = ref(false)
const savedName = ref('')
const savedColor = ref('')
const showNameInput = ref(false)
const createName = ref('')

const ITEMS = [
  { emoji: '🥩', name: 'beef' },
  { emoji: '🍤', name: 'shrimp' },
  { emoji: '🍄', name: 'mushrooms' },
  { emoji: '🥬', name: 'napa cabbage' },
  { emoji: '🍜', name: 'noodles' },
  { emoji: '🧊', name: 'tofu' },
  { emoji: '🌽', name: 'corn' },
  { emoji: '🥚', name: 'quail eggs' },
  { emoji: '🦑', name: 'squid' },
]

const lastFeedId = ref(null)

watch(
  () => store.roomKey,
  (val) => {
    if (!val && phase.value === 'playing') {
      phase.value = 'choose'
      window.history.replaceState(null, '', window.location.pathname)
    }
  }
)

watch(
  () => store.feed,
  (newFeed) => {
    if (!newFeed.length) return
    const latest = newFeed[newFeed.length - 1]
    if (latest.id === lastFeedId.value) return
    lastFeedId.value = latest.id
    if (latest.type === 'drop' && latest.emoji) {
      spawnFloatingItem(latest.emoji)
    }
  },
  { immediate: true }
)

const shareUrl = computed(() => {
  const base = window.location.origin + window.location.pathname
  return `${base}?room=${createdKey.value}`
})

async function fetchRooms() {
  loadingRooms.value = true
  try {
    const res = await listRooms('hotpot')
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
    const saved = localStorage.getItem(`hotpot-player-${room}`)
    if (saved) {
      try {
        const player = JSON.parse(saved)
        const result = await store.join(player, room)
        if (result?.error) {
          savedName.value = player.name || ''
          savedColor.value = player.color || ''
          joinKey.value = room
          phase.value = 'joinOverlay'
          return
        }
        phase.value = 'playing'
        return
      } catch {}
    }
    joinKey.value = room
    phase.value = 'joinOverlay'
  }
})

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

function copyKey() {
  navigator.clipboard.writeText(createdKey.value)
  toast.success('Room key copied!')
}

function copyUrl() {
  navigator.clipboard.writeText(shareUrl.value)
  toast.success('Link copied!')
}

function copyRoomKey() {
  navigator.clipboard.writeText(store.roomKey)
  toast.success('Room key copied!')
}

async function onCreateJoin(player) {
  localStorage.setItem(`hotpot-player-${createdKey.value}`, JSON.stringify(player))
  const result = await store.join(player, createdKey.value)
  if (result?.error) {
    toast.error(result.error.charAt(0).toUpperCase() + result.error.slice(1))
    return
  }
  phase.value = 'playing'
}

async function onJoin(player) {
  localStorage.setItem(`hotpot-player-${joinKey.value}`, JSON.stringify(player))
  window.history.replaceState(null, '', `${window.location.pathname}?room=${joinKey.value}`)
  savedName.value = ''
  savedColor.value = ''
  const result = await store.join(player, joinKey.value)
  if (result?.error) {
    toast.error(result.error.charAt(0).toUpperCase() + result.error.slice(1))
    return
  }
  phase.value = 'playing'
}

function onLeave() {
  if (store.roomKey) {
    localStorage.removeItem(`hotpot-player-${store.roomKey}`)
  }
  store.leave()
  window.location.href = '/hotpot'
}

async function onDelete() {
  if (!confirm('Delete this room? Everyone will be kicked out.')) return
  if (store.roomKey) {
    localStorage.removeItem(`hotpot-player-${store.roomKey}`)
  }
  await store.deleteRoom()
  phase.value = 'choose'
  window.history.replaceState(null, '', window.location.pathname)
}

async function onDeleteFromList(roomKey) {
  if (!confirm('Delete this room?')) return
  await deleteRoom('hotpot', roomKey)
  await fetchRooms()
}

function onDropItem(item) {
  store.dropItem(item)
}

function onCheer(emoji) {
  store.cheer(emoji)
}

function onSendChat() {
  if (!chatText.value.trim()) return
  store.sendMessage(chatText.value.trim())
  chatText.value = ''
}

function seatStyle(name) {
  if (!tableZone.value) return {}
  const zone = tableZone.value
  const names = store.players.map((p) => p.player_name)
  const idx = names.indexOf(name)
  const cx = zone.clientWidth / 2
  const cy = zone.clientHeight / 2
  const rx = zone.clientWidth * 0.42
  const ry = zone.clientHeight * 0.38
  const angle = (idx / names.length) * Math.PI * 2 - Math.PI / 2
  const x = cx + Math.cos(angle) * rx
  const y = cy + Math.sin(angle) * ry
  return { left: x + 'px', top: y + 'px' }
}

const lastBubble = ref({ name: '', text: '', show: false })

function showBubble(name) {
  return lastBubble.value.show && lastBubble.value.name === name
}

function bubbleText(name) {
  return lastBubble.value.name === name ? lastBubble.value.text : ''
}

function formatTime(ts) {
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function spawnFloatingItem(emoji) {
  if (!tableZone.value) return
  const el = document.createElement('div')
  el.className = 'float-item'
  el.textContent = emoji
  const angle = Math.random() * Math.PI * 2
  const r = 150
  const x = tableZone.value.clientWidth / 2 + Math.cos(angle) * r
  const y = tableZone.value.clientHeight / 2 + Math.sin(angle) * r * 0.5 - 60
  el.style.left = x + 'px'
  el.style.top = y + 'px'
  tableZone.value.appendChild(el)
  setTimeout(() => el.remove(), 1500)
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

.close-btn {
  position: absolute;
  top: 12px;
  right: 12px;
  background: none;
  border: 1px solid var(--line);
  color: var(--mild-cream);
  width: 30px;
  height: 30px;
  border-radius: 50%;
  font-size: 0.85rem;
  cursor: pointer;
  font-family: inherit;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.15s ease, color 0.15s ease;
}

.close-btn:hover {
  border-color: var(--broth-red);
  color: var(--broth-red);
}

.choose-card,
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

.choose-card h1,
.create-card h1 {
  font-size: 1.7rem;
  margin-bottom: 8px;
}

.choose-card p,
.create-card p {
  color: var(--mild-cream);
  opacity: 0.8;
  font-size: 0.92rem;
  line-height: 1.5;
  margin-bottom: 22px;
}

.choose-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.choose-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 18px 20px;
  border-radius: 14px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--steam-cream);
  cursor: pointer;
  transition: all 0.15s ease;
  font-family: inherit;
}

.choose-btn:hover {
  border-color: var(--chili-orange);
  transform: translateY(-2px);
}

.choose-btn span {
  font-weight: 700;
  font-size: 1rem;
}

.choose-btn small {
  font-size: 0.78rem;
  color: var(--mild-cream);
  opacity: 0.6;
}

.room-key-display {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: center;
  margin-bottom: 16px;
}

.room-key {
  font-family: 'Baloo 2', monospace;
  font-size: 1.8rem;
  font-weight: 800;
  letter-spacing: 0.15em;
  color: var(--gold);
  background: rgba(238, 194, 92, 0.1);
  padding: 10px 22px;
  border-radius: 12px;
  border: 1px solid rgba(238, 194, 92, 0.3);
}

.copy-btn {
  padding: 10px 18px;
  border-radius: 10px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--steam-cream);
  font-weight: 600;
  font-size: 0.85rem;
  cursor: pointer;
  font-family: inherit;
  transition: border-color 0.15s ease;
}

.copy-btn:hover {
  border-color: var(--chili-orange);
}

.share-url {
  margin-bottom: 18px;
}

.share-url input {
  width: 100%;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--mild-cream);
  font-size: 0.82rem;
  text-align: center;
  cursor: pointer;
  outline: none;
  font-family: inherit;
}

.share-url input:focus {
  border-color: var(--chili-orange);
}

.share-url small {
  display: block;
  margin-top: 6px;
  font-size: 0.72rem;
  color: var(--mild-cream);
  opacity: 0.5;
}

.room-key-input {
  width: 100%;
  padding: 14px 16px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: #1c1512;
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
  color: var(--mild-cream);
  opacity: 0.5;
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
  background: #1c1512;
  color: var(--steam-cream);
  transition: all 0.15s ease;
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
  transition: all 0.15s ease;
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
  color: var(--mild-cream);
  opacity: 0.7;
}

.rooms-empty {
  margin-top: 16px;
  font-size: 0.85rem;
  color: var(--mild-cream);
  opacity: 0.5;
  text-align: center;
  font-style: italic;
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

.app {
  max-width: 1100px;
  margin: 0 auto;
  padding: 26px 20px 60px;
}

.game-info {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 26px;
  flex-wrap: wrap;
}

.room-key-pill {
  font-family: 'Baloo 2', monospace;
  font-size: 0.78rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: rgba(238, 194, 92, 0.12);
  color: var(--gold);
  border: 1px solid rgba(238, 194, 92, 0.3);
  letter-spacing: 0.08em;
  font-weight: 700;
  cursor: pointer;
  transition: border-color 0.15s ease;
}

.room-key-pill:hover {
  border-color: var(--gold);
}

.room-name-pill {
  font-size: 0.85rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: rgba(238, 194, 92, 0.12);
  color: var(--steam-cream);
  border: 1px solid rgba(238, 194, 92, 0.3);
  font-weight: 700;
}

.status-pill {
  font-size: 0.78rem;
  padding: 6px 14px;
  border-radius: 100px;
  background: rgba(238, 194, 92, 0.12);
  color: var(--gold);
  border: 1px solid rgba(238, 194, 92, 0.3);
}

.leave-btn {
  background: transparent;
  border: 1px solid var(--line);
  color: var(--mild-cream);
  padding: 8px 16px;
  border-radius: 100px;
  font-size: 0.82rem;
  cursor: pointer;
  font-family: inherit;
}

.leave-btn:hover {
  border-color: var(--chili-orange);
}

.delete-btn {
  background: transparent;
  border: 1px solid var(--broth-red);
  color: var(--broth-red);
  padding: 8px 16px;
  border-radius: 100px;
  font-size: 0.82rem;
  cursor: pointer;
  font-family: inherit;
}

.delete-btn:hover {
  background: var(--broth-red);
  color: var(--steam-cream);
}

.table-zone {
  position: relative;
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 24px;
  height: 440px;
  margin-bottom: 22px;
  overflow: hidden;
}

.table-oval {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 70%;
  max-width: 340px;
  aspect-ratio: 1;
  border-radius: 50%;
  background: radial-gradient(circle at 35% 30%, #3a2a1e, #1c1512 75%);
  border: 1px solid var(--line);
}

.pot-svg {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  width: 150px;
  height: 150px;
}

.steam {
  transform-origin: center bottom;
  animation: rise 4.5s ease-in-out infinite;
  opacity: 0;
}
.steam.s2 { animation-delay: 1.1s; }
.steam.s3 { animation-delay: 2.2s; }
@keyframes rise {
  0% { opacity: 0; transform: translateY(0); }
  15% { opacity: 0.6; }
  80% { opacity: 0; }
  100% { opacity: 0; transform: translateY(-40px); }
}

.bubble { animation: bub 2.4s ease-in-out infinite; }
.bubble.b2 { animation-delay: .5s; }
.bubble.b3 { animation-delay: 1.1s; }
@keyframes bub {
  0%, 100% { transform: translateY(0); opacity: 0.5; }
  50% { transform: translateY(-5px); opacity: 1; }
}

.seat {
  position: absolute;
  width: 64px;
  text-align: center;
  transform: translate(-50%, -50%);
  transition: opacity 0.3s ease;
}

.seat .avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  margin: 0 auto 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: 'Baloo 2', sans-serif;
  font-weight: 700;
  font-size: 1.1rem;
  color: #1c1512;
  box-shadow: 0 6px 18px -6px rgba(0, 0, 0, 0.6);
  border: 2px solid rgba(253, 243, 228, 0.4);
}

.seat .avatar.me {
  box-shadow: 0 0 0 3px var(--gold), 0 6px 18px -6px rgba(0, 0, 0, 0.6);
}

.seat .name {
  font-size: 0.72rem;
  color: var(--mild-cream);
  opacity: 0.85;
}

.seat .bubble-msg {
  position: absolute;
  bottom: 100%;
  left: 50%;
  transform: translateX(-50%);
  background: var(--steam-cream);
  color: #1c1512;
  padding: 5px 10px;
  border-radius: 10px;
  font-size: 0.72rem;
  white-space: nowrap;
  margin-bottom: 6px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.3s ease;
}

.seat .bubble-msg.show {
  opacity: 1;
}

.empty-seats {
  position: absolute;
  bottom: 16px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 0.78rem;
  color: var(--mild-cream);
  opacity: 0.5;
  text-align: center;
}

.panels {
  display: grid;
  grid-template-columns: 1.1fr 0.9fr;
  gap: 20px;
}

@media (max-width: 760px) {
  .panels { grid-template-columns: 1fr; }
}

.panel {
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 18px;
  padding: 22px;
}

.panel h3 {
  font-size: 0.78rem;
  margin-bottom: 14px;
  color: var(--gold);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.picker-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.pick-chip {
  padding: 9px 14px;
  border-radius: 100px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--mild-cream);
  font-size: 0.86rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.pick-chip:hover {
  border-color: var(--chili-orange);
  transform: translateY(-1px);
}

.cheers-row {
  display: flex;
  gap: 10px;
}

.cheers-btn {
  flex: 1;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--steam-cream);
  font-size: 1.3rem;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.cheers-btn:hover {
  transform: scale(1.06);
  border-color: var(--gold);
}

.chat-form {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.chat-form input {
  flex: 1;
  padding: 11px 14px;
  border-radius: 100px;
  border: 1px solid var(--line);
  background: #1c1512;
  color: var(--steam-cream);
  font-size: 0.88rem;
  outline: none;
  font-family: inherit;
}

.chat-form input:focus {
  border-color: var(--chili-orange);
}

.chat-form button {
  padding: 11px 18px;
  border-radius: 100px;
  border: none;
  background: var(--chili-orange);
  color: var(--steam-cream);
  font-weight: 600;
  font-size: 0.86rem;
  font-family: inherit;
}

.feed {
  max-height: 280px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.feed-item {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  font-size: 0.86rem;
  line-height: 1.4;
}

.feed-avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 1px;
}

.feed-text {
  color: var(--mild-cream);
}

.feed-text b {
  color: var(--steam-cream);
}

.feed-time {
  font-size: 0.7rem;
  opacity: 0.4;
  margin-left: 6px;
}

.feed-empty {
  font-size: 0.85rem;
  opacity: 0.5;
  font-style: italic;
}

:deep(.float-item) {
  position: absolute;
  font-size: 1.3rem;
  opacity: 0;
  pointer-events: none;
  animation: drop 1.4s ease-out forwards;
}

@keyframes drop {
  0% { opacity: 0; transform: translate(-50%, -50%) scale(0.6); }
  20% { opacity: 1; transform: translate(-50%, -50%) scale(1.1); }
  100% { opacity: 0; transform: translate(-50%, -50%) translateY(60px) scale(0.7); }
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
