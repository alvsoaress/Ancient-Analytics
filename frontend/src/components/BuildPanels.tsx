import { useState } from 'react'
import type { AbilityBuild, BuildItem, HeroBuild, StartingInventory, TalentDefinition } from '../builds'
import { buildCount, buildPercentage, formatPurchaseTime, talentText } from '../builds'

export function DotaImage({ src, name, ability = false }: { src: string | null; name: string; ability?: boolean }) {
  const [failedSrc, setFailedSrc] = useState<string>()
  const className = `dota-image${ability ? ' dota-ability-image' : ''}`
  if (!src || failedSrc === src) return <span className={`${className} dota-image-missing`} role="img" aria-label={`Imagem de ${name} indisponível`}>?</span>
  return <img className={className} src={src} alt="" width={ability ? 48 : 64} height="48" loading="lazy" onError={() => setFailedSrc(src)} />
}

function InventoryRow({ inventory, rank }: { inventory: StartingInventory; rank: number }) {
  return (
    <li className="starting-inventory">
      <div className="inventory-heading"><span>{rank === 0 ? 'Mais comum' : `Opção ${rank + 1}`}</span><small>{buildCount(inventory.matches)} {inventory.matches === 1 ? 'partida' : 'partidas'} · {buildPercentage(inventory.winRate)} vitórias</small></div>
      {inventory.items.length === 0 ? <p className="build-muted">Sem compras registradas antes do início.</p> : (
        <ul className="inventory-items">{inventory.items.map(({ item, quantity }) => <li key={item.id}>
          <span className="inventory-image"><DotaImage src={item.imageUrl} name={item.name} />{quantity > 1 && <span className="item-quantity" aria-label={`Quantidade: ${quantity}`}>×{quantity}</span>}</span>
          <span>{item.name}</span>
        </li>)}</ul>
      )}
    </li>
  )
}

export function StartingItemsPanel({ inventories }: { inventories: StartingInventory[] }) {
  return <section className="build-panel" aria-labelledby="starting-items-title">
    <h3 id="starting-items-title">Itens iniciais</h3><p className="build-panel-note">Combinações compradas até o início da partida.</p>
    {inventories.length === 0 ? <p className="build-muted">Inventários iniciais indisponíveis nesta sequência.</p> : <>
      <ol className="starting-inventories">{inventories.slice(0, 3).map((inventory, index) => <InventoryRow key={index} inventory={inventory} rank={index} />)}</ol>
      {inventories.length > 3 && <details className="build-more"><summary>Mais {inventories.length - 3} inventários</summary><ol className="starting-inventories" start={4}>{inventories.slice(3).map((inventory, index) => <InventoryRow key={index} inventory={inventory} rank={index + 3} />)}</ol></details>}
    </>}
  </section>
}

export function PurchasePanel({ items }: { items: HeroBuild['coreItems'] }) {
  const phases = [{ key: 'early', title: 'Início de jogo', timing: 'Até 10 minutos' }, { key: 'core', title: 'Itens principais', timing: 'De 10 a 25 minutos' }, { key: 'late', title: 'Final de jogo', timing: 'A partir de 25 minutos' }]
  return <section className="build-panel" aria-labelledby="purchase-order-title">
    <h3 id="purchase-order-title">Progressão de itens</h3><p className="build-panel-note">Itens de uso e itens montados, do início ao fim da partida.</p>
    {phases.map(phase => <div className="purchase-phase" key={phase.key}>
      <div className="purchase-phase-heading"><h4>{phase.title}</h4><span>{phase.timing}</span></div>
      {items.some(entry => entry.stage === phase.key) ? <ol className="build-purchases">{items.map((entry, index) => ({ ...entry, order: index + 1 })).filter(entry => entry.stage === phase.key).map(({ item, medianPurchaseTimeSeconds, order }) => <li key={item.id}>
        <span className="purchase-order" aria-label={`Compra ${order}`}>{String(order).padStart(2, '0')}</span>
        <DotaImage src={item.imageUrl} name={item.name} />
        <span className="purchase-name">{item.name}</span>
        <span className="purchase-time">{formatPurchaseTime(medianPurchaseTimeSeconds)}</span>
      </li>)}</ol> : <p className="build-muted">Nenhum item adicional registrado nesta fase.</p>}
    </div>)}
    <p className="build-inline-note">Os tempos são medianas das compras observadas. As fases acompanham o tempo de jogo.</p>
  </section>
}

