export const GAMES = {
  hotpot: {
    code: 'hotpot',
    name: 'Hotpot Night',
    route: '/hotpot',
  },
  ek: {
    code: 'ek',
    name: 'Exploding Kitchen',
    route: '/exploding-kitchen',
  },
}

export function getGame(code) {
  return GAMES[code] || null
}
