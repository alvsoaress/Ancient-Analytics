import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useApi } from '../api'
import type { Hero } from '../api'
import { navigate, useDocumentTitle } from '../navigation'
import { describeRank, parseAccountId } from '../players'
import type { PlayerProfile } from '../players'
import { PlayerMatches } from './PlayerMatches'
import { ErrorState, LoadingState } from './RequestState'

const queryParam = 'jogador'
const integer = new Intl.NumberFormat('pt-BR')

function accountFromUrl() {
  const value = new URLSearchParams(window.location.search).get(queryParam)
  const parsed = value ? parseAccountId(value) : null
  return parsed && 'accountId' in parsed ? parsed.accountId : null
}

function Avatar({ src, name }: { src: string | null; name: string }) {
  const [failedSrc, setFailedSrc] = useState<string>()
  if (!src || failedSrc === src) return <span className="player-avatar avatar-fallback" aria-hidden="true">{name.charAt(0).toUpperCase() || '?'}</span>
  return <img className="player-avatar" src={src} alt="" width="112" height="112" onError={() => setFailedSrc(src)} />
}

function RankBadge({ rankTier, leaderboardRank }: { rankTier: number | null; leaderboardRank: number | null }) {
  const rank = rankTier === null ? null : describeRank(rankTier)
  return (
    <dl className="player-stats">
      <div>
        <dt>Medalha</dt>
        <dd>
          {rank ? (
            <span className="rank-medal">
              {rank.label}
              {rank.stars > 0 && <span className="rank-stars" aria-hidden="true">{Array.from({ length: 5 }, (_, index) => <i key={index} className={index < rank.stars ? 'is-filled' : undefined} />)}</span>}
            </span>
          ) : <span className="unavailable">Indisponível</span>}
        </dd>
      </div>
      <div>
        <dt>Ranking</dt>
        <dd>{leaderboardRank === null ? <span className="unavailable">Fora do ranking ou indisponível</span> : `#${integer.format(leaderboardRank)}`}</dd>
      </div>
    </dl>
  )
}

function PlayerProfile({ accountId, heroes }: { accountId: number; heroes: Map<number, Hero> }) {
  const [profile, retry] = useApi<PlayerProfile>(`/api/players/${accountId}`)
  useDocumentTitle(profile.status === 'success' && profile.data.name ? profile.data.name : `Jogador ${accountId}`)

  if (profile.status === 'loading') return <LoadingState>Consultando o perfil na OpenDota.</LoadingState>
  if (profile.status === 'error') {
    if (profile.error.code === 'PLAYER_NOT_FOUND') {
      return (
        <div className="request-state" role="alert">
          <h3>Perfil não disponível na OpenDota</h3>
          <p>Confira o ID de amigo informado. A OpenDota só exibe contas com dados públicos disponíveis.</p>
        </div>
      )
    }
    return <ErrorState title="Não foi possível consultar o jogador" error={profile.error} onRetry={retry} />
  }

  const player = profile.data
  const name = player.name ?? 'Nome indisponível'
  return (
    <>
      <article className="player-card" aria-labelledby="player-name">
        <Avatar src={player.avatarUrl} name={player.name ?? ''} />
        <div className="player-identity">
          <p className="eyebrow">Perfil público</p>
          <h3 id="player-name" className={player.name ? undefined : 'unavailable'}>{name}</h3>
          <p className="player-ids">
            <span>ID de amigo <strong>{player.accountId}</strong></span>
            {player.steamId && <span>SteamID64 <strong>{player.steamId}</strong></span>}
          </p>
          <div className="player-links">
            {player.profileUrl && <a className="text-link" href={player.profileUrl} target="_blank" rel="noreferrer">Perfil na Steam <span aria-hidden="true">↗</span></a>}
            <a className="text-link" href={`https://www.opendota.com/players/${player.accountId}`} target="_blank" rel="noreferrer">Ver na OpenDota <span aria-hidden="true">↗</span></a>
          </div>
        </div>
        <RankBadge rankTier={player.rankTier} leaderboardRank={player.leaderboardRank} />
      </article>
      <PlayerMatches accountId={player.accountId} heroes={heroes} />
    </>
  )
}

export function PlayerSection({ heroes }: { heroes: Map<number, Hero> }) {
  const [accountId, setAccountId] = useState(accountFromUrl)
  const [input, setInput] = useState(() => accountId?.toString() ?? '')
  const [formError, setFormError] = useState<string>()

  useEffect(() => {
    const syncWithUrl = () => {
      const fromUrl = accountFromUrl()
      setAccountId(fromUrl)
      setInput(fromUrl?.toString() ?? '')
      setFormError(undefined)
    }
    window.addEventListener('popstate', syncWithUrl)
    return () => window.removeEventListener('popstate', syncWithUrl)
  }, [])

  const search = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const parsed = parseAccountId(input)
    if ('error' in parsed) {
      setFormError(parsed.error)
      return
    }
    setFormError(undefined)
    setInput(String(parsed.accountId))
    if (parsed.accountId !== accountId) {
      setAccountId(parsed.accountId)
      navigate(`?${queryParam}=${parsed.accountId}#jogador`)
    }
  }

  return (
    <section className="player-section section-width" id="jogador" aria-labelledby="player-title">
      <div className="player-heading">
        <div>
          <p className="eyebrow">Jogadores</p>
          <h2 id="player-title">Consultar jogador</h2>
          <p>Busque pelo ID de amigo do Dota 2. Também aceitamos o SteamID64 e links de perfil da Steam, OpenDota ou Dotabuff.</p>
        </div>
        <form className="player-search" onSubmit={search} noValidate>
          <label htmlFor="player-id">ID do jogador</label>
          <div>
            <input id="player-id" type="text" inputMode="text" autoComplete="off" spellCheck={false} placeholder="Ex.: 123456789" value={input} aria-invalid={formError ? true : undefined} aria-describedby={formError ? 'player-id-error' : 'player-id-hint'} onChange={(event) => setInput(event.target.value)} />
            <button className="button button-primary" type="submit">Consultar</button>
          </div>
          {formError
            ? <p className="field-error" id="player-id-error" role="alert">{formError}</p>
            : <p className="field-hint" id="player-id-hint">No Dota 2, o ID de amigo aparece no seu perfil, ao lado do nome.</p>}
        </form>
      </div>
      {accountId !== null && <PlayerProfile key={accountId} accountId={accountId} heroes={heroes} />}
    </section>
  )
}
