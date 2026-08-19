import { useState } from 'react';
import { Phone, Key, ShieldAlert, ArrowLeft, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

interface VerifyCodeResult {
  step: string;
  error?: string;
}

interface AuthFlowProps {
  initialStep?: 'phone' | 'code' | 'password';
  error?: string;
  onSendPhone: (phone: string) => Promise<void>;
  onVerifyCode: (code: string) => Promise<VerifyCodeResult>;
  onSendPassword: (password: string) => Promise<void>;
  onCancel: () => void;
}

/** Formata numero de telefone visualmente: 99 99999-9999 */
function formatPhone(value: string): string {
  // Remove tudo que nao for digito ou + inicial
  const cleaned = value.replace(/[^\d+]/g, '');
  const hasPlus = cleaned.startsWith('+');
  const digits = cleaned.replace(/\D/g, '').slice(0, 13); // Max 13 (ex: +55 11 91234-5678)

  if (digits.length <= 2) {
    return hasPlus ? `+${digits}` : digits;
  }
  if (digits.length <= 4) {
    return hasPlus
      ? `+${digits.slice(0, 2)} ${digits.slice(2)}`
      : `${digits.slice(0, 2)} ${digits.slice(2)}`;
  }

  // A partir de 5 digitos: separa DDI + DDD + numero
  const prefix = hasPlus ? '+' : '';
  const country = digits.slice(0, 2);
  const area = digits.slice(2, 4);
  const number = digits.slice(4);

  if (number.length <= 4) {
    return `${prefix}${country} ${area} ${number}`;
  }

  // Numero com 5+ digitos: separa com hifen apos o quinto digito
  const first = number.slice(0, 5);
  const rest = number.slice(5, 9);
  return `${prefix}${country} ${area} ${first}${rest ? '-' + rest : ''}`;
}

export function AuthFlow({
  initialStep = 'phone',
  error: serverError,
  onSendPhone,
  onVerifyCode,
  onSendPassword,
  onCancel,
}: AuthFlowProps) {
  const [step, setStep] = useState(initialStep);
  const [phone, setPhone] = useState('');
  const [code, setCode] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(serverError || null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const handleSendPhone = async () => {
    // Remove espacos e tracos para validar so os digitos
    const rawPhone = phone.replace(/[\s-]/g, '');
    if (rawPhone.length < 8) {
      setError('Número de telefone inválido');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await onSendPhone(rawPhone);
      setStep('code');
    } catch (err: any) {
      setError(err.message || 'Erro ao enviar código');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyCode = async () => {
    if (code.length < 3) {
      setError('Código inválido');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await onVerifyCode(code);
      // Usa o resultado real da API, nao a prop (que pode estar defasada)
      if (result.step === 'password') {
        setStep('password');
      } else if (result.step === 'done') {
        setSuccessMessage('Conta conectada com sucesso!');
      } else {
        // step === 'code' ou outro - continua no mesmo passo
      }
    } catch (err: any) {
      setError(err.message || 'Erro ao verificar código');
    } finally {
      setLoading(false);
    }
  };

  const handleSendPassword = async () => {
    if (!password) {
      setError('Senha é obrigatória');
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await onSendPassword(password);
      setSuccessMessage('Conta conectada com sucesso!');
    } catch (err: any) {
      setError(err.message || 'Erro ao verificar senha');
    } finally {
      setLoading(false);
    }
  };

  if (successMessage) {
    return (
      <div className="text-center py-8 flex flex-col items-center gap-4">
        <div className="flex size-[72px] items-center justify-center rounded-full bg-accent/10">
          <ShieldAlert size={36} className="text-accent" />
        </div>
        <h3 className="text-lg font-bold">{successMessage}</h3>
        <p className="text-sm text-muted-foreground">
          Sua conta Telegram está conectada e pronta para uso.
        </p>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        {step !== 'phone' && (
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setStep(step === 'code' ? 'phone' : 'code')}
            title="Voltar"
          >
            <ArrowLeft size={20} />
          </Button>
        )}
        <div>
          <h3 className="text-base font-bold">
            {step === 'phone' && 'Conectar Conta Telegram'}
            {step === 'code' && 'Verificar Código'}
            {step === 'password' && 'Senha 2FA'}
          </h3>
          <p className="text-xs text-muted-foreground">
            {step === 'phone' && 'Digite seu número de telefone'}
            {step === 'code' && 'Digite o código recebido no Telegram'}
            {step === 'password' && 'Sua conta possui verificação em duas etapas'}
          </p>
        </div>
      </div>

      {/* Error message */}
      {error && (
        <div className="mb-4 rounded-lg bg-destructive/10 px-4 py-3 text-sm font-medium text-destructive">
          {error}
        </div>
      )}

      {/* Step: Phone */}
      {step === 'phone' && (
        <div className="flex flex-col gap-4">
          <div className="relative">
            <Phone size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="tel"
              placeholder="+55 11 99999-9999"
              value={phone}
              onChange={(e) => setPhone(formatPhone(e.target.value))}
              onKeyDown={(e) => e.key === 'Enter' && handleSendPhone()}
              autoFocus
              disabled={loading}
              className="pl-10"
            />
          </div>
          <p className="text-xs text-muted-foreground leading-relaxed">
            Inclua o código do país (ex: +55 para Brasil).
            Um código de verificação será enviado para este número no Telegram.
          </p>
          <Button
            onClick={handleSendPhone}
            disabled={loading || phone.replace(/[\s-]/g, '').length < 8}
            className="w-full"
          >
            {loading ? <Loader2 size={18} className="animate-spin" /> : <Phone size={18} />}
            {loading ? 'Enviando...' : 'Enviar Código'}
          </Button>
        </div>
      )}

      {/* Step: Code */}
      {step === 'code' && (
        <div className="flex flex-col gap-4">
          <div className="relative">
            <Key size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="text"
              inputMode="numeric"
              placeholder="00000"
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
              onKeyDown={(e) => e.key === 'Enter' && handleVerifyCode()}
              autoFocus
              disabled={loading}
              className="pl-10 tracking-widest"
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Digite o código de 5 ou 6 dígitos enviado para seu Telegram.
          </p>
          <Button
            onClick={handleVerifyCode}
            disabled={loading || code.length < 3}
            className="w-full"
          >
            {loading ? <Loader2 size={18} className="animate-spin" /> : <Key size={18} />}
            {loading ? 'Verificando...' : 'Verificar Código'}
          </Button>
        </div>
      )}

      {/* Step: Password */}
      {step === 'password' && (
        <div className="flex flex-col gap-4">
          <div className="relative">
            <ShieldAlert size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="password"
              placeholder="Sua senha 2FA"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSendPassword()}
              autoFocus
              disabled={loading}
              className="pl-10"
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Sua senha 2FA não será armazenada. Ela é usada apenas neste momento para autenticar.
          </p>
          <Button
            onClick={handleSendPassword}
            disabled={loading || !password}
            className="w-full"
          >
            {loading ? <Loader2 size={18} className="animate-spin" /> : <ShieldAlert size={18} />}
            {loading ? 'Autenticando...' : 'Entrar'}
          </Button>
        </div>
      )}

      {/* Cancel */}
      <Button
        variant="ghost"
        onClick={onCancel}
        className="w-full mt-3"
      >
        Cancelar
      </Button>
    </div>
  );
}
