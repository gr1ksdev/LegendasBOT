import { useState } from 'react';
import { Shield, Check } from 'lucide-react';
import { Button } from './ui/button';

interface AuthDisclaimerProps {
  onAccept: () => void;
  onCancel: () => void;
}

export function AuthDisclaimer({ onAccept, onCancel }: AuthDisclaimerProps) {
  const [agreeTerms, setAgreeTerms] = useState(false);
  const [agreePrivacy, setAgreePrivacy] = useState(false);

  const isAgreed = agreeTerms && agreePrivacy;

  return (
    <div>
      <div className="text-center mb-6">
        <div
          className="flex size-16 items-center justify-center rounded-[20px] mx-auto mb-4"
          style={{ background: 'var(--accent-soft)' }}
        >
          <Shield size={32} style={{ color: 'var(--accent)' }} />
        </div>
        <h3 className="text-[18px] font-bold mb-2">
          Sua Conta Telegram
        </h3>
        <p className="text-[13px] text-muted-foreground leading-relaxed">
          Sua conta Telegram será utilizada exclusivamente nos canais autorizados por você.
        </p>
      </div>

      <div
        className="p-4 mb-5 rounded-xl"
        style={{ background: 'var(--danger-soft)' }}
      >
        <p className="text-[12px] font-medium" style={{ color: 'var(--danger)', lineHeight: 1.6 }}>
          <strong>Não armazenamos:</strong>
        </p>
        <ul className="text-[12px]" style={{ color: 'var(--danger)', lineHeight: 1.8, opacity: 0.8, marginTop: 4 }}>
          <li>• Código de autenticação</li>
          <li>• Senha 2FA</li>
        </ul>
        <p className="text-[12px] text-muted-foreground leading-relaxed mt-2">
          Apenas uma sessão autenticada criptografada será armazenada para funcionamento do serviço.
        </p>
      </div>

      <p className="mb-4 text-[13px] text-muted-foreground leading-relaxed">
        Você poderá desconectar sua conta a qualquer momento.
      </p>

      <label
        className={`flex items-center justify-between px-[18px] py-3 rounded-[20px] gap-3 min-h-[52px] cursor-pointer transition-all mb-2 ${agreeTerms ? 'bg-accent/10' : ''}`}
        onClick={() => setAgreeTerms(!agreeTerms)}
      >
        <div className="flex items-center gap-3">
          <div
            className="flex size-[22px] items-center justify-center rounded-[6px] border-2 transition-all"
            style={{
              background: agreeTerms ? 'var(--accent)' : 'transparent',
              borderColor: agreeTerms ? 'var(--accent)' : 'var(--border)',
            }}
          >
            {agreeTerms && <Check size={14} className="text-white" />}
          </div>
          <span className="text-[13px]">
            Li e concordo com os{' '}
            <strong className="text-accent">Termos de Uso</strong>
          </span>
        </div>
      </label>

      <label
        className={`flex items-center justify-between px-[18px] py-3 rounded-[20px] gap-3 min-h-[52px] cursor-pointer transition-all ${agreePrivacy ? 'bg-accent/10' : ''}`}
        onClick={() => setAgreePrivacy(!agreePrivacy)}
      >
        <div className="flex items-center gap-3">
          <div
            className="flex size-[22px] items-center justify-center rounded-[6px] border-2 transition-all"
            style={{
              background: agreePrivacy ? 'var(--accent)' : 'transparent',
              borderColor: agreePrivacy ? 'var(--accent)' : 'var(--border)',
            }}
          >
            {agreePrivacy && <Check size={14} className="text-white" />}
          </div>
          <span className="text-[13px]">
            Li e concordo com a{' '}
            <strong className="text-accent">Política de Privacidade</strong>
          </span>
        </div>
      </label>

      <div className="flex gap-2 mt-6">
        <Button variant="ghost" className="flex-1" onClick={onCancel}>
          Cancelar
        </Button>
        <Button
          variant="default"
          className="flex-1"
          disabled={!isAgreed}
          onClick={onAccept}
        >
          Continuar
        </Button>
      </div>
    </div>
  );
}
