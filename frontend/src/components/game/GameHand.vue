<template>
  <div class="hand-zone">
    <!-- Spectating message -->
    <div v-if="store.amIEliminated" class="spectating-banner">
      <Bomb class="spectating-icon" :size="18" aria-hidden="true" />
      <span>You exploded{{ store.amISpectating ? ' — watching the rest play out' : '' }}</span>
      <button class="leave-spectate-btn" @click="$emit('leave')">
        <LogOut :size="14" />
        Leave room
      </button>
    </div>

    <!-- Card hand with floating selection pill -->
    <div v-else class="hand-cards-wrap">
      <!-- Floating selection pill -->
      <Transition name="pill">
        <div v-if="store.selectedCardIndices.length > 0" class="selection-pill">
          <span class="pill-count">{{ store.selectedCardIndices.length }} selected</span>
          <span class="pill-sep"></span>
          <button class="pill-clear" data-testid="hand-clear-selection" @click="store.clearSelection()">Clear</button>
          <span v-if="store.selectedCombo" class="pill-combo">
            {{ store.selectedCombo.count }}x {{ store.categoryLabel(store.selectedCombo.category) }}
          </span>
          <button
            v-if="store.isMyTurn && !store.nopeWindow.active"
            class="pill-play"
            data-testid="hand-play-selected"
            :disabled="store.selectedCardIndices.length === 0"
            @click="$emit('playSelected')"
          >
            <Play :size="12" />
            <span>Play{{ store.selectedCombo ? ` ${store.selectedCombo.count}x` : '' }}</span>
          </button>
        </div>
      </Transition>

      <TransitionGroup tag="div" class="hand-cards" name="card">
        <div
          v-for="({ cardId, uid }, i) in store.handCards"
          :key="uid"
          :ref="el => setCardEl(uid, el)"
          class="hand-card"
          :data-testid="'hand-card-' + i"
          :data-card-category="store.getCategoryName(cardId)"
          :class="{
            selected: store.selectedCardIndices.includes(i),
            disabled: !store.isMyTurn && !(store.nopeWindow.active && store.getCategoryName(cardId) === 'Nope'),
            'in-flight': store.drawFlight?.uid === uid,
            landed: landedUid === uid,
            [`cat-${getCategoryKey(cardId)}`]: true
          }"
          :style="{ '--card-index': i, '--total': store.handCards.length }"
          @click="$emit('cardClick', i)"
        >
          <div class="card-badge" :class="`badge-${getCategoryKey(cardId)}`">
            <component :is="getCategoryIcon(cardId)" class="badge-icon" :size="10" aria-hidden="true" />
            <span class="badge-label">{{ store.getCategoryName(cardId) }}</span>
          </div>
          <div class="card-icon-area">
            <img
              v-if="store.getCardData(cardId)"
              :src="`/cards/${store.getCardData(cardId).file}`"
              :alt="store.getCardData(cardId)?.name"
              class="card-image"
              loading="lazy"
            >
          </div>
          <div class="card-title">{{ getCardDisplayName(cardId) }}</div>
        </div>
      </TransitionGroup>
    </div>

    <!-- Nope hint -->
    <div v-if="store.canINope" class="nope-hint" aria-live="polite">
      Select a Nope card to play!
    </div>

    <!-- Action buttons (only nope, pass and leave remain) -->
    <div v-if="!store.amIEliminated && store.nopeWindow.active" class="hand-actions">
      <button
        v-if="store.canINope"
        class="action-btn nope-btn"
        data-testid="hand-nope-btn"
        :disabled="!hasNopeSelected"
        @click="$emit('playNope')"
      >
        <ShieldOff class="btn-icon" :size="16" />
        <span>Nope!</span>
      </button>
      <!-- Everyone who may answer gets this, Nope in hand or not: passing is
           what closes the window early, and it has to mean nothing about the
           cards of whoever pressed it. -->
      <button
        v-if="store.canIAnswerNope"
        class="action-btn pass-btn"
        data-testid="hand-pass-btn"
        @click="$emit('passNope')"
      >
        <Check class="btn-icon" :size="16" />
        <span>Let it through</span>
        <span class="nope-countdown">{{ store.nopeTimeLeft }}s</span>
      </button>
      <div v-else class="pass-waiting">
        Waiting for {{ store.nopeWaitingCount }} player{{ store.nopeWaitingCount === 1 ? '' : 's' }}… {{ store.nopeTimeLeft }}s
      </div>
    </div>

    <!-- Spectating leave -->
    <div v-if="store.amIEliminated" class="hand-actions">
      <button class="action-btn leave-btn" @click="$emit('leave')">
        <LogOut class="btn-icon" :size="16" />
        <span>Leave room</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, onBeforeUpdate } from 'vue'
