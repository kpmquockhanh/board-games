// The secret the server hands a player when they take a seat. It is what
// proves the seat is theirs: the name alone used to be enough to act as
// someone and to see their hand.
//
// Kept like savedPlayer.js keeps who you are: per tab in sessionStorage, which
// is authoritative, so two players in two tabs of one browser each keep their
// own, with a localStorage copy so the room's link opened in a fresh tab still
// finds it.

const keyFor = (roomKey) => `seat-token-${roomKey}`

export function loadSeatToken(roomKey) {
  if (!roomKey) return ''
  for (const store of [sessionStorage, localStorage]) {
    try {
      const token = store.getItem(keyFor(roomKey))
      if (token) return token
    } catch {}
  }
  return ''
}

export function saveSeatToken(roomKey, token) {
  if (!roomKey || !token) return
  for (const store of [sessionStorage, localStorage]) {
    try {
      store.setItem(keyFor(roomKey), token)
    } catch {}
  }
}

export function forgetSeatToken(roomKey) {
  const key = keyFor(roomKey)
  let mine = ''
  try {
    mine = sessionStorage.getItem(key) || ''
    sessionStorage.removeItem(key)
  } catch {}
  // The shared copy may be another tab's by now; only this tab's goes.
  try {
    if (mine && localStorage.getItem(key) === mine) localStorage.removeItem(key)
  } catch {}
}
