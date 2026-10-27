import { useEffect, useMemo, useRef } from 'react'
import { portraitUrl, useApi } from './api'
import type { Hero } from './api'
import { parseMatchId } from './matches'
import { navigate, useSearchParam } from './navigation'
import { parseAccountId } from './players'
import { Header } from './components/Header'
import { StatsPreview } from './components/StatsPreview'
import { MetaSection } from './components/MetaSection'
import { ErrorBoundary } from './components/ErrorBoundary'
import { HeroCatalog } from './components/HeroCatalog'
import { MatchView } from './components/MatchView'
import { PlayerSection } from './components/PlayerSection'
import './App.css'

function App() {
  const [heroes, retryHeroes] = useApi<Hero[]>('/api/heroes')
  const matchParam = useSearchParam('partida')
  const accountParam = useSearchParam('jogador')
  const parsedAccount = accountParam === null ? null : parseAccountId(accountParam)
  const highlightedAccount = parsedAccount && 'accountId' in parsedAccount ? parsedAccount.accountId : null
  // Ao sair da página da partida para a inicial, a seção indicada no endereço (como #jogador) ainda não
  // existia; depois de exibida, a página rola até ela.
  const previousMatch = useRef(matchParam)
  useEffect(() => {
    if (previousMatch.current !== null && matchParam === null) {
      const target = window.location.hash ? document.getElementById(window.location.hash.slice(1)) : null
      if (target) target.scrollIntoView()
      else window.scrollTo(0, 0)
    }
    previousMatch.current = matchParam
  }, [matchParam])
  const heroesById = useMemo(() => new Map(heroes.status === 'success' ? heroes.data.map((hero) => [hero.id, hero] as const) : []), [heroes])
  const portraits = useMemo(() => {
    const entries = heroes.status === 'success' ? heroes.data.map((hero) => [hero.id, portraitUrl(hero)] as const) : []
    return new Map(entries.filter((entry): entry is readonly [number, string] => entry[1] !== undefined))
  }, [heroes])

  return (
    <>
      <a className="skip-link" href="#conteudo">Pular para o conteúdo</a>
      <Header prefix={matchParam === null ? '' : '/'} />
      <main id="conteudo">
        <ErrorBoundary>
        {matchParam !== null ? (
          <MatchView matchId={parseMatchId(matchParam)} heroes={heroesById} highlightedAccount={highlightedAccount} />
        ) : (
          <>
            <section className="hero section-width" id="inicio" aria-labelledby="hero-title">
              <div className="hero-copy">
                <p className="eyebrow"><span className="status-dot" /> Estatísticas. Meta. Partidas. Dota 2.</p>
                <h1 id="hero-title">Menos achismo.<br /><span>Mais jogo.</span></h1>
                <p className="hero-description">Entenda o meta. Conheça seus números.<br />Veja o jogo por outra perspectiva.</p>
                <div className="hero-actions">
                  <a className="button button-primary" href="#meta">Explorar o meta <span aria-hidden="true">↗</span></a>
                  <a className="button button-secondary" href="#jogador">Consultar jogador <svg className="button-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" aria-hidden="true"><circle cx="10.5" cy="10.5" r="6.5" /><path d="m15.5 15.5 5 5" /></svg></a>
                </div>
                <p className="hero-note">Dados públicos. Sua curiosidade é o ponto de partida.</p>
              </div>
              <StatsPreview portraits={portraits} />
            </section>
            <MetaSection portraits={portraits} />
            <PlayerSection heroes={heroesById} />
            <section className="features section-width" id="recursos" aria-labelledby="features-title">
              <h2 className="sr-only" id="features-title">Um olhar completo sobre o jogo</h2>
              <article className="feature">
                <svg viewBox="0 0 32 32" fill="none" stroke="currentColor" strokeWidth="1.4" aria-hidden="true"><circle cx="16" cy="10" r="5" /><path d="M5 28v-3c0-6 5-9 11-9s11 3 11 9v3" /></svg>
                <div><h3>Perfil público</h3><p>Conheça o jogador por trás das partidas, com identificação e rank disponíveis.</p></div>
              </article>
              <article className="feature" id="historico">
                <svg viewBox="0 0 32 32" fill="none" stroke="currentColor" strokeWidth="1.4" aria-hidden="true"><path d="M7 3h13l6 6v20H7zM20 3v7h6M12 15h9M12 20h9M12 25h6" /></svg>
                <div><h3>Histórico de partidas</h3><p>Revisite suas partidas e acompanhe heróis, resultados e desempenho.</p></div>
              </article>
              <article className="feature" id="partidas">
                <svg viewBox="0 0 32 32" fill="none" stroke="currentColor" strokeWidth="1.4" aria-hidden="true"><path d="M4 4v24h25M8 22l6-9 6 4 7-10" /><circle cx="14" cy="13" r="2" /><circle cx="20" cy="17" r="2" /><circle cx="27" cy="7" r="2" /></svg>
                <div><h3>Detalhes da partida</h3><p>De cada lado do mapa: times, resultado e os números de cada jogador.</p></div>
              </article>
            </section>
            <HeroCatalog heroes={heroes} onRetry={retryHeroes} />
          </>
        )}
        </ErrorBoundary>
      </main>
      <footer className="footer section-width">
        <a className="footer-brand" href={matchParam === null ? '#inicio' : '/'} onClick={(event) => { if (matchParam !== null) { event.preventDefault(); navigate('/') } }}>Ancient Analytics <span aria-hidden="true">↗</span></a>
        <p>Feito para quem quer entender o jogo.</p>
        <a href="https://www.opendota.com/" target="_blank" rel="noreferrer">Fonte de dados: OpenDota <span aria-hidden="true">↗</span></a>
      </footer>
    </>
  )
}

export default App
