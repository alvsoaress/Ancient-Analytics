import { useState } from 'react'
import { portraitUrl } from '../api'
import type { ApiState, Hero } from '../api'
import { findHeroes, heroBuildUrl } from '../heroes'
import { navigate } from '../navigation'
import { HeroLink } from './HeroLink'
import { HeroPortrait } from './HeroPortrait'

export function HeroSearch({ heroes, query, onChange, onRetry }: { heroes: ApiState<Hero[]>; query: string; onChange: (value: string) => void; onRetry: () => void }) {
  const [expanded, setExpanded] = useState(false)
  const results = heroes.status === 'success' ? findHeroes(heroes.data, query) : []
  const searching = query.trim().length > 0 && expanded
  return (
    <form className="build-search" role="search" aria-label="Buscar builds de heróis" onBlur={event => {
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setExpanded(false)
    }} onSubmit={event => {
      event.preventDefault()
      const exact = results.find(hero => hero.localized_name.toLocaleLowerCase('pt-BR') === query.trim().toLocaleLowerCase('pt-BR'))
      const hero = exact ?? (results.length === 1 ? results[0] : undefined)
      setExpanded(false)
      if (hero) navigate(heroBuildUrl(hero.id), { scrollToTop: true })
      else document.getElementById('herois')?.scrollIntoView({ behavior: 'instant' })
    }}>
      <label htmlFor="build-hero-search">Buscar build por herói</label>
      <div className="build-search-field">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m15.5 15.5 5 5" /></svg>
        <input id="build-hero-search" type="search" value={query} onChange={event => { onChange(event.target.value); setExpanded(true) }} onFocus={() => setExpanded(true)} onKeyDown={event => { if (event.key === 'Escape') setExpanded(false) }} placeholder="Digite o nome do herói" autoComplete="off" autoCapitalize="none" spellCheck={false} enterKeyHint="search" />
        <button type="submit">Buscar <span aria-hidden="true">↗</span></button>
      </div>
      {searching && (
        <div className="build-search-results">
          {heroes.status === 'loading' ? <p role="status">Carregando os heróis…</p> : heroes.status === 'error'
            ? <p role="alert">Catálogo indisponível. <button type="button" className="inline-button" onClick={onRetry}>Tentar novamente</button></p>
            : <><p className="search-result-count" role="status">{results.length ? `${results.length} ${results.length === 1 ? 'herói encontrado' : 'heróis encontrados'}` : 'Nenhum herói encontrado. Tente outro nome.'}</p>
              <ul aria-label="Resultados da busca por herói">{results.slice(0, 5).map(hero => <li key={hero.id}><HeroLink heroId={hero.id} name={hero.localized_name}><HeroPortrait src={portraitUrl(hero)} name={hero.localized_name} /><span>{hero.localized_name}<small>Ver builds</small></span><span aria-hidden="true">↗</span></HeroLink></li>)}</ul>
            </>}
        </div>
      )}
    </form>
  )
}
