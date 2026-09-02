<template>
  <div
    class="card-view"
    :class="{
      selected: selected,
      disabled: disabled,
      'card-back': back,
      [`size-${size}`]: true,
    }"
    :tabindex="disabled ? -1 : 0"
    :role="back ? undefined : 'button'"
    :aria-pressed="!back ? selected : undefined"
    :aria-label="!back && cardData ? `${categoryName || cardData.name}${selected ? ' (selected)' : ''}` : undefined"
    @click="$emit('click')"
    @keydown.enter="$emit('click')"
    @keydown.space.prevent="$emit('click')"
    @mouseenter="showPreview = true"
    @mouseleave="showPreview = false"
  >
    <template v-if="!back && cardData">
      <div class="card-image">
        <img :src="`/cards/${cardData.file}`" :alt="cardData.name" loading="lazy">
      </div>
      <div class="card-name">{{ categoryName || cardData.name }}</div>
    </template>
    <template v-else-if="back">
      <div class="card-pattern"></div>
    </template>

    <Transition name="preview">
      <div v-if="showPreview && !back && cardData && size === 'normal'" class="card-preview" aria-hidden="true">
        <img :src="`/cards/${cardData.file}`" :alt="cardData.name">
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref } from 'vue'

defineProps({
  cardData: { type: Object, default: null },
  categoryName: { type: String, default: '' },
  selected: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  back: { type: Boolean, default: false },
  size: { type: String, default: 'normal' },
})

defineEmits(['click'])

const showPreview = ref(false)
</script>

<style scoped>
.card-view {
  width: 80px;
  height: 112px;
  border-radius: 10px;
  cursor: pointer;
  border: 2px solid var(--line);
  background: var(--charcoal);
  position: relative;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: transform var(--ease-standard), border-color var(--ease-standard), box-shadow var(--ease-standard);
  user-select: none;
  overflow: visible;
}

.card-view:focus-visible {
  outline: 2px solid var(--chili-orange);
  outline-offset: 2px;
}

.card-view:not(.card-back):hover {
  transform: translateY(-6px);
  border-color: var(--chili-orange);
  z-index: var(--z-cards);
}

.card-view:not(.card-back):active {
  transform: translateY(-2px);
}

.card-view.selected {
  border-color: var(--gold);
  box-shadow: 0 0 20px rgba(238, 194, 92, 0.45), 0 0 40px rgba(238, 194, 92, 0.15);
  transform: translateY(-6px);
  z-index: var(--z-cards);
}

.card-view.disabled {
  opacity: 0.35;
  cursor: default;
  pointer-events: none;
}

.card-image {
  width: 100%;
  height: 86px;
  overflow: hidden;
}

.card-image img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.card-name {
  font-size: 0.55rem;
  letter-spacing: 0.06em;
  color: var(--mild-cream);
  opacity: 0.8;
  text-align: center;
  padding: 0 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
}

.card-pattern {
  width: 100%;
  height: 100%;
  background: repeating-conic-gradient(#2a1d16 0% 25%, var(--charcoal) 0% 50%) 50%/16px 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* Floating preview on hover */
.card-preview {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 50%;
  transform: translateX(-50%) scale(1.8);
  transform-origin: bottom center;
  pointer-events: none;
  z-index: var(--z-cards);
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5), 0 0 0 2px var(--chili-orange);
}

.card-preview img {
  width: 80px;
  height: 112px;
  object-fit: contain;
  display: block;
  background: var(--charcoal);
}

.preview-enter-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.preview-leave-active {
  transition: opacity 0.1s ease, transform 0.1s ease;
}

.preview-enter-from {
  opacity: 0;
  transform: translateX(-50%) scale(1.6);
}

.preview-leave-to {
  opacity: 0;
  transform: translateX(-50%) scale(1.6);
}

/* Size variants */
.card-view.size-small {
  width: 60px;
  height: 84px;
}

.card-view.size-small .card-image {
  height: 62px;
}

.card-view.size-small .card-name {
  font-size: 0.45rem;
}

.card-view.size-mini {
  width: 48px;
  height: 68px;
}

.card-view.size-mini .card-image {
  height: 48px;
}

.card-view.size-mini .card-name {
  font-size: 0.4rem;
}
</style>
