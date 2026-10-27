import { Component } from 'react'
import type { ReactNode } from 'react'

// Evita a tela em branco se um componente falhar ao exibir dados inesperados.
export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false }

  static getDerivedStateFromError() {
    return { failed: true }
  }

  render() {
    if (!this.state.failed) return this.props.children
    return (
      <div className="request-state error-state section-width page-error" role="alert">
        <h2>Algo deu errado ao exibir esta página</h2>
        <p>Recarregue a página para tentar novamente.</p>
        <button className="button button-secondary" type="button" onClick={() => window.location.reload()}>Recarregar</button>
      </div>
    )
  }
}
