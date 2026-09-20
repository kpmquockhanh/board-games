// A page reload opens its new socket while the one the old page left behind is
// still registered on the server, so for a moment the returning player looks
// like a second person using their name — which is what used to throw them
// back to the join screen. This id is what tells the two apart: it survives a
// reload, because sessionStorage does, and a different tab or browser gets its
// own, so someone else typing your name is still refused while you are here.
const KEY = 'bg-session-id'

let cached = ''

function newId() {
  if (globalThis.crypto?.randomUUID) return crypto.randomUUID()
  return `s-${Math.random().toString(36).slice(2)}-${Date.now().toString(36)}`
}

export function getSessionId() {
  if (cached) return cached
  try {
    cached = sessionStorage.getItem(KEY) || ''
    if (!cached) {
      cached = newId()
      sessionStorage.setItem(KEY, cached)
    }
  } catch {
    // Private browsing and blocked site data: an id that lasts as long as this
    // page still keeps the socket it opens from fighting with itself.
    cached = newId()
  }
  return cached
}
