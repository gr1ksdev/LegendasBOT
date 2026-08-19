import { useState, useEffect } from 'react';
import {
  fetchAdminAccounts,
  adminConnectAccount,
  adminVerifyCode,
  adminSendPassword,
  adminDeleteAccount,
  adminToggleAccount,
} from '../api';
import { AdminMTProtoAccount, AdminAuthStep } from '../types';
import { useToast } from './Toast';
import { Button } from './ui/button';
import { Switch } from './ui/switch';
import { Input } from './ui/input';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from './ui/card';
import { UserCheck, UserX, Trash2, Phone, Key, Lock, CheckCircle, Loader2, Plus, ChevronRight, Clock, Database } from 'lucide-react';

type AuthPhase = 'idle' | 'phone' | 'code' | 'password' | 'done';

export function AdminMTProtoAccountsTab() {
  const [accounts, setAccounts] = useState<AdminMTProtoAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [authPhase, setAuthPhase] = useState<AuthPhase>('idle');
  const [authSessionId, setAuthSessionId] = useState<string>('');
  const [authLabel, setAuthLabel] = useState('');
  const [authPhone, setAuthPhone] = useState('');
  const [authCode, setAuthCode] = useState('');
  const [authPassword, setAuthPassword] = useState('');
  const [authLoading, setAuthLoading] = useState(false);
  const [authError, setAuthError] = useState('');
  const toast = useToast();

  const loadAccounts = async () => {
    try {
      const data = await fetchAdminAccounts();
      setAccounts(Array.isArray(data) ? data : []);
    } catch {
      toast('Erro ao carregar contas MTProto', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAccounts();
  }, []);

  const handleStartConnect = () => {
    setAuthPhase('phone');
    setAuthLabel('');
    setAuthPhone('');
    setAuthCode('');
    setAuthPassword('');
    setAuthError('');
  };

  const handleCancelAuth = () => {
    setAuthPhase('idle');
    setAuthSessionId('');
  };

  const handleSendCode = async () => {
    if (!authLabel.trim() || !authPhone.trim()) {
      setAuthError('Preencha o label e o número de telefone');
      return;
    }

    setAuthLoading(true);
    setAuthError('');
    try {
      const result: AdminAuthStep = await adminConnectAccount(authLabel.trim(), authPhone.trim());
      if (result.step === 'error') {
        setAuthError(result.error || 'Erro ao enviar código');
      } else if (result.step === 'code' && result.sessionId) {
        setAuthSessionId(result.sessionId);
        setAuthPhase('code');
      }
    } catch (err: any) {
      setAuthError(err.message || 'Erro ao conectar');
    } finally {
      setAuthLoading(false);
    }
  };

  const handleVerifyCode = async () => {
    if (!authCode.trim()) {
      setAuthError('Informe o código recebido');
      return;
    }

    setAuthLoading(true);
    setAuthError('');
    try {
      const result: AdminAuthStep = await adminVerifyCode(authSessionId, authCode.trim());
      if (result.step === 'error') {
        setAuthError(result.error || 'Código inválido');
      } else if (result.step === 'password') {
        setAuthPhase('password');
      } else if (result.step === 'done') {
        setAuthPhase('done');
        toast('Conta conectada com sucesso!', 'success');
        loadAccounts();
        setTimeout(() => setAuthPhase('idle'), 1500);
      }
    } catch (err: any) {
      setAuthError(err.message || 'Erro ao verificar código');
    } finally {
      setAuthLoading(false);
    }
  };

  const handleSendPassword = async () => {
    if (!authPassword.trim()) {
      setAuthError('Informe a senha 2FA');
      return;
    }

    setAuthLoading(true);
    setAuthError('');
    try {
      const result: AdminAuthStep = await adminSendPassword(authSessionId, authPassword.trim());
      if (result.step === 'error') {
        setAuthError(result.error || 'Senha inválida');
      } else if (result.step === 'done') {
        setAuthPhase('done');
        toast('Conta conectada com sucesso!', 'success');
        loadAccounts();
        setTimeout(() => setAuthPhase('idle'), 1500);
      }
    } catch (err: any) {
      setAuthError(err.message || 'Erro ao verificar senha');
    } finally {
      setAuthLoading(false);
    }
  };

  const handleToggle = async (account: AdminMTProtoAccount) => {
    try {
      const updated = await adminToggleAccount(account.id, !account.enabled);
      if (updated) {
        setAccounts(prev => prev.map(a => a.id === account.id ? { ...a, enabled: updated.enabled } : a));
        toast(updated.enabled ? 'Conta ativada' : 'Conta desativada', 'success');
      }
    } catch {
      toast('Erro ao atualizar conta', 'error');
    }
  };

  const handleDelete = async (account: AdminMTProtoAccount) => {
    const tg = window.Telegram?.WebApp;
    if (tg) {
      tg.showConfirm(`Remover conta "${account.label || account.username}"?`, async (ok) => {
        if (!ok) return;
        await doDelete(account);
      });
    } else {
      if (!confirm(`Remover conta "${account.label || account.username}"?`)) return;
      await doDelete(account);
    }
  };

  const doDelete = async (account: AdminMTProtoAccount) => {
    try {
      await adminDeleteAccount(account.id);
      setAccounts(prev => prev.filter(a => a.id !== account.id));
      toast('Conta removida', 'success');
    } catch {
      toast('Erro ao remover conta', 'error');
    }
  };

  const formatDate = (d: string | null) => {
    if (!d) return '—';
    try {
      return new Date(d).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' });
    } catch {
      return d;
    }
  };

  // ── Loading ──
  if (loading) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center py-12 gap-3">
          <div className="auth-spinner" />
          <p className="text-sm text-muted-foreground">Carregando contas MTProto...</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="admin-accounts-page grid gap-5">

      <Card>
        <CardHeader>
          <CardTitle>Contas MTProto</CardTitle>
          <CardDescription>Gerencie contas Telegram para edição de postagens</CardDescription>
        </CardHeader>
        <CardContent>

        {/* Auth flow modal inline */}
        {authPhase !== 'idle' && authPhase !== 'done' && (
          <div className="cfg-auth-flow">
            {/* Steps indicator */}
            <div className="flex items-center gap-2 mb-4">
              {['phone', 'code', 'password'].map((step, i) => {
                const isActive = authPhase === step;
                const isPast = ['phone', 'code', 'password'].indexOf(authPhase) > i;
                return (
                  <div key={step} className="flex items-center gap-2">
                    <div className={`cfg-step-dot ${isActive ? 'active' : ''} ${isPast ? 'past' : ''}`}>
                      {isPast ? <CheckCircle size={14} /> : i + 1}
                    </div>
                    <span className={`text-[11px] font-medium ${isActive ? 'text-foreground' : 'text-muted-foreground'}`}>
                      {step === 'phone' ? 'Telefone' : step === 'code' ? 'Código' : 'Senha 2FA'}
                    </span>
                    {i < 2 && <ChevronRight size={12} className="text-muted-foreground/30" />}
                  </div>
                );
              })}
            </div>

            {/* Phone step */}
            {authPhase === 'phone' && (
              <div className="space-y-3">
                <div className="cfg-auth-field">
                  <label className="text-xs font-medium text-muted-foreground mb-1 block">Identificação (label)</label>
                  <Input
                    value={authLabel}
                    onChange={e => setAuthLabel(e.target.value)}
                    placeholder="Ex: Conta do João"
                    disabled={authLoading}
                    className="h-9"
                  />
                </div>
                <div className="cfg-auth-field">
                  <label className="text-xs font-medium text-muted-foreground mb-1 block">Número de Telefone</label>
                  <Input
                    value={authPhone}
                    onChange={e => setAuthPhone(e.target.value)}
                    placeholder="+5511999999999"
                    disabled={authLoading}
                    className="h-9"
                  />
                </div>
                {authError && <p className="text-[12px] text-destructive">{authError}</p>}
                <div className="flex gap-2">
                  <Button variant="secondary" size="sm" onClick={handleCancelAuth} disabled={authLoading}>
                    Cancelar
                  </Button>
                  <Button variant="default" size="sm" onClick={handleSendCode} disabled={authLoading} className="flex-1">
                    {authLoading ? <Loader2 size={14} className="animate-spin" /> : <Phone size={14} />}
                    {authLoading ? 'Enviando...' : 'Enviar Código'}
                  </Button>
                </div>
              </div>
            )}

            {/* Code step */}
            {authPhase === 'code' && (
              <div className="space-y-3">
                <div className="cfg-auth-field">
                  <label className="text-xs font-medium text-muted-foreground mb-1 block">Código de Verificação</label>
                  <Input
                    value={authCode}
                    onChange={e => setAuthCode(e.target.value)}
                    placeholder="12345"
                    disabled={authLoading}
                    className="h-9"
                  />
                  <p className="text-[11px] text-muted-foreground mt-1">
                    Digite o código enviado para o Telegram
                  </p>
                </div>
                {authError && <p className="text-[12px] text-destructive">{authError}</p>}
                <div className="flex gap-2">
                  <Button variant="secondary" size="sm" onClick={handleCancelAuth} disabled={authLoading}>
                    Cancelar
                  </Button>
                  <Button variant="default" size="sm" onClick={handleVerifyCode} disabled={authLoading} className="flex-1">
                    {authLoading ? <Loader2 size={14} className="animate-spin" /> : <Key size={14} />}
                    {authLoading ? 'Verificando...' : 'Verificar Código'}
                  </Button>
                </div>
              </div>
            )}

            {/* Password step */}
            {authPhase === 'password' && (
              <div className="space-y-3">
                <div className="cfg-auth-field">
                  <label className="text-xs font-medium text-muted-foreground mb-1 block">Senha 2FA</label>
                  <Input
                    type="password"
                    value={authPassword}
                    onChange={e => setAuthPassword(e.target.value)}
                    placeholder="Senha de duas etapas"
                    disabled={authLoading}
                    className="h-9"
                  />
                  <p className="text-[11px] text-muted-foreground mt-1">
                    Esta conta possui verificação em duas etapas
                  </p>
                </div>
                {authError && <p className="text-[12px] text-destructive">{authError}</p>}
                <div className="flex gap-2">
                  <Button variant="secondary" size="sm" onClick={handleCancelAuth} disabled={authLoading}>
                    Cancelar
                  </Button>
                  <Button variant="default" size="sm" onClick={handleSendPassword} disabled={authLoading} className="flex-1">
                    {authLoading ? <Loader2 size={14} className="animate-spin" /> : <Lock size={14} />}
                    {authLoading ? 'Verificando...' : 'Confirmar Senha'}
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Done state */}
        {authPhase === 'done' && (
          <div className="cfg-auth-flow">
            <div className="flex flex-col items-center py-4 gap-2">
              <div className="flex items-center justify-center size-10 rounded-full" style={{ background: 'var(--success-soft)' }}>
                <CheckCircle size={20} style={{ color: 'var(--success)' }} />
              </div>
              <p className="text-[13px] font-semibold text-success">Conta conectada com sucesso!</p>
            </div>
          </div>
        )}

        {/* Connect button */}
        {authPhase === 'idle' && (
          <div className="px-6 py-3">
            <Button variant="default" size="sm" onClick={handleStartConnect} className="w-full cfg-connect-btn">
              <Plus size={15} />
              Conectar Nova Conta
            </Button>
          </div>
        )}

        <div className="h-px bg-border mx-6" />

        {/* Accounts list */}
        <div className="px-6 py-3">
          <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
            Contas Conectadas ({accounts.length})
          </span>

          {accounts.length === 0 ? (
            <div className="flex flex-col items-center py-8 text-muted-foreground">
              <Database size={28} className="opacity-30 mb-2" />
              <p className="text-[13px] font-medium">Nenhuma conta conectada</p>
              <p className="text-[11px] mt-1">Conecte contas Telegram para editar postagens via MTProto</p>
            </div>
          ) : (
            <div className="space-y-2">
              {accounts.map(account => (
                <div key={account.id} className="cfg-account-card">
                  <div className="cfg-account-left">
                    {/* Avatar */}
                    <div
                      className="cfg-account-avatar"
                      style={{
                        background: account.enabled ? 'var(--accent-soft)' : 'var(--muted)',
                        color: account.enabled ? 'var(--accent)' : 'var(--text-muted)',
                      }}
                    >
                      {account.telegramUserId > 0 ? (
                        account.username ? (
                          <span className="text-sm font-bold">
                            {(account.firstName || account.username || '?')[0].toUpperCase()}
                          </span>
                        ) : (
                          <UserCheck size={18} />
                        )
                      ) : (
                        <UserX size={18} />
                      )}
                    </div>

                    {/* Info */}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="cfg-account-label">{account.label}</span>
                        <span className={`cfg-status-badge ${account.enabled ? 'enabled' : 'disabled'}`}>
                          {account.enabled ? 'Ativa' : 'Inativa'}
                        </span>
                        <span className={`cfg-status-badge ${account.status === 'connected' ? 'connected' : 'error'}`}>
                          {account.status === 'connected' ? 'Conectada' : account.status === 'error' ? 'Erro' : 'Desconectada'}
                        </span>
                      </div>
                      <div className="cfg-account-meta">
                        {account.username && <span>@{account.username}</span>}
                        {account.firstName && <span>{account.firstName}</span>}
                        <span>ID: {account.telegramUserId}</span>
                        {account.lastUsedAt && (
                          <span className="flex items-center gap-1">
                            <Clock size={10} />
                            {formatDate(account.lastUsedAt)}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="cfg-account-actions">
                    <Switch
                      checked={account.enabled}
                      onCheckedChange={() => handleToggle(account)}
                    />
                    <button
                      className="cfg-icon-btn danger"
                      title="Remover conta"
                      onClick={() => handleDelete(account)}
                    >
                      <Trash2 size={15} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

      <style>{`
        .cfg-auth-flow {
          padding: 16px;
          margin-bottom: 8px;
          border-radius: 10px;
          background: var(--muted);
        }
        .cfg-step-dot {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 22px;
          height: 22px;
          border-radius: 50%;
          font-size: 11px;
          font-weight: 700;
          background: var(--muted);
          color: var(--text-muted);
          transition: all 0.2s;
        }
        .cfg-step-dot.active {
          background: var(--accent);
          color: white;
        }
        .cfg-step-dot.past {
          background: var(--success-soft);
          color: var(--success);
        }
        .cfg-auth-field {
          margin-bottom: 8px;
        }
        .cfg-connect-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          gap: 6px;
          padding: 10px;
          border-radius: 10px;
        }
        .cfg-account-card {
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 12px;
          padding: 12px;
          border-radius: 10px;
          border: 1px solid var(--border);
          transition: border-color 0.15s;
        }
        .cfg-account-card:hover {
          border-color: var(--accent);
        }
        .cfg-account-left {
          display: flex;
          align-items: center;
          gap: 12px;
          min-width: 0;
          flex: 1;
        }
        .cfg-account-avatar {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 38px;
          height: 38px;
          border-radius: 10px;
          flex-shrink: 0;
        }
        .cfg-account-label {
          font-size: 13px;
          font-weight: 600;
          color: var(--text-primary);
        }
        .cfg-account-meta {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-wrap: wrap;
          font-size: 11px;
          color: var(--text-muted);
          margin-top: 2px;
        }
        .cfg-status-badge {
          font-size: 10px;
          font-weight: 600;
          padding: 2px 7px;
          border-radius: 6px;
          text-transform: uppercase;
          letter-spacing: 0.04em;
        }
        .cfg-status-badge.enabled {
          background: var(--success-soft);
          color: var(--success);
        }
        .cfg-status-badge.disabled {
          background: var(--muted);
          color: var(--text-muted);
        }
        .cfg-status-badge.connected {
          background: rgba(99, 102, 241, 0.08);
          color: var(--accent);
        }
        .cfg-status-badge.error {
          background: var(--danger-soft);
          color: var(--danger);
        }
        .cfg-account-actions {
          display: flex;
          align-items: center;
          gap: 8px;
          flex-shrink: 0;
        }
        .cfg-icon-btn {
          display: flex;
          align-items: center;
          justify-content: center;
          width: 32px;
          height: 32px;
          border: none;
          border-radius: 8px;
          background: transparent;
          color: var(--text-muted);
          cursor: pointer;
          transition: all 0.15s;
        }
        .cfg-icon-btn.danger:hover {
          background: var(--danger-soft);
          color: var(--danger);
        }
      `}</style>
        </CardContent>
      </Card>
    </div>
  );
}
