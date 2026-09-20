<template>
  <div class="room-settings" :class="{ collapsed: !expanded }">
    <button class="settings-toggle" data-testid="settings-toggle" @click="expanded = !expanded">
      <span class="toggle-icon"><ChevronDown v-if="expanded" :size="14" /><ChevronRight v-else :size="14" /></span>
      <span class="toggle-label">Room Settings</span>
      <span class="toggle-hint" v-if="!expanded && !isDefault">customized</span>
    </button>

    <div v-if="expanded" class="settings-body">

      <!-- PLAYER COUNT + HAND SIZE -->
      <div class="setting-group setting-group-split">
        <div class="split-half">
          <div class="setting-label">Players</div>
          <div class="setting-row">
            <label class="setting-sublabel">Min</label>
            <div class="stepper">
              <button class="stepper-btn" @click="dec('minPlayers', 2, 6)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.minPlayers }}</span>
              <button class="stepper-btn" @click="inc('minPlayers', 2, settings.maxPlayers)" :disabled="!editable">+</button>
            </div>
            <label class="setting-sublabel">Max</label>
            <div class="stepper">
              <button class="stepper-btn" @click="dec('maxPlayers', settings.minPlayers, 6)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.maxPlayers }}</span>
              <button class="stepper-btn" @click="inc('maxPlayers', settings.minPlayers, 6)" :disabled="!editable">+</button>
            </div>
          </div>
        </div>
        <div class="split-half">
          <div class="setting-label">Starting Hand</div>
          <div class="setting-row">
            <div class="stepper full">
              <button class="stepper-btn" data-testid="handsize-dec" @click="dec('handSize', 4, 12)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.handSize }} cards</span>
              <button class="stepper-btn" data-testid="handsize-inc" @click="inc('handSize', 4, 12)" :disabled="!editable">+</button>
            </div>
          </div>
        </div>
      </div>

      <!-- DEFENSE + EXPLOSIVE CARDS -->
      <div class="setting-group setting-group-split">
        <div class="split-half">
          <div class="setting-label">Defense Cards at Start</div>
          <div class="setting-row">
            <div class="stepper full">
              <button class="stepper-btn" @click="dec('startingDefense', 0, 3)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.startingDefense }}</span>
              <button class="stepper-btn" @click="inc('startingDefense', 0, 3)" :disabled="!editable">+</button>
            </div>
          </div>
        </div>
        <div class="split-half">
          <div class="setting-label">Explosive Cards</div>
          <div class="setting-row">
            <div class="stepper full">
              <button class="stepper-btn" data-testid="explosive-dec" @click="dec('explosiveCount', 1, 10)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.explosiveCount }}</span>
              <button class="stepper-btn" data-testid="explosive-inc" @click="inc('explosiveCount', 1, 10)" :disabled="!editable">+</button>
            </div>
          </div>
          <div class="setting-hint">Default: player count - 1</div>
        </div>
      </div>

      <!-- DECK SIZE + TURN TIMER -->
      <div class="setting-group setting-group-split">
        <div class="split-half">
          <div class="setting-label">Deck Size</div>
          <div class="deck-sizes">
            <button
              v-for="opt in deckSizeOptions"
              :key="opt.value"
              class="deck-size-btn"
              :class="{ active: settings.deckSizeMultiplier === opt.value }"
              :disabled="!editable"
              @click="settings.deckSizeMultiplier = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>
        <div class="split-half">
          <div class="setting-label">Turn Timer</div>
          <div class="deck-sizes">
            <button
              v-for="opt in timerOptions"
              :key="opt.value"
              class="deck-size-btn"
              :class="{ active: settings.turnTimer === opt.value }"
              :disabled="!editable"
              @click="settings.turnTimer = opt.value"
            >{{ opt.label }}</button>
          </div>
        </div>
      </div>

      <!-- ROOM VISIBILITY -->
      <div class="setting-group">
        <div class="setting-label">Room Visibility</div>
        <div class="deck-sizes">
          <button
            class="deck-size-btn"
            :class="{ active: settings.isPublic }"
            :disabled="!editable"
            @click="settings.isPublic = true"
          >Public</button>
          <button
            class="deck-size-btn"
            :class="{ active: !settings.isPublic }"
            :disabled="!editable"
            @click="settings.isPublic = false"
          >Private</button>
        </div>
      </div>

      <!-- SPECTATOR MODE -->
      <div class="setting-group">
        <div class="setting-label-row">
          <span class="setting-label">Spectator Mode</span>
          <label class="toggle-switch">
            <input type="checkbox" v-model="settings.allowSpectators" :disabled="!editable">
            <span class="toggle-track"></span>
          </label>
        </div>
        <div class="setting-hint">Eliminated players can watch the game</div>
      </div>

      <!-- CARD CATEGORIES -->
      <div class="setting-group">
        <div class="setting-label">Card Categories</div>
        <div class="category-list">
          <div
            v-for="(cat, key) in categories"
            :key="key"
            class="category-item"
          >
            <label class="category-toggle">
              <input
                type="checkbox"
                :data-testid="'cat-toggle-' + key"
                :checked="settings.enabledCategories[key]"
                @change="toggleCategory(key)"
                @click.stop
              >
              <span class="cat-icon"><component :is="cat.icon" :size="14" /></span>
              <span class="cat-name">{{ cat.name }}</span>
              <span class="cat-count">{{ cat.cards.length }}</span>
            </label>
          </div>
        </div>
      </div>

    </div>

    <!-- CARD PREVIEW LIGHTBOX -->
    <teleport to="body">
      <div v-if="previewCard" class="card-lightbox" @click="closePreview">
        <div class="lightbox-card" @click.stop>
          <img :src="`/cards/${previewCard.file}`" :alt="previewCard.name">
          <div class="lightbox-name">{{ previewCard.name }}</div>
          <button class="lightbox-close" @click="closePreview"><X :size="14" /></button>
        </div>
      </div>
    </teleport>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import {
  ChevronDown, ChevronRight, X,
  Swords, SkipForward, RotateCcw, Eye, Hand, Shuffle,
  ArrowDownToLine, ArrowDownUp, Trash2, Bomb, MapPin,
  Archive, Pickaxe, Gift, Copy, PawPrint, Flame
} from '@lucide/vue'
import cardsData from '../data/cards.json'

