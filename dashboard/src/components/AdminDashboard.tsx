import { useState, useMemo, useTransition, useEffect, Dispatch, SetStateAction, Suspense, lazy } from 'react';
import { AdminDashboardData, User, Channel, AuditResult } from '../types';
import { NoticeButton, NoticeTarget, updateUserAdmin, updateUserBlacklist } from '../api';

const AdminNoticeTab = lazy(() => import('./AdminNoticeTab').then(m => ({ default: m.AdminNoticeTab })));
const AdminConfigTab = lazy(() => import('./AdminConfigTab').then(m => ({ default: m.AdminConfigTab })));
const AdminAuditTab = lazy(() => import('./AdminAuditTab').then(m => ({ default: m.AdminAuditTab })));
const AdminLogsTab = lazy(() => import('./AdminLogsTab').then(m => ({ default: m.AdminLogsTab })));
const AdminMTProtoAccountsTab = lazy(() => import('./AdminMTProtoAccountsTab').then(m => ({ default: m.AdminMTProtoAccountsTab })));
const AdminPremiumFeaturesTab = lazy(() => import('./AdminPremiumFeaturesTab').then(m => ({ default: m.AdminPremiumFeaturesTab })));
const AdminSubscriptionsTab = lazy(() => import('./AdminSubscriptionsTab').then(m => ({ default: m.AdminSubscriptionsTab })));
import { Hash, ArrowLeft, ChevronRight, User as UserIcon, ShieldCheck, UserX, UserCheck, MessageSquare, Calendar, ExternalLink, Users, Radio, Filter, ArrowUpDown, Info, Clock, SortAsc, Tv, Hourglass } from 'lucide-react';
import { useToast } from './Toast';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from './ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { DataTable, Column } from './admin/DataTable';
import { StatusBadge } from './admin/StatusBadge';
import { OperationsOverview } from './admin/OperationsOverview';
import { AdminPageHeader } from './admin/AdminPageHeader';
import { useAdminCrmControls } from './admin/AdminCrmContext';
import { filterAndSortChannels, filterAndSortUsers } from './admin/crmSelectors';

const USER_FILTER_CONFIG: Record<string, { label: string; icon: any }> = {
  all: { label: 'Todos os Usuários', icon: Users },
  recent: { label: 'Mais Recentes', icon: Clock },
  name: { label: 'Nome (A-Z)', icon: SortAsc },
  channels: { label: 'Mais Canais', icon: Tv },
  admins: { label: 'Apenas Admins', icon: ShieldCheck },
  blacklisted: { label: 'Apenas Bloqueados', icon: UserX },
  'with-channels': { label: 'Com Canais', icon: Tv },
  'without-channels': { label: 'Sem Canais', icon: Hourglass },
};

interface AdminDashboardProps {
  adminData: AdminDashboardData;
  activeTab: 'overview' | 'users' | 'channels' | 'notice' | 'config' | 'audit' | 'logs' | 'accounts' | 'premium-features' | 'subscriptions';
  initialLogsChannelId?: string;
  navigateToChannel: (id: number) => void;
  selectedUserId: number | null;
  onSelectUser: (id: number | null) => void;
  onOpenUserDetail: (id: number) => void;
  onMessageUser: (id: number) => void;
  // Notice tab props
  noticeMessage: string;
  setNoticeMessage: Dispatch<SetStateAction<string>>;
  noticeImageUrl: string;
  setNoticeImageUrl: Dispatch<SetStateAction<string>>;
  noticeTarget: NoticeTarget;
  setNoticeTarget: Dispatch<SetStateAction<NoticeTarget>>;
  noticeTargetId: string;
  setNoticeTargetId: Dispatch<SetStateAction<string>>;
  noticeButtons: NoticeButton[];
  handleAddNoticeButton: () => void;
  updateNoticeButton: (index: number, field: keyof NoticeButton, value: string) => void;
  removeNoticeButton: (index: number) => void;
  handleSendNotice: () => void;
  isSendingNotice: boolean;
  auditResults: AuditResult[] | null;
  setAuditResults: Dispatch<SetStateAction<AuditResult[] | null>>;
  auditLoading: boolean;
  handleRunAudit: () => void;
  toast: (message: string, type: 'success' | 'error' | 'info') => void;
}

