import { defineStore } from 'pinia'
import { toast } from 'vue-sonner'
import { useWsStore } from './ws'
import { createRoom, joinRoom, getRoomState, saveRoomState, getPlayers, deleteRoom, leaveRoom } from '../api'

const GAME_CODE = 'hotpot'

export const useHotpotStore = defineStore('hotpot', {
  state: () => ({
    players: [],
    feed: [],
    me: null,
    pollTimer: null,
    roomKey: '',
    roomName: '',
  }),

  getters: {
    playerCount: (state) => state.players.length,
    recentFeed: (state) => state.feed.slice(-25).reverse(),
    _ws: () => useWsStore(),
  },

  actions: {
    async create(roomName) {
      const res = await createRoom(GAME_CODE, roomName)
      if (!res) return null
      this.roomName = res.room_name
      return res.room_key
    },

    async join(player, roomKey) {
      this.me = player
      this.roomKey = roomKey

      const result = await joinRoom(GAME_CODE, roomKey, player.name, player.color)
      if (result.error) {
        return { error: result.error }
      }

      const [roomState, playersData] = await Promise.all([
        getRoomState(GAME_CODE, this.roomKey),
        getPlayers(GAME_CODE, this.roomKey),
      ])
      if (roomState?.feed) this.feed = roomState.feed

      this.players = playersData.players || []

      toast.success('Joined the pot!')

      this._ws.connect(roomKey, player.name)
      this._ws.onMessage(this._handleMessage)
      this._ws.send('join', { name: player.name, color: player.color })

      this._startPoll()

      return { players: result.players }
    },

    async leave() {
      this._stopPoll()

      if (this.me && this.roomKey) {
        await leaveRoom(GAME_CODE, this.roomKey, this.me.name)
      }
      this._ws.send('leave', { name: this.me?.name, color: this.me?.color })
      this._ws.disconnect()
      this.me = null
      this.players = []
      this.feed = []
      this.roomKey = ''
      this.roomName = ''
      toast('Left the table')
    },

    async deleteRoom() {
      this._stopPoll()

      if (this.me && this.roomKey) {
        await deleteRoom(GAME_CODE, this.roomKey, this.me.name)
      }
      this._ws.disconnect()
      this.me = null
      this.players = []
      this.feed = []
      this.roomKey = ''
      this.roomName = ''
      toast('Room deleted')
    },
    async dropItem(item) {
      this._ws.send('drop', { name: this.me.name, color: this.me.color, emoji: item.emoji, itemName: item.name })

      await saveRoomState(GAME_CODE, this.roomKey, 'drop', { name: this.me.name, color: this.me.color, text: item.name, emoji: item.emoji }, this.me.name)
    },

    async cheer(emoji) {
      this._ws.send('cheer', { name: this.me.name, color: this.me.color, emoji })

      await saveRoomState(GAME_CODE, this.roomKey, 'cheer', { name: this.me.name, color: this.me.color, text: emoji }, this.me.name)
    },

    async sendMessage(text) {
      this._ws.send('chat', { name: this.me.name, color: this.me.color, text })

      await saveRoomState(GAME_CODE, this.roomKey, 'chat', { name: this.me.name, color: this.me.color, text }, this.me.name)
    },

    _handleMessage(msg) {
      const { type, payload } = msg

      if (type === 'room_deleted') {
        this._stopPoll()
        this._ws.disconnect()
        this.me = null
        this.players = []
        this.feed = []
        this.roomKey = ''
        this.roomName = ''
        toast('Room has been deleted')
        return
      }

      if (type === 'player_left') {
        const id = Date.now() + '-' + Math.random().toString(36).slice(2, 7)
        const name = msg.player
        const existing = this.players.find((p) => p.name === name)
        this.players = this.players.filter((p) => p.name !== name)
        this.feed.push({ id, name, color: existing?.color || '#888', type: 'leave', text: '', ts: Date.now() })
        return
      }

      if (!payload) return

      let data
      try {
        data = typeof payload === 'string' ? JSON.parse(payload) : payload
      } catch {
        return
      }

      const id = Date.now() + '-' + Math.random().toString(36).slice(2, 7)

      switch (type) {
        case 'join': {
          this.players.push({ name: data.name, color: data.color })
          this.feed.push({ id, name: data.name, color: data.color, type: 'join', text: '', ts: Date.now() })
          break
        }
        case 'leave': {
          this.players = this.players.filter((p) => p.name !== data.name)
          this.feed.push({ id, name: data.name, color: data.color, type: 'leave', text: '', ts: Date.now() })
          break
        }
        case 'drop': {
          this.feed.push({ id, name: data.name, color: data.color, type: 'drop', text: data.itemName, emoji: data.emoji, ts: Date.now() })
          break
        }
        case 'cheer': {
          this.feed.push({ id, name: data.name, color: data.color, type: 'cheer', text: data.emoji, ts: Date.now() })
          break
        }
        case 'chat': {
          this.feed.push({ id, name: data.name, color: data.color, type: 'chat', text: data.text, ts: Date.now() })
          break
        }
      }

      if (this.feed.length > 40) this.feed = this.feed.slice(-40)
    },

    _startPoll() {
      this._stopPoll()
      this.pollTimer = setInterval(async () => {
        const [feedData, playersData] = await Promise.all([
          getRoomState(GAME_CODE, this.roomKey),
          getPlayers(GAME_CODE, this.roomKey),
        ])
        if (feedData?.state) this.feed = feedData.state.feed || []
        if (playersData) this.players = playersData.players || []
      }, 5000)
    },

    _stopPoll() {
      if (this.pollTimer) {
        clearInterval(this.pollTimer)
        this.pollTimer = null
      }
    },
  },
})
