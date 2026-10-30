import { useApi } from '../api'
import type { MetaResponse } from '../api'
import { HeroLink } from './HeroLink'
import { HeroPortrait } from './HeroPortrait'

const percentage = new Intl.NumberFormat('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
const integer = new Intl.NumberFormat('pt-BR')
const circumference = 2 * Math.PI * 94
const barCount = 7

// Destaque do topo: o herói mais escolhido no agregado profissional da OpenDota.
export function StatsPreview({ portraits }: { portraits: Map<number, string> }) {
  const [meta, retry] = useApi<MetaResponse>('/api/meta/heroes?sortBy=picks&minPicks=1')
  const heroes = meta.status === 'success' ? meta.data.heroes.slice(0, barCount) : []
  const top = heroes.at(0)
  const winRate = top?.winRate ?? null
  const maxPickRate = Math.max(0, ...heroes.map((hero) => hero.pickRate ?? 0))
  const losses = top ? top.picks - top.wins : 0

  return (
    <figure className="stats-preview" aria-label="Destaque do meta profissional">
      <div className="chart-grid" aria-hidden="true" />
      <div className="donut-container">
        <svg className="donut" viewBox="0 0 224 224" role="img" aria-label={top && winRate !== null ? `Taxa de vitória de ${top.name}: ${percentage.format(winRate)} por cento` : 'Taxa de vitória indisponível'}>
          <circle className="donut-track" cx="112" cy="112" r="94" />
          {winRate !== null && <circle className="donut-value" cx="112" cy="112" r="94" strokeDasharray={`${circumference * winRate / 100} ${circumference}`} transform="rotate(-90 112 112)" />}
        </svg>
        <div className="donut-label">
          <span>Taxa de vitória</span>
          <strong>{winRate === null ? '—' : <>{percentage.format(winRate)}<span>%</span></>}</strong>
          <small>{top ? top.name : meta.status === 'loading' ? 'Carregando…' : 'Indisponível'}</small>
        </div>
      </div>
      <div className="mini-charts">
        <div className="mini-chart">
          <p className="mini-chart-label">Taxa de escolha</p>
          <p className="mini-chart-value">{top?.pickRate == null ? '—' : `${percentage.format(top.pickRate)}%`}</p>
          <div className="chart-bars" aria-hidden="true">{heroes.map((hero) => <span key={hero.heroId} style={{ height: `${maxPickRate > 0 ? (hero.pickRate ?? 0) / maxPickRate * 100 : 0}%` }} />)}</div>
          <div className="chart-axis"><span>1º</span><span>{barCount}º mais escolhido</span></div>
          {heroes.length > 0 && (
            <ol className="sr-only" aria-label="Heróis mais escolhidos">
              {heroes.map((hero) => <li key={hero.heroId}>{hero.name}: {hero.pickRate === null ? 'taxa de escolha indisponível' : `${percentage.format(hero.pickRate)}% de escolha`}</li>)}
            </ol>
          )}
        </div>
        <div className="mini-chart">
          <p className="mini-chart-label">Vitórias e derrotas</p>
          <div className="outcome-bar" aria-hidden="true">{top && top.picks > 0 && <><span style={{ width: `${top.wins / top.picks * 100}%` }} /><span /></>}</div>
          <div className="outcome-legend">
            <span><i /> {top ? integer.format(top.wins) : '—'} vitórias</span>
            <span><i /> {top ? integer.format(losses) : '—'} derrotas</span>
          </div>
        </div>
      </div>
      <figcaption>
        {meta.status === 'error' ? (
          <>Destaque indisponível no momento. <button className="inline-button" type="button" onClick={retry}>Tentar novamente</button></>
        ) : top ? (
          <><HeroLink heroId={top.heroId} name={top.name}><HeroPortrait src={portraits.get(top.heroId)} name={top.name} /></HeroLink> {top.name}, o herói mais escolhido em partidas profissionais · OpenDota</>
        ) : (
          <><span className="status-dot" /> Carregando o destaque do meta</>
        )}
      </figcaption>
    </figure>
  )
}
