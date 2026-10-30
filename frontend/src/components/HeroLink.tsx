import type { ReactNode } from 'react'
import { heroBuildUrl } from '../heroes'
import { navigate } from '../navigation'

export function HeroLink({ heroId, name, children, className }: { heroId: number; name: string; children: ReactNode; className?: string }) {
  const url = heroBuildUrl(heroId)
  return <a className={className} href={url} aria-label={`Ver builds de ${name}`} onClick={event => {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    event.preventDefault()
    navigate(url, { scrollToTop: true })
  }}>{children}</a>
}