import {
  Play, ShieldOff, LogOut, Check, Swords, Shield, SkipForward, RotateCcw, Eye,
  Hand, Shuffle, ArrowDownToLine, ArrowDownUp, Trash2, Bomb, MapPin, Archive,
  Pickaxe, Gift, Copy, Flame, Users, Sparkles, PawPrint,
} from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()

defineEmits(['cardClick', 'playSelected', 'playNope', 'passNope', 'leave'])

// The table needs the on-screen box of a specific card to fly a drawn card
// into it, and only the hand knows where its cards ended up in the fan.
const cardEls = new Map()

onBeforeUpdate(() => cardEls.clear())

function setCardEl(uid, el) {
  if (el) cardEls.set(uid, el)
  else cardEls.delete(uid)
}

// The flight clearing is the card landing, so that is when it lights up.
const landedUid = ref(null)
let landedTimer = null

watch(
  () => store.drawFlight,
  (now, before) => {
    if (now || !before) return
    landedUid.value = before.uid
    clearTimeout(landedTimer)
    landedTimer = setTimeout(() => { landedUid.value = null }, 600)
  }
)

defineExpose({
  getCardRect: (uid) => cardEls.get(uid)?.getBoundingClientRect() || null,
})

const hasNopeSelected = computed(() => {
  if (store.selectedCardIndices.length === 0) return false
  const hand = store.myHand
  return store.selectedCardIndices.some(i => {
    const cardId = hand[i]
    return store.getCardData(cardId) && store.getCategoryName(cardId) === 'Nope'
  })
})

// Every card category gets its own badge: its own name, colour and icon.
// The icons match the ones the room settings screen already uses for the
// same categories, so a category looks the same wherever it is named. The
// four categories that never appear in settings (Defense, Explosive, Group
// Effects, Special Power) pick up icons in the same vocabulary.
//
// Colour alone never carries the category — the icon and the name carry it
// too, which is what lets the phone tier drop the word and stay legible.
const CATEGORIES = {
  'Attack': { key: 'attack', icon: Swords },
  'Defense': { key: 'defense', icon: Shield },
  'Skip': { key: 'skip', icon: SkipForward },
  'Super Skip': { key: 'super-skip', icon: SkipForward },
  'Reverse': { key: 'reverse', icon: RotateCcw },
  'Future Vision': { key: 'future', icon: Eye },
  'Nope': { key: 'nope', icon: Hand },
  'Shuffle': { key: 'shuffle', icon: Shuffle },
  'Draw From Bottom': { key: 'draw-bottom', icon: ArrowDownToLine },
  'Swap Top & Bottom': { key: 'swap', icon: ArrowDownUp },
  'Garbage Collection': { key: 'garbage', icon: Trash2 },
  'Catomic Bomb': { key: 'catomic', icon: Bomb },
  'Mark': { key: 'mark', icon: MapPin },
  'Bury': { key: 'bury', icon: Archive },
  'Dig Deeper': { key: 'dig', icon: Pickaxe },
  'Favor': { key: 'favor', icon: Gift },
  'Clone': { key: 'clone', icon: Copy },
  'Explosive': { key: 'explosive', icon: Flame },
  'Group Effects': { key: 'group', icon: Users },
  'Special Power': { key: 'special', icon: Sparkles },
  'Cat Cards': { key: 'cat', icon: PawPrint },
}

const FALLBACK_CATEGORY = { key: 'cat', icon: PawPrint }

function getCategory(cardId) {
  return CATEGORIES[store.getCategoryName(cardId)] || FALLBACK_CATEGORY
}

function getCategoryKey(cardId) {
  return getCategory(cardId).key
}

function getCategoryIcon(cardId) {
  return getCategory(cardId).icon
}

// The badge above already names the category, so the title says which card
// this actually is. Most names in cards.json are "Category - Variant"
// ("Shuffle - Transdimensional Litter Box"), and repeating the category
// under its own badge wastes the only two lines the card has, so the prefix
// is dropped and the variant kept. The 70 names with no prefix — the cat
// cards and the one-of-a-kind actions — are shown whole.
function getCardDisplayName(cardId) {
  const name = store.getCardData(cardId)?.name
  if (!name) return 'Unknown'
  const dash = name.indexOf(' - ')
  return dash === -1 ? name : name.slice(dash + 3)
}
</script>

