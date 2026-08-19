import { useState, useEffect, useCallback } from 'react';
import {
  fetchAccountStatus,
  fetchAuthStatus,
  connectAccount,
  verifyCode,
  sendPassword,
  disconnectAccount,
} from '../api';
import { AccountStatus, AuthStatus } from '../types';
import { ConnectedAccountCard } from './ConnectedAccountCard';
import { AuthDisclaimer } from './AuthDisclaimer';
import { AuthFlow } from './AuthFlow';
import { Loader2, CheckCircle2 } from 'lucide-react';
import { Button } from './ui/button';

type FlowStep = 'idle' | 'disclaimer' | 'auth' | 'done';

interface ContaTelegramTabProps {
  /** Se true, pula o estado idle e vai direto pro disclaimer (usado no modal) */
  startConnecting?: boolean;
}

export function ContaTelegramTab({ startConnecting }: ContaTelegramTabProps) {
  const [loading, setLoading] = useState(true);
  const [accountStatus, setAccountStatus] = useState<AccountStatus>({ status: 'disconnected' });
  const [authStatus, setAuthStatus] = useState<AuthStatus>({ step: 'phone' });
  const [flowStep, setFlowStep] = useState<FlowStep>(startConnecting ? 'disclaimer' : 'idle');

  // Load initial status
  const loadStatus = useCallback(async () => {
    setLoading(true);
    try {
      const status = await fetchAccountStatus();
      setAccountStatus(status);
      // Se ja estava conectado e tentou conectar, volta pra idle
      if (startConnecting && status.status === 'connected') {
        setFlowStep('idle');
      }
    } catch (err) {
      console.error('Erro ao carregar status da conta:', err);
    } finally {
      setLoading(false);
    }
  }, [startConnecting]);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const handleConnect = async () => {
    setFlowStep('disclaimer');
  };

  const handleAcceptTerms = async () => {
    setFlowStep('auth');
    // Check current auth step status
    try {
      const status = await fetchAuthStatus();
      setAuthStatus(status);
    } catch {
      setAuthStatus({ step: 'phone' });
    }
  };

  const handleCancel = () => {
    setFlowStep('idle');
    setAuthStatus({ step: 'phone' });
  };

  const handleSendPhone = async (phone: string) => {
    const result = await connectAccount(phone);
    if (result.step === 'error') {
      throw new Error(result.error || 'Erro ao enviar código');
    }
    setAuthStatus(result);
  };

  const handleVerifyCode = async (code: string) => {
    const result = await verifyCode(code);
    if (result.step === 'error') {
      throw new Error(result.error || 'Erro ao verificar código');
    }
    setAuthStatus(result);
    if (result.step === 'done') {
      setFlowStep('done');
      await loadStatus(); // Reload account status
    }
    return result;
  };

  const handleSendPassword = async (password: string) => {
    const result = await sendPassword(password);
    if (result.step === 'error') {
      throw new Error(result.error || 'Erro ao verificar senha');
    }
    if (result.step === 'done') {
      setFlowStep('done');
      await loadStatus();
    }
  };

  const handleDisconnect = async () => {
    if (!confirm('Tem certeza que deseja desconectar sua conta Telegram?')) {
      return;
    }
    try {
      await disconnectAccount();
      setAccountStatus({ status: 'disconnected' });
      setFlowStep('idle');
    } catch (err) {
      console.error('Erro ao desconectar:', err);
      alert('Erro ao desconectar conta. Tente novamente.');
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center py-10 gap-3">
        <Loader2 size={32} className="animate-spin text-accent" />
        <p className="text-[13px] text-muted-foreground">Carregando...</p>
      </div>
    );
  }

  return (
    <div className="space-y-5 px-4">
      <div>
        <h3 className="text-[15px] font-semibold">Conta Telegram</h3>
        <p className="text-xs text-muted-foreground mt-0.5">
          Conecte sua conta pessoal para recursos exclusivos
        </p>
      </div>

      {/* Só mostra o card de status se estiver em idle (não iniciou conexão) */}
      {flowStep === 'idle' && (
        <ConnectedAccountCard
          status={accountStatus}
          onReconnect={handleConnect}
          onDisconnect={handleDisconnect}
        />
      )}

      {flowStep === 'disclaimer' && (
        <AuthDisclaimer
          onAccept={handleAcceptTerms}
          onCancel={handleCancel}
        />
      )}

      {flowStep === 'auth' && (
        <AuthFlow
          initialStep={authStatus.step === 'password' ? 'password' : authStatus.step === 'code' ? 'code' : 'phone'}
          error={authStatus.error}
          onSendPhone={handleSendPhone}
          onVerifyCode={handleVerifyCode}
          onSendPassword={handleSendPassword}
          onCancel={handleCancel}
        />
      )}

      {flowStep === 'done' && accountStatus.status === 'connected' && (
        <div className="flex flex-col items-center py-6 gap-3 text-center">
          <div className="flex items-center gap-2 text-sm font-semibold text-accent">
            <CheckCircle2 size={20} />
            <span>Conta conectada com sucesso!</span>
          </div>
          <p className="text-xs text-muted-foreground">
            Sua conta Telegram agora será usada automaticamente nos canais autorizados.
          </p>
          <Button
            variant="default"
            className="w-full mt-2"
            onClick={() => setFlowStep('idle')}
          >
            Concluído
          </Button>
        </div>
      )}
    </div>
  );
}
