import { useEffect, useState } from 'react'

// Navegação sem recarregar: grava a URL no histórico e avisa os componentes, que ouvem `popstate`
// tanto para o botão voltar do navegador quanto para estas mudanças.
export function navigate(url: string, { scrollToTop = false } = {}) {
  window.history.pushState({ inApp: true }, '', url)
  window.dispatchEvent(new PopStateEvent('popstate', { state: { inApp: true } }))
  if (scrollToTop) window.scrollTo(0, 0)
}

export function useSearchParam(name: string) {
  const read = () => new URLSearchParams(window.location.search).get(name)
  const [value, setValue] = useState(read)
  useEffect(() => {
    const sync = () => setValue(new URLSearchParams(window.location.search).get(name))
    window.addEventListener('popstate', sync)
    return () => window.removeEventListener('popstate', sync)
  }, [name])
  return value
}

// Indica se há uma página anterior desta aplicação para onde o "voltar" pode retornar.
export const canGoBack = () => (window.history.state as { inApp?: boolean } | null)?.inApp === true

const defaultTitle = document.title

// Título da aba conforme a tela; volta ao título padrão quando a tela sai.
export function useDocumentTitle(title: string) {
  useEffect(() => {
    document.title = `${title} — Ancient Analytics`
    return () => { document.title = defaultTitle }
  }, [title])
}
