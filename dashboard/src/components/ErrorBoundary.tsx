import { Component, ErrorInfo, ReactNode } from 'react';
import { AlertTriangle, RefreshCw } from 'lucide-react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null, errorInfo: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error, errorInfo: null };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    this.setState({ errorInfo });

    // Log to console for debugging
    console.error('[ErrorBoundary]', error.message, error.stack);
    console.error('[ErrorBoundary] Component stack:', errorInfo.componentStack);

    // Try to send error to backend for remote debugging
    try {
      const body = JSON.stringify({
        message: error.message,
        stack: error.stack,
        componentStack: errorInfo.componentStack,
        url: window.location.href,
        userAgent: navigator.userAgent,
      });
      // Use sendBeacon for fire-and-forget
      navigator.sendBeacon?.('/api/log/client-error', body);
    } catch {
      // Best effort
    }
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null, errorInfo: null });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div
          className="app-layout"
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: '100dvh',
            padding: 24,
            textAlign: 'center',
            gap: 16,
            background: 'var(--bg, #fff)',
            color: 'var(--text, #111)',
          }}
        >
          <div
            style={{
              width: 64,
              height: 64,
              borderRadius: 20,
              background: 'rgba(239,68,68,0.1)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <AlertTriangle size={32} style={{ color: '#ef4444' }} />
          </div>

          <h2 style={{ fontSize: 20, fontWeight: 800, margin: 0 }}>
            Ops! Algo deu errado
          </h2>

          <p
            style={{
              fontSize: 14,
              color: 'var(--hint, #888)',
              maxWidth: 360,
              lineHeight: 1.6,
              margin: 0,
            }}
          >
            Ocorreu um erro inesperado ao carregar esta página.
            O erro foi registrado e será analisado.
          </p>

          <div
            style={{
              background: 'rgba(0,0,0,0.05)',
              borderRadius: 12,
              padding: '12px 16px',
              maxWidth: '100%',
              overflow: 'auto',
              fontSize: 11,
              fontFamily: 'monospace',
              textAlign: 'left',
              color: 'var(--hint, #888)',
              wordBreak: 'break-word',
            }}
          >
            <strong>Erro:</strong> {this.state.error?.message || 'Desconhecido'}
          </div>

          <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
            <button
              onClick={this.handleReload}
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 6,
                padding: '10px 20px',
                borderRadius: 12,
                border: 'none',
                background: 'var(--accent, #6c5ce7)',
                color: '#fff',
                fontSize: 14,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              <RefreshCw size={16} />
              Recarregar
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
