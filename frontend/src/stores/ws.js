import { defineStore } from 'pinia'

function getWsUrl(room, player) {
  const env = import.meta.env.VITE_WS_URL
  if (env) {
    return `${env}/ws?room=${encodeURIComponent(room)}&player=${encodeURIComponent(player)}`
  }
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${location.host}/ws?room=${encodeURIComponent(room)}&player=${encodeURIComponent(player)}`
}

export const useWsStore = defineStore('ws', {
  state: () => ({
    connected: false,
    room: '',
    player: '',
  }),

  actions: {
    connect(room, player) {
      this.room = room
      this.player = player
      if (!this._handlers) this._handlers = []

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
        this.connected = true
        this._reconnectDelay = 1000
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

    disconnect() {
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
      this._intentionalClose = false
    },

    _scheduleReconnect() {
      if (this._reconnectTimer) return
      this._reconnectTimer = setTimeout(() => {
        this._reconnectTimer = null
        this.connect(this.room, this.player)
      }, this._reconnectDelay)
      this._reconnectDelay = Math.min(this._reconnectDelay * 2, 10000)
    },
  },
})