<style scoped>
.hand-zone {
  position: relative;
  padding: 14px 20px 12px;
  background: var(--surface);
  border-top: var(--edge-w) solid var(--edge);
  z-index: 60;
  flex-shrink: 0;
}

/* ─── Spectating ─── */

.spectating-banner {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 12px 14px;
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  background: var(--dim-fill);
  color: var(--ink);
  font-size: 0.9rem;
  font-weight: 800;
}

.spectating-icon {
  flex-shrink: 0;
  color: var(--broth);
}

.leave-spectate-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  min-height: 44px;
  border-radius: var(--radius-sm);
  border: 2.5px solid var(--edge);
  background: var(--surface);
  color: var(--ink);
  font-size: 0.85rem;
  font-weight: 800;
  font-family: inherit;
  cursor: pointer;
  transition: var(--transition-interactive);
}

.leave-spectate-btn:hover {
  background: var(--broth);
  color: var(--surface);
}

/* ─── Card Hand Wrap ─── */

.hand-cards-wrap {
  position: relative;
  padding-top: 10px;
}

/* ─── Floating Selection Pill ─── */

.selection-pill {
  position: absolute;
  top: -8px;
  left: 0;
  right: 0;
  width: fit-content;
  margin: 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px;
  border-radius: 100px;
  background: var(--surface);
  border: 2.5px solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  z-index: 30;
  white-space: nowrap;
  font-size: 0.78rem;
}

.pill-count {
  color: var(--ink);
  font-weight: 800;
}

.pill-sep {
  width: 2px;
  height: 14px;
  background: var(--edge);
}

.pill-clear {
  background: none;
  border: none;
  color: var(--muted-cream);
  font-size: 0.75rem;
  font-weight: 800;
  font-family: inherit;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
  transition: var(--transition-interactive);
}

.pill-clear:hover {
  color: var(--broth);
}

.pill-combo {
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--ink);
  background: var(--gold);
  padding: 2px 9px;
  border-radius: 100px;
  border: 2px solid var(--edge);
}

.pill-play {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 13px;
  border-radius: 100px;
  border: 2.5px solid var(--edge);
  background: var(--chili);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.78rem;
  font-weight: 800;
  cursor: pointer;
  transition: var(--transition-interactive);
  margin-left: 2px;
}

.pill-play:hover:not(:disabled) {
  background: #ff7448;
}

.pill-play:disabled {
  background: var(--dim-fill);
  border-color: var(--dim-edge);
  color: var(--dim-text);
  cursor: default;
}

.pill-play:active:not(:disabled) {
  transform: translateY(2px);
}

/* Pill transition */
.pill-enter-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.pill-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}
.pill-enter-from {
  opacity: 0;
  transform: translateY(4px);
}
.pill-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

/* ─── Card Hand ─── */

.hand-cards {
  display: flex;
  justify-content: center;
  gap: 0;
  padding: 8px 0 4px;
  min-height: 140px;
  perspective: 800px;
}

.hand-card {
  width: 90px;
  height: 130px;
  background: var(--surface);
  border-radius: var(--radius-sm);
  border: 2.5px solid var(--edge);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 0;
  cursor: pointer;
  flex-shrink: 0;
  margin-left: -40px;
  position: relative;
  transition: transform 0.2s var(--ease-out), box-shadow 0.2s var(--ease-out);
  transform-origin: bottom center;
  transform: rotate(calc((var(--card-index) - var(--total) / 2) * 4deg));
  box-shadow: 2px 3px 0 var(--edge);
  overflow: hidden;
}

.hand-card:first-child {
  margin-left: 0;
}

.hand-card:hover {
  transform: rotate(0deg) translateY(-20px) scale(1.08);
  z-index: 20;
  box-shadow: 0 8px 0 var(--edge);
}

/* Unplayable cards wash out onto the tan rather than dimming, which is
   invisible against a light ground. */
.hand-card.disabled {
  filter: grayscale(0.85) contrast(0.92) brightness(1.04);
  cursor: default;
  pointer-events: none;
}

.hand-card.selected {
  transform: rotate(0deg) translateY(-28px) scale(1.1);
  z-index: 25;
  box-shadow: 0 0 0 4px var(--gold), 0 8px 0 var(--edge);
}

/* ─── Arriving Cards ─── */

/* Held invisible while the ghost card is flying to this slot. Overrides the
   enter transition below, which would otherwise fade it in underneath. */
