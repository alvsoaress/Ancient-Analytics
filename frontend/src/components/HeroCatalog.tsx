import { useState } from 'react'
import { findHeroes } from '../heroes'
import { HeroLink } from './HeroLink'
import { portraitUrl } from '../api'
import type { ApiState, Attribute, Hero } from '../api'
import { HeroPortrait } from './HeroPortrait'
import { ErrorState, LoadingState } from './RequestState'

const attributes: { value: Attribute | ''; label: string }[] = [
  { value: '', label: 'Todos' }, { value: 'str', label: 'Força' }, { value: 'agi', label: 'Agilidade' },
  { value: 'int', label: 'Inteligência' }, { value: 'all', label: 'Universal' },
]
const attackTypes = { Melee: 'Corpo a corpo', Ranged: 'À distância' }

export function HeroCatalog({ heroes, onRetry, search, onSearch }: { heroes: ApiState<Hero[]>; onRetry: () => void; search: string; onSearch: (value: string) => void }) {
  const [attribute, setAttribute] = useState<Attribute | ''>('')

  let content
  if (heroes.status === 'loading') {
    content = <LoadingState>Carregando o catálogo de heróis.</LoadingState>
  } else if (heroes.status === 'error') {
    content = <ErrorState title="Não foi possível carregar os heróis" error={heroes.error} onRetry={onRetry} />
  } else {
    const visible = findHeroes(heroes.data, search, attribute)
    content = (
      <>
        <p className="data-note" role="status" aria-live="polite">{visible.length} de {heroes.data.length} heróis</p>
        {visible.length ? (
          <ul className="hero-catalog">{visible.map((hero) => (
            <li key={hero.id}>
              <HeroLink className="catalog-hero-link" heroId={hero.id} name={hero.localized_name}>
                <HeroPortrait src={portraitUrl(hero)} name={hero.localized_name} />
                <span className="hero-name">{hero.localized_name}</span>
                <span className="hero-meta">{attributes.find((item) => item.value === hero.primary_attr)?.label ?? 'Atributo indisponível'} · {attackTypes[hero.attack_type] ?? 'Ataque indisponível'}</span>
                <span className="catalog-build-link">Ver builds <span aria-hidden="true">↗</span></span>
              </HeroLink>
            </li>
          ))}</ul>
        ) : (
          <div className="request-state">
            <h3>Nenhum herói encontrado</h3>
            <p>Tente outro nome ou limpe os filtros.</p>
            <button className="button button-secondary" type="button" onClick={() => { onSearch(''); setAttribute('') }}>Limpar filtros</button>
          </div>
        )}
      </>
    )
  }

  return (
    <section className="catalog section-width" id="herois" aria-labelledby="catalog-title">
      <div className="catalog-heading">
        <div><p className="eyebrow">Cada escolha muda a partida</p><h2 id="catalog-title">Os rostos do jogo.</h2><p>Escolha um herói para explorar itens, habilidades e talentos.</p></div>
        <div className="hero-search"><label htmlFor="hero-search">Filtrar heróis</label><div><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m15.5 15.5 5 5" /></svg><input id="hero-search" type="search" placeholder="Nome do herói" autoComplete="off" value={search} onChange={(event) => onSearch(event.target.value)} /></div></div>
      </div>
      <div className="position-filters" role="group" aria-label="Filtrar por atributo principal">
        {attributes.map((item) => <button key={item.value} type="button" aria-pressed={attribute === item.value} onClick={() => setAttribute(item.value)}>{item.label}</button>)}
      </div>
      {content}
    </section>
  )
}
