import { memo, useMemo } from 'react';
import { ArrowRight, ChartNoAxesColumnIncreasing, ChevronRight, Clock3, Info, Radio, Send, Settings, ShieldCheck, TrendingUp, UserRoundCheck, Users } from 'lucide-react';
import { Channel, User } from '../../types';
import { getOverviewMetrics, OperationalAlertKind } from './crmSelectors';

interface OperationsOverviewProps {
  users: User[];
  channels: Channel[];
  onOpenUser: (id: number) => void;
  onViewUsers: () => void;
  onReviewAlert: (kind: OperationalAlertKind) => void;
  onNavigate: (tab: 'notice' | 'users' | 'config') => void;
}

function compact(value: number) {
  return new Intl.NumberFormat('pt-BR', { notation: 'compact', maximumFractionDigits: 1 }).format(value);
}

function userPriority(user: User) {
  if (user.is_blacklisted) return 0;
  if (user.is_admin) return 1;
  return 2;
}

export const OperationsOverview = memo(function OperationsOverview({ users, channels, onOpenUser, onViewUsers, onReviewAlert, onNavigate }: OperationsOverviewProps) {
  const metrics = useMemo(() => getOverviewMetrics(users, channels), [channels, users]);
  const reviewUsers = useMemo(() => [...users]
    .filter((user) => user.is_blacklisted || user.is_admin || !(user.channels?.length))
    .sort((a, b) => userPriority(a) - userPriority(b)), [users]);
  const visibleReviewUsers = reviewUsers.slice(0, 3);
  const remainingReviews = Math.max(0, reviewUsers.length - visibleReviewUsers.length);
  const updatedAt = useMemo(() => new Intl.DateTimeFormat('pt-BR', { hour: '2-digit', minute: '2-digit' }).format(new Date()), []);

  const items = [
    { key: 'users', label: 'Usuários', value: compact(metrics.totalUsers), note: 'base cadastrada', icon: Users },
    { key: 'channels', label: 'Canais', value: compact(metrics.totalChannels), note: 'conectados', icon: Radio },
    { key: 'admins', label: 'Admins', value: compact(metrics.admins), note: 'com acesso', icon: ShieldCheck },
    { key: 'activation', label: 'Ativação', value: `${metrics.activationRate}%`, note: `${compact(metrics.activatedUsers)} com canais`, icon: ChartNoAxesColumnIncreasing },
  ];
  const attention = [
    { id: 'blacklisted' as const, count: metrics.blacklisted, title: 'Usuários bloqueados', description: 'Novos bloqueios abaixo confirmam revisões.', tone: 'danger' },
    { id: 'without-channels' as const, count: metrics.withoutChannels, title: 'Ativação pendente', description: 'Usuários ainda não possuem canal vinculado.', tone: 'warning' },
    { id: 'new-users' as const, count: metrics.recentUsers, title: 'Novos cadastros', description: 'Entraram na base nos últimos sete dias.', tone: 'info' },
  ];

  return (
    <section className="operations-overview" aria-label="Resumo operacional">
      <header className="operations-overview-heading">
        <span>Administração</span><h1>Visão geral</h1><p>Estado atual da base e acessos que merecem revisão.</p>
      </header>

      <nav className="operations-quick-actions" aria-label="Atalhos administrativos">
        <button type="button" onClick={() => onNavigate('notice')}><Send size={15} /><span>Novo broadcast</span></button>
        <button type="button" onClick={() => onNavigate('users')}><Users size={15} /><span>Usuários</span></button>
        <button type="button" onClick={() => onNavigate('config')}><Settings size={15} /><span>Configurações</span></button>
      </nav>

      <section className="operations-metrics" aria-label="Métricas globais">
        {items.map(({ key, label, value, note, icon: Icon }) => (
          <article className={`operations-metric metric-${key}`} key={key}><Icon size={15} /><strong>{value}</strong><span>{label}</span><small>{note}</small></article>
        ))}
      </section>

      <section className="operations-health">
        <header className="operations-section-heading"><div><span>Saúde da base</span><h2>Ativação e acompanhamento</h2></div><strong>{metrics.activationRate}%</strong></header>
        <div className="operations-progress" role="progressbar" aria-label="Taxa de ativação" aria-valuemin={0} aria-valuemax={100} aria-valuenow={metrics.activationRate}><span style={{ width: `${metrics.activationRate}%` }} /></div>
        <div className="operations-health-grid">
          <div className="health-active"><UserRoundCheck size={15} /><strong>{compact(metrics.activatedUsers)}</strong><span>usuários<br />com canal</span></div>
          <div className="health-pending"><TrendingUp size={15} /><strong>{compact(metrics.withoutChannels)}</strong><span>aguardando<br />ativação</span></div>
          <div className="health-recent"><Clock3 size={15} /><strong>{compact(metrics.recentUsers)}</strong><span>novos em<br />7 dias</span></div>
        </div>
      </section>

      <section className="operations-alerts" aria-labelledby="operations-alerts-title">
        <header><span id="operations-alerts-title">Pontos que pedem atenção</span></header>
        <div className="operations-alert-list">
          {attention.map((alert) => (
            <button type="button" key={alert.id} className={`operations-alert is-${alert.tone}`} onClick={() => onReviewAlert(alert.id)}>
              <i /><strong>{compact(alert.count)}</strong><span><b>{alert.title}</b><small>{alert.description}</small></span><ChevronRight size={15} />
            </button>
          ))}
        </div>
        <button type="button" className="operations-card-link" onClick={onViewUsers}>Ver todas notificações <ArrowRight size={14} /></button>
      </section>

      <section className="operations-queue" aria-labelledby="operations-queue-title">
        <header><div><span id="operations-queue-title">Fila de revisão</span><p>Usuários com acesso administrativo, bloqueado ou sem canais vinculados.</p></div><b>{reviewUsers.length} pendente{reviewUsers.length === 1 ? '' : 's'}</b></header>
        {visibleReviewUsers.length ? (
          <div className="operations-queue-list">
            {visibleReviewUsers.map((user) => (
              <button type="button" key={user.id} onClick={() => onOpenUser(user.id)} className="operations-queue-item">
                <span className="operations-avatar">{Array.from((user.first_name || user.username || '?').trim())[0]?.toLocaleUpperCase('pt-BR') || '?'}</span>
                <span className="operations-user"><strong>{user.first_name || user.username || 'Sem nome'}</strong><small>ID {user.id} · {user.channels?.length || 0} {(user.channels?.length || 0) === 1 ? 'canal' : 'canais'}</small></span><ChevronRight size={15} />
              </button>
            ))}
          </div>
        ) : <div className="operations-queue-empty"><strong>✓ Fila em dia</strong><span>Nenhum usuário precisa de revisão.</span></div>}
        {reviewUsers.length > 0 && <button type="button" className="operations-card-link" onClick={onViewUsers}>{remainingReviews > 0 ? `Ver mais ${remainingReviews}` : 'Ver fila completa'} <ArrowRight size={14} /></button>}
      </section>

      <aside className="operations-updated"><Info size={16} /><div><strong>Última atualização: {updatedAt}</strong><span>Dados carregados nesta sessão administrativa.</span></div></aside>
    </section>
  );
});
