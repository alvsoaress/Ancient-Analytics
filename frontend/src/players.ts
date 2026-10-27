export type PlayerProfile = {
  accountId: number
  steamId: string | null
  name: string | null
  avatarUrl: string | null
  profileUrl: string | null
  rankTier: number | null
  leaderboardRank: number | null
}

const steamId64Base = 76561197960265728n
const maxAccountId = 4294967295n

// Aceita o ID de amigo (Steam32), o SteamID64 ou links de perfil da Steam, OpenDota e Dotabuff,
// e devolve sempre o ID de amigo, que é o formato aceito pela API.
export function parseAccountId(input: string): { accountId: number } | { error: string } {
  const value = input.trim()
  if (!value) return { error: 'Informe o ID de amigo do jogador.' }
  if (/steamcommunity\.com\/id\//i.test(value)) {
    return { error: 'Endereços personalizados da Steam (/id/nome) não podem ser convertidos. Use o ID de amigo ou um link /profiles/ com números.' }
  }
  const digits = /^\d+$/.test(value) ? value : value.match(/\/(?:players|profiles)\/(\d+)/i)?.[1]
  if (!digits) return { error: 'Use o ID de amigo do Dota 2, o SteamID64 ou um link de perfil com o número da conta.' }

  let id = BigInt(digits)
  if (id >= steamId64Base && id - steamId64Base <= maxAccountId) id -= steamId64Base
  if (id < 1n || id > maxAccountId) return { error: 'Este número não corresponde a uma conta do Dota 2.' }
  return { accountId: Number(id) }
}

const medals = ['Arauto', 'Guardião', 'Cruzado', 'Arconte', 'Lenda', 'Ancestral', 'Divino', 'Imortal']

// rankTier da OpenDota: dezena = medalha (1 a 8), unidade = estrelas (0 a 5).
export function describeRank(rankTier: number) {
  const medal = Math.floor(rankTier / 10)
  const stars = rankTier % 10
  if (medal < 1 || medal > medals.length || stars > 5) return null
  const name = medals[medal - 1]
  return { name, stars: medal === 8 ? 0 : stars, label: medal === 8 || stars === 0 ? name : `${name} ${stars}` }
}
