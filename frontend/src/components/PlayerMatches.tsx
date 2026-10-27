import { useEffect, useState } from 'react'
import { portraitUrl, useApi } from '../api'
import type { Hero } from '../api'
import { describeMode, formatDuration } from '../matches'
import { navigate } from '../navigation'
import type { PlayerMatchesPage } from '../matches'
import { HeroPortrait } from './HeroPortrait'
import { ErrorState, LoadingState, Unavailable } from './RequestState'

const pageSize = 20
const dateTime = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: '2-digit', year: '2-digit', hour: '2-digit', minute: '2-digit' })

const value = (number: number | null) => (number === null ? '—' : String(number))

// A página do histórico fica em ?pagina=N (a partir de 1), ao lado de ?jogador=ID.
const pageParam = 'pagina'
const maxPage = 100000

function offsetFromUrl() {
  const page = Number(new URLSearchParams(window.location.search).get(pageParam))
  return Number.isInteger(page) && page > 1 && page <= maxPage ? (page - 1) * pageSize : 0
}

export function PlayerMatches({ accountId, heroes }: { accountId: number; heroes: Map<number, Hero> }) {
  const [offset, setOffset] = useState(offsetFromUrl)

  useEffect(() => {
    const syncWithUrl = () => setOffset(offsetFromUrl())
    window.addEventListener('popstate', syncWithUrl)
    return () => window.removeEventListener('popstate', syncWithUrl)
  }, [])

  const matchUrl = (matchId: number) => `?partida=${matchId}&jogador=${accountId}`

  const goTo = (nextOffset: number) => {
    const params = new URLSearchParams(window.location.search)
    if (nextOffset > 0) params.set(pageParam, String(nextOffset / pageSize + 1))
    else params.delete(pageParam)
    navigate(`?${params}${window.location.hash}`)
    document.getElementById('matches-title')?.scrollIntoView({ block: 'start' })
  }
  const [page, retry] = useApi<PlayerMatchesPage>(`/api/players/${accountId}/matches?limit=${pageSize}&offset=${offset}`)

  const data = page.status === 'success' ? page.data : page.status === 'loading' ? page.previous : undefined
  const refreshing = page.status === 'loading' && data !== undefined

  let content
  if (page.status === 'error') {
    content = <ErrorState title="Não foi possível carregar o histórico" error={page.error} onRetry={retry} />
  } else if (!data) {
    content = <LoadingState>Carregando partidas.</LoadingState>
  } else if (data.matches.length === 0) {
    content = (
      <div className="request-state">
        <h3>{offset === 0 ? 'Nenhuma partida pública disponível' : 'Não há mais partidas'}</h3>
        <p>{offset === 0 ? 'A OpenDota não retornou partidas para esta conta. Ela pode não ter partidas ou não compartilhar o histórico.' : 'O histórico pode ter mudado desde a página anterior.'}</p>
        {offset > 0 && <button className="button button-secondary" type="button" onClick={() => goTo(0)}>Voltar ao início</button>}
      </div>
    )
  } else {
    const { matches, pagination } = data
    const wins = matches.filter((match) => match.won === true).length
    const losses = matches.filter((match) => match.won === false).length
    content = (
      <>
        <p className="data-note">
          Partidas {pagination.offset + 1}–{pagination.offset + pagination.count}, da mais recente para a mais antiga. Nesta página: {wins} {wins === 1 ? 'vitória' : 'vitórias'} e {losses} {losses === 1 ? 'derrota' : 'derrotas'}.
        </p>
        <div className="table-scroll" tabIndex={0} role="region" aria-label="Tabela de partidas">
          <table className="matches-table">
            <caption className="sr-only">Partidas do jogador: herói, resultado, abates, mortes e assistências, duração, modo e data.</caption>
            <thead><tr>
              <th scope="col">Herói</th><th scope="col">Resultado</th><th scope="col">K / D / A</th><th scope="col">Duração</th>
              <th scope="col" className="optional-column">Modo</th><th scope="col">Data</th><th scope="col">Partida</th>
            </tr></thead>
            <tbody>{matches.map((match) => {
              const hero = match.heroId === null ? undefined : heroes.get(match.heroId)
              const heroName = hero?.localized_name ?? (match.heroId === null ? 'Herói indisponível' : `Herói ${match.heroId}`)
              const { mode, lobby } = describeMode(match.gameMode, match.lobbyType)
              const result = match.won === null ? 'unknown' : match.won ? 'win' : 'loss'
              return (
                <tr key={match.matchId} className={`result-${result}`}>
                  <th scope="row"><span className="hero-cell"><HeroPortrait src={hero && portraitUrl(hero)} name={heroName} /><span>{heroName}</span></span></th>
                  <td><span className="result-label">{result === 'win' ? 'Vitória' : result === 'loss' ? 'Derrota' : <Unavailable />}</span></td>
                  <td className="kda">{value(match.kills)} / {value(match.deaths)} / {value(match.assists)}</td>
                  <td>{match.durationSeconds === null ? <Unavailable /> : formatDuration(match.durationSeconds)}</td>
                  <td className="optional-column">{mode ?? <Unavailable />}{lobby && <small>{lobby}</small>}</td>
                  <td>{match.startTime === null ? <Unavailable /> : <time dateTime={new Date(match.startTime * 1000).toISOString()}>{dateTime.format(match.startTime * 1000)}</time>}</td>
                  <td><a className="text-link" href={matchUrl(match.matchId)} onClick={(event) => { event.preventDefault(); navigate(matchUrl(match.matchId), { scrollToTop: true }) }}><span className="match-id">{match.matchId}</span> Detalhes <span aria-hidden="true">→</span><span className="sr-only">da partida {match.matchId}</span></a></td>
                </tr>
              )
            })}</tbody>
          </table>
        </div>
        <nav className="pagination" aria-label="Páginas do histórico">
          <button className="button button-secondary" type="button" disabled={refreshing || offset === 0} onClick={() => goTo(Math.max(0, offset - pageSize))}>← Anteriores</button>
          <span>Página {Math.floor(offset / pageSize) + 1}</span>
          <button className="button button-secondary" type="button" disabled={refreshing || !pagination.hasMore || pagination.nextOffset === null} onClick={() => pagination.nextOffset !== null && goTo(pagination.nextOffset)}>Próximas →</button>
        </nav>
      </>
    )
  }

  return (
    <section className="player-matches" aria-labelledby="matches-title">
      <h3 id="matches-title">Histórico de partidas</h3>
      {refreshing && <p className="refresh-note" role="status"><span className="loading-bar" aria-hidden="true" />Carregando partidas.</p>}
      <div className={refreshing ? 'is-refreshing' : undefined} aria-busy={refreshing}>{content}</div>
    </section>
  )
}
