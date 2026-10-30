import { useEffect, useRef, useState } from 'react'
import type { Hero, Position } from '../api'
import { portraitUrl, useApi } from '../api'
import type { HeroBuildsResponse } from '../builds'
import { buildCount, buildPercentage, sampleExclusions } from '../builds'
import { buildPositions, heroBuildUrl, parseBuildPosition } from '../heroes'
import { navigate, useDocumentTitle } from '../navigation'
import { HeroPortrait } from './HeroPortrait'
import { ErrorState, LoadingState } from './RequestState'
import { AbilitiesPanel, BuildItemStrip, PurchasePanel, StartingItemsPanel, TalentsPanel } from './BuildPanels'
import { AlternativeItemsPanel, FinalInventoryPanel, PurchaseHistoryPanel } from './BuildGuidePanels'
import './HeroBuildView.css'

const date = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' })
const dateTime = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })

export function HeroBuildView({ heroId, position, hero }: { heroId: number | null; position: string | null; hero?: Hero }) {
  const section = useRef<HTMLElement>(null)
  useEffect(() => { section.current?.focus() }, [heroId])
  const selectedPosition = parseBuildPosition(position)
  return <section className="hero-builds section-width" ref={section} tabIndex={-1} aria-labelledby="hero-build-title">
    <a className="text-link back-link" href="/#herois" onClick={event => { if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return; event.preventDefault(); navigate('/#herois') }}><span aria-hidden="true">←</span> Voltar aos heróis</a>
    {heroId === null ? <InvalidBuildPage title="Identificador de herói inválido" message="Confira o endereço do herói ou escolha um rosto no catálogo." />
      : selectedPosition === null ? <InvalidBuildPage title="Posição inválida" message="Escolha uma das posições disponíveis para consultar as builds." heroId={heroId} />
      : <HeroBuildContent key={heroId} heroId={heroId} hero={hero} position={selectedPosition} />}
  </section>
}

function InvalidBuildPage({ title, message, heroId }: { title: string; message: string; heroId?: number }) {
  useDocumentTitle(title)
  return <div className="request-state" role="alert"><h1 id="hero-build-title">{title}</h1><p>{message}</p>{heroId && <a className="button button-secondary" href={heroBuildUrl(heroId)} onClick={event => { event.preventDefault(); navigate(heroBuildUrl(heroId)) }}>Ver todas as posições</a>}</div>
}

