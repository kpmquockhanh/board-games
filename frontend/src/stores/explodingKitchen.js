import { defineStore } from 'pinia'
import { toast } from 'vue-sonner'
import cardsData from '../data/cards.json'
import { createRoom, joinRoom, getRoomState, saveRoomState, deleteRoom, leaveRoom, getPlayers } from '../api'
import { useWsStore } from './ws'

const GAME_CODE = 'ek'

export function defaultRoomSettings() {
  return {
    minPlayers: 2,
    maxPlayers: 6,
    handSize: 8,
    startingDefense: 1,
    explosiveCount: -1,
    deckSizeMultiplier: 1,
    turnTimer: 0,
    isPublic: true,
    allowSpectators: false,
    enabledCategories: {
      explosive: true,
      attack: true,
      skip: true,
      super_skip: true,
      reverse: true,
      future_vision: true,
      nope: true,
      shuffle: true,
      draw_from_bottom: true,
      swap_top_and_bottom: true,
      garbage_collection: true,
      catomic_bomb: true,
      mark: true,
      bury: true,
      dig_deeper: true,
      favor: true,
      clone: true,
      group_effects: false,
      special_power: false,
      cat_cards: true,
    },
  }
}

function getCardCategory(cardId) {
  for (const [catName, cat] of Object.entries(cardsData.categories)) {
    if (cat.cards.find((c) => c.id === cardId)) return catName
  }
  return 'unknown'
}

function getCardData(cardId) {
  for (const cat of Object.values(cardsData.categories)) {
    const found = cat.cards.find((c) => c.id === cardId)
    if (found) return found
  }
  return null
}

function getCardIdByName(name) {
  for (const cat of Object.values(cardsData.categories)) {
    const found = cat.cards.find((c) => c.name === name)
    if (found) return found.id
  }
  return null
}

const categoryNameMap = {
  attack: 'Attack',
  skip: 'Skip',
  super_skip: 'Super Skip',
  reverse: 'Reverse',
  future_vision: 'Future Vision',
  nope: 'Nope',
  shuffle: 'Shuffle',
  draw_from_bottom: 'Draw From Bottom',
  swap_top_and_bottom: 'Swap Top & Bottom',
  garbage_collection: 'Garbage Collection',
  catomic_bomb: 'Catomic Bomb',
  mark: 'Mark',
  bury: 'Bury',
  dig_deeper: 'Dig Deeper',
  favor: 'Favor',
  clone: 'Clone',
  group_effects: 'Group Effects',
  special_power: 'Special Power',
  cat_cards: 'Cat Cards',
  explosive: 'Explosive',
  defense: 'Defense',
}

function getCategoryName(cardId) {
  const cat = getCardCategory(cardId)
  return categoryNameMap[cat] || cat
}

