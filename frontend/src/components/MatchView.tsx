import { portraitUrl, useApi } from '../api'
import type { Hero } from '../api'
import { describeMode, formatDuration, teamNames } from '../matches'
import type { MatchDetails, MatchPlayer, Team } from '../matches'
import { useEffect, useRef } from 'react'
import { canGoBack, navigate, useDocumentTitle } from '../navigation'
import { HeroPortrait } from './HeroPortrait'
import { ErrorState, LoadingState, Unavailable } from './RequestState'

const integer = new Intl.NumberFormat('pt-BR')
const dateTime = new Intl.DateTimeFormat('pt-BR', { day: '2-digit', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit' })

const number = (value: number | null) => (value === null ? <Unavailable /> : integer.format(value))
const plain = (value: number | null) => (value === null ? '—' : String(value))

function PlayerLink({ player }: { player: MatchPlayer }) {
  if (player.accountId === null) return <span className="unavailable">Perfil anônimo</span>
  const url = `?jogador=${player.accountId}#jogador`
  return (
    <a className="player-link" href={url} onClick={(event) => { event.preventDefault(); navigate(url) }}>
      {player.name ?? `Conta ${player.accountId}`}
    </a>
  )
}

function TeamTable({ title, team, players, winner, heroes, maxDamage, highlightedAccount }: {
  title: string; team: Team | null; players: MatchPlayer[]; winner: Team | null
  heroes: Map<number, Hero>; maxDamage: number; highlightedAccount: number | null
}) {
  const kills = players.reduce((sum, player) => sum + (player.kills ?? 0), 0)
  return (
    <section className={`team team-${team ?? 'unknown'}`} aria-labelledby={`team-${team ?? 'unknown'}`}>
      <div className="team-heading">
        <h2 id={`team-${team ?? 'unknown'}`}>{title}</h2>
        {team && winner && <span className={team === winner ? 'team-result is-winner' : 'team-result'}>{team === winner ? 'Vitória' : 'Derrota'}</span>}
        <span className="team-kills">{integer.format(kills)} abates</span>
      </div>
      <div className="table-scroll" tabIndex={0} role="region" aria-label={`Desempenho dos jogadores: ${title}`}>
        <table className="match-table">
          <thead><tr>
            <th scope="col">Jogador</th><th scope="col">Nível</th><th scope="col">K / D / A</th>
            <th scope="col">GPM</th><th scope="col" className="optional-column">XPM</th>
            <th scope="col">Dano em heróis</th><th scope="col" className="optional-column">Dano em torres</th>
            <th scope="col" className="optional-column">Cura</th><th scope="col" className="optional-column">Últimos golpes / Negados</th>
          </tr></thead>
          <tbody>{players.map((player, index) => {
            const hero = player.heroId === null ? undefined : heroes.get(player.heroId)
            const heroName = hero?.localized_name ?? (player.heroId === null ? 'Herói indisponível' : `Herói ${player.heroId}`)
            const highlighted = highlightedAccount !== null && player.accountId === highlightedAccount
            return (
              <tr key={player.playerSlot ?? `sem-slot-${index}`} className={highlighted ? 'is-highlighted' : undefined}>
                <th scope="row">
                  <span className="hero-cell">
                    <HeroPortrait src={hero && portraitUrl(hero)} name={heroName} />
                    <span className="match-player">
                      <span>{heroName}{highlighted && <span className="sr-only"> (jogador consultado)</span>}</span>
                      <PlayerLink player={player} />
                    </span>
                  </span>
                </th>
                <td>{number(player.level)}</td>
                <td className="kda">{plain(player.kills)} / {plain(player.deaths)} / {plain(player.assists)}</td>
                <td>{number(player.goldPerMinute)}</td>
                <td className="optional-column">{number(player.xpPerMinute)}</td>
                <td>
                  {player.heroDamage === null ? <Unavailable /> : (
                    <span className="damage">
                      {integer.format(player.heroDamage)}
                      <span className="stat-track" aria-hidden="true"><span style={{ width: `${maxDamage > 0 ? player.heroDamage / maxDamage * 100 : 0}%` }} /></span>
                    </span>
                  )}
                </td>
                <td className="optional-column">{number(player.towerDamage)}</td>
                <td className="optional-column">{number(player.heroHealing)}</td>
                <td className="optional-column">{plain(player.lastHits)} / {plain(player.denies)}</td>
              </tr>
            )
          })}</tbody>
        </table>
      </div>
    </section>
  )
}

export function MatchView({ matchId, heroes, highlightedAccount }: { matchId: number | null; heroes: Map<number, Hero>; highlightedAccount: number | null }) {
  const back = highlightedAccount === null
    ? { label: 'Voltar ao início', url: '/' }
    : { label: 'Voltar ao jogador', url: `?jogador=${highlightedAccount}#jogador` }
  const section = useRef<HTMLElement>(null)
  useDocumentTitle(matchId === null ? 'Partida inválida' : `Partida ${matchId}`)
  // Ao abrir outra partida, o foco vai para a nova página, e leitores de tela anunciam seu título.
  useEffect(() => section.current?.focus(), [matchId])

  return (
    <section ref={section} className="match-view section-width" aria-labelledby="match-title" tabIndex={-1}>
      <a className="text-link back-link" href={back.url} onClick={(event) => {
        event.preventDefault()
        if (canGoBack()) window.history.back()
        else navigate(back.url)
      }}><span aria-hidden="true">←</span> {back.label}</a>
      {matchId === null
        ? (
          <div className="request-state" role="alert">
            <h1 id="match-title">Identificador de partida inválido</h1>
            <p>O endereço deve conter um número de partida, por exemplo <code>?partida=9025128045</code>.</p>
          </div>
        )
        : <MatchContent key={matchId} matchId={matchId} heroes={heroes} highlightedAccount={highlightedAccount} />}
    </section>
  )
}

function MatchContent({ matchId, heroes, highlightedAccount }: { matchId: number; heroes: Map<number, Hero>; highlightedAccount: number | null }) {
  const [match, retry] = useApi<MatchDetails>(`/api/matches/${matchId}`)

  if (match.status === 'loading') return <><h1 className="sr-only" id="match-title">Partida {matchId}</h1><LoadingState>Carregando a partida.</LoadingState></>
  if (match.status === 'error') {
    return (
      <>
        <h1 className="sr-only" id="match-title">Partida {matchId}</h1>
        {match.error.code === 'MATCH_NOT_FOUND'
          ? <div className="request-state" role="alert"><h2>Partida não encontrada na OpenDota</h2><p>Confira o número da partida.</p></div>
          : <ErrorState title="Não foi possível carregar a partida" error={match.error} onRetry={retry} />}
      </>
    )
  }

  const data = match.data
  const { mode, lobby } = describeMode(data.gameMode, data.lobbyType)
  const maxDamage = Math.max(0, ...data.players.map((player) => player.heroDamage ?? 0))
  const groups: { title: string; team: Team | null; players: MatchPlayer[] }[] = [
    { title: teamNames.radiant, team: 'radiant', players: data.players.filter((player) => player.team === 'radiant') },
    { title: teamNames.dire, team: 'dire', players: data.players.filter((player) => player.team === 'dire') },
    { title: 'Sem lado identificado', team: null, players: data.players.filter((player) => player.team === null) },
  ]

  return (
    <>
      <header className="match-header">
        <div>
          <p className="eyebrow">Partida {data.matchId}</p>
          <h1 id="match-title">{data.winner ? <>Vitória dos <span>{teamNames[data.winner]}</span></> : 'Resultado indisponível'}</h1>
          <dl className="match-facts">
            <div><dt>Duração</dt><dd>{data.durationSeconds === null ? <Unavailable /> : formatDuration(data.durationSeconds)}</dd></div>
            <div><dt>Início</dt><dd>{data.startTime === null ? <Unavailable /> : <time dateTime={new Date(data.startTime * 1000).toISOString()}>{dateTime.format(data.startTime * 1000)}</time>}</dd></div>
            <div><dt>Modo</dt><dd>{mode ?? <Unavailable />}{lobby && ` · ${lobby}`}</dd></div>
          </dl>
          <a className="text-link" href={`https://www.opendota.com/matches/${data.matchId}`} target="_blank" rel="noreferrer">Ver na OpenDota <span aria-hidden="true">↗</span></a>
        </div>
        <div className="scoreboard" role="group" aria-label={`Placar: ${teamNames.radiant} ${plain(data.radiantScore)}, ${teamNames.dire} ${plain(data.direScore)}`}>
          <div className={data.winner === 'radiant' ? 'is-winner' : undefined}><span>{teamNames.radiant}</span><strong>{plain(data.radiantScore)}</strong></div>
          <span className="score-separator" aria-hidden="true">×</span>
          <div className={data.winner === 'dire' ? 'is-winner' : undefined}><span>{teamNames.dire}</span><strong>{plain(data.direScore)}</strong></div>
        </div>
      </header>
      {data.players.length === 0
        ? <div className="request-state"><h2>Jogadores indisponíveis</h2><p>A OpenDota não forneceu os jogadores desta partida.</p></div>
        : groups.filter((group) => group.players.length > 0).map((group) => (
          <TeamTable key={group.title} {...group} winner={data.winner} heroes={heroes} maxDamage={maxDamage} highlightedAccount={highlightedAccount} />
        ))}
      <p className="data-note">Estatísticas que dependem da análise do replay podem não estar disponíveis e aparecem como "—".</p>
    </>
  )
}