const props = defineProps({
  modelValue: { type: Object, required: true },
  editable: { type: Boolean, default: true },
})

const emit = defineEmits(['update:modelValue'])

const expanded = ref(false)
const expandedCategory = ref(null)
const previewCard = ref(null)

const settings = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})

const defaults = {
  minPlayers: 2,
  maxPlayers: 6,
  handSize: 8,
  startingDefense: 1,
  explosiveCount: -1,
  deckSizeMultiplier: 1,
  turnTimer: 0,
  isPublic: true,
  allowSpectators: false,
  enabledCategories: {
    explosive: true,
    attack: true,
    skip: true,
    super_skip: true,
    reverse: true,
    future_vision: true,
    nope: true,
    shuffle: true,
    draw_from_bottom: true,
    swap_top_and_bottom: true,
    garbage_collection: true,
    catomic_bomb: true,
    mark: true,
    bury: true,
    dig_deeper: true,
    favor: true,
    clone: true,
    special_power: false,
    cat_cards: true,
  },
}

const isDefault = computed(() => {
  if (settings.value.minPlayers !== defaults.minPlayers) return false
  if (settings.value.maxPlayers !== defaults.maxPlayers) return false
  if (settings.value.handSize !== defaults.handSize) return false
  if (settings.value.startingDefense !== defaults.startingDefense) return false
  if (settings.value.explosiveCount !== defaults.explosiveCount) return false
  if (settings.value.deckSizeMultiplier !== defaults.deckSizeMultiplier) return false
  if (settings.value.turnTimer !== defaults.turnTimer) return false
  if (settings.value.isPublic !== defaults.isPublic) return false
  if (settings.value.allowSpectators !== defaults.allowSpectators) return false
  for (const k of Object.keys(defaults.enabledCategories)) {
    if (settings.value.enabledCategories[k] !== defaults.enabledCategories[k]) return false
  }
  return true
})

const categories = {
  attack: { name: 'Attack', icon: Swords, cards: cardsData.categories.attack?.cards || [] },
  skip: { name: 'Skip', icon: SkipForward, cards: cardsData.categories.skip?.cards || [] },
  super_skip: { name: 'Super Skip', icon: SkipForward, cards: cardsData.categories.super_skip?.cards || [] },
  reverse: { name: 'Reverse', icon: RotateCcw, cards: cardsData.categories.reverse?.cards || [] },
  future_vision: { name: 'Future Vision', icon: Eye, cards: cardsData.categories.future_vision?.cards || [] },
  nope: { name: 'Nope', icon: Hand, cards: cardsData.categories.nope?.cards || [] },
  shuffle: { name: 'Shuffle', icon: Shuffle, cards: cardsData.categories.shuffle?.cards || [] },
  draw_from_bottom: { name: 'Draw From Bottom', icon: ArrowDownToLine, cards: cardsData.categories.draw_from_bottom?.cards || [] },
  swap_top_and_bottom: { name: 'Swap Top & Bottom', icon: ArrowDownUp, cards: cardsData.categories.swap_top_and_bottom?.cards || [] },
  garbage_collection: { name: 'Garbage Collection', icon: Trash2, cards: cardsData.categories.garbage_collection?.cards || [] },
  catomic_bomb: { name: 'Catomic Bomb', icon: Bomb, cards: cardsData.categories.catomic_bomb?.cards || [] },
  mark: { name: 'Mark', icon: MapPin, cards: cardsData.categories.mark?.cards || [] },
  bury: { name: 'Bury', icon: Archive, cards: cardsData.categories.bury?.cards || [] },
  dig_deeper: { name: 'Dig Deeper', icon: Pickaxe, cards: cardsData.categories.dig_deeper?.cards || [] },
  favor: { name: 'Favor', icon: Gift, cards: cardsData.categories.favor?.cards || [] },
  clone: { name: 'Clone', icon: Copy, cards: cardsData.categories.clone?.cards || [] },
  cat_cards: { name: 'Cat Cards', icon: PawPrint, cards: cardsData.categories.cat_cards?.cards || [] },
}

