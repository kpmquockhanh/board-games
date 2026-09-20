// Who you are in a room, remembered across a reload.
//
// This used to live only in localStorage, which every tab of a browser shares.
// Two people at one table in two tabs — the usual way of trying the game out,
// and of playing it on one machine — overwrote each other's entry under the
// same room key. Reloading either tab then came back as the *other* player,
// asked for a name that player was already using, was refused it, and dropped
// to the join screen: the reload kicked you out of your own game.
//
// sessionStorage is per tab, so that is where the authoritative copy lives. The
// localStorage copy is kept as a fallback, so opening the room's link in a
// fresh tab or window still remembers who you were.

function read(store, key) {
  try {
    const raw = store.getItem(key)
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function loadSavedPlayer(key) {
  return read(sessionStorage, key) || read(localStorage, key)
}

export function saveSavedPlayer(key, player) {
  const raw = JSON.stringify(player)
  try {
    sessionStorage.setItem(key, raw)
  } catch {}
  try {
    localStorage.setItem(key, raw)
  } catch {}
}

export function forgetSavedPlayer(key) {
  try {
    sessionStorage.removeItem(key)
  } catch {}
  try {
    localStorage.removeItem(key)
  } catch {}
}
