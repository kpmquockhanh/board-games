<template>
  <Teleport to="body">
    <div v-if="flight" ref="ghostEl" class="draw-flight" :style="ghostStyle" aria-hidden="true">
      <!-- The pile's back, facing the player until the card turns over. -->
      <div class="flight-face flight-back">
        <Flame :size="30" />
      </div>
      <div class="flight-face flight-front">
        <div class="flight-badge">{{ store.getCategoryName(flight.cardId) }}</div>
        <div class="flight-art">
          <img
            v-if="store.getCardData(flight.cardId)"
            :src="`/cards/${store.getCardData(flight.cardId).file}`"
            :alt="''"
          >
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { Flame } from '@lucide/vue'
import { useEkStore } from '../../stores/explodingKitchen'

const store = useEkStore()

const flight = ref(null)
const ghostEl = ref(null)

const ghostStyle = computed(() => {
  if (!flight.value) return {}
  const { to } = flight.value
  return {
    width: `${to.width}px`,
    height: `${to.height}px`,
    // The ghost is placed entirely by the animation's transform, so it starts
    // pinned at the viewport origin.
    transform: frameTransform(flight.value.from.x, flight.value.from.y, flight.value.startScale, 0, 0),
  }
})

const DURATION = 440

function prefersReducedMotion() {
  return window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false
}

function centerOf(rect) {
  return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
}

// Puts the ghost's centre at (cx, cy) — translate moves its top-left corner,
// so half its own size comes back off.
function frameTransform(cx, cy, scale, tilt, flip) {
  const f = flight.value
  const w = f ? f.to.width : 0
  const h = f ? f.to.height : 0
  return `perspective(900px) translate(${cx - w / 2}px, ${cy - h / 2}px) scale(${scale}) rotateZ(${tilt}deg) rotateY(${flip}deg)`
}

/**
 * Flies a card from the draw pile into the slot it now occupies in the hand.
 * Resolves once the ghost has landed, at which point the real card should be
 * revealed — the ghost's last frame is exactly the card's own box, so the
 * handoff is invisible.
 */
async function run(cardId, deckRect, cardRect) {
  if (!deckRect || !cardRect || prefersReducedMotion()) return

  flight.value = {
    cardId,
    from: centerOf(deckRect),
    to: cardRect,
    startScale: deckRect.width / cardRect.width,
  }

  await nextTick()
  const el = ghostEl.value
  if (!el) {
    flight.value = null
    return
  }

  const { from, to, startScale } = flight.value
  const dest = centerOf(to)
  // Arcs up and over rather than sliding flat, and swells slightly at the peak
  // so the card reads as passing nearer the player.
  const midX = (from.x + dest.x) / 2
  const midY = (from.y + dest.y) / 2 - 56
  const midScale = ((startScale + 1) / 2) * 1.18

  const anim = el.animate(
    [
      {
        transform: frameTransform(from.x, from.y, startScale, 0, 0),
        easing: 'cubic-bezier(0.38, 0, 0.6, 1)',
      },
      {
        transform: frameTransform(midX, midY, midScale, -7, 90),
        offset: 0.5,
        easing: 'cubic-bezier(0.25, 0.7, 0.3, 1)',
      },
      {
        transform: frameTransform(dest.x, dest.y, 1, 0, 180),
      },
    ],
    { duration: DURATION, fill: 'forwards' }
  )

  try {
    await anim.finished
  } catch {
    // Cancelled — the hand changed under us, so just drop the ghost.
  }
  flight.value = null
}

defineExpose({ run })
</script>

<style scoped>
.draw-flight {
  position: fixed;
  left: 0;
  top: 0;
  z-index: var(--z-overlay);
  pointer-events: none;
  transform-style: preserve-3d;
  will-change: transform;
}

.flight-face {
  position: absolute;
  inset: 0;
  border-radius: var(--radius-sm);
  overflow: hidden;
  backface-visibility: hidden;
  border: 2.5px solid var(--edge);
  /* The ghost is mid-air, so this is the one shadow in the theme allowed to
     blur — a hard offset would read as a card lying flat on the table. */
  box-shadow: 0 12px 22px rgba(27, 14, 6, 0.35);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

/* Matches the draw pile's back so the card appears to lift off it. */
.flight-back {
  background: var(--chili);
  color: var(--ink);
}

/* Pre-turned, so it faces the player only once the card has flipped. */
.flight-front {
  transform: rotateY(180deg);
  background: var(--surface);
  justify-content: flex-start;
}

.flight-badge {
  width: 100%;
  text-align: center;
  font-family: 'Baloo 2', sans-serif;
  font-size: 0.55rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 4px 0;
  color: var(--ink);
  background: var(--gold);
  border-bottom: 2.5px solid var(--edge);
}

.flight-art {
  flex: 1;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 0;
  background: var(--sand);
}

.flight-art img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
</style>
