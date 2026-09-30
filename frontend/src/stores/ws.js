import { defineStore } from 'pinia'
import { getSessionId } from '../session'
import { loadSeatToken } from '../seatToken'

function getWsUrl(room, player) {
  // The session id travels with the socket so the server can recognise the one
  // this tab left open before a reload and drop it, instead of reading it as a
  // second player on the seat. The seat token proves the seat is this tab's;
  // it is read on every (re)connect, so a rejoin that issued a new one is used.
  const query = `room=${encodeURIComponent(room)}&player=${encodeURIComponent(player)}&session=${encodeURIComponent(getSessionId())}&token=${encodeURIComponent(loadSeatToken(room))}`
  const env = import.meta.env.VITE_WS_URL
  if (env) {
    return `${env}/ws?${query}`
  }
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws?${query}`
}

export const useWsStore = defineStore('ws', {
  state: () => ({
    connected: false,
    room: '',
    player: '',
  }),

  actions: {
    connect(room, player) {
      this._generation = (this._generation || 0) + 1
      this.room = room
      this.player = player
      if (!this._handlers) this._handlers = []
      if (!this._openHandlers) this._openHandlers = []

      if (this._ws) {
        this._ws.onopen = null
        this._ws.onmessage = null
        this._ws.onclose = null
        this._ws.onerror = null
        this._ws.close()
      }

      const url = getWsUrl(room, player)
      const socket = new WebSocket(url)
      this._ws = socket

      socket.onopen = () => {
        const reopened = this._everConnected === true
        this._everConnected = true
        this.connected = true
        this._reconnectDelay = 1000
        this._openHandlers.forEach((h) => h(reopened))
      }

      socket.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          this._handlers.forEach((h) => h(msg))
        } catch (e) {
          console.error('[ws] parse error:', e)
        }
      }

      socket.onclose = () => {
        this.connected = false
        this._ws = null
        if (!this._intentionalClose) {
          this._scheduleReconnect()
        }
      }

      socket.onerror = (err) => {
        console.error('[ws] error:', err)
      }
    },

    send(type, payload = {}) {
      if (!this._ws || this._ws.readyState !== WebSocket.OPEN) return false
      this._ws.send(JSON.stringify({ type, room: this.room, player: this.player, payload }))
      return true
    },

    onMessage(handler) {
      this._handlers.push(handler)
      return () => {
        this._handlers = this._handlers.filter((h) => h !== handler)
      }
    },

    // Called on every open. The argument is true when the socket had been
    // connected before, i.e. this is a reconnect and state may have been missed.
    onOpen(handler) {
      if (!this._openHandlers) this._openHandlers = []
      this._openHandlers.push(handler)
      return () => {
        this._openHandlers = this._openHandlers.filter((h) => h !== handler)
      }
    },

    // Called before every reconnect, and awaited: whatever has to happen before
    // the socket opens again, such as sitting back down in the room.
    beforeReconnect(handler) {
      if (!this._reconnectHandlers) this._reconnectHandlers = []
      this._reconnectHandlers.push(handler)
      return () => {
        this._reconnectHandlers = this._reconnectHandlers.filter((h) => h !== handler)
      }
    },

    disconnect() {
      this._generation = (this._generation || 0) + 1
      this._intentionalClose = true
      if (this._reconnectTimer) {
        clearTimeout(this._reconnectTimer)
        this._reconnectTimer = null
      }
      if (this._ws) {
        this._ws.onclose = null
        this._ws.close()
        this._ws = null
      }
      this.connected = false
      this._everConnected = false
      this._intentionalClose = false
    },

    _scheduleReconnect() {
      if (this._reconnectTimer) return
      const generation = this._generation
      this._reconnectTimer = setTimeout(async () => {
        this._reconnectTimer = null
        for (const h of this._reconnectHandlers || []) {
          try {
            await h()
          } catch (e) {
            console.error('[ws] before reconnect:', e)
          }
        }
        // The page left the room, or went to another, while that ran.
        if (this._generation !== generation) return
        this.connect(this.room, this.player)
      }, this._reconnectDelay)
      this._reconnectDelay = Math.min(this._reconnectDelay * 2, 10000)
    },
  },
})
