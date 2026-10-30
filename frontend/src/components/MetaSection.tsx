import { useState } from 'react'
import { useApi } from '../api'
import type { MetaResponse, Position } from '../api'
import { HeroLink } from './HeroLink'
import { HeroPortrait } from './HeroPortrait'
import { ErrorState, LoadingState, Unavailable } from './RequestState'

const positions: { value: Position | ''; label: string }[] = [
  { value: '', label: 'Geral' }, { value: 'carry', label: 'HC' }, { value: 'mid', label: 'Mid' },
  { value: 'offlane', label: 'Offlane' }, { value: 'support4', label: 'Sup 4' }, { value: 'support5', label: 'Sup 5' },
]
const sorts = [{ value: 'picks', label: 'Escolhas' }, { value: 'winRate', label: 'Taxa de vitória' }] as const
const minPicksOptions = [1, 3, 5, 10, 25, 50]
// A amostra por posição tem só 20 partidas; no agregado geral, poucas escolhas distorcem a taxa de vitória.
const defaultMinPicks = (position: Position | '') => (position ? 1 : 10)
const initialRows = 10

const percentage = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const integer = new Intl.NumberFormat('pt-BR')
const ratio = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })

function RateBar({ value, max, className = '' }: { value: number | null; max: number; className?: string }) {
  if (value === null) return <Unavailable />
  return <div className="bar-value"><span className={`stat-track ${className}`} aria-hidden="true"><span style={{ width: `${max > 0 ? value / max * 100 : 0}%` }} /></span><strong>{percentage.format(value)}%</strong></div>
}

export function MetaSection({ portraits }: { portraits: Map<number, string> }) {
  const [position, setPosition] = useState<Position | ''>('')
  const [sortBy, setSortBy] = useState<(typeof sorts)[number]['value']>('picks')
  const [minPicks, setMinPicks] = useState(defaultMinPicks(''))
  const [showAll, setShowAll] = useState(false)
  const params = new URLSearchParams({ sortBy, minPicks: String(minPicks) })
  if (position) params.set('position', position)
  const [meta, retry] = useApi<MetaResponse>(`/api/meta/heroes?${params}`)

  const selectPosition = (value: Position | '') => {
    setPosition(value)
    setMinPicks(defaultMinPicks(value))
    setShowAll(false)
  }

  const data = meta.status === 'success' ? meta.data : meta.status === 'loading' ? meta.previous : undefined
  const refreshing = meta.status === 'loading' && data !== undefined
  const loadingMessage = position ? 'Montando a amostra de partidas profissionais. A primeira consulta pode levar alguns segundos.' : 'Carregando estatísticas profissionais.'

  let content
  if (meta.status === 'error') {
    content = <ErrorState title="Não foi possível carregar o meta" error={meta.error} onRetry={retry} />
  } else if (!data) {
    content = <LoadingState>{loadingMessage}</LoadingState>
  } else if (data.heroes.length === 0) {
    content = (
      <div className="request-state">
        <h3>Nenhum herói com pelo menos {minPicks} {minPicks === 1 ? 'escolha' : 'escolhas'}</h3>
        <p>Reduza o mínimo de escolhas para ampliar a lista.</p>
        {minPicks > 1 && <button className="button button-secondary" type="button" onClick={() => setMinPicks(1)}>Mostrar a partir de 1 escolha</button>}
      </div>
    )
  } else {
    const { heroes } = data
    const maxPickRate = Math.max(0, ...heroes.map((hero) => hero.pickRate ?? 0))
    const rows = showAll ? heroes : heroes.slice(0, initialRows)
    // Colunas e legenda seguem os dados exibidos, que podem ser os anteriores enquanto a nova consulta carrega.
    const showKda = Boolean(data.position)
    content = (
      <>
        <div className="table-scroll" tabIndex={0} role="region" aria-label="Estatísticas profissionais dos heróis">
          <table className="meta-table">
            <caption className="sr-only">Taxa de escolha, taxa de vitória{showKda && ', KDA'} e escolhas dos heróis em partidas profissionais{data.position && ` na posição ${positions.find((item) => item.value === data.position)?.label}`}.</caption>
            <thead><tr>
              <th scope="col">#</th><th scope="col">Herói</th><th scope="col" className="rate-column">Taxa de escolha</th><th scope="col" className="rate-column">Taxa de vitória</th>
              {showKda && <th scope="col" className="kda-column">KDA</th>}
              <th scope="col" className="optional-column">Escolhas</th>
            </tr></thead>
            <tbody>{rows.map((hero, index) => (
              <tr key={hero.heroId}>
                <td className="rank">{String(index + 1).padStart(2, '0')}</td>
                <th scope="row"><HeroLink className="hero-cell" heroId={hero.heroId} name={hero.name}><HeroPortrait src={portraits.get(hero.heroId)} name={hero.name} /><span>{hero.name}</span></HeroLink></th>
                <td className="rate-column"><RateBar value={hero.pickRate} max={maxPickRate} /></td>
                <td className="rate-column"><RateBar value={hero.winRate} max={100} className="win-track" /></td>
                {showKda && <td className="kda-column">{hero.kda === null ? <Unavailable /> : ratio.format(hero.kda)}</td>}
                <td className="optional-column">{integer.format(hero.picks)}</td>
              </tr>
            ))}</tbody>
          </table>
        </div>
        {heroes.length > initialRows && (
          <button className="text-link table-toggle" type="button" aria-expanded={showAll} onClick={() => setShowAll(!showAll)}>
            {showAll ? 'Mostrar menos' : `Mostrar todos os ${heroes.length} heróis`}
          </button>
        )}
      </>
    )
  }

  return (
    <section className="meta-section" id="meta" aria-labelledby="meta-title">
      <div className="section-width">
        <div className="meta-heading">
          <div className="section-title-group"><h2 id="meta-title">Meta profissional</h2><span className="source-label">Dados OpenDota</span></div>
          <a className="text-link" href="#herois">Conhecer os heróis <span aria-hidden="true">↗</span></a>
        </div>
        <div className="position-filters" role="group" aria-label="Filtrar por posição">
          {positions.map((item) => <button key={item.value} type="button" aria-pressed={position === item.value} onClick={() => selectPosition(item.value)}>{item.label}</button>)}
        </div>
        <div className="meta-controls">
          <div className="segmented" role="group" aria-label="Ordenar por">
            <span aria-hidden="true">Ordenar por</span>
            {sorts.map((item) => <button key={item.value} type="button" aria-pressed={sortBy === item.value} onClick={() => setSortBy(item.value)}>{item.label}</button>)}
          </div>
          <label className="select-field">
            Mínimo de escolhas
            <select value={minPicks} onChange={(event) => { setMinPicks(Number(event.target.value)); setShowAll(false) }}>
              {minPicksOptions.map((value) => <option key={value} value={value}>{value}</option>)}
            </select>
          </label>
        </div>
        {refreshing && <p className="refresh-note" role="status"><span className="loading-bar" aria-hidden="true" />{loadingMessage}</p>}
        <div className={refreshing ? 'is-refreshing' : undefined} aria-busy={refreshing}>{content}</div>
      </div>
    </section>
  )
}