.hand-card.in-flight {
  opacity: 0 !important;
}

.hand-card.landed {
  animation: cardLanded 0.6s ease-out;
}

@keyframes cardLanded {
  0% { box-shadow: 0 0 0 5px var(--gold), 2px 3px 0 var(--edge); }
  100% { box-shadow: 2px 3px 0 var(--edge); }
}

/* Cards that arrive from somewhere other than the deck — a Favor, a steal —
   get no flight, so they fade up in place instead of blinking into the fan.
   No move transition is defined on purpose: the fan's rotation is a transform
   too, and Vue's FLIP would overwrite it mid-slide. */
.card-enter-active {
  transition: opacity var(--duration-slow) ease, transform var(--duration-slow) ease;
}

.card-leave-active {
  transition: opacity var(--duration-fast) ease, transform var(--duration-fast) ease;
}

.card-enter-from {
  opacity: 0;
  transform: rotate(calc((var(--card-index) - var(--total) / 2) * 4deg)) translateY(18px) scale(0.9);
}

.card-leave-to {
  opacity: 0;
  transform: rotate(calc((var(--card-index) - var(--total) / 2) * 4deg)) translateY(-14px) scale(0.92);
}

/* ─── Category Badge ─── */

.card-badge {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.56rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 4px 5px;
  border-bottom: 2.5px solid var(--edge);
}

.badge-icon {
  flex-shrink: 0;
}

.badge-label {
  line-height: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* One fill per category. Each fill takes whichever of ink or cream reads
   against it — every pair below clears 4.5:1, so no badge depends on the
   reader being able to pick a light word off a light band. */
.badge-attack {
  background: #ef5350;
  color: var(--ink);
}

.badge-defense {
  background: #14b8a6;
  color: var(--ink);
}

.badge-skip {
  background: #3b82f6;
  color: var(--ink);
}

.badge-super-skip {
  background: #4f46e5;
  color: var(--surface);
}

.badge-reverse {
  background: #06b6d4;
  color: var(--ink);
}

.badge-future {
  background: #a78bfa;
  color: var(--ink);
}

.badge-nope {
  background: #4a3726;
  color: var(--surface);
}

.badge-shuffle {
  background: #c026d3;
  color: var(--surface);
}

.badge-draw-bottom {
  background: #f59e0b;
  color: var(--ink);
}

.badge-swap {
  background: #eab308;
  color: var(--ink);
}

.badge-garbage {
  background: #f97316;
  color: var(--ink);
}

.badge-catomic {
  background: #be123c;
  color: var(--surface);
}

.badge-mark {
  background: #ec4899;
  color: var(--ink);
}

.badge-bury {
  background: #ab7c52;
  color: var(--ink);
}

.badge-dig {
  background: #65a30d;
  color: var(--ink);
}

.badge-favor {
  background: #10b981;
  color: var(--ink);
}

.badge-clone {
  background: #0ea5e9;
  color: var(--ink);
}

.badge-explosive {
  background: #991b1b;
  color: var(--surface);
}

.badge-group {
  background: #7c3aed;
  color: var(--surface);
}

.badge-special {
  background: #f472b6;
  color: var(--ink);
}

.badge-cat {
  background: #a16207;
  color: var(--surface);
}

/* ─── Card Icon ─── */

.card-icon-area {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  background: var(--sand);
  overflow: hidden;
}

.card-image {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

/* ─── Card Title ─── */

.card-title {
  width: 100%;
  text-align: center;
  font-size: 0.64rem;
  font-weight: 800;
  color: var(--ink);
  border-top: 2.5px solid var(--edge);
  padding: 5px 4px 6px;
  line-height: 1.2;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
}

/* ─── Nope Hint ─── */

.nope-hint {
  text-align: center;
  font-size: 0.82rem;
  color: var(--chili-deep);
  font-weight: 800;
  padding: 4px 0;
  animation: pulse 1s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}

/* ─── Action Buttons ─── */

.hand-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
  padding-top: 8px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 22px;
  min-height: 48px;
  border-radius: var(--radius-sm);
  font-family: 'Baloo 2', sans-serif;
  font-weight: 800;
  font-size: 0.95rem;
  border: var(--edge-w) solid var(--edge);
  box-shadow: 0 3px 0 var(--edge);
  cursor: pointer;
  transition: var(--transition-interactive);
  white-space: nowrap;
}

.action-btn:active:not(:disabled) {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}

.action-btn:disabled {
  background: var(--dim-fill);
  border-color: var(--dim-edge);
  color: var(--dim-text);
  box-shadow: 0 3px 0 var(--dim-edge);
  cursor: default;
}

.btn-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

/* Pass Button */
.pass-btn {
  background: var(--surface);
  color: var(--ink);
}

.pass-btn:hover:not(:disabled) {
  transform: translateY(-1px);
}

.pass-waiting {
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--ink);
  opacity: 0.75;
  align-self: center;
}

