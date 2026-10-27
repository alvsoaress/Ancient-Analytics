import { useEffect, useState } from 'react'

export type Attribute = 'str' | 'agi' | 'int' | 'all'
export type Hero = { id: number; name: string; localized_name: string; primary_attr: Attribute; attack_type: 'Melee' | 'Ranged'; roles: string[] }
export type Position = 'carry' | 'mid' | 'offlane' | 'support4' | 'support5'
export type HeroPerformance = { heroId: number; name: string; picks: number; wins: number; winRate: number | null; pickRate: number | null; kda: number | null }
export type PositionSample = { matchesIncluded: number; matchesUnavailable: number; matchesWithoutPositions: number; from: string; to: string }
export type MetaResponse = { scope: 'professional'; heroes: HeroPerformance[]; position?: Position; sample?: PositionSample }

export class ApiError extends Error {
  readonly code: string
  constructor(code: string, message: string) {
    super(message)
    this.code = code
  }
}

// Durante o carregamento, `previous` traz o último resultado bem-sucedido, para manter a tela estável ao trocar filtros ou páginas.
export type ApiState<T> = { status: 'loading'; previous?: T } | { status: 'success'; data: T } | { status: 'error'; error: ApiError }

const unreachable = 'Não foi possível conectar à API. Verifique se o backend está em execução.'

async function getJson<T>(path: string, signal: AbortSignal): Promise<T> {
  let response: Response
  try {
    response = await fetch(path, { signal, headers: { Accept: 'application/json' } })
  } catch (error) {
    if (signal.aborted) throw error
    throw new ApiError('NETWORK_ERROR', unreachable)
  }
  const body = await response.json().catch(() => null)
  if (!response.ok) throw new ApiError(body?.error?.code ?? 'HTTP_ERROR', body?.error?.message ?? unreachable)
  if (body === null) throw new ApiError('INVALID_RESPONSE', 'A API retornou uma resposta inválida.')
  return body as T
}

// O estado guarda a chave da requisição; ao mudar o endereço, o resultado anterior deixa de valer e volta a carregar.
export function useApi<T>(path: string) {
  const [attempt, setAttempt] = useState(0)
  const key = `${attempt}:${path}`
  const [result, setResult] = useState<{ key: string; state: ApiState<T> }>()

  useEffect(() => {
    const controller = new AbortController()
    getJson<T>(path, controller.signal).then(
      (data) => setResult({ key, state: { status: 'success', data } }),
      (error: unknown) => {
        if (controller.signal.aborted) return
        setResult({ key, state: { status: 'error', error: error instanceof ApiError ? error : new ApiError('UNKNOWN', unreachable) } })
      },
    )
    return () => controller.abort()
  }, [key, path])

  const previous = result?.state.status === 'success' ? result.state.data : undefined
  const state: ApiState<T> = result?.key === key ? result.state : { status: 'loading', previous }
  return [state, () => setAttempt((value) => value + 1)] as const
}

const heroPrefix = 'npc_dota_hero_'

// Retrato oficial na CDN Steam, identificado pelo nome interno do herói no catálogo OpenDota.
export function portraitUrl(hero: Hero) {
  return hero.name?.startsWith(heroPrefix) ? `https://cdn.cloudflare.steamstatic.com/apps/dota2/images/dota_react/heroes/${hero.name.slice(heroPrefix.length)}.png` : undefined
}