const deckSizeOptions = [
  { label: '0.5×', value: 0.5 },
  { label: '1×', value: 1 },
  { label: '1.5×', value: 1.5 },
  { label: '2×', value: 2 },
]

const timerOptions = [
  { label: 'Off', value: 0 },
  { label: '30s', value: 30 },
  { label: '60s', value: 60 },
  { label: '90s', value: 90 },
]

function inc(key, min, max) {
  if (settings.value[key] < max) settings.value[key]++
}

function dec(key, min, max) {
  if (settings.value[key] > min) settings.value[key]--
}

function toggleCategory(key) {
  settings.value.enabledCategories[key] = !settings.value.enabledCategories[key]
}

function togglePreview(key) {
  expandedCategory.value = expandedCategory.value === key ? null : key
}

function openPreview(card) {
  previewCard.value = card
}

function closePreview() {
  previewCard.value = null
}
</script>

<style scoped>
.room-settings {
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-md);
  box-shadow: var(--lift);
  overflow: hidden;
  margin-bottom: 12px;
}

.settings-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 12px 14px;
  min-height: 52px;
  background: var(--gold);
  border: none;
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.95rem;
  font-weight: 800;
  cursor: pointer;
  transition: var(--transition-interactive);
}

/* The header keeps its bottom rule only while the body is open, so a collapsed
   panel reads as one solid chip rather than a lidded box. */
.room-settings:not(.collapsed) .settings-toggle {
  border-bottom: var(--edge-w) solid var(--edge);
}

.settings-toggle:hover {
  background: #ffd469;
}

.toggle-icon {
  display: inline-flex;
  color: var(--ink);
}

.toggle-hint {
  margin-left: auto;
  font-size: 0.68rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--ink);
  background: var(--chili);
  border: 2px solid var(--edge);
  padding: 2px 9px;
  border-radius: 100px;
}

.settings-body {
  padding: 4px 14px 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.setting-group {
  border-top: 2px solid rgba(27, 14, 6, 0.14);
  padding-top: 12px;
}

.setting-group-split {
  display: flex;
  gap: 16px;
}

.setting-group-split .split-half {
  flex: 1;
  min-width: 0;
}

.setting-group-split .setting-row {
  flex-wrap: nowrap;
}

.setting-label {
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--chili-deep);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.setting-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.setting-label-row .setting-label {
  margin-bottom: 0;
}

.setting-sublabel {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--mild-cream);
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.setting-hint {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--muted-cream);
  margin-top: 6px;
}

.stepper {
  display: flex;
  align-items: center;
  gap: 0;
  border: 2.5px solid var(--edge);
  border-radius: var(--radius-sm);
  background: var(--sand);
  overflow: hidden;
}

.stepper.full {
  flex: 1;
}

.stepper-btn {
  width: 44px;
  height: 44px;
  border: none;
  background: var(--surface);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 1.3rem;
  font-weight: 800;
  line-height: 1;
  cursor: pointer;
  transition: var(--transition-interactive);
}

.stepper-btn:hover:not(:disabled) {
  background: var(--chili);
}

.stepper-btn:disabled {
  background: var(--dim-fill);
  color: var(--dim-text);
  cursor: default;
}

.stepper-val {
  flex: 1;
  text-align: center;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.9rem;
  font-weight: 800;
  color: var(--ink);
  min-width: 44px;
  padding: 0 6px;
}

.deck-sizes {
  display: flex;
  gap: 6px;
}

.deck-size-btn {
  flex: 1;
  min-width: 0;
  padding: 10px 2px;
  min-height: 44px;
  border-radius: var(--radius-sm);
  border: 2.5px solid var(--edge);
  background: var(--surface);
  color: var(--ink);
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.8rem;
  font-weight: 800;
  cursor: pointer;
  transition: var(--transition-interactive);
}