/* Nope Button */
.nope-btn {
  background: var(--broth);
  color: var(--surface);
}

.nope-btn:hover:not(:disabled) {
  background: #b81f1f;
}

.nope-countdown {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 20px;
  padding: 0 6px;
  border-radius: 100px;
  border: 2px solid var(--edge);
  background: var(--surface);
  color: var(--ink);
  font-size: 0.68rem;
  font-weight: 800;
  line-height: 1;
}

.nope-btn:disabled .nope-countdown {
  background: var(--surface);
  border-color: var(--dim-edge);
  color: var(--dim-text);
}

/* Leave Button */
.leave-btn {
  background: var(--surface);
  color: var(--ink);
}

.leave-btn:hover {
  background: var(--sand);
}

/* ─── Responsive ─── */

@media (max-width: 640px) {
  .hand-zone {
    padding: 10px 8px 6px;
  }

  .selection-pill {
    font-size: 0.68rem;
    padding: 4px 9px;
    gap: 5px;
  }

  .pill-play {
    font-size: 0.66rem;
    padding: 3px 9px;
  }

  .hand-cards {
    justify-content: center;
    padding: 6px 0 2px;
    min-height: 96px;
  }

  .hand-card {
    width: 58px;
    height: 84px;
    margin-left: -33px;
    border-width: 2px;
  }

  .hand-card:first-child {
    margin-left: 0;
  }

  .hand-card:hover {
    transform: rotate(0deg) translateY(-12px) scale(1.05);
  }

  .hand-card.selected {
    transform: rotate(0deg) translateY(-16px) scale(1.08);
    box-shadow: 0 0 0 3px var(--gold), 0 6px 0 var(--edge);
  }

  /* At this size the label would be under 8px, so the badge keeps the icon —
     which still carries the category — and drops the word. The card title
     underneath continues to name the card. */
  .card-badge {
    padding: 3px 0;
    border-bottom-width: 2px;
  }

  .badge-label {
    display: none;
  }

  .card-title {
    font-size: 0.5rem;
    padding: 3px 2px 4px;
    border-top-width: 2px;
  }

  .hand-actions {
    padding-top: 4px;
  }

  .action-btn {
    padding: 8px 14px;
    font-size: 0.85rem;
  }
}

/* ─── Desktop ───
   The hand was built at phone size and then simply centred on wide screens,
   which left the cards small and the fan marooned in empty space. From 900px
   the cards grow with the viewport, and the selection pill moves out of the
   fan's way into the gutter beside it instead of floating over the cards. */

@media (min-width: 900px) {
  .hand-zone {
    padding: 16px 24px;
  }

  .selection-pill {
    left: auto;
    right: 28px;
    margin: 0;
    top: 16px;
    font-size: 0.88rem;
    padding: 7px 14px;
    gap: 10px;
  }

  .pill-clear,
  .pill-play {
    min-height: 44px;
  }

  .pill-play {
    padding: 8px 18px;
    font-size: 0.88rem;
  }

  .pill-clear {
    padding: 4px 10px;
    font-size: 0.82rem;
  }

  .hand-cards {
    min-height: 196px;
    padding: 10px 0 16px;
  }

  .hand-card {
    width: 112px;
    height: 162px;
    margin-left: -46px;
  }

  .hand-card:first-child {
    margin-left: 0;
  }

  .hand-card:hover {
    transform: rotate(0deg) translateY(-26px) scale(1.08);
  }

  .hand-card.selected {
    transform: rotate(0deg) translateY(-34px) scale(1.1);
  }

  .card-badge {
    font-size: 0.64rem;
    padding: 5px 0;
    gap: 4px;
  }

  .card-title {
    font-size: 0.76rem;
    padding: 6px 5px 7px;
  }
}

@media (min-width: 1280px) {
  .hand-cards {
    min-height: 224px;
  }

  .hand-card {
    width: 128px;
    height: 186px;
    margin-left: -48px;
  }

  .hand-card:first-child {
    margin-left: 0;
  }

  .card-badge {
    font-size: 0.7rem;
  }

  .card-title {
    font-size: 0.84rem;
  }
}
</style>
