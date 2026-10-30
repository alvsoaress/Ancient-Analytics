import { useState } from 'react'
import type { HeroBuild, HeroBuildsResponse } from '../builds'
import { buildCount, buildPercentage, formatPurchaseTime } from '../builds'
import { DotaImage } from './BuildPanels'

export function FinalInventoryPanel({ inventories }: { inventories: HeroBuild['finalInventories'] }) {
  const [selected, setSelected] = useState(0)
  const inventory = inventories[selected] ?? inventories[0]
  return <section className="build-panel" aria-labelledby="final-inventory-title">
    <h3 id="final-inventory-title">Inventário final</h3><p className="build-panel-note">Os seis espaços de itens no encerramento das partidas.</p>
    {!inventory ? <p className="build-muted">A fonte não informou o inventário ao final destas partidas.</p> : <>
      {inventories.length > 1 && <label className="build-select">Inventário observado<select value={selected} onChange={event => setSelected(Number(event.target.value))}>{inventories.map((option, index) => <option key={index} value={index}>Opção {index + 1} · {option.matches} {option.matches === 1 ? 'partida' : 'partidas'}</option>)}</select></label>}
      <ul className="final-inventory-slots">{inventory.slots.map((item, index) => <li key={index}>{item ? <><DotaImage src={item.imageUrl} name={item.name} /><span>{item.name}</span></> : <><span className="empty-inventory-slot" aria-hidden="true" /><span>Espaço vazio</span></>}</li>)}</ul>
      <p className="build-inline-note">{buildCount(inventory.matches)} {inventory.matches === 1 ? 'partida' : 'partidas'} · {buildPercentage(inventory.winRate)} vitórias. Espaços vazios refletem o inventário real.</p>
    </>}
  </section>
}

export function AlternativeItemsPanel({ items }: { items: HeroBuildsResponse['alternativeItems'] }) {
  const [expanded, setExpanded] = useState(false)
  const visible = expanded ? items : items.slice(0, 8)
  return <section className="build-panel build-alternatives" aria-labelledby="alternative-items-title">
    <h3 id="alternative-items-title">Situacionais e alternativas</h3><p className="build-panel-note">Itens presentes em outras progressões da amostra. Escolha conforme a partida.</p>
    {items.length === 0 ? <p className="build-muted">Não há outras opções de itens registradas nesta amostra.</p> : <>
      <ul className="alternative-items">{visible.map(entry => <li key={entry.item.id}><DotaImage src={entry.item.imageUrl} name={entry.item.name} /><span>{entry.item.name}<small>{entry.matches} {entry.matches === 1 ? 'partida' : 'partidas'} · {buildPercentage(entry.purchaseRate)} de presença</small></span><span className="alternative-rate">{buildPercentage(entry.winRate)}<small>vitórias</small></span></li>)}</ul>
      {items.length > 8 && <button className="text-link build-toggle" type="button" aria-expanded={expanded} onClick={() => setExpanded(!expanded)}>{expanded ? 'Mostrar menos alternativas' : `Ver todas as ${items.length} alternativas`}</button>}
    </>}
  </section>
}

export function PurchaseHistoryPanel({ history, example }: { history: HeroBuild['purchaseHistory']; example: HeroBuild['exampleMatch'] }) {
  const [expanded, setExpanded] = useState(false)
  return <details className="purchase-history" onToggle={event => setExpanded(event.currentTarget.open)}>
    <summary>Histórico completo de compras · {history.length} registros</summary>
    <p className="build-muted">Compras da partida de exemplo, incluindo componentes, receitas e consumíveis. <a className="text-link" href={`https://www.opendota.com/matches/${example.matchId}`} target="_blank" rel="noreferrer">Partida {example.matchId} ↗</a></p>
    {expanded && <ol className="purchase-history-list">{history.map((entry, index) => <li key={index}><span className="purchase-time">{formatPurchaseTime(entry.timeSeconds)}</span><DotaImage src={entry.item.imageUrl} name={entry.item.name} /><span>{entry.item.name}</span></li>)}</ol>}
  </details>
}