export function AbilitiesPanel({ paths }: { paths: AbilityBuild[] }) {
  const [selected, setSelected] = useState(0)
  const path = paths[selected] ?? paths[0]
  const abilities = path ? [...new Map(path.upgrades.map(upgrade => [upgrade.ability.id, upgrade.ability])).values()] : []
  return <section className="build-panel" aria-labelledby="ability-order-title">
    <h3 id="ability-order-title">Ordem das habilidades</h3><p className="build-panel-note">Sequência completa das melhorias registradas.</p>
    {!path ? <p className="build-muted">A fonte não informou melhorias de habilidades para esta sequência.</p> : <>
      {paths.length > 1 && <label className="build-select">Sequência de habilidades<select value={selected} onChange={event => setSelected(Number(event.target.value))}>{paths.map((option, index) => <option key={index} value={index}>Opção {index + 1} · {option.matches} {option.matches === 1 ? 'partida' : 'partidas'}</option>)}</select></label>}
      <p className="build-muted">{buildCount(path.matches)} {path.matches === 1 ? 'partida' : 'partidas'} · {buildPercentage(path.winRate)} vitórias</p>
      <ol className="ability-order">{path.upgrades.map(({ order, ability }) => <li key={order}>
        <span className="ability-step">{order}</span><DotaImage src={ability.imageUrl} name={talentText(ability.name).label ?? `Habilidade ${ability.id}`} ability /><span className="sr-only">{talentText(ability.name).label ?? `Habilidade ${ability.id}`}</span>
      </li>)}</ol>
      <ul className="ability-legend">{abilities.map(ability => <li key={ability.id}><DotaImage src={ability.imageUrl} name={talentText(ability.name).label ?? `Habilidade ${ability.id}`} ability /><span>{talentText(ability.name).label ?? `Habilidade ${ability.id}`}</span></li>)}</ul>
      <p className="build-inline-note">Os números indicam a ordem das melhorias. Pontos guardados podem mudar o nível em que cada habilidade foi aprendida.</p>
    </>}
  </section>
}

export function TalentsPanel({ options, choices }: { options: TalentDefinition[]; choices: HeroBuild['talents'] }) {
  const levels = [...new Set(options.map(option => option.level))].sort((a, b) => b - a)
  return <section className="build-panel" aria-labelledby="talents-title">
    <h3 id="talents-title">Talentos</h3><p className="build-panel-note">Escolhas observadas nesta build, por nível.</p>
    {levels.length === 0 ? <p className="build-muted">O catálogo não informou talentos para este herói.</p> : <div className="talent-tree">{levels.map(level => {
      const talents = options.filter(option => option.level === level)
      const mostPicked = Math.max(0, ...talents.map(talent => choices.find(choice => choice.ability.id === talent.ability.id)?.matches ?? 0))
      return <div className="talent-tier" key={level}><span className="talent-level">Nível {level}</span><div className="talent-choices">{talents.map((talent, index) => {
        const observed = choices.find(choice => choice.ability.id === talent.ability.id)
        const text = talentText(talent.ability.name)
        return <div className={`talent-choice${observed && observed.matches === mostPicked ? ' is-observed' : ''}`} key={talent.ability.id}>
          <span>{text.label ?? `Opção ${index + 1} do nível ${level}`}</span>
          {text.hasMissingValue && <small>Valor do talento não informado pela fonte</small>}
          <small>{observed ? `${observed.matches} ${observed.matches === 1 ? 'escolha' : 'escolhas'} · ${buildPercentage(observed.winRate)} vitórias` : 'Não observado nesta amostra'}</small>
        </div>
      })}</div></div>
    })}</div>}
  </section>
}

export function BuildItemStrip({ items }: { items: BuildItem[] }) {
  return <span className="build-item-strip">{items.slice(0, 6).map(item => <DotaImage key={item.id} src={item.imageUrl} name={item.name} />)}{items.length > 6 && <span className="build-strip-more">+{items.length - 6}</span>}</span>
}
