<template>
  <div class="room-settings" :class="{ collapsed: !expanded }">
    <button class="settings-toggle" @click="expanded = !expanded">
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
              <button class="stepper-btn" @click="dec('handSize', 4, 12)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.handSize }} cards</span>
              <button class="stepper-btn" @click="inc('handSize', 4, 12)" :disabled="!editable">+</button>
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
              <button class="stepper-btn" @click="dec('explosiveCount', 1, 10)" :disabled="!editable">-</button>
              <span class="stepper-val">{{ settings.explosiveCount }}</span>
              <button class="stepper-btn" @click="inc('explosiveCount', 1, 10)" :disabled="!editable">+</button>
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
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 14px;
  overflow: hidden;
  margin-bottom: 12px;
}

.settings-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 10px 14px;
  background: none;
  border: none;
  color: var(--steam-cream);
  font-family: inherit;
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease;
}

.settings-toggle:hover {
  background: rgba(255, 255, 255, 0.03);
}

.toggle-icon {
  font-size: 0.75rem;
  color: var(--gold);
}

.toggle-hint {
  margin-left: auto;
  font-size: 0.72rem;
  font-weight: 400;
  color: var(--chili-orange);
  background: rgba(226, 99, 44, 0.12);
  padding: 2px 8px;
  border-radius: 100px;
}

.settings-body {
  padding: 0 14px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.setting-group {
  border-top: 1px solid var(--line);
  padding-top: 10px;
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
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--gold);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.setting-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.setting-label-row .setting-label {
  margin-bottom: 0;
}

.setting-sublabel {
  font-size: 0.78rem;
  color: var(--mild-cream);
  opacity: 0.7;
}

.setting-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.setting-hint {
  font-size: 0.7rem;
  color: var(--mild-cream);
  opacity: 0.45;
  margin-top: 4px;
}

.stepper {
  display: flex;
  align-items: center;
  gap: 0;
  border: 1px solid var(--line);
  border-radius: 10px;
  overflow: hidden;
}

.stepper.full {
  flex: 1;
}

.stepper-btn {
  width: 34px;
  height: 34px;
  border: none;
  background: rgba(255, 255, 255, 0.04);
  color: var(--steam-cream);
  font-size: 1rem;
  font-family: inherit;
  cursor: pointer;
  transition: background 0.15s ease;
}

.stepper-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.08);
}

.stepper-btn:disabled {
  opacity: 0.3;
  cursor: default;
}

.stepper-val {
  flex: 1;
  text-align: center;
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--steam-cream);
  min-width: 36px;
}

.deck-sizes {
  display: flex;
  gap: 5px;
}

.deck-size-btn {
  flex: 1;
  padding: 6px 0;
  border-radius: 8px;
  border: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.03);
  color: var(--mild-cream);
  font-size: 0.78rem;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  transition: all 0.15s ease;
}

.deck-size-btn:hover:not(:disabled) {
  border-color: var(--chili-orange);
}

.deck-size-btn.active {
  background: rgba(226, 99, 44, 0.15);
  border-color: var(--chili-orange);
  color: var(--chili-orange);
}

.deck-size-btn:disabled {
  opacity: 0.4;
  cursor: default;
}

.toggle-switch {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-track {
  position: absolute;
  inset: 0;
  background: rgba(255, 255, 255, 0.1);
  border-radius: 100px;
  transition: background 0.2s ease;
}

.toggle-track::after {
  content: '';
  position: absolute;
  width: 16px;
  height: 16px;
  left: 3px;
  bottom: 3px;
  background: var(--mild-cream);
  border-radius: 50%;
  transition: transform 0.2s ease;
}

.toggle-switch input:checked + .toggle-track {
  background: var(--chili-orange);
}

.toggle-switch input:checked + .toggle-track::after {
  transform: translateX(18px);
}

.toggle-switch input:disabled + .toggle-track {
  opacity: 0.4;
}

.category-list {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 4px;
}

.category-item {
  border: 1px solid var(--line);
  border-radius: 8px;
  overflow: hidden;
  transition: border-color 0.15s ease;
}

.category-item:hover {
  border-color: var(--chili-orange);
}

.category-item:has(.category-toggle input:checked) {
  border-color: var(--chili-orange);
  background: rgba(226, 99, 44, 0.05);
}

.category-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  cursor: pointer;
  transition: background 0.15s ease;
}

.category-toggle:hover {
  background: rgba(255, 255, 255, 0.02);
}

.category-toggle input {
  display: none;
}

.cat-icon {
  font-size: 0.85rem;
}

.cat-name {
  font-size: 0.75rem;
  color: var(--steam-cream);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.cat-count {
  font-size: 0.65rem;
  color: var(--mild-cream);
  opacity: 0.4;
  font-weight: 600;
}

.cat-preview-btn {
  width: 22px;
  height: 22px;
  border: none;
  background: rgba(255, 255, 255, 0.05);
  color: var(--mild-cream);
  border-radius: 4px;
  font-size: 0.65rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s ease;
  flex-shrink: 0;
}

.cat-preview-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.1);
}

.cat-preview-btn:disabled {
  opacity: 0.3;
  cursor: default;
}

.category-preview {
  padding: 8px 10px 10px;
  border-top: 1px solid var(--line);
  background: rgba(0, 0, 0, 0.15);
}

.preview-scroll {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  padding-bottom: 4px;
  scrollbar-width: thin;
  scrollbar-color: var(--line) transparent;
}

.preview-scroll::-webkit-scrollbar {
  height: 4px;
}

.preview-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.preview-scroll::-webkit-scrollbar-thumb {
  background: var(--line);
  border-radius: 4px;
}

.preview-card {
  flex-shrink: 0;
  width: 48px;
  text-align: center;
  cursor: pointer;
  transition: transform 0.15s ease;
}

.preview-card:hover {
  transform: translateY(-3px);
}

.preview-card img {
  width: 48px;
  height: 66px;
  object-fit: cover;
  border-radius: 4px;
  border: 1px solid var(--line);
  display: block;
}

.preview-card-name {
  display: block;
  font-size: 0.38rem;
  color: var(--mild-cream);
  opacity: 0.6;
  margin-top: 3px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.category-toggle:has(input:disabled) {
  opacity: 0.4;
  cursor: default;
}

/* CARD PREVIEW LIGHTBOX */
.card-lightbox {
  position: fixed;
  inset: 0;
  z-index: var(--z-lightbox);
  background: rgba(21, 15, 12, 0.85);
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
  background: var(--panel);
  border: 1px solid var(--line);
  border-radius: 16px;
  padding: 20px;
  text-align: center;
  max-width: 220px;
  width: 100%;
  animation: lbPop 0.2s ease;
}

@keyframes lbPop {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.lightbox-card img {
  width: 180px;
  height: 250px;
  object-fit: cover;
  border-radius: 10px;
  border: 1px solid var(--line);
  display: block;
  margin: 0 auto;
}

.lightbox-name {
  margin-top: 12px;
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--steam-cream);
}

.lightbox-close {
  position: absolute;
  top: 10px;
  right: 10px;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 1px solid var(--line);
  background: rgba(21, 15, 12, 0.8);
  color: var(--mild-cream);
  font-size: 0.75rem;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.15s ease, color 0.15s ease;
}

.lightbox-close:hover {
  border-color: var(--broth-red);
  color: var(--broth-red);
}
</style>
