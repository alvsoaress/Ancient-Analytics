import { useRef, useState } from 'react'
import type { MouseEvent } from 'react'
import { navigate } from '../navigation'

const links = [['#herois', 'Heróis'], ['#meta', 'Meta'], ['#jogador', 'Jogadores'], ['#partidas', 'Partidas']]

// Nas páginas de detalhes, as seções da página inicial não estão presentes; os links levam de volta a ela.
export function Header({ prefix = '' }: { prefix?: string }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const menuButton = useRef<HTMLButtonElement>(null)
  // Fora da página inicial, volta a ela sem recarregar; o App rola até a seção indicada.
  const goTo = (event: MouseEvent<HTMLAnchorElement>) => {
    setMenuOpen(false)
    if (!prefix) return
    event.preventDefault()
    navigate(event.currentTarget.getAttribute('href') ?? '/')
  }

  return (
    <header className="site-header">
      <div className="header-content section-width" onKeyDown={(event) => {
        if (event.key === 'Escape' && menuOpen) {
          setMenuOpen(false)
          menuButton.current?.focus()
        }
      }}>
        <a className="brand" href={`${prefix}#inicio`} aria-label="Ancient Analytics — início" onClick={goTo}>
          <img src="/favicon.svg" width="44" height="44" alt="" /><span>Ancient<br />Analytics</span>
        </a>
        <button ref={menuButton} className="menu-toggle" type="button" aria-expanded={menuOpen} aria-controls="main-navigation" onClick={() => setMenuOpen(!menuOpen)}>
          {menuOpen ? 'Fechar' : 'Menu'}<span className={menuOpen ? 'menu-lines is-open' : 'menu-lines'} aria-hidden="true"><i /><i /></span>
        </button>
        <nav className={menuOpen ? 'main-navigation is-open' : 'main-navigation'} id="main-navigation" aria-label="Navegação principal">
          {links.map(([href, label]) => <a key={href} href={prefix + href} onClick={goTo}>{label}</a>)}
        </nav>
        <a className="button button-primary header-action" href={`${prefix}#meta`} onClick={goTo}>Explorar <span aria-hidden="true">↗</span></a>
      </div>
    </header>
  )
}
