/**
 * Picks a readable foreground for a player-chosen background colour.
 *
 * Seat colours are supplied by whoever joined — and old rooms carry colours
 * chosen before the kitchen palette existed — so no single fixed text colour
 * works across all of them. Ink is unreadable on the dark reds, cream is
 * unreadable on the golds. This picks whichever of the two has more contrast
 * against the actual background.
 */

const INK = '#1b0e06'
const CREAM = '#fffdf6'

function channel(v) {
  const c = v / 255
  return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
}

function luminance(hex) {
  const m = /^#?([0-9a-f]{6})$/i.exec(String(hex).trim())
  if (!m) return null
  const n = parseInt(m[1], 16)
  return (
    0.2126 * channel((n >> 16) & 255) +
    0.7152 * channel((n >> 8) & 255) +
    0.0722 * channel(n & 255)
  )
}

export function readableInk(bg) {
  const l = luminance(bg)
  // Unparseable or missing colour: ink on the theme's light surfaces is the
  // safer guess than cream.
  if (l === null) return INK
  // 0.179 is the luminance at which black and white are equally readable.
  return l > 0.179 ? INK : CREAM
}
