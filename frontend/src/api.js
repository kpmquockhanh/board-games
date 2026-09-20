import { getSessionId } from './session'
const API_URL = import.meta.env.VITE_API_URL ?? ''

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
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ room_key: roomKey, player_name: playerName, color, session: getSessionId() }),
    })
    const data = await res.json()
    if (!res.ok) return { error: data.error || 'join failed' }
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

export async function getRoomState(gameCode, roomKey, playerName = '') {
  try {
    const player = playerName ? `?player=${encodeURIComponent(playerName)}` : ''
    const res = await fetch(
      `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}/state${player}`
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
        headers: { 'Content-Type': 'application/json' },
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

export async function deleteRoom(gameCode, roomKey, playerName) {
  try {
    const url = `${API_URL}/api/${encodeURIComponent(gameCode)}/rooms/${encodeURIComponent(roomKey)}`
    const query = playerName ? `?player=${encodeURIComponent(playerName)}` : ''
    const res = await fetch(url + query, { method: 'DELETE' })
    if (!res.ok) return { error: 'failed to delete room' }
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
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ player_name: playerName }),
      }
    )
    if (!res.ok) return { error: 'failed to leave room' }
    return await res.json()
  } catch {
    return { error: 'network error' }
  }
}
