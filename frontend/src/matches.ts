export type PlayerMatch = {
  matchId: number
  heroId: number | null
  won: boolean | null
  durationSeconds: number | null
  startTime: number | null
  kills: number | null
  deaths: number | null
  assists: number | null
  gameMode: number | null
  lobbyType: number | null
}

export type PlayerMatchesPage = {
  accountId: number
  matches: PlayerMatch[]
  pagination: { limit: number; offset: number; count: number; hasMore: boolean; nextOffset: number | null }
}

// Identificadores da OpenDota (dotaconstants: game_mode e lobby_type), com os nomes do cliente em português.
const gameModes: Record<number, string> = {
  1: 'Todos Escolhem', 2: 'Modo Capitão', 3: 'Seleção Aleatória', 4: 'Seleção Única', 5: 'Todos Aleatórios',
  11: 'Somente Meio', 12: 'Menos Jogados', 16: 'Seleção do Capitão', 18: 'Seleção de Habilidades',
  20: 'Todos Aleatórios Mata-Mata', 21: '1x1 no Meio', 22: 'Todos Escolhem', 23: 'Turbo',
}
const lobbyTypes: Record<number, string> = {
  0: 'Normal', 1: 'Prática', 2: 'Torneio', 4: 'Contra bots', 5: 'Ranqueada em equipe', 6: 'Ranqueada solo',
  7: 'Ranqueada', 8: '1x1 no Meio', 9: 'Battle Cup', 12: 'Evento',
}

export function describeMode(gameMode: number | null, lobbyType: number | null) {
  const mode = gameMode === null ? null : gameModes[gameMode] ?? `Modo ${gameMode}`
  const lobby = lobbyType === null ? null : lobbyTypes[lobbyType] ?? `Fila ${lobbyType}`
  return { mode, lobby }
}

export function formatDuration(seconds: number) {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const rest = String(seconds % 60).padStart(2, '0')
  return hours > 0 ? `${hours}:${String(minutes).padStart(2, '0')}:${rest}` : `${minutes}:${rest}`
}

export type Team = 'radiant' | 'dire'

export type MatchPlayer = {
  accountId: number | null
  name: string | null
  heroId: number | null
  playerSlot: number | null
  team: Team | null
  won: boolean | null
  kills: number | null
  deaths: number | null
  assists: number | null
  goldPerMinute: number | null
  xpPerMinute: number | null
  heroDamage: number | null
  towerDamage: number | null
  heroHealing: number | null
  lastHits: number | null
  denies: number | null
  level: number | null
}

export type MatchDetails = {
  matchId: number
  durationSeconds: number | null
  startTime: number | null
  gameMode: number | null
  lobbyType: number | null
  winner: Team | null
  radiantScore: number | null
  direScore: number | null
  players: MatchPlayer[]
}

// Nomes dos lados no cliente em português.
export const teamNames: Record<Team, string> = { radiant: 'Iluminados', dire: 'Temidos' }

// Mesmo limite do backend: inteiro positivo representável com exatidão no JavaScript.
export function parseMatchId(value: string | null) {
  if (!value || !/^\d+$/.test(value)) return null
  const id = Number(value)
  return id >= 1 && Number.isSafeInteger(id) ? id : null
}
