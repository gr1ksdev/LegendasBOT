import { useState } from 'react';
import { UserCheck, UserX, Clock, Calendar, RefreshCw, Trash2 } from 'lucide-react';
import { AccountStatus } from '../types';
import { Button } from './ui/button';
import { Badge } from './ui/badge';

interface ConnectedAccountCardProps {
  status: AccountStatus;
  onReconnect: () => void;
  onDisconnect: () => void;
}

/** Formata data ISO para padrao brasileiro: DD/MM/YYYY HH:mm */
function formatDateBR(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

export function ConnectedAccountCard({
  status,
  onReconnect,
  onDisconnect,
}: ConnectedAccountCardProps) {
  const [imgError, setImgError] = useState(false);
  const isConnected = status.status === 'connected';
  const hasPhoto = isConnected && status.avatarUrl && !imgError;
  const initials = status.firstName
    ? status.firstName[0].toUpperCase()
    : status.username
      ? status.username[0].toUpperCase()
      : status.telegramId
        ? `U${String(status.telegramId).slice(-1)}`
        : '?';

  return (
    <div>
      {/* Status Header */}
      <div className="flex items-center gap-4 mb-5">
        <div
          className="size-14 rounded-full overflow-hidden shrink-0 flex items-center justify-center"
          style={{ background: isConnected ? 'var(--accent-soft)' : 'var(--danger-soft)' }}
        >
          {hasPhoto ? (
            <img
              src={status.avatarUrl}
              alt=""
              className="w-full h-full object-cover"
              onError={() => setImgError(true)}
            />
          ) : isConnected ? (
            <span className="font-bold text-[20px]" style={{ color: 'var(--accent)' }}>
              {initials}
            </span>
          ) : (
            <UserX size={28} style={{ color: 'var(--danger)' }} />
          )}
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-[16px] font-bold truncate">
            {isConnected && status.firstName ? status.firstName : 'Conta Telegram'}
          </h3>
          <div className="flex items-center gap-2 mt-1">
            <Badge variant={isConnected ? "default" : "secondary"} className="text-[11px]">
              {isConnected ? 'Conectada' : 'Desconectada'}
            </Badge>
          </div>
        </div>
      </div>

      {/* Connected Details */}
      {isConnected && status.telegramId && (
        <div className="flex flex-col gap-3 mb-5">
          <div className="flex items-center gap-3 text-[13px]">
            <div className="size-8 rounded-full flex items-center justify-center shrink-0" style={{ background: 'var(--accent-soft)' }}>
              <UserCheck size={16} style={{ color: 'var(--accent)' }} />
            </div>
            <div className="min-w-0 flex-1">
              <span className="text-muted-foreground text-[11px]">Username</span>
              <p className="font-medium truncate" style={status.username ? {} : { color: 'var(--hint)', fontStyle: 'italic' }}>
                {status.username || 'Sem username'}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3 text-[13px]">
            <div className="size-8 rounded-full flex items-center justify-center shrink-0" style={{ background: 'var(--accent-soft)' }}>
              <Calendar size={16} style={{ color: 'var(--accent)' }} />
            </div>
            <div className="min-w-0 flex-1">
              <span className="text-muted-foreground text-[11px]">Conectada em</span>
              <p className="font-medium truncate">{status.connectedAt ? formatDateBR(status.connectedAt) : '-'}</p>
            </div>
          </div>

          {status.lastUsedAt && (
            <div className="flex items-center gap-3 text-[13px]">
              <div className="size-8 rounded-full flex items-center justify-center shrink-0" style={{ background: 'var(--accent-soft)' }}>
                <Clock size={16} style={{ color: 'var(--accent)' }} />
              </div>
              <div className="min-w-0 flex-1">
                <span className="text-muted-foreground text-[11px]">Último uso</span>
                <p className="font-medium truncate">{formatDateBR(status.lastUsedAt)}</p>
              </div>
            </div>
          )}

          <div className="flex items-center gap-3 text-[13px]">
            <div className="size-8 rounded-full flex items-center justify-center shrink-0" style={{ background: 'var(--accent-soft)' }}>
              <Clock size={16} style={{ color: 'var(--accent)' }} />
            </div>
            <div className="min-w-0 flex-1">
              <span className="text-muted-foreground text-[11px]">Telegram ID</span>
              <p className="font-medium truncate">{status.telegramId}</p>
            </div>
          </div>
        </div>
      )}

      {/* Disconnected Info */}
      {!isConnected && (
        <div className="mb-5">
          <p className="text-[13px] text-muted-foreground leading-relaxed">
            Conecte sua conta Telegram para utilizar recursos exclusivos como emojis Premium,
            reações Premium e outras funcionalidades nos seus canais.
          </p>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-2">
        {isConnected ? (
          <>
            <Button variant="ghost" className="flex-1" onClick={onReconnect}>
              <RefreshCw size={16} />
              Reconectar
            </Button>
            <Button variant="destructive" className="flex-1" onClick={onDisconnect}>
              <Trash2 size={16} />
              Desconectar
            </Button>
          </>
        ) : (
          <Button variant="default" className="w-full" onClick={onReconnect}>
            <RefreshCw size={16} />
            Conectar Conta Telegram
          </Button>
        )}
      </div>
    </div>
  );
}
