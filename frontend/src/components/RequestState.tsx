import type { ApiError } from '../api'

export function Unavailable() {
  return <span className="unavailable"><span aria-hidden="true">—</span><span className="sr-only">Indisponível</span></span>
}

export function LoadingState({ children }: { children: string }) {
  return <div className="request-state" role="status"><span className="loading-bar" aria-hidden="true" /><p>{children}</p></div>
}

export function ErrorState({ title, error, onRetry }: { title: string; error: ApiError; onRetry: () => void }) {
  return (
    <div className="request-state error-state" role="alert">
      <h3>{title}</h3>
      <p>{error.message}</p>
      <button className="button button-secondary" type="button" onClick={onRetry}>Tentar novamente</button>
    </div>
  )
}