function HeroBuildContent({ heroId, position, hero }: { heroId: number; position: Position | ''; hero?: Hero }) {
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showAll, setShowAll] = useState(false)
  const [response, retry] = useApi<HeroBuildsResponse>(`/api/heroes/${heroId}/builds${position ? `?position=${position}` : ''}`)
  const data = response.status === 'success' ? response.data : undefined
  const name = data?.hero.name ?? hero?.localized_name ?? `Herói ${heroId}`
  useDocumentTitle(`${name} · Builds`)
  const selected = data?.builds.find(build => build.id === selectedId) ?? data?.builds[0]
  const selectedIndex = data?.builds.findIndex(build => build.id === selected?.id) ?? -1
  const options = showAll || selectedIndex >= 4 ? data?.builds : data?.builds.slice(0, 4)

  return <>
    <header className="build-hero-heading">
      <HeroPortrait src={data?.hero.imageUrl ?? (hero && portraitUrl(hero))} name={name} />
      <div><p className="eyebrow">Heróis / Builds</p><h1 id="hero-build-title">{name}</h1><p className="build-subtitle">Itens, habilidades e talentos em partidas profissionais.</p></div>
    </header>
    <div className="position-filters build-position-filters" role="group" aria-label="Posição da build">{buildPositions.map(item => <button type="button" key={item.value} aria-pressed={position === item.value} onClick={() => { setShowAll(false); navigate(heroBuildUrl(heroId, item.value)) }}>{item.label}</button>)}</div>
    {response.status === 'loading' ? <LoadingState>Consultando as builds do herói. A primeira consulta pode levar alguns segundos.</LoadingState> : response.status === 'error'
      ? response.error.code === 'HERO_NOT_FOUND' ? <div className="request-state" role="alert"><h2>Herói não encontrado</h2><p>Esse identificador não está no catálogo. Volte aos heróis e escolha outro.</p></div>
        : <ErrorState title="Não foi possível carregar as builds" error={response.error} onRetry={retry} />
      : data && <>
        <div className="build-sample-line"><span className="source-label">Amostra profissional · OpenDota</span><p>{buildCount(data.sample.matchesIncluded)} {data.sample.matchesIncluded === 1 ? 'partida incluída' : 'partidas incluídas'}{data.sample.from && data.sample.to && <> · {date.format(new Date(data.sample.from))} a {date.format(new Date(data.sample.to))}</>}</p></div>
        {data.builds.length === 0 ? <div className="request-state" role="status"><h2>Sem builds disponíveis nesta amostra</h2><p>Não há sequências de compras utilizáveis{position ? ' para esta posição' : ' para este herói'}. Tente outra posição ou consulte mais tarde.</p>{position && <button type="button" className="button button-secondary" onClick={() => navigate(heroBuildUrl(heroId))}>Ver todas as posições</button>}</div> : <>
          <div className="build-section-heading"><div><p className="eyebrow">Do registro de compras à partida</p><h2>Builds observadas</h2></div><p className="build-muted">{data.builds.length} {data.builds.length === 1 ? 'sequência encontrada' : 'sequências encontradas'}</p></div>
          <div className="build-options" role="group" aria-label="Escolher uma build">{options?.map((build, index) => <button type="button" key={build.id} className="build-option" aria-pressed={selected?.id === build.id} onClick={() => setSelectedId(build.id)}>
            <span className="build-option-heading"><strong>Build {index + 1}</strong>{index === 0 && <span className="build-option-badge">Mais jogada</span>}</span>
            <BuildItemStrip items={build.coreItems.map(core => core.item)} />
            <span className="sr-only">{build.coreItems.map(core => core.item.name).join(', ')}</span>
            <span className="build-option-stats">{buildCount(build.matches)} {build.matches === 1 ? 'partida' : 'partidas'}<strong>{buildPercentage(build.winRate)} vitórias</strong></span>
          </button>)}</div>
          {data.builds.length > 4 && <button type="button" className="text-link build-toggle" aria-expanded={showAll || selectedIndex >= 4} onClick={() => { if (showAll || selectedIndex >= 4) { setSelectedId(null); setShowAll(false) } else setShowAll(true) }}>{showAll || selectedIndex >= 4 ? 'Mostrar as quatro primeiras' : `Ver todas as ${data.builds.length} sequências`} <span aria-hidden="true">{showAll || selectedIndex >= 4 ? '↑' : '↓'}</span></button>}
          {selected && <section className="selected-build" aria-labelledby="selected-build-title">
            <div className="selected-build-heading"><h2 id="selected-build-title">Build {selectedIndex + 1}<span>{buildCount(selected.matches)} {selected.matches === 1 ? 'partida' : 'partidas'} · {buildCount(selected.wins)} {selected.wins === 1 ? 'vitória' : 'vitórias'}</span></h2><p className="selected-build-rate"><strong>{buildPercentage(selected.winRate)}</strong><span>de vitórias nesta sequência</span></p></div>
            <p className="build-small-sample" role="status">{selected.matches === 1 ? 'Esta sequência aparece em apenas uma partida.' : `Resultados de ${selected.matches} partidas.`} Considere o tamanho da amostra ao comparar as taxas.</p>
            <div className="build-columns"><div className="build-column"><StartingItemsPanel inventories={selected.startingInventories} /><PurchasePanel items={selected.coreItems} /></div><div className="build-column"><FinalInventoryPanel key={`inventory-${selected.id}`} inventories={selected.finalInventories} /><AbilitiesPanel key={`abilities-${selected.id}`} paths={selected.abilityBuilds} /><TalentsPanel options={data.hero.talents} choices={selected.talents} /></div></div>
            <AlternativeItemsPanel key={`alternatives-${selected.id}`} items={data.alternativeItems.filter(entry => !selected.coreItems.some(core => core.item.id === entry.item.id))} />
            <PurchaseHistoryPanel key={`history-${selected.id}`} history={selected.purchaseHistory} example={selected.exampleMatch} />
          </section>}
        </>}
        <SampleDetails data={data} />
      </>}
  </>
}

function SampleDetails({ data }: { data: HeroBuildsResponse }) {
  const sample = data.sample
  const excluded = sampleExclusions(sample)
  const reasons: [number, string][] = [[sample.matchesUnavailable, 'partidas indisponíveis'], [sample.matchesWithoutPurchaseLog, 'partidas sem registro de compras'], [sample.matchesOutsidePosition, 'partidas em outras posições'], [sample.matchesWithoutPosition, 'partidas sem posição identificada'], [sample.matchesWithUnknownItems, 'partidas com itens fora do catálogo'], [sample.matchesWithoutCoreItems, 'partidas sem progressão de itens disponível']]
  return <aside className="build-sample-details" aria-label="Sobre os dados desta build">
    <p>A progressão acompanha os itens de uso e itens montados ao longo de toda a partida. Componentes, receitas e consumíveis também podem ser consultados no histórico completo de compras.</p>
    <p>{sample.matchesGrouped} de {sample.matchesRequested} partidas consultadas entraram nos grupos.{sample.patchIds.length > 1 && ` A amostra abrange ${sample.patchIds.length} patches.`} Atualizado em <time dateTime={sample.fetchedAt}>{dateTime.format(new Date(sample.fetchedAt))}</time>.</p>
    {excluded > 0 && <details><summary>Por que {excluded} {excluded === 1 ? 'partida ficou de fora' : 'partidas ficaram de fora'}?</summary><ul>{reasons.filter(([count]) => count > 0).map(([count, reason]) => <li key={reason}>{count} {reason}</li>)}</ul></details>}
  </aside>
}