export const useEkStore = defineStore('explodingKitchen', {
  state: () => ({
    me: null,
    roomKey: '',
    roomName: '',
    phase: 'join',
    lobby: { players: {}, started: false },
    roomState: {
      roomSettings: defaultRoomSettings(),
      gameState: null,
    },
    selectedCardIndices: [],
    turnTimer: null,
    turnTimeLeft: 0,
    modal: { show: false, title: '', desc: '', cards: [], resolve: null },
    defusePositionModal: { show: false, deckLength: 0, resolve: null },
    garbageCollectionModal: { show: false, resolve: null },
    favorModal: { show: false, step: 'target', targetName: null, resolve: null },
    targetSelectModal: { show: false, players: [], resolve: null },
    cardNameInputModal: { show: false, title: '', desc: '', resolve: null },
    discardPickerModal: { show: false, resolve: null },
    localTopCards: null,
    nopeWindow: {
      active: false,
      playerId: null,
      cardId: null,
      cardName: null,
      cat: null,
      expiresAt: null,
      nopeCount: 0,
      lastNopePlayer: null,
    },
    nopeTimeLeft: 0,
    _nopeTimer: null,
    _gameEndShown: false,
    _ws: null,
  }),

  getters: {
    roomSettings: (state) => state.roomState.roomSettings,
    gameState: (state) => state.roomState.gameState,
    isMyTurn: (state) => state.gameState?.turn === state.me?.name,
    myHand: (state) => state.gameState?.players[state.me?.name]?.hand || [],
    drawPileCount: (state) => state.gameState?.deck?.length || 0,
    discardTop: (state) => {
      const d = state.gameState?.discard
      if (!d || d.length === 0) return null
      return getCardData(d[d.length - 1])
    },
    recentlyPlayed: (state) => {
      const d = state.gameState?.discard
      if (!d || d.length === 0) return []
      return d.slice(-5).reverse().map(id => getCardData(id)).filter(Boolean)
    },
    selectedCards: (state) => {
      const hand = state.gameState?.players[state.me?.name]?.hand || []
      return state.selectedCardIndices.map(i => hand[i]).filter(Boolean)
    },
    selectedCombo: (state) => {
      if (state.selectedCardIndices.length < 2) return null
      const hand = state.gameState?.players[state.me?.name]?.hand || []
      const cards = state.selectedCardIndices.map(i => hand[i])

      if (cards.some(c => getCardCategory(c) === 'explosive')) return null

      if (cards.length === 5) {
        let allDifferent = true
        const seenIDs = {}
        const seenCats = {}
        for (let i = 0; i < cards.length; i++) {
          const cat = getCardCategory(cards[i])
          if (cat === 'cat_cards') {
            const cardData = getCardData(cards[i])
            const key = cardData?.id || cards[i]
            if (seenIDs[key]) { allDifferent = false; break }
            seenIDs[key] = true
          } else {
            if (seenCats[cat]) { allDifferent = false; break }
            seenCats[cat] = true
          }
        }
        if (allDifferent) {
          return { category: 'combo_5x', count: 5 }
        }
      }

      const catCounts = {}
      for (let i = 0; i < state.selectedCardIndices.length; i++) {
        const cat = getCardCategory(cards[i])
        if (!catCounts[cat]) catCounts[cat] = []
        catCounts[cat].push(state.selectedCardIndices[i])
      }

      for (const [cat, catIndices] of Object.entries(catCounts)) {
        if (catIndices.length < 2) continue

        if (cat === 'cat_cards') {
          const idGroups = {}
          for (const idx of catIndices) {
            const cardData = getCardData(hand[idx])
            const cardId = cardData?.id || hand[idx]
            if (!idGroups[cardId]) idGroups[cardId] = []
            idGroups[cardId].push(idx)
          }
          for (const idCards of Object.values(idGroups)) {
            if (idCards.length >= 2) {
              return { category: cat, count: idCards.length }
            }
          }
        } else {
          return { category: cat, count: catIndices.length }
        }
      }
      return null
    },
    attackStackCount: (state) => state.gameState?.attackStack || 0,
    playerList: (state) => Object.keys(state.lobby.players),
    turnOrder: (state) => state.gameState?.turnOrder || [],
    winner: (state) => state.gameState?.winner,
    recentLog: (state) => (state.gameState?.log || []).slice(-15).reverse(),
    amISpectating: (state) => {
      if (!state.me || !state.gameState) return false
      const me = state.gameState.players[state.me.name]
      return me && !me.alive && state.roomSettings.allowSpectators
    },
    canStart: (state) => {
      const count = Object.keys(state.lobby.players).length
      return count >= state.roomSettings.minPlayers
    },
    allReady: (state) => {
      const players = Object.values(state.lobby.players)
      if (players.length < state.roomSettings.minPlayers) return false
      return players.every((p) => p.ready)
    },
    canINope: (state) => {
      if (!state.nopeWindow.active || !state.me || !state.gameState) return false
      if (state.nopeWindow.lastNopePlayer === state.me.name) return false
      const myHand = state.gameState.players[state.me.name]?.hand || []
      return myHand.some(c => getCardCategory(c) === 'nope')
    },
  },

  actions: {
    getCardData,
    getCategoryName,

    async create(roomName) {
      const res = await createRoom(GAME_CODE, roomName)
      if (!res) return null
      this.roomKey = res.room_key
      this.roomName = res.room_name
      return res.room_key
    },

    async deleteRoom() {

      if (this.me && this.roomKey) {
        await deleteRoom(GAME_CODE, this.roomKey, this.me.name)
      }
      this._disconnectWs()
      this.me = null
      this.roomKey = ''
      this.roomName = ''
      this.phase = 'join'
      this.lobby = { players: {}, started: false }
      this.roomState = {
        roomSettings: defaultRoomSettings(),
        gameState: null,
      }
      this.selectedCardIndices = []
      this.nopeWindow = { active: false, playerId: null, cardId: null, cardName: null, cat: null, expiresAt: null, nopeCount: 0, lastNopePlayer: null }
      this._stopNopeTimer()
      toast('Room deleted')
    },

    async leaveRoom() {

      this._gameEndShown = false
      if (this.me && this.roomKey) {
        await leaveRoom(GAME_CODE, this.roomKey, this.me.name)
      }
      this._disconnectWs()
      this.me = null
      this.roomKey = ''
      this.roomName = ''
      this.phase = 'join'
      this.lobby = { players: {}, started: false }
      this.roomState = {
        roomSettings: defaultRoomSettings(),
        gameState: null,
      }
      this.selectedCardIndices = []
      this.nopeWindow = { active: false, playerId: null, cardId: null, cardName: null, cat: null, expiresAt: null, nopeCount: 0, lastNopePlayer: null }
      this._stopNopeTimer()
      toast('Left the kitchen')
    },

    async joinLobby(player, roomKey) {
      this.me = player
      this.roomKey = roomKey || this.roomKey
      this.phase = 'lobby'

      const result = await joinRoom(GAME_CODE, this.roomKey, player.name, player.color)
      if (result.error) {
        return { error: result.error }
      }


      const [stateRes, playersData] = await Promise.all([
        getRoomState(GAME_CODE, this.roomKey),
        getPlayers(GAME_CODE, this.roomKey),
      ])

      if (stateRes?.state) {
        const saved = stateRes.state
        this.roomState = {
          roomSettings: saved.roomSettings ? { ...defaultRoomSettings(), ...saved.roomSettings } : defaultRoomSettings(),
          gameState: saved.gameState || null,
        }
      } else {
        this.roomState = {
          roomSettings: defaultRoomSettings(),
          gameState: null,
        }
      }
      console.log('playersData:', { playersData, stateRes })

      this.lobby = { players: {}, started: false }
      if (playersData?.players) {
        for (const p of playersData.players) {
          this.lobby.players[p.player_name] = {
            color: p.color,
            joined: p.joined_at,
            ready: p.ready || false,
          }
        }
      }

      if (!this.lobby.players[player.name]) {
        this.lobby.players[player.name] = { color: player.color, joined: Date.now(), ready: false }
      }

      this._connectWs()
      toast.success('Joined the kitchen!')
      return { players: result.players }
    },

    async updateSettings(newSettings) {
      this.roomState.roomSettings = { ...this.roomState.roomSettings, ...newSettings }
      await this._sendAction('updateSettings', { roomSettings: this.roomState.roomSettings })
    },

    async toggleReady() {
      if (!this.me) return
      await this._sendAction('toggleReady', {})
    },

    async startGame() {
      const names = Object.keys(this.lobby.players)
      if (names.length < this.roomSettings.minPlayers) return

      this._gameEndShown = false

      const data = {
        players: {},
        turnOrder: names,
        handSize: this.roomSettings.handSize,
        defenseCount: this.roomSettings.startingDefense,
        multiplier: this.roomSettings.deckSizeMultiplier || 1,
        enabledCats: this.roomSettings.enabledCategories || {},
      }
      names.forEach((n) => {
        data.players[n] = { color: this.lobby.players[n].color }
      })

      this.lobby.started = true
      this.phase = 'game'

      await this._sendAction('startGame', data)
      toast.success('Game started!')
    },

    async drawCard() {
      if (!this.isMyTurn || !this.gameState) return
      if (this.nopeWindow.active) return
      await this._sendAction('drawCard', { turnTimer: this.roomSettings.turnTimer })
    },

    async playSelected() {
      if (!this.isMyTurn || !this.gameState) return
      if (this.nopeWindow.active) return
      if (this.selectedCardIndices.length === 0) return

      const hand = this.gameState?.players?.[this.me?.name]?.hand
      if (!hand) return
      const cards = this.selectedCardIndices.map(i => hand[i]).filter(Boolean)
      const cats = cards.map(c => getCardCategory(c))

      if (cats.some(c => c === 'explosive')) {
        toast("Can't play an Explosive card!")
        return
      }

      const combo = this.detectCombo(this.selectedCardIndices)
      if (combo) {
        const comboCards = combo.indices.map(i => hand[i]).filter(Boolean)

        if (combo.category === 'combo_5x') {
          const discardCardId = await this._showDiscardPicker()
          if (discardCardId === null) return
          await this._sendAction('playCombo', {
            cards: comboCards,
            discardCardId,
            turnTimer: this.roomSettings.turnTimer,
          })
          this.selectedCardIndices = []
          return
        }

        const alivePlayers = this.turnOrder.filter(
          p => p !== this.me?.name && this.gameState?.players?.[p]?.alive
        )
        if (alivePlayers.length === 0) {
          toast("No valid targets!")
          return
        }

        const target = await this._showTargetPicker(alivePlayers)
        if (!target) return

        if (combo.count >= 3) {
          const cardId = await this._showCardNameInput(combo.category)
          if (cardId === null) return
          await this._sendAction('playCombo', {
            cards: comboCards,
            target,
            cardId,
            turnTimer: this.roomSettings.turnTimer,
          })
        } else {
          await this._sendAction('playCombo', {
            cards: comboCards,
            target,
            turnTimer: this.roomSettings.turnTimer,
          })
        }

        this.selectedCardIndices = []
        return
      }

      await this._sendAction('playCard', { cards, turnTimer: this.roomSettings.turnTimer })
      this.selectedCardIndices = []
    },

    selectCard(idx) {
      const i = this.selectedCardIndices.indexOf(idx)
      if (i >= 0) {
        this.selectedCardIndices.splice(i, 1)
      } else {
        this.selectedCardIndices.push(idx)
      }
    },

    clearSelection() {
      this.selectedCardIndices = []
    },

    _showTargetPicker(players) {
      return new Promise((resolve) => {
        this.targetSelectModal = {
          show: true,
          players,
          resolve: (val) => {
            this.targetSelectModal.show = false
            resolve(val)
          },
        }
      })
    },

    _showCardNameInput(category) {
      let title, desc
      if (category === 'combo_5x') {
        title = 'Name a card to pick from the discard pile'
        desc = 'Type the exact card name. If it\'s not in the discard pile, the combo is wasted.'
      } else {
        const catName = categoryNameMap[category] || category
        title = `Name a ${catName} card to steal`
        desc = 'Type the exact card name. If the target doesn\'t have it, the combo is wasted.'
      }
      return new Promise((resolve) => {
        this.cardNameInputModal = {
          show: true,
          title,
          desc,
          resolve: (val) => {
            this.cardNameInputModal.show = false
            if (val) {
              const cardId = getCardIdByName(val)
              if (cardId) {
                resolve(cardId)
              } else {
                toast('Card not found. Check the name and try again.')
                resolve(null)
              }
            } else {
              resolve(null)
            }
          },
        }
      })
    },

    _showDiscardPicker() {
      return new Promise((resolve) => {
        this.discardPickerModal = {
          show: true,
          resolve: (val) => {
            this.discardPickerModal.show = false
            resolve(val)
          },
        }
      })
    },

    async playNope() {
      if (!this.nopeWindow.active || !this.me || !this.gameState) return
      if (this.nopeWindow.lastNopePlayer === this.me.name) return
      const myHand = this.gameState?.players?.[this.me?.name]?.hand
      if (!myHand) return
      const hasNope = myHand.some(c => getCardCategory(c) === 'nope')
      if (!hasNope) return

      await this._sendAction('playNope', {})
    },

    detectCombo(indices) {
      if (indices.length < 2) return null
      const hand = this.gameState?.players[this.me?.name]?.hand || []
      const cards = indices.map(i => hand[i])

      if (cards.some(c => getCardCategory(c) === 'explosive')) return null

      if (cards.length === 5) {
        let allDifferent = true
        const seenIDs = {}
        const seenCats = {}
        for (let i = 0; i < cards.length; i++) {
          const cat = getCardCategory(cards[i])
          if (cat === 'cat_cards') {
            const cardData = getCardData(cards[i])
            const key = cardData?.id || cards[i]
            if (seenIDs[key]) { allDifferent = false; break }
            seenIDs[key] = true
          } else {
            if (seenCats[cat]) { allDifferent = false; break }
            seenCats[cat] = true
          }
        }
        if (allDifferent) {
          return { category: 'combo_5x', count: 5, indices }
        }
      }

      const catCounts = {}
      for (let i = 0; i < indices.length; i++) {
        const cat = getCardCategory(cards[i])
        if (!catCounts[cat]) catCounts[cat] = []
        catCounts[cat].push(indices[i])
      }

      for (const [cat, catIndices] of Object.entries(catCounts)) {
        if (catIndices.length < 2) continue

        if (cat === 'cat_cards') {
          const idGroups = {}
          for (const idx of catIndices) {
            const cardData = getCardData(hand[idx])
            const cardId = cardData?.id || hand[idx]
            if (!idGroups[cardId]) idGroups[cardId] = []
            idGroups[cardId].push(idx)
          }
          for (const idCards of Object.values(idGroups)) {
            if (idCards.length >= 2) {
              return { category: cat, count: idCards.length, indices: idCards }
            }
          }
        } else {
          return { category: cat, count: catIndices.length, indices: catIndices }
        }
      }
      return null
    },

    closeTopCards() {
      this.localTopCards = null
    },

    async resolveDefuse(useDefuse, position) {
      await this._sendAction('resolveDefuse', { useDefuse, position })
    },

    _startNopeTimer(expiresAt) {
      this._stopNopeTimer()
      const update = () => {
        const left = Math.max(0, Math.ceil((expiresAt - Date.now()) / 1000))
        this.nopeTimeLeft = left
        if (left <= 0) {
          this._stopNopeTimer()
        }
      }
      update()
      this._nopeTimer = setInterval(update, 250)
    },

    _stopNopeTimer() {
      if (this._nopeTimer) {
        clearInterval(this._nopeTimer)
        this._nopeTimer = null
      }
      this.nopeTimeLeft = 0
    },

    _connectWs() {
      if (this._ws) {
        this._ws.disconnect()
      }
      const ws = useWsStore()
      this._ws = ws
      ws.connect(this.roomKey, this.me.name)
      ws.onMessage((msg) => this._handleWsMessage(msg))
    },

    _disconnectWs() {
      if (this._ws) {
        this._ws.disconnect()
        this._ws = null
      }
    },

    _handleWsMessage(msg) {
      console.log('[ws] received message:', msg)
      if (msg.type === 'state_updated') {
        this._handleStateUpdated(msg)
      } else if (msg.type === 'prompt_defuse') {
        this._handleDefusePrompt(msg)
      } else if (msg.type === 'peek_cards') {
        this._handlePeekCards(msg)
      } else if (msg.type === 'player_joined') {
        this._handlePlayerJoined(msg)
      } else if (msg.type === 'player_left') {
        this._handlePlayerLeft(msg)
      } else if (msg.type === 'player_ready') {
        this._handlePlayerReady(msg)
      } else if (msg.type === 'room_settings_updated') {
        this._handleRoomSettingsUpdated(msg)
      } else if (msg.type === 'room_deleted') {
        this._handleRoomDeleted()
      } else if (msg.type === 'game_ended') {
        this._handleGameEnd()
      }
    },

    _handleStateUpdated(msg) {
      let state
      try {
        state = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        return
      }

      if (!state) return

      if (state.players) {
        this.roomState.gameState = state
        if (state.phase === 'playing') {
          this.phase = 'game'
        }
      }

      const gs = this.roomState.gameState
      if (!gs) return

      if (gs.NopeWindow || gs.nopeWindow) {
        const nw = gs.NopeWindow || gs.nopeWindow
        const expiresAt = nw.ExpiredAt || nw.expiredAt
        if (nw && expiresAt) {
          const expiresAtMs = typeof expiresAt === 'string' ? new Date(expiresAt).getTime() : expiresAt
          this.nopeWindow = {
            active: true,
            playerId: nw.PlayerID || nw.playerId,
            cardId: nw.CardID || nw.cardId,
            cardName: nw.CardName || nw.cardName,
            cat: nw.Category || nw.cat,
            expiresAt: expiresAtMs,
            nopeCount: nw.NopeCount || nw.nopeCount || 0,
            lastNopePlayer: nw.LastNopePlayer || nw.lastNopePlayer || null,
          }
          this._startNopeTimer(expiresAtMs)
        }
      } else if (this.nopeWindow.active) {
        this.nopeWindow = { active: false, playerId: null, cardId: null, cardName: null, cat: null, expiresAt: null, nopeCount: 0, lastNopePlayer: null }
        this._stopNopeTimer()
      }

      if (gs.PendingGarbage || gs.pendingGarbage) {
        const pg = gs.PendingGarbage || gs.pendingGarbage
        const myName = this.me?.name
        if (myName && !pg.Responded?.[myName] && !pg.responded?.[myName]) {
          this.garbageCollectionModal = {
            show: true,
            resolve: async (cardId) => {
              this.garbageCollectionModal.show = false
              if (cardId) {
                await this._sendAction('resolveGarbageCollection', { cardId })
              }
            },
          }
        }
      }

      const nopeActive = !!(gs.NopeWindow || gs.nopeWindow)

      if (gs.PendingFavor || gs.pendingFavor) {
        const pf = gs.PendingFavor || gs.pendingFavor
        const myName = this.me?.name
        if (pf.PlayerID === myName && !pf.TargetID && !nopeActive) {
          const alivePlayers = this.turnOrder.filter(
            p => p !== myName && this.gameState?.players?.[p]?.alive && (this.gameState?.players?.[p]?.hand?.length || 0) > 0
          )
          if (alivePlayers.length > 0 && !this.favorModal.show) {
            this.favorModal = {
              show: true,
              step: 'target',
              targetName: null,
              resolve: async (targetId) => {
                this.favorModal.show = false
                if (targetId) {
                  await this._sendAction('resolveFavor', { targetId })
                }
              },
            }
          }
        } else if (pf.TargetID === myName && pf.PlayerID !== myName && !nopeActive) {
          const myHand = this.gameState?.players?.[myName]?.hand || []
          if (myHand.length > 0 && !this.favorModal.show) {
            this.favorModal = {
              show: true,
              step: 'card',
              targetName: pf.TargetID,
              resolve: async (cardId) => {
                this.favorModal.show = false
                if (cardId) {
                  await this._sendAction('resolveFavor', { cardId })
                }
              },
            }
          }
        }
      }

      if (gs.Log && gs.Log.length > 0) {
        const lastLog = gs.Log[gs.Log.length - 1]
        if (lastLog?.Text?.includes('wins the game') || lastLog?.text?.includes('wins the game')) {
          if (!this._gameEndShown) {
            this._gameEndShown = true
            const winner = gs.Winner || gs.winner
            if (winner) {
              toast(winner === this.me?.name ? '🎉 You win!' : `💀 ${winner} won the game!`)
            }
          }
        }
      }
    },

    _handleDefusePrompt(msg) {
      let payload
      try {
        payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        return
      }

      if (payload.player !== this.me?.name) return

      this.modal = {
        show: true,
        title: '💥 Explosive Card!',
        desc: 'You drew an Explosive card! Do you want to use your Defense card to defuse it?',
        cards: [],
        resolve: async (useDefuse) => {
          this.modal.show = false
          if (useDefuse) {
            const deckLength = this.gameState?.deck?.length || 0
            this.defusePositionModal = {
              show: true,
              deckLength,
              resolve: async (position) => {
                this.defusePositionModal.show = false
                await this.resolveDefuse(true, position)
              },
            }
          } else {
            await this.resolveDefuse(false, 0)
          }
        },
      }
    },

    _handlePeekCards(msg) {
      let payload
      try {
        payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        return
      }
      if (payload?.cards) {
        this.localTopCards = payload.cards
      }
    },

    _handleGameEnd() {

      if (!this._gameEndShown) {
        this._gameEndShown = true
        const w = this.gameState?.winner
        if (w) {
          toast(w === this.me?.name ? '🎉 You win!' : `💀 ${w} won the game!`)
        } else {
          toast('💀 Game ended — no winner')
        }
      }
    },

    _handlePlayerJoined(msg) {
      let payload
      try {
        payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        return
      }
      if (!payload) return

      const { player, color, players, colors } = payload
      if (players && Array.isArray(players)) {
        for (const name of players) {
          if (!this.lobby.players[name]) {
            this.lobby.players[name] = {
              color: colors?.[name] || '#888',
              joined: Date.now(),
              ready: false,
            }
          }
        }
      } else if (player) {
        if (!this.lobby.players[player]) {
          this.lobby.players[player] = {
            color: color || '#888',
            joined: Date.now(),
            ready: false,
          }
        }
      }
      if (player && player !== this.me?.name) {
        toast(`${player} joined the kitchen!`)
      }
    },

    _handlePlayerLeft(msg) {
      if (msg.player && this.lobby?.players?.[msg.player]) {
        delete this.lobby.players[msg.player]
        toast(`${msg.player} left the kitchen`)
      }
    },

    _handlePlayerReady(msg) {
      let payload
      try {
        payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        return
      }
      if (!payload) return
      const { player, ready } = payload
      if (player && this.lobby?.players?.[player]) {
        this.lobby.players[player].ready = ready
      }
    },

    _handleRoomSettingsUpdated(msg) {
      let settings
      try {
        settings = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        return
      }
      if (settings) {
        this.roomState.roomSettings = { ...defaultRoomSettings(), ...settings }
      }
    },

    _handleRoomDeleted() {

      this._disconnectWs()
      this.me = null
      this.roomKey = ''
      this.roomName = ''
      this.phase = 'join'
      this.lobby = { players: {}, started: false }
      this.roomState = {
        roomSettings: defaultRoomSettings(),
        gameState: null,
      }
      this.selectedCardIndices = []
      this.nopeWindow = { active: false, playerId: null, cardId: null, cardName: null, cat: null, expiresAt: null, nopeCount: 0, lastNopePlayer: null }
      this._stopNopeTimer()
      toast('Room has been deleted')
    },

    async _sendAction(action, data = {}) {
      const result = await saveRoomState(GAME_CODE, this.roomKey, action, data, this.me.name)
      if (result?.state) {
        const saved = result.state
        if (saved.players) {
          this.roomState.gameState = saved
        } else if (saved.minPlayers !== undefined) {
          this.roomState.roomSettings = { ...defaultRoomSettings(), ...saved }
        }
      }
    },
  },
})
