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
  border-radius: var(--radius-sm);
  cursor: pointer;
  border: 2.5px solid var(--edge);
  background: var(--surface);
  box-shadow: 0 3px 0 var(--edge);
  position: relative;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition: transform var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out), filter var(--duration-fast) ease;
  user-select: none;
  overflow: visible;
}

.card-view:not(.card-back):hover {
  transform: translateY(-6px);
  box-shadow: 0 9px 0 var(--edge);
  z-index: var(--z-cards);
}

.card-view:not(.card-back):active {
  transform: translateY(-2px);
  box-shadow: 0 5px 0 var(--edge);
}

/* Selection is a gold ring drawn inside the outline, so the card keeps its
   exact footprint and nothing around it reflows when one is picked. */
.card-view.selected {
  transform: translateY(-6px);
  box-shadow: 0 9px 0 var(--edge), inset 0 0 0 4px var(--gold);
  z-index: var(--z-cards);
}

/* Washed out rather than faded: on a light ground low opacity reads as
   half-erased instead of unavailable. */
.card-view.disabled {
  filter: grayscale(0.85) contrast(0.92) brightness(1.04);
  background: var(--dim-fill);
  border-color: var(--dim-edge);
  box-shadow: 0 3px 0 var(--dim-edge);
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
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.58rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: var(--ink);
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
  background: repeating-conic-gradient(var(--chili) 0% 25%, var(--chili-deep) 0% 50%) 50%/16px 16px;
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
  border-radius: var(--radius-sm);
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(27, 14, 6, 0.32), 0 0 0 2.5px var(--edge);
}

.card-preview img {
  width: 80px;
  height: 112px;
  object-fit: contain;
  display: block;
  background: var(--surface);
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
  font-size: 0.48rem;
}

.card-view.size-mini {
  width: 48px;
  height: 68px;
}

.card-view.size-mini .card-image {
  height: 48px;
}

.card-view.size-mini .card-name {
  font-size: 0.42rem;
}
</style>
