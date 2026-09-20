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

// Resolves what the player typed to a card id. The server matches by card type,
// so any card of the named kind will do — which is just as well, since most
// kinds exist under several ids with different names on them.
function getCardIdByName(name) {
  const needle = String(name || '').trim().toLowerCase()
  if (!needle) return null

  for (const cat of Object.values(cardsData.categories)) {
    const found = cat.cards.find((c) => c.name.toLowerCase() === needle)
    if (found) return found.id
  }

  // What the card does, rather than the joke printed on it: "Attack", "Nope".
  for (const [key, cat] of Object.entries(cardsData.categories)) {
    const label = (categoryNameMap[key] || key).toLowerCase()
    if (label === needle && cat.cards.length > 0) return cat.cards[0].id
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
  cat_cards: 'Cat Cards',
  explosive: 'Explosive',
  defense: 'Defense',
}

function getCategoryName(cardId) {
  const cat = getCardCategory(cardId)
  return categoryNameMap[cat] || cat
}

// A window nobody is answering. Kept in one place because the store clears it
// from half a dozen spots (leaving, being kicked, a new game).
function emptyNopeWindow() {
  return {
    active: false,
    playerId: null,
    cardId: null,
    cardName: null,
    cat: null,
    expiresAt: null,
    nopeCount: 0,
    lastNopePlayer: null,
    passed: [],
  }
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
    turnTimeLeft: 0,
    modal: { show: false, title: '', desc: '', cards: [], resolve: null },
    positionModal: { show: false, title: '', desc: '', deckLength: 0, resolve: null },
    chooseCardModal: { show: false, title: '', desc: '', cards: [], resolve: null },
    garbageCollectionModal: { show: false, resolve: null },
    favorModal: { show: false, step: 'target', targetName: null, resolve: null },
    targetSelectModal: { show: false, players: [], resolve: null },
    cardNameInputModal: { show: false, title: '', desc: '', resolve: null },
    discardPickerModal: { show: false, resolve: null },
    localTopCards: null,
    nopeWindow: emptyNopeWindow(),
    nopeTimeLeft: 0,
    offlinePlayers: [],
    // Stable identities for the cards in my hand. Card ids repeat, so the
    // position in the hand is the only thing that tells two Skips apart —
    // which is exactly what breaks when a card is drawn or played. These
    // uids survive the reshuffle so Vue can animate a card rather than
    // patching a new image into an existing node.
    handSlots: [],
    // The card currently flying from the deck into my hand, if any.
    drawFlight: null,
    _uidSeq: 0,
    _nopeTimer: null,
    _turnTimer: null,
    _gameEndShown: false,
    _ws: null,
  }),

  getters: {
    roomSettings: (state) => state.roomState.roomSettings,
    gameState: (state) => state.roomState.gameState,
    isMyTurn: (state) => state.gameState?.turn === state.me?.name,
    myHand: (state) => state.gameState?.players[state.me?.name]?.hand || [],
    drawPileCount: (state) => state.gameState?.deckCount || 0,
    discardTop: (state) => {
      const d = state.gameState?.discard
      if (!d || d.length === 0) return null
      return getCardData(d[d.length - 1])
    },
    recentlyPlayed: (state) => {
      const d = state.gameState?.discard
      if (!d || d.length === 0) return []
      // The position in the discard pile is carried along as a key: keying by
      // the loop index makes Vue patch a new image into the node that is
      // already on top, so a card landing on the pile can never animate.
      return d
        .slice(-5)
        .map((id, i) => ({ card: getCardData(id), key: d.length - 5 + i }))
        .filter(e => e.card)
        .reverse()
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
    // More than one means the deck ran out and the survivors shared the win.
    winners: (state) => state.gameState?.winners || (state.gameState?.winner ? [state.gameState.winner] : []),
    didIWin: (state) => {
      const list = state.gameState?.winners || (state.gameState?.winner ? [state.gameState.winner] : [])
      return !!state.me && list.includes(state.me.name)
    },
    amIEliminated: (state) => {
      if (!state.me || !state.gameState) return false
      return state.gameState.players?.[state.me.name]?.alive === false
    },
    isOffline: (state) => (name) => state.offlinePlayers.includes(name),
    // The hand as the fan renders it: each card with the stable uid of the
    // slot it lives in. Falls back to the index if the slots have not been
    // synced yet, which only happens on the very first paint.
    handCards: (state) => {
      const hand = state.gameState?.players[state.me?.name]?.hand || []
      return hand.map((cardId, i) => ({
        cardId,
        uid: state.handSlots[i]?.uid ?? `i${i}`,
      }))
    },
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
    // Whether the open window is still mine to answer — with a Nope or with a
    // pass. Deliberately says nothing about my cards: everyone who may answer
    // gets the same prompt, so a window that closes early never reveals who
    // was holding a Nope.
    canIAnswerNope: (state) => {
      if (!state.nopeWindow.active || !state.me || !state.gameState) return false
      const me = state.gameState.players[state.me.name]
      if (!me || me.alive === false) return false
      if (state.nopeWindow.lastNopePlayer === state.me.name) return false
      // You may counter a Nope played on your card, but not Nope yourself.
      if (state.nopeWindow.playerId === state.me.name && state.nopeWindow.nopeCount === 0) return false
      return !state.nopeWindow.passed.includes(state.me.name)
    },
    canINope() {
      if (!this.canIAnswerNope) return false
      const myHand = this.gameState.players[this.me.name]?.hand || []
      return myHand.some(c => getCardCategory(c) === 'nope')
    },
    // How many players the window is still waiting on, for the countdown
    // banner. A head count, never a hand count.
    nopeWaitingCount: (state) => {
      if (!state.nopeWindow.active || !state.gameState) return 0
      const players = state.gameState.players || {}
      return Object.entries(players).filter(([name, p]) => {
        if (!p || p.alive === false) return false
        if (name === state.nopeWindow.lastNopePlayer) return false
        if (name === state.nopeWindow.playerId && state.nopeWindow.nopeCount === 0) return false
        return !state.nopeWindow.passed.includes(name)
      }).length
    },
  },

  actions: {
    getCardData,
    getCategoryName,

    categoryLabel(cat) {
      if (cat === 'combo_5x') return 'Rainbow'
      return categoryNameMap[cat] || cat
    },

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
      this.handSlots = []
      this.drawFlight = null
      this.nopeWindow = emptyNopeWindow()
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
      this.handSlots = []
      this.drawFlight = null
      this.nopeWindow = emptyNopeWindow()
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
        getRoomState(GAME_CODE, this.roomKey, player.name),
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
      this.handSlots = []
      this.drawFlight = null
      this._syncHandSlots()

      // A page that reloads mid-game must land back at the table, not in the
      // lobby. The restored snapshot already says whether a game is running,
      // so the phase comes from it rather than from waiting for the socket to
      // push one: a lobby screen that needs a Ready click to get past is not a
      // reconnect.
      if (this.roomState.gameState?.phase) {
        this.phase = 'game'
      }

      this.lobby = { players: {}, started: false }
      this.offlinePlayers = []
      if (playersData?.players) {
        for (const p of playersData.players) {
          this.lobby.players[p.player_name] = {
            color: p.color,
            joined: p.joined_at,
            ready: p.ready || false,
          }
          // A page that has just loaded was never told about the drops that
          // happened before it, so it takes them from the roster.
          if (p.disconnected_at && p.player_name !== player.name) {
            this.offlinePlayers.push(p.player_name)
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
        title = `Name a card to steal with your ${catName} combo`
        desc = 'Name the kind of card you want — "Attack", "Nope", "Tacocat". If the target has none, the combo is wasted.'
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
      if (!this.canINope) return
      await this._sendAction('playNope', {})
    },

    // Letting the play through. Passing is what closes the window early, so
    // everyone is offered it — holding a Nope has nothing to do with it.
    async passNope() {
      if (!this.canIAnswerNope) return
      // Optimistic, so the button stops asking before the server answers.
      this.nopeWindow.passed = [...this.nopeWindow.passed, this.me.name]
      await this._sendAction('passNope', {})
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

    // Bury and Dig Deeper each stop for a decision only this player can make.
    _openChoice(choice) {
      const kind = choice.Kind || choice.kind
      if (kind === 'bury') {
        if (this.positionModal.show) return
        this.positionModal = {
          show: true,
          title: 'Bury a card',
          desc: 'You took the top card without looking at it. Choose where it goes back:',
          deckLength: choice.DeckSize ?? choice.deckSize ?? 0,
          resolve: async (position) => {
            this.positionModal.show = false
            await this._sendAction('resolveChoice', { position })
          },
        }
        return
      }

      if (this.chooseCardModal.show) return
      this.chooseCardModal = {
        show: true,
        title: 'Dig Deeper',
        desc: 'Keep one of these cards. The rest go back on top of the deck, in the same order.',
        cards: choice.Cards || choice.cards || [],
        resolve: async (index) => {
          this.chooseCardModal.show = false
          await this._sendAction('resolveChoice', { index })
        },
      }
    },

    // Back to the lobby with the same people and the same settings.
    async requestRematch() {
      // The reset itself comes back over the socket, so every client at the
      // table goes to the lobby together.
      await this._sendAction('rematch', {})
    },

    _handleRematch() {
      this._gameEndShown = false
      this._stopNopeTimer()
      this._stopTurnCountdown()
      this.roomState.gameState = null
      this.phase = 'lobby'
      this.selectedCardIndices = []
      this.handSlots = []
      this.drawFlight = null
      this.offlinePlayers = []
      this.localTopCards = null
      this.modal.show = false
      this.positionModal.show = false
      this.chooseCardModal.show = false
      this.favorModal.show = false
      this.garbageCollectionModal.show = false
      this.targetSelectModal.show = false
      this.cardNameInputModal.show = false
      this.discardPickerModal.show = false
      this.nopeWindow = emptyNopeWindow()
      for (const p of Object.values(this.lobby.players)) p.ready = false
      this.lobby.started = false
      this._refreshPlayers()
      toast('Rematch! Ready up when you are.')
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

    _startTurnCountdown(endsAt) {
      this._stopTurnCountdown()
      const update = () => {
        const left = Math.max(0, Math.ceil((endsAt - Date.now()) / 1000))
        this.turnTimeLeft = left
        if (left <= 0) this._stopTurnCountdown()
      }
      update()
      this._turnTimer = setInterval(update, 250)
    },

    _stopTurnCountdown() {
      if (this._turnTimer) {
        clearInterval(this._turnTimer)
        this._turnTimer = null
      }
      this.turnTimeLeft = 0
    },

    _stopNopeTimer() {
      if (this._nopeTimer) {
        clearInterval(this._nopeTimer)
        this._nopeTimer = null
      }
      this.nopeTimeLeft = 0
    },

    _connectWs() {
      this._disconnectWs()
      const ws = useWsStore()
      this._ws = ws
      ws.connect(this.roomKey, this.me.name)
      // Keep the unsubscribers: leaving and rejoining a room otherwise stacks
      // another copy of every handler on the shared socket store.
      this._wsOff = [
        ws.onMessage((msg) => this._handleWsMessage(msg)),
        ws.onOpen((reopened) => { if (reopened) this._rejoinAfterReconnect() }),
      ]
    },

    _disconnectWs() {
      if (this._wsOff) {
        this._wsOff.forEach((off) => off())
        this._wsOff = null
      }
      if (this._ws) {
        this._ws.disconnect()
        this._ws = null
      }
    },

    // A dropped socket takes the player out of the room on the server, so every
    // action afterwards is refused until they are put back in it.
    async _rejoinAfterReconnect() {
      if (!this.me || !this.roomKey) return

      const result = await joinRoom(GAME_CODE, this.roomKey, this.me.name, this.me.color)
      if (result?.error && !/already connected/i.test(result.error)) {
        toast.error(`Couldn't rejoin: ${result.error}`)
        return
      }

      // The server pushes the game state on connect; this is for the room
      // settings, which it does not.
      const stateRes = await getRoomState(GAME_CODE, this.roomKey, this.me.name)
      if (stateRes?.state?.roomSettings) {
        this.roomState.roomSettings = { ...defaultRoomSettings(), ...stateRes.state.roomSettings }
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
      } else if (msg.type === 'player_disconnected') {
        this._handlePlayerDisconnected(msg)
      } else if (msg.type === 'player_reconnected') {
        this._handlePlayerReconnected(msg)
      } else if (msg.type === 'player_ready') {
        this._handlePlayerReady(msg)
      } else if (msg.type === 'room_settings_updated') {
        this._handleRoomSettingsUpdated(msg)
      } else if (msg.type === 'room_deleted') {
        this._handleRoomDeleted()
      } else if (msg.type === 'game_ended') {
        this._handleGameEnd()
      } else if (msg.type === 'rematch') {
        this._handleRematch()
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
        // Selections are hand positions, so anything that reshuffles the hand
        // (a draw, a steal, a Favor) would leave them pointing at other cards.
        const before = this.myHand.join(',')
        const prevDeckCount = this.drawPileCount
        this.roomState.gameState = state
        this._syncHandSlots(prevDeckCount)
        if (this.myHand.join(',') !== before) {
          this.selectedCardIndices = []
        }
        if (state.phase === 'playing') {
          this.phase = 'game'
        }
      }

      const gs = this.roomState.gameState
      if (!gs) return

      const turnEndsAt = gs.turnEndsAt ? new Date(gs.turnEndsAt).getTime() : null
      if (turnEndsAt) {
        this._startTurnCountdown(turnEndsAt)
      } else {
        this._stopTurnCountdown()
      }

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
            passed: nw.Passed || nw.passed || [],
          }
          this._startNopeTimer(expiresAtMs)
        }
      } else if (this.nopeWindow.active) {
        this.nopeWindow = emptyNopeWindow()
        this._stopNopeTimer()
      }

      const pendingChoice = gs.PendingChoice || gs.pendingChoice
      if (pendingChoice && pendingChoice.PlayerID === this.me?.name) {
        this._openChoice(pendingChoice)
      } else if (!pendingChoice) {
        this.positionModal.show = this.positionModal.show && !!(gs.PendingDefuse || gs.pendingDefuse)
        this.chooseCardModal.show = false
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
      } else if (this.garbageCollectionModal.show) {
        // Same as the Favor picker below: once the round is resolved the
        // prompt is stale, and answering it sends an action the server drops.
        this.garbageCollectionModal.show = false
      }

      const nopeActive = !!(gs.NopeWindow || gs.nopeWindow)

      if (gs.PendingFavor || gs.pendingFavor) {
        const pf = gs.PendingFavor || gs.pendingFavor
        const myName = this.me?.name
        if (pf.PlayerID === myName && !pf.TargetID && !nopeActive) {
          const alivePlayers = this.turnOrder.filter(
            p => p !== myName && this.gameState?.players?.[p]?.alive && (this.gameState?.players?.[p]?.handCount || 0) > 0
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
      } else if (this.favorModal.show) {
        // The Favor is settled — the target chose, or the prompt timed out and
        // a card was given for them. Leaving the picker up asks for a card that
        // has already been handed over, and the click does nothing.
        this.favorModal.show = false
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
            this.positionModal = {
              show: true,
              title: 'Place the Explosive',
              desc: 'Choose where to put the Explosive card back in the deck:',
              deckLength: this.gameState?.deckCount || 0,
              resolve: async (position) => {
                this.positionModal.show = false
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
      if (this._gameEndShown) return
      this._gameEndShown = true
      this._stopTurnCountdown()

      const winners = this.winners
      if (this.didIWin) {
        toast(winners.length > 1 ? '🏁 You survived the deck — shared win!' : '🎉 You win!')
      } else if (winners.length === 1) {
        toast(`💀 ${winners[0]} won the game!`)
      } else if (winners.length > 1) {
        toast(`🏁 The deck ran out — ${winners.join(', ')} survive`)
      } else {
        toast('💀 Game ended — no winner')
      }
    },

    // Rebuilds the roster from the server. Every client then lists players in
    // the same order, so they agree on who the host is — it used to depend on
    // the order each client happened to hear about people joining.
    async _refreshPlayers() {
      if (!this.roomKey) return
      const data = await getPlayers(GAME_CODE, this.roomKey)
      if (!data?.players) return
      const roster = {}
      for (const p of data.players) {
        roster[p.player_name] = {
          color: p.color,
          joined: p.joined_at,
          ready: p.ready || false,
        }
      }
      this.lobby.players = roster
      // The server is the authority on who is away. A tab that has just
      // reloaded starts with an empty list and was never told about the drops
      // that happened while it was gone, so take them from the roster.
      this.offlinePlayers = data.players
        .filter((p) => p.disconnected_at && p.player_name !== this.me?.name)
        .map((p) => p.player_name)
    },

    _handlePlayerJoined(msg) {
      let payload
      try {
        payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
      } catch {
        payload = null
      }
      const player = payload?.player || msg.player

      if (player) {
        this.offlinePlayers = this.offlinePlayers.filter((name) => name !== player)
      }
      this._refreshPlayers()

      if (player && player !== this.me?.name) {
        toast(`${player} joined the kitchen!`)
      }
    },

    _handlePlayerLeft(msg) {
      this._refreshPlayers()
      if (msg.player && msg.player !== this.me?.name) {
        toast(`${msg.player} left the kitchen`)
      }
    },

    // Dropped mid-game: they keep their seat and their cards, so say so rather
    // than announcing that they left.
    _handlePlayerDisconnected(msg) {
      if (!msg.player || msg.player === this.me?.name) return
      if (!this.offlinePlayers.includes(msg.player)) {
        this.offlinePlayers.push(msg.player)
      }
      toast(`${msg.player} lost connection`)
    },

    // They are back — from a reload, or from whatever took their connection
    // away. Nothing else clears the offline mark, so without this the rest of
    // the table keeps them greyed out for the rest of the game.
    _handlePlayerReconnected(msg) {
      if (!msg.player) return
      const wasOffline = this.offlinePlayers.includes(msg.player)
      this.offlinePlayers = this.offlinePlayers.filter((name) => name !== msg.player)
      if (wasOffline && msg.player !== this.me?.name) {
        toast(`${msg.player} is back`)
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
      this.handSlots = []
      this.drawFlight = null
      this.nopeWindow = emptyNopeWindow()
      this._stopNopeTimer()
      toast('Room has been deleted')
    },

    // Re-pairs the hand with its stable slot uids after the server sends new
    // state, and notices when a card arrived from the deck so the table can
    // fly it into the fan.
    _syncHandSlots(prevDeckCount) {
      const hand = this.gameState?.players[this.me?.name]?.hand || []
      const prev = this.handSlots
      const taken = new Array(prev.length).fill(false)
      const next = new Array(hand.length).fill(null)

      // A card still sitting where it was keeps its slot. Drawing appends, so
      // this pass alone resolves the common case.
      for (let i = 0; i < hand.length && i < prev.length; i++) {
        if (prev[i].cardId === hand[i]) {
          next[i] = prev[i]
          taken[i] = true
        }
      }
      // Anything left takes the first free slot holding the same card id, so a
      // card that only shifted position is still the same card to Vue.
      for (let i = 0; i < hand.length; i++) {
        if (next[i]) continue
        const j = prev.findIndex((p, k) => !taken[k] && p.cardId === hand[i])
        if (j !== -1) {
          next[i] = prev[j]
          taken[j] = true
        }
      }
      const fresh = []
      for (let i = 0; i < hand.length; i++) {
        if (!next[i]) {
          next[i] = { uid: `c${++this._uidSeq}`, cardId: hand[i] }
          fresh.push(next[i])
        }
      }
      this.handSlots = next

      // One new card while the deck shrank is a draw — the only case where the
      // card can be shown travelling from the pile. A Favor or a steal also
      // adds a card, but it comes from a hand, not the deck.
      const drewFromDeck =
        fresh.length === 1 &&
        typeof prevDeckCount === 'number' &&
        this.drawPileCount < prevDeckCount
      if (drewFromDeck) {
        // Never cleared here: the action's own reply and the broadcast both
        // land, and the second one must not cancel a flight already running.
        this.drawFlight = { uid: fresh[0].uid, cardId: fresh[0].cardId }
      }
    },

    clearDrawFlight() {
      this.drawFlight = null
    },

    async _sendAction(action, data = {}) {
      const result = await saveRoomState(GAME_CODE, this.roomKey, action, data, this.me.name)
      if (result?.error) {
        // The server refuses an action without changing anything, so the only
        // sign the player gets is this.
        toast.error(result.error)
        return { error: result.error }
      }
      if (result?.state) {
        const saved = result.state
        if (saved.players) {
          const prevDeckCount = this.drawPileCount
          this.roomState.gameState = saved
          this._syncHandSlots(prevDeckCount)
        } else if (saved.minPlayers !== undefined) {
          this.roomState.roomSettings = { ...defaultRoomSettings(), ...saved }
        }
      }
      return result
    },
  },
})
