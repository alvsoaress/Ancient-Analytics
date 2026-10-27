import { useState } from 'react'

export function HeroPortrait({ src, name }: { src?: string; name: string }) {
  const [failedSrc, setFailedSrc] = useState<string>()
  if (!src || failedSrc === src) return <span className="hero-portrait portrait-unavailable" role="img" aria-label={`Retrato de ${name} indisponível`}>Sem imagem</span>
  return <img className="hero-portrait" src={src} alt="" width="96" height="54" loading="lazy" onError={() => setFailedSrc(src)} />
}
