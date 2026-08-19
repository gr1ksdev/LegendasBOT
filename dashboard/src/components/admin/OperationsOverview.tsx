import { memo, useMemo } from 'react';
import { ArrowRight, BellRing, ChartNoAxesColumnIncreasing, Radio, ShieldCheck, Users } from 'lucide-react';
import { Channel, User } from '../../types';
import { getOperationalAlerts, getOverviewMetrics, OperationalAlertKind } from './crmSelectors';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

interface OperationsOverviewProps {
  users: User[];
  channels: Channel[];
  onOpenUser: (id: number) => void;
  onViewUsers: () => void;
  onReviewAlert: (kind: OperationalAlertKind) => void;
}

function compact(value: number) {
  return new Intl.NumberFormat('pt-BR', { notation: 'compact', maximumFractionDigits: 1 }).format(value);
}

export const OperationsOverview = memo(function OperationsOverview({ users, channels, onOpenUser, onViewUsers, onReviewAlert }: OperationsOverviewProps) {
  const metrics = useMemo(() => getOverviewMetrics(users, channels), [channels, users]);
  const alerts = useMemo(() => getOperationalAlerts(users, channels), [channels, users]);
  const reviewUsers = useMemo(
    () => [...users].filter((user) => user.is_blacklisted || user.is_admin || !(user.channels?.length)).slice(0, 6),
    [users],
  );

  const items = [
    { label: 'Usuários', value: compact(metrics.totalUsers), note: 'base cadastrada', icon: Users },
    { label: 'Canais', value: compact(metrics.totalChannels), note: 'conectados', icon: Radio },
    { label: 'Administradores', value: compact(metrics.admins), note: 'com acesso', icon: ShieldCheck },
    { label: 'Ativação', value: `${metrics.activationRate}%`, note: `${compact(metrics.activatedUsers)} com canais`, icon: ChartNoAxesColumnIncreasing },
  ];

  return (
    <section className="operations-overview" aria-label="Resumo operacional">
      <header className="operations-overview-heading mb-4">
        <div>
          <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Administração</span>
          <div className="operations-title text-2xl font-extrabold text-foreground text-slate-100 tracking-tight mt-0.5">Visão geral</div>
          <p className="text-xs text-muted-foreground mt-1">Estado atual da base e acessos que merecem revisão.</p>
        </div>
        <Button variant="ghost" type="button" className="operations-overview-link" onClick={onViewUsers}>
          Ver usuários <ArrowRight size={15} aria-hidden="true" />
        </Button>
      </header>

      <div className="operations-metrics">
        {items.map(({ label, value, note, icon: Icon }) => (
          <article className="operations-metric" key={label}>
            <Icon size={16} aria-hidden="true" />
            <span>{label}</span>
            <strong>{value}</strong>
            <small>{note}</small>
          </article>
        ))}
      </div>

      <div className="operations-insights gap-6 mt-6" aria-label="Indicadores operacionais">
        <section className="operations-health">
          <div className="operations-section-heading">
            <div>
              <span>Saúde da base</span>
              <h2>Ativação e acompanhamento</h2>
            </div>
            <strong>{metrics.activationRate}%</strong>
          </div>
          <div className="operations-progress" role="progressbar" aria-label="Taxa de ativação" aria-valuemin={0} aria-valuemax={100} aria-valuenow={metrics.activationRate}>
            <span style={{ width: `${metrics.activationRate}%` }} />
          </div>
          <div className="operations-health-grid">
            <div><strong>{compact(metrics.activatedUsers)}</strong><span>usuários com canal</span></div>
            <div><strong>{compact(metrics.withoutChannels)}</strong><span>aguardando ativação</span></div>
            <div><strong>{compact(metrics.recentUsers)}</strong><span>novos em 7 dias</span></div>
          </div>
        </section>

        <section className="operations-alerts" aria-labelledby="operations-alerts-title">
          <div className="operations-section-heading mb-3">
            <div>
              <span>Notificações</span>
              <h2 id="operations-alerts-title" className="text-base font-bold text-foreground">Pontos que pedem atenção</h2>
            </div>
            <BellRing size={17} aria-hidden="true" className="text-amber-400" />
          </div>
          {alerts.length ? (
            <div className="operations-alert-list space-y-2.5">
              {alerts.map((alert) => (
                <button
                  type="button"
                  key={alert.id}
                  className={`operations-alert is-${alert.severity}`}
                  onClick={() => onReviewAlert(alert.id)}
                >
                  <strong>{alert.count}</strong>
                  <span>
                    <b>{alert.title}</b>
                    <small>{alert.description}</small>
                  </span>
                  <ArrowRight size={15} aria-hidden="true" className="text-muted-foreground shrink-0" />
                </button>
              ))}
            </div>
          ) : (
            <p className="operations-alert-empty">Nenhuma notificação operacional no momento.</p>
          )}
        </section>
      </div>

      <section className="operations-queue" aria-labelledby="operations-queue-title">
        <header>
          <div>
            <div id="operations-queue-title" className="operations-queue-title">Fila de revisão</div>
            <p>Usuários com acesso administrativo, bloqueio ou sem canais vinculados.</p>
          </div>
          <span>{reviewUsers.length} exibidos</span>
        </header>
        {reviewUsers.length ? (
          <div className="operations-queue-list">
            {reviewUsers.map((user) => (
              <button type="button" key={user.id} onClick={() => onOpenUser(user.id)} className="operations-queue-item">
                <span className="operations-avatar">{(user.first_name || '?')[0].toUpperCase()}</span>
                <span className="operations-user">
                  <strong>{user.first_name || 'Sem nome'}</strong>
                  <small>ID {user.id} · {user.channels?.length || 0} canais</small>
                </span>
                <Badge variant={user.is_blacklisted ? 'destructive' : user.is_admin ? 'default' : 'outline'} className="justify-self-end">
                  {user.is_blacklisted ? 'Bloqueado' : user.is_admin ? 'Admin' : 'Sem canais'}
                </Badge>
                <ArrowRight size={15} aria-hidden="true" className="justify-self-end text-muted-foreground" />
              </button>
            ))}
          </div>
        ) : (
          <p className="operations-empty">Nenhuma revisão pendente na base atual.</p>
        )}
      </section>
    </section>
  );
});
