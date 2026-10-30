import type { Position } from './api'

export type BuildItem = { id: number; key: string; name: string; imageUrl: string | null }
export type BuildAbility = { id: number; key: string | null; name: string | null; imageUrl: string | null }
export type TalentDefinition = { level: number; ability: BuildAbility }
export type BuildPerformance = { matches: number; wins: number; winRate: number }
export type StartingInventory = BuildPerformance & { items: { item: BuildItem; quantity: number }[] }
export type AbilityBuild = BuildPerformance & { upgrades: { order: number; ability: BuildAbility }[] }
export type HeroBuild = BuildPerformance & {
  id: string
  pickRate: number
  coreItems: { item: BuildItem; medianPurchaseTimeSeconds: number; stage: 'early' | 'core' | 'late' }[]
  exampleMatch: { matchId: number; durationSeconds: number; won: boolean }
  purchaseHistory: { item: BuildItem; timeSeconds: number }[]
  finalInventories: (BuildPerformance & { slots: (BuildItem | null)[] })[]
  startingInventories: StartingInventory[]
  abilityBuilds: AbilityBuild[]
  talents: (BuildPerformance & TalentDefinition)[]
}
export type BuildSample = {
  matchLimit: number; matchesRequested: number; matchesUnavailable: number; matchesWithoutPurchaseLog: number
  matchesOutsidePosition: number; matchesWithoutPosition: number; matchesIncluded: number
  matchesWithUnknownItems: number; matchesWithoutCoreItems: number; matchesGrouped: number; matchesWithAbilityUpgrades: number
  matchIds: number[]; patchIds: number[]; from: string | null; to: string | null; fetchedAt: string
}
export type HeroBuildsResponse = {
  hero: { heroId: number; name: string; imageUrl: string | null; abilities: BuildAbility[]; talents: TalentDefinition[] }
  alternativeItems: (BuildPerformance & { item: BuildItem; medianPurchaseTimeSeconds: number; purchaseRate: number })[]
  source: 'opendota'; scope: 'professional_sample'; position: Position | null; positionSource: string | null
  grouping: { method: string; earlyGameEndsAtSeconds: number; lateGameStartsAtSeconds: number }
  sample: BuildSample
  builds: HeroBuild[]
}

export function formatPurchaseTime(seconds: number) {
  if (seconds <= 0) return 'Antes do início'
  const rounded = Math.round(seconds)
  return `${Math.floor(rounded / 60)}:${String(rounded % 60).padStart(2, '0')}`
}

export function sampleExclusions(sample: Pick<BuildSample, 'matchesUnavailable' | 'matchesWithoutPurchaseLog' | 'matchesOutsidePosition' | 'matchesWithoutPosition' | 'matchesWithUnknownItems' | 'matchesWithoutCoreItems'>) {
  return sample.matchesUnavailable + sample.matchesWithoutPurchaseLog + sample.matchesOutsidePosition + sample.matchesWithoutPosition + sample.matchesWithUnknownItems + sample.matchesWithoutCoreItems
}

const percentage = new Intl.NumberFormat('pt-BR', { maximumFractionDigits: 1 })
const count = new Intl.NumberFormat('pt-BR')
export const buildPercentage = (value: number) => `${percentage.format(value)}%`
export const buildCount = (value: number) => count.format(value)

export function talentText(name: string | null) {
  if (name === null) return { label: null, hasMissingValue: false }
  const hasMissingValue = /\{s:[^}]+\}/.test(name)
  const label = name.replace(/[+-]?\{s:[^}]+\}(?:%|s)?/g, '').replace(/\s+/g, ' ').trim()
  return { label: label || null, hasMissingValue }
}
