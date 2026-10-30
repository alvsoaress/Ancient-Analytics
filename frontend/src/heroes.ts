import type { Attribute, Hero, Position } from './api'

export const buildPositions: { value: Position | ''; label: string }[] = [
  { value: '', label: 'Todas as posições' }, { value: 'carry', label: 'HC' }, { value: 'mid', label: 'Mid' },
  { value: 'offlane', label: 'Offlane' }, { value: 'support4', label: 'Suporte 4' }, { value: 'support5', label: 'Suporte 5' },
]

const normalizeName = (value: string) => value.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLocaleLowerCase('pt-BR').replace(/[^\p{L}\p{N}]/gu, '')

export function findHeroes(heroes: Hero[], query: string, attribute: Attribute | '' = '') {
  const term = normalizeName(query)
  return heroes.filter(hero => (!attribute || hero.primary_attr === attribute) && normalizeName(hero.localized_name).includes(term))
    .sort((a, b) => a.localized_name.localeCompare(b.localized_name, 'pt-BR'))
}

export function parseHeroId(value: string | null) {
  if (!value || !/^\d+$/.test(value)) return null
  const id = Number(value)
  return Number.isInteger(id) && id >= 1 && id <= 2147483647 ? id : null
}

export function parseBuildPosition(value: string | null): Position | '' | null {
  if (value === null) return ''
  return value && buildPositions.some(position => position.value === value) ? value as Position : null
}

export function heroBuildUrl(heroId: number, position: Position | '' = '') {
  const params = new URLSearchParams({ heroi: String(heroId) })
  if (position) params.set('posicao', position)
  return `/?${params}`
}
