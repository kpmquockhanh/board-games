import { getSessionId } from './session'
import { loadSeatToken, saveSeatToken, forgetSeatToken } from './seatToken'
const API_URL = import.meta.env.VITE_API_URL ?? ''

// Every call made as a player carries the token of their seat in the room.
function seatHeaders(roomKey, headers = {}) {
  const token = loadSeatToken(roomKey)
  return token ? { ...headers, 'X-Seat-Token': token } : headers
}

export async function createRoom(gameCode, name) {
  try {
    const res = await fetch(`${API_URL}/api/${encodeURIComponent(gameCode)}/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    })
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

export async function listRooms(gameCode) {
  try {
    const res = await fetch(`${API_URL}/api/${encodeURIComponent(gameCode)}/rooms`)
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

export async function joinRoom(gameCode, roomKey, playerName, color) {
  try {
    const res = await fetch(`${API_URL}/api/${encodeURIComponent(gameCode)}/join`, {
      method: 'POST',
      headers: seatHeaders(roomKey, { 'Content-Type': 'application/json' }),
      body: JSON.stringify({ room_key: roomKey, player_name: playerName, color, session: getSessionId() }),
    })
    const data = await res.json()
    if (!res.ok) return { error: data.error || 'join failed' }
    saveSeatToken(roomKey, data.seat_token)
    return data
  } catch {
    return { error: 'network error' }
  }
}

export async function getRoom(gameCode, roomKey) {
  try {
    const res = await fetch(`${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}`)
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

export async function getRoomTimeline(gameCode, roomKey, limit = 100) {
  try {
    const res = await fetch(
      `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}/timeline?limit=${limit}`
    )
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

// The server builds the view for the seat the token proves, so there is no
// player to name here.
export async function getRoomState(gameCode, roomKey) {
  try {
    const res = await fetch(
      `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}/state`,
      { headers: seatHeaders(roomKey) }
    )
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

export async function saveRoomState(gameCode, roomKey, action, data, player = '') {
  try {
    const res = await fetch(
      `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}/state`,
      {
        method: 'POST',
        headers: seatHeaders(roomKey, { 'Content-Type': 'application/json' }),
        body: JSON.stringify({ action, data, player }),
      }
    )
    const result = await res.json().catch(() => ({}))
    if (!res.ok) return { error: result.error || 'that action was rejected' }
    return { state: result.state ?? null }
  } catch {
    return { error: 'network error' }
  }
}

export async function getPlayers(gameCode, roomKey) {
  try {
    const res = await fetch(
      `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}/players`
    )
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

export async function deleteRoom(gameCode, roomKey) {
  try {
    const url = `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}`
    const res = await fetch(url, { method: 'DELETE', headers: seatHeaders(roomKey) })
    if (!res.ok) return { error: 'failed to delete room' }
    forgetSeatToken(roomKey)
    return await res.json()
  } catch {
    return { error: 'network error' }
  }
}

export async function leaveRoom(gameCode, roomKey, playerName) {
  try {
    const res = await fetch(
      `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}/leave`,
      {
        method: 'POST',
        headers: seatHeaders(roomKey, { 'Content-Type': 'application/json' }),
        body: JSON.stringify({ player_name: playerName }),
      }
    )
    if (!res.ok) return { error: 'failed to leave room' }
    forgetSeatToken(roomKey)
    return await res.json()
  } catch {
    return { error: 'network error' }
  }
}

// Who this browser is signed in as ({ user, providers }). Never makes a guest,
// so it is safe to ask on every page load.
export async function getAuthSession() {
  try {
    const res = await fetch(`${API_URL}/api/auth/session`)
    if (!res.ok) return null
    return await res.json()
  } catch {
    return null
  }
}

// Login leaves the page for the provider's, and comes back to returnTo.
export function loginUrl(provider, returnTo) {
  return `${API_URL}/api/auth/${encodeURIComponent(provider)}/login?return=${encodeURIComponent(returnTo)}`
}

export async function logout() {
  try {
    const res = await fetch(`${API_URL}/api/auth/logout`, { method: 'POST' })
    return res.ok
  } catch {
    return false
  }
}

// The tables a signed-in account is sitting at, to go back to from any
// device. Empty for a guest.
export async function getMySeats() {
  try {
    const res = await fetch(`${API_URL}/api/me/seats`)
    if (!res.ok) return []
    return (await res.json()).seats || []
  } catch {
    return []
  }
}

// The account's finished games, newest first.
export async function getMyMatches() {
  try {
    const res = await fetch(`${API_URL}/api/me/matches`)
    if (!res.ok) return []
    return (await res.json()).matches || []
  } catch {
    return []
  }
}

export async function getPlayedWith() {
  try {
    const res = await fetch(`${API_URL}/api/me/played-with`)
    if (!res.ok) return []
    return (await res.json()).played_with || []
  } catch {
    return []
  }
}