// ───── Main Component ─────

export function AdminDashboard({
  adminData,
  activeTab,
  navigateToChannel,
  selectedUserId,
  onSelectUser,
  onOpenUserDetail,
  onMessageUser,
  noticeMessage, setNoticeMessage,
  noticeImageUrl, setNoticeImageUrl,
  noticeTarget, setNoticeTarget,
  noticeTargetId, setNoticeTargetId,
  noticeButtons, handleAddNoticeButton,
  updateNoticeButton, removeNoticeButton,
  handleSendNotice,
  isSendingNotice,
  auditResults, setAuditResults, auditLoading, handleRunAudit,
  initialLogsChannelId
}: AdminDashboardProps) {
  const toast = useToast();
  const { searchQuery, sortBy, filterBy, setFilterBy, navigateToTab } = useAdminCrmControls();

  const [localActiveTab, setLocalActiveTab] = useState(activeTab);
  const [isPending, startTransition] = useTransition();
  const [expandedChannelId, setExpandedChannelId] = useState<number | null>(null);
  const [userFilterOption, setUserFilterOption] = useState<string>('all');

  const [localUsers, setLocalUsers] = useState<User[]>(adminData.users || []);

  useEffect(() => {
    setLocalUsers(adminData.users || []);
  }, [adminData.users]);

  useEffect(() => {
    startTransition(() => {
      setLocalActiveTab(activeTab);
    });
  }, [activeTab]);

  const usersList = localUsers;
  const channelsList = adminData.channels || [];
  const visibleUsers = useMemo(() => {
    let fBy: AdminCrmFilter = 'all';
    let sBy: AdminCrmSort = 'recent';

    if (userFilterOption === 'recent' || userFilterOption === 'name' || userFilterOption === 'channels') {
      sBy = userFilterOption as AdminCrmSort;
    } else {
      fBy = userFilterOption as AdminCrmFilter;
    }

    return filterAndSortUsers(usersList, searchQuery, fBy, sBy);
  }, [usersList, searchQuery, userFilterOption]);
  const visibleChannels = useMemo(
    () => filterAndSortChannels(channelsList, searchQuery, sortBy),
    [channelsList, searchQuery, sortBy],
  );

  const adminSelectedUser = useMemo(() =>
    selectedUserId ? usersList.find(u => u.id === selectedUserId) : null,
    [selectedUserId, usersList]);

  const setAdminSelectedUser = (user: User | null) => onSelectUser(user ? user.id : null);

  // ── User Actions ──

  const handleToggleAdmin = async (uid: number) => {
    try {
      const res = await updateUserAdmin(uid);
      if (res.success) {
        const isAdmin = res.data?.isAdmin;
        setLocalUsers(prev => prev.map(u => u.id === uid ? { ...u, is_admin: isAdmin } : u));
        toast(isAdmin ? "Usuário promovido a Admin" : "Privilégios de Admin removidos", "success");
      }
    } catch (err: any) {
      toast(err.message || "Erro ao atualizar status de admin", "error");
    }
  };

  const handleToggleBlacklist = async (uid: number) => {
    try {
      const res = await updateUserBlacklist(uid);
      if (res.success) {
        const isBlacklisted = res.data?.isBlacklisted;
        setLocalUsers(prev => prev.map(u => u.id === uid ? { ...u, is_blacklisted: isBlacklisted } : u));
        toast(isBlacklisted ? "Usuário adicionado à Blacklist" : "Usuário removido da Blacklist", isBlacklisted ? "error" : "success");
      }
    } catch (err: any) {
      toast(err.message || "Erro ao atualizar status de blacklist", "error");
    }
  };

  // ── Overview Tab ──

  const renderOverviewTab = () => (
    <div className="admin-overview-page">
      <OperationsOverview
        users={usersList}
        channels={channelsList}
        onOpenUser={(id) => {
          onOpenUserDetail(id);
          navigateToTab('users');
        }}
        onViewUsers={() => navigateToTab('users')}
        onReviewAlert={(kind) => {
          if (kind === 'blacklisted') setFilterBy('blacklisted');
          if (kind === 'without-channels') setFilterBy('without-channels');
          if (kind === 'new-users') setFilterBy('all');
          navigateToTab('users');
        }}
      />
    </div>
  );

  // ── User Detail ──

  const renderUserDetail = () => {
    if (!adminSelectedUser) return null;
    const name = adminSelectedUser.first_name || 'Sem nome';
    return (
      <div className="space-y-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setAdminSelectedUser(null)}
          className="text-muted-foreground"
        >
          <ArrowLeft size={16} className="mr-1.5" /> Voltar para usuários
        </Button>

        <div className="rounded-xl border border-border p-4">
          <div className="flex items-center gap-3 mb-4">
            <div className="flex items-center justify-center size-12 rounded-xl shrink-0" style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}>
              <UserIcon size={24} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="text-base font-bold truncate">{name}</h2>
                {adminSelectedUser.is_admin && <Badge variant="default" className="text-[10px]">Admin</Badge>}
                {adminSelectedUser.is_blacklisted && <Badge variant="destructive" className="text-[10px]">Bloqueado</Badge>}
              </div>
              <p className="text-xs text-muted-foreground">ID: {adminSelectedUser.id} • {adminSelectedUser.channels?.length || 0} canais</p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
            <Button
              variant="default"
              size="sm"
              className="w-full"
              onClick={() => {
                onMessageUser(adminSelectedUser.id);
                navigateToTab('notice');
              }}
            >
              <MessageSquare size={16} />
              Mensagem de Suporte
            </Button>
            <Button
              variant={adminSelectedUser.is_admin ? "secondary" : "default"}
              size="sm"
              className="w-full"
              onClick={() => handleToggleAdmin(adminSelectedUser.id)}
            >
              <ShieldCheck size={16} />
              {adminSelectedUser.is_admin ? "Remover Admin" : "Tornar Admin"}
            </Button>
            <Button
              variant={adminSelectedUser.is_blacklisted ? "secondary" : "destructive"}
              size="sm"
              className="w-full"
              onClick={() => handleToggleBlacklist(adminSelectedUser.id)}
            >
              {adminSelectedUser.is_blacklisted ? <UserCheck size={16} /> : <UserX size={16} />}
              {adminSelectedUser.is_blacklisted ? "Remover Blacklist" : "Add Blacklist"}
            </Button>
          </div>
        </div>

        <div className="space-y-2">
          <h3 className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">Canais do Usuário</h3>
          {adminSelectedUser.channels && adminSelectedUser.channels.length > 0 ? (
            adminSelectedUser.channels.map((c: Channel) => (
              <button
                key={c.id}
                className="flex items-center w-full text-left gap-3 rounded-xl border border-border p-3 hover:bg-muted/30 transition-colors"
                onClick={() => navigateToChannel(c.id)}
              >
                <div className="flex items-center justify-center size-9 rounded-lg shrink-0" style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}>
                  <Hash size={16} />
                </div>
                <div className="min-w-0 flex-1">
                  <h3 className="text-[13px] font-semibold truncate">{c.title}</h3>
                  <p className="text-[11px] text-muted-foreground mt-0.5">ID: {c.id}</p>
                </div>
                <ChevronRight size={16} className="shrink-0 text-muted-foreground/30" />
              </button>
            ))
          ) : (
            <div className="flex flex-col items-center py-6 text-muted-foreground rounded-xl border border-border">
              <Hash size={28} className="opacity-30 mb-2" />
              <p className="text-[13px] font-medium">Este usuário não possui canais</p>
            </div>
          )}
        </div>
      </div>
    );
  };

  // ── Users Tab ──

  const renderUsersTab = () => {
    return (
      <div className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2 px-1">
          <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
            Total de Usuários ({visibleUsers.length})
          </span>
          <div className="flex items-center gap-2">
            {/* Seletor Único de Filtro & Ordenação */}
            <Select
              value={userFilterOption}
              onValueChange={(val) => setUserFilterOption(val)}
            >
              <SelectTrigger size="sm" className="min-w-[210px] h-9 text-xs font-bold rounded-xl bg-card text-foreground border-border cursor-pointer shadow-xs">
                <div className="flex items-center gap-2 truncate">
                  {(() => {
                    const cfg = USER_FILTER_CONFIG[userFilterOption] || USER_FILTER_CONFIG.all;
                    const IconComp = cfg.icon;
                    return (
                      <>
                        <IconComp size={14} className="text-accent shrink-0" />
                        <span className="text-foreground font-bold">{cfg.label}</span>
                      </>
                    );
                  })()}
                </div>
              </SelectTrigger>
              <SelectContent className="bg-[#12141a] text-slate-100 border border-slate-800 rounded-xl shadow-2xl z-[99999]">
                <SelectItem value="all" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <Users size={13} className="text-accent shrink-0" />
                    <span>Todos os Usuários</span>
                  </div>
                </SelectItem>
                <SelectItem value="recent" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <Clock size={13} className="text-accent shrink-0" />
                    <span>Mais Recentes</span>
                  </div>
                </SelectItem>
                <SelectItem value="name" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <SortAsc size={13} className="text-accent shrink-0" />
                    <span>Nome (A-Z)</span>
                  </div>
                </SelectItem>
                <SelectItem value="channels" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <Tv size={13} className="text-accent shrink-0" />
                    <span>Mais Canais</span>
                  </div>
                </SelectItem>
                <SelectItem value="admins" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <ShieldCheck size={13} className="text-accent shrink-0" />
                    <span>Apenas Admins</span>
                  </div>
                </SelectItem>
                <SelectItem value="blacklisted" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <UserX size={13} className="text-accent shrink-0" />
                    <span>Apenas Bloqueados</span>
                  </div>
                </SelectItem>
                <SelectItem value="with-channels" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <Tv size={13} className="text-accent shrink-0" />
                    <span>Com Canais</span>
                  </div>
                </SelectItem>
                <SelectItem value="without-channels" className="text-xs font-medium cursor-pointer py-2">
                  <div className="flex items-center gap-2">
                    <Hourglass size={13} className="text-accent shrink-0" />
                    <span>Sem Canais</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        {visibleUsers.length === 0 ? (
          <div className="text-center py-10 text-xs text-muted-foreground border border-border rounded-xl bg-card">
            Nenhum usuário encontrado com os filtros aplicados.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2.5">
            {visibleUsers.map(user => (
              <div
                key={user.id}
                onClick={() => setAdminSelectedUser(user)}
                className="flex items-center justify-between p-3 rounded-xl border border-border bg-card hover:bg-muted/30 transition-all cursor-pointer shadow-xs group"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <div className="flex items-center justify-center size-8 rounded-full shrink-0 text-xs font-bold bg-accent/15 text-accent">
                    {(user.first_name || '?')[0].toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <p className="text-xs font-bold text-foreground truncate group-hover:text-accent transition-colors">
                      {user.first_name || 'Sem nome'}
                    </p>
                    <p className="text-[10px] text-muted-foreground truncate">
                      ID: {user.id} • {user.channels?.length || 0} canal(is)
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-1.5 shrink-0">
                  {user.is_admin && <Badge variant="default" className="text-[9px] px-1.5 py-0.2">Admin</Badge>}
                  {user.is_blacklisted && <Badge variant="destructive" className="text-[9px] px-1.5 py-0.2">Bloqueado</Badge>}
                  <ChevronRight size={15} className="text-muted-foreground/40 group-hover:text-foreground transition-colors" />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    );
  };

  // ── Channels Tab ──

  const renderChannelsTab = () => {
    return (
      <div className="space-y-3">
        <div className="flex flex-wrap items-center justify-between gap-2 px-1">
          <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
            Canais Conectados ({visibleChannels.length})
          </span>
          <div className="flex items-center gap-2">
            <span className="text-[11px] text-muted-foreground">Clique no canal para expandir detalhes</span>
          </div>
        </div>
        {visibleChannels.length === 0 ? (
          <div className="text-center py-10 text-xs text-muted-foreground border border-border rounded-xl bg-card">
            Nenhum canal encontrado.
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2.5 items-start">
            {visibleChannels.map(channel => {
              const isExpanded = expandedChannelId === channel.id;
              const ownerUser = usersList.find(u => u.id === channel.ownerId);

              return (
                <div
                  key={channel.id}
                  className={`rounded-xl border transition-all shadow-xs overflow-hidden ${
                    isExpanded
                      ? 'border-accent/60 bg-card ring-1 ring-accent/20'
                      : 'border-border bg-card hover:bg-muted/30 cursor-pointer'
                  }`}
                >
                  {/* Cabeçalho do Card */}
                  <div
                    onClick={() => setExpandedChannelId(isExpanded ? null : channel.id)}
                    className="flex items-center justify-between p-3 cursor-pointer group"
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      <div className="flex items-center justify-center size-8 rounded-lg shrink-0 bg-accent/15 text-accent">
                        <Radio size={16} />
                      </div>
                      <div className="min-w-0">
                        <p className="text-xs font-bold text-foreground truncate group-hover:text-accent transition-colors">
                          {channel.title}
                        </p>
                        <p className="text-[10px] font-mono text-muted-foreground">
                          ID: {channel.id}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-2 shrink-0">
                      {channel.subscriberCount ? (
                        <Badge variant="secondary" className="text-[9px] font-mono px-1.5 py-0.2">
                          👥 {channel.subscriberCount}
                        </Badge>
                      ) : null}
                      <ChevronRight
                        size={16}
                        className={`text-muted-foreground/50 transition-transform duration-200 ${
                          isExpanded ? 'rotate-90 text-accent' : ''
                        }`}
                      />
                    </div>
                  </div>

                  {/* Painel Expansível Inline (Sanfona) */}
                  {isExpanded && (
                    <div className="px-3 pb-3.5 pt-1 border-t border-border/60 bg-muted/20 space-y-2.5 animate-in fade-in slide-in-from-top-1 duration-150">
                      <div className="grid grid-cols-1 gap-2 text-xs pt-1">
                        <div className="flex flex-col gap-1 p-2 rounded-lg bg-card/90 border border-border/50">
                          <span className="text-[10px] uppercase font-bold tracking-wider text-muted-foreground flex items-center gap-1.5">
                            <Radio size={12} className="text-accent" />
                            ID do Canal
                          </span>
                          <span className="font-mono font-bold text-foreground text-xs break-all">
                            {channel.id}
                          </span>
                        </div>

                        <div className="flex flex-col gap-1 p-2 rounded-lg bg-card/90 border border-border/50">
                          <span className="text-[10px] uppercase font-bold tracking-wider text-muted-foreground flex items-center gap-1.5">
                            <UserIcon size={12} className="text-accent" />
                            Dono do Canal
                          </span>
                          <span className="font-semibold text-foreground text-xs leading-tight break-words">
                            {ownerUser?.first_name || 'Desconhecido'}{' '}
                            <span className="text-[11px] text-muted-foreground font-mono font-normal">
                              (ID: {channel.ownerId || 'N/A'})
                            </span>
                          </span>
                        </div>

                        <div className="flex items-center justify-between p-2 rounded-lg bg-card/90 border border-border/50">
                          <span className="text-[10px] uppercase font-bold tracking-wider text-muted-foreground flex items-center gap-1.5">
                            <Calendar size={12} className="text-accent" />
                            Adicionado em
                          </span>
                          <span className="font-medium text-foreground text-xs font-mono">
                            {channel.created_at
                              ? new Date(channel.created_at).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' })
                              : 'Não informado'}
                          </span>
                        </div>
                      </div>

                      <Button
                        size="sm"
                        className="w-full h-8 rounded-lg text-xs font-bold bg-accent hover:bg-accent/90 text-accent-foreground flex items-center justify-center gap-1.5 cursor-pointer shadow-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          navigateToChannel(channel.id);
                        }}
                      >
                        <ExternalLink size={14} />
                        <span>Ir para Dashboard do Canal</span>
                      </Button>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    );
  };

  // ── Notice Tab ──

  const renderNoticeTab = () => {
    return (
      <div className="space-y-4">
        <AdminNoticeTab
          noticeMessage={noticeMessage}
          setNoticeMessage={setNoticeMessage}
          noticeImageUrl={noticeImageUrl}
          setNoticeImageUrl={setNoticeImageUrl}
          noticeTarget={noticeTarget}
          setNoticeTarget={setNoticeTarget}
          noticeTargetId={noticeTargetId}
          setNoticeTargetId={setNoticeTargetId}
          noticeButtons={noticeButtons}
          handleAddNoticeButton={handleAddNoticeButton}
          updateNoticeButton={updateNoticeButton}
          removeNoticeButton={removeNoticeButton}
          handleSendNotice={handleSendNotice}
          isSendingNotice={isSendingNotice}
          users={usersList}
          channels={channelsList}
        />
      </div>
    );
  };

  // ── Render ──

  const tabCopy: Record<typeof localActiveTab, { title: string; description: string }> = {
    overview: { title: 'Visão geral', description: 'Acompanhamento operacional da base' },
    users: { title: 'Usuários', description: 'Gerencie usuários, acessos e canais vinculados' },
    channels: { title: 'Canais', description: 'Todos os canais conectados ao FreddyBot' },
    notice: { title: 'Broadcast', description: 'Envie comunicações para usuários e canais' },
    audit: { title: 'Auditoria', description: 'Verifique a presença e o estado do bot nos canais' },
    logs: { title: 'Logs', description: 'Investigue o histórico operacional do sistema' },
    config: { title: 'Configurações', description: 'Defina o comportamento global do FreddyBot' },
    accounts: { title: 'Contas MTProto', description: 'Gerencie contas usadas na edição de postagens' },
    'premium-features': { title: 'Features premium', description: 'Controle recursos e preços premium' },
    subscriptions: { title: 'Assinaturas', description: 'Gerencie assinaturas e pagamentos dos usuários' },
  };

  return (
    <div className={`admin-crm-page ${localActiveTab === 'overview' ? 'is-overview' : ''} ${isPending ? 'is-pending' : ''}`}>
      {localActiveTab !== 'overview' && (
        <AdminPageHeader
          eyebrow="Painel administrativo"
          title={tabCopy[localActiveTab].title}
          description={tabCopy[localActiveTab].description}
        />
      )}

      {localActiveTab === 'overview' && renderOverviewTab()}
      {localActiveTab === 'users' && !adminSelectedUser && renderUsersTab()}
      {localActiveTab === 'users' && adminSelectedUser && renderUserDetail()}
      {localActiveTab === 'channels' && renderChannelsTab()}
      {localActiveTab === 'audit' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando auditoria...</div>}>
            <AdminAuditTab
              navigateToChannel={navigateToChannel}
              onOpenUser={(id) => {
                onOpenUserDetail(id);
                navigateToTab('users');
              }}
              results={auditResults}
              setResults={setAuditResults}
              loading={auditLoading}
              onRunAudit={handleRunAudit}
            />
          </Suspense>
        </div>
      )}
      {localActiveTab === 'notice' && (
        <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando aviso...</div>}>
          {renderNoticeTab()}
        </Suspense>
      )}
      {localActiveTab === 'logs' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando logs...</div>}>
            <AdminLogsTab navigateToChannel={navigateToChannel} initialChannelId={initialLogsChannelId} />
          </Suspense>
        </div>
      )}
      {localActiveTab === 'config' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando configurações...</div>}>
            <AdminConfigTab />
          </Suspense>
        </div>
      )}
      {localActiveTab === 'accounts' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando contas MTProto...</div>}>
            <AdminMTProtoAccountsTab />
          </Suspense>
        </div>
      )}
      {localActiveTab === 'premium-features' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando recursos premium...</div>}>
            <AdminPremiumFeaturesTab toast={toast} />
          </Suspense>
        </div>
      )}
      {localActiveTab === 'subscriptions' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando assinaturas...</div>}>
            <AdminSubscriptionsTab toast={toast} />
          </Suspense>
        </div>
      )}
    </div>
  );
}