.deck-size-btn:hover:not(:disabled) {
  background: var(--sand);
}

/* Selected is a filled chip, not a tinted border — at this size a 2px colour
   change is not a strong enough signal for which option is live. */
.deck-size-btn.active {
  background: var(--chili);
  color: var(--ink);
  box-shadow: inset 0 0 0 2px var(--surface);
}

.deck-size-btn:disabled {
  background: var(--dim-fill);
  border-color: var(--dim-edge);
  color: var(--dim-text);
  cursor: default;
}

.deck-size-btn.active:disabled {
  background: var(--chili);
  border-color: var(--edge);
  color: var(--ink);
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 56px;
  height: 32px;
  cursor: pointer;
  flex-shrink: 0;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-track {
  position: absolute;
  inset: 0;
  background: var(--sand);
  border: 2.5px solid var(--edge);
  border-radius: 100px;
  transition: background var(--duration-normal) ease;
}

.toggle-track::after {
  content: '';
  position: absolute;
  width: 20px;
  height: 20px;
  left: 3px;
  top: 3px;
  background: var(--surface);
  border: 2.5px solid var(--edge);
  border-radius: 50%;
  transition: transform var(--duration-normal) var(--ease-out);
}

.toggle-switch input:checked + .toggle-track {
  background: var(--mint);
}

.toggle-switch input:checked + .toggle-track::after {
  transform: translateX(24px);
}

.toggle-switch input:disabled + .toggle-track {
  background: var(--dim-fill);
  border-color: var(--dim-edge);
}

.toggle-switch input:focus-visible + .toggle-track {
  outline: 3px solid var(--chili);
  outline-offset: 3px;
}

.category-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 6px;
}

.category-item {
  border: 2.5px solid var(--edge);
  border-radius: var(--radius-sm);
  background: var(--surface);
  overflow: hidden;
  transition: var(--transition-interactive);
}

.category-item:has(.category-toggle input:checked) {
  background: var(--mint);
}

.category-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 8px;
  min-height: 44px;
  cursor: pointer;
}

.category-toggle input {
  display: none;
}

.cat-icon {
  display: inline-flex;
  flex-shrink: 0;
  color: var(--ink);
}

.cat-name {
  font-size: 0.76rem;
  font-weight: 800;
  color: var(--ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  min-width: 0;
}

.cat-count {
  flex-shrink: 0;
  font-size: 0.66rem;
  font-weight: 800;
  color: var(--ink);
  background: rgba(27, 14, 6, 0.12);
  border-radius: 100px;
  padding: 1px 7px;
}

/* Locked settings wash out to tan — on a light ground a low opacity reads as
   half-erased rather than unavailable. */
.category-toggle:has(input:disabled) {
  background: var(--dim-fill);
  cursor: default;
}

.category-toggle:has(input:disabled) .cat-name,
.category-toggle:has(input:disabled) .cat-icon,
.category-toggle:has(input:disabled) .cat-count {
  color: var(--dim-text);
}

/* ─── Card preview lightbox ───
   Teleported to <body>, so it relies on the theme living on <html>. */

.card-lightbox {
  position: fixed;
  inset: 0;
  z-index: var(--z-lightbox);
  background: rgba(27, 14, 6, 0.58);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  animation: lbFadeIn 0.15s ease;
}

@keyframes lbFadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

.lightbox-card {
  position: relative;
  background: var(--surface);
  border: var(--edge-w) solid var(--edge);
  border-radius: var(--radius-lg);
  box-shadow: var(--lift-lg);
  padding: 20px;
  text-align: center;
  max-width: 240px;
  width: 100%;
  animation: lbPop 0.2s var(--ease-out-back);
}

@keyframes lbPop {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.lightbox-card img {
  width: 180px;
  height: 250px;
  object-fit: cover;
  border-radius: var(--radius-sm);
  border: 2.5px solid var(--edge);
  background: var(--sand);
  display: block;
  margin: 0 auto;
}

.lightbox-name {
  margin-top: 12px;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.95rem;
  font-weight: 800;
  color: var(--ink);
}

.lightbox-close {
  position: absolute;
  top: -14px;
  right: -14px;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: var(--edge-w) solid var(--edge);
  background: var(--broth);
  color: var(--surface);
  box-shadow: 0 3px 0 var(--edge);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: var(--transition-interactive);
}

.lightbox-close:hover {
  background: #b81f1f;
}

.lightbox-close:active {
  transform: translateY(2px);
  box-shadow: 0 1px 0 var(--edge);
}
</style>
