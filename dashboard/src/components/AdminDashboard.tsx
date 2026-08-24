import { useState, useMemo, useTransition, useEffect, Dispatch, SetStateAction, Suspense, lazy } from 'react';
import { AdminDashboardData, User, Channel, AuditResult } from '../types';
import { NoticeButton, NoticeTarget, updateUserAdmin, updateUserBlacklist, fetchServerConfig } from '../api';
import { AdminNoticeTab } from './AdminNoticeTab';
import { AdminConfigTab } from './AdminConfigTab';

const AdminAuditTab = lazy(() => import('./AdminAuditTab').then(m => ({ default: m.AdminAuditTab })));
const AdminLogsTab = lazy(() => import('./AdminLogsTab').then(m => ({ default: m.AdminLogsTab })));
const AdminMTProtoAccountsTab = lazy(() => import('./AdminMTProtoAccountsTab').then(m => ({ default: m.AdminMTProtoAccountsTab })));
const AdminPremiumFeaturesTab = lazy(() => import('./AdminPremiumFeaturesTab').then(m => ({ default: m.AdminPremiumFeaturesTab })));
const AdminSubscriptionsTab = lazy(() => import('./AdminSubscriptionsTab').then(m => ({ default: m.AdminSubscriptionsTab })));
import { Hash, ArrowLeft, ChevronRight, User as UserIcon, ShieldCheck, UserX, UserCheck, MessageSquare, Calendar, ExternalLink, Users, Radio, Info, Loader2, MoreVertical, Copy, Link2, SlidersHorizontal, ArrowUpDown } from 'lucide-react';
import { useToast } from './Toast';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from './ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { ConfirmModal } from './ConfirmModal';
import { StatusBadge } from './admin/StatusBadge';
import { OperationsOverview } from './admin/OperationsOverview';
import { AdminPageHeader } from './admin/AdminPageHeader';
import { useAdminCrmControls } from './admin/AdminCrmContext';
import { filterAndSortChannels, filterAndSortUsers } from './admin/crmSelectors';

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
  noticeMediaType: 'photo' | 'video' | 'animation';
  setNoticeMediaType: Dispatch<SetStateAction<'photo' | 'video' | 'animation'>>;
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
  noticeMediaType, setNoticeMediaType,
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
  const { searchQuery, setFilterBy, navigateToTab } = useAdminCrmControls();

  const [localActiveTab, setLocalActiveTab] = useState(activeTab);
  const [isPending, startTransition] = useTransition();
  const [expandedChannelId, setExpandedChannelId] = useState<number | null>(null);
  const [channelFilter, setChannelFilter] = useState<'all' | 'with-owner' | 'with-data' | 'recent'>('all');
  const [channelSort, setChannelSort] = useState<AdminCrmSort>('recent');
  const [channelsPage, setChannelsPage] = useState(1);
  const [userFilterOption, setUserFilterOption] = useState<string>('all');
  const [userSortOption, setUserSortOption] = useState<AdminCrmSort>('recent');
  const [openUserMenu, setOpenUserMenu] = useState<number | null>(null);
  const [usersPage, setUsersPage] = useState(1);
  const [listConfirmation, setListConfirmation] = useState<{ user: User; action: 'admin' | 'blacklist' } | null>(null);

  const [localUsers, setLocalUsers] = useState<User[]>(adminData.users || []);

  useEffect(() => {
    setLocalUsers(adminData.users || []);
  }, [adminData.users]);

  useEffect(() => {
    // Pré-carrega as configurações do servidor em segundo plano para abertura instantânea da aba
    fetchServerConfig().catch(() => {});
  }, []);

  useEffect(() => {
    startTransition(() => {
      setLocalActiveTab(activeTab);
    });
  }, [activeTab]);

  const usersList = localUsers;
  const channelsList = adminData.channels || [];
  const visibleUsers = useMemo(() => {
    return filterAndSortUsers(usersList, searchQuery, userFilterOption as AdminCrmFilter, userSortOption);
  }, [usersList, searchQuery, userFilterOption, userSortOption]);
  const visibleChannels = useMemo(() => {
    const sorted = filterAndSortChannels(channelsList, searchQuery, channelSort);
    const weekAgo = Date.now() - 7 * 24 * 60 * 60 * 1000;
    return sorted.filter((channel) => {
      if (channelFilter === 'with-owner') return Boolean(channel.ownerId);
      if (channelFilter === 'with-data') return Boolean(channel.defaultCaption?.caption || channel.buttons?.length || channel.customCaptions?.length);
      if (channelFilter === 'recent') return new Date(channel.created_at).getTime() >= weekAgo;
      return true;
    });
  }, [channelsList, searchQuery, channelFilter, channelSort]);
  const channelsPerPage = 10;
  const channelsPageCount = Math.max(1, Math.ceil(visibleChannels.length / channelsPerPage));
  const paginatedChannels = visibleChannels.slice((channelsPage - 1) * channelsPerPage, channelsPage * channelsPerPage);

  useEffect(() => setChannelsPage(1), [searchQuery, channelFilter, channelSort]);
  useEffect(() => {
    if (channelsPage > channelsPageCount) setChannelsPage(channelsPageCount);
  }, [channelsPage, channelsPageCount]);
  const usersPerPage = 10;
  const usersPageCount = Math.max(1, Math.ceil(visibleUsers.length / usersPerPage));
  const paginatedUsers = visibleUsers.slice((usersPage - 1) * usersPerPage, usersPage * usersPerPage);

  useEffect(() => setUsersPage(1), [searchQuery, userFilterOption, userSortOption]);
  useEffect(() => {
    if (usersPage > usersPageCount) setUsersPage(usersPageCount);
  }, [usersPage, usersPageCount]);
  useEffect(() => {
    if (openUserMenu === null) return;
    const closeOnEscape = (event: KeyboardEvent) => { if (event.key === 'Escape') setOpenUserMenu(null); };
    document.addEventListener('keydown', closeOnEscape);
    window.addEventListener('scroll', () => setOpenUserMenu(null), { once: true, passive: true });
    return () => {
      document.removeEventListener('keydown', closeOnEscape);
    };
  }, [openUserMenu]);

  const adminSelectedUser = useMemo(() =>
    selectedUserId ? usersList.find(u => u.id === selectedUserId) : null,
    [selectedUserId, usersList]);

  const setAdminSelectedUser = (user: User | null) => onSelectUser(user ? user.id : null);

  const [isUpdatingAdmin, setIsUpdatingAdmin] = useState(false);
  const [isUpdatingBlacklist, setIsUpdatingBlacklist] = useState(false);
  const [confirmAdminOpen, setConfirmAdminOpen] = useState(false);
  const [confirmBlacklistOpen, setConfirmBlacklistOpen] = useState(false);

  // ── User Actions ──

  const handleToggleAdmin = async (uid: number) => {
    setIsUpdatingAdmin(true);
    try {
      const res = await updateUserAdmin(uid);
      if (res.success) {
        const isAdmin = res.data?.isAdmin;
        setLocalUsers(prev => prev.map(u => u.id === uid ? { ...u, is_admin: isAdmin } : u));
        toast(isAdmin ? "Usuário promovido a Admin" : "Privilégios de Admin removidos", "success");
      }
    } catch (err: any) {
      toast(err.message || "Erro ao atualizar status de admin", "error");
    } finally {
      setIsUpdatingAdmin(false);
    }
  };

  const handleToggleBlacklist = async (uid: number) => {
    setIsUpdatingBlacklist(true);
    try {
      const res = await updateUserBlacklist(uid);
      if (res.success) {
        const isBlacklisted = res.data?.isBlacklisted;
        setLocalUsers(prev => prev.map(u => u.id === uid ? { ...u, is_blacklisted: isBlacklisted } : u));
        toast(isBlacklisted ? "Usuário adicionado à Blacklist" : "Usuário removido da Blacklist", isBlacklisted ? "error" : "success");
      }
    } catch (err: any) {
      toast(err.message || "Erro ao atualizar status de blacklist", "error");
    } finally {
      setIsUpdatingBlacklist(false);
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
        onNavigate={(tab) => navigateToTab(tab)}
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
    const channelCount = adminSelectedUser.channels?.length || 0;
    const channelCountText = channelCount === 1 ? '1 canal' : `${channelCount} canais`;

    let statusVariant: 'success' | 'danger' | 'accent' = 'success';
    let statusLabel = 'Ativo';

    if (adminSelectedUser.is_blacklisted) {
      statusVariant = 'danger';
      statusLabel = 'Bloqueado';
    } else if (adminSelectedUser.is_admin) {
      statusVariant = 'accent';
      statusLabel = 'Admin';
    }

    return (
      <div className="admin-user-detail">
        {/* Back Navigation */}
        <div className="admin-user-detail-back-row">
          <button
            type="button"
            onClick={() => setAdminSelectedUser(null)}
            className="admin-user-detail-back"
          >
            <ArrowLeft size={15} />
            <span>Voltar para usuários</span>
          </button>
        </div>

        {/* User Profile Card */}
        <section className="admin-user-detail-panel">
          <div className="admin-user-detail-identity">
            <div className="admin-user-detail-person">
              <div className="admin-user-detail-avatar">
                <UserIcon size={20} />
                {!adminSelectedUser.is_blacklisted && <i aria-hidden="true" />}
              </div>
              <div className="admin-user-detail-copy">
                <h2>{name}</h2>
                <p>
                  ID: {adminSelectedUser.id} · {channelCountText}
                </p>
              </div>
            </div>
            <div className="admin-user-detail-status">
              <StatusBadge label={statusLabel} variant={statusVariant} dot size="sm" />
            </div>
          </div>

          {/* Administrative Actions */}
          <div className="admin-user-detail-actions">
            {/* Primary Action: Mensagem de Suporte */}
            <Button
              variant="default"
              className="admin-user-action admin-user-action-support"
              onClick={() => {
                onMessageUser(adminSelectedUser.id);
                navigateToTab('notice');
              }}
            >
              <MessageSquare size={15} />
              <span>Mensagem de Suporte</span>
            </Button>

            {/* Secondary Action: Tornar Admin / Remover Admin */}
            <Button
              variant="outline"
              className={`admin-user-action admin-user-action-admin ${adminSelectedUser.is_admin ? 'is-remove' : ''}`}
              disabled={isUpdatingAdmin}
              onClick={() => setConfirmAdminOpen(true)}
            >
              {isUpdatingAdmin ? (
                <Loader2 size={15} className="animate-spin" />
              ) : (
                <ShieldCheck size={15} />
              )}
              <span>{adminSelectedUser.is_admin ? "Remover Admin" : "Tornar Admin"}</span>
            </Button>

            {/* Destructive / Restrictive Action: Adicionar à blacklist / Remover da blacklist */}
            <Button
              variant="outline"
              className={`admin-user-action admin-user-action-blacklist ${adminSelectedUser.is_blacklisted ? 'is-restore' : ''}`}
              disabled={isUpdatingBlacklist}
              onClick={() => setConfirmBlacklistOpen(true)}
            >
              {isUpdatingBlacklist ? (
                <Loader2 size={15} className="animate-spin" />
              ) : adminSelectedUser.is_blacklisted ? (
                <UserCheck size={15} />
              ) : (
                <UserX size={15} />
              )}
              <span>{adminSelectedUser.is_blacklisted ? "Remover da blacklist" : "Adicionar à blacklist"}</span>
            </Button>
          </div>
        </section>

        {/* User Channels Section */}
        <section className="admin-user-detail-channels">
          <h3>
            Canais do Usuário
          </h3>
          {adminSelectedUser.channels && adminSelectedUser.channels.length > 0 ? (
            <div className="admin-user-channel-list">
              {adminSelectedUser.channels.map((c: Channel) => (
                <button
                  key={c.id}
                  type="button"
                  className="admin-user-channel"
                  onClick={() => navigateToChannel(c.id)}
                >
                  <div className="admin-user-channel-icon">
                    <Hash size={17} />
                  </div>
                  <div className="admin-user-channel-copy">
                    <h4>{c.title || 'Canal sem título'}</h4>
                    <p>ID: {c.id}</p>
                  </div>
                  <ChevronRight size={16} />
                </button>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-8 px-4 text-center rounded-2xl border border-white/10 bg-white/5">
              <div className="flex items-center justify-center size-12 rounded-xl bg-muted/20 text-muted-foreground/40 mb-2.5">
                <Hash size={24} />
              </div>
              <p className="text-sm font-semibold text-foreground">Nenhum canal vinculado</p>
              <p className="text-xs text-muted-foreground mt-0.5">Este usuário ainda não adicionou canais ao bot.</p>
            </div>
          )}
        </section>

        {/* Administrative Information Notice */}
        <aside className="admin-user-detail-notice">
          <Info size={15} />
          <p>
            As alterações feitas aqui são aplicadas imediatamente e refletem no acesso do usuário.
          </p>
        </aside>

        {/* Modais de Confirmação para Ações Administrativas */}
        <ConfirmModal
          open={confirmBlacklistOpen}
          onClose={() => setConfirmBlacklistOpen(false)}
          onConfirm={() => handleToggleBlacklist(adminSelectedUser.id)}
          title={adminSelectedUser.is_blacklisted ? "Remover da blacklist?" : `Adicionar ${name} à blacklist?`}
          message={
            adminSelectedUser.is_blacklisted
              ? "O usuário voltará a ter acesso normal aos recursos e comandos do FreddyBot."
              : "O usuário poderá perder acesso imediato a todos os recursos do FreddyBot."
          }
          confirmText={adminSelectedUser.is_blacklisted ? "Remover" : "Adicionar à blacklist"}
          danger={!adminSelectedUser.is_blacklisted}
        />

        <ConfirmModal
          open={confirmAdminOpen}
          onClose={() => setConfirmAdminOpen(false)}
          onConfirm={() => handleToggleAdmin(adminSelectedUser.id)}
          title={adminSelectedUser.is_admin ? "Remover privilégios de Admin?" : `Tornar ${name} Administrador?`}
          message={
            adminSelectedUser.is_admin
              ? "O usuário deixará de ter acesso ao painel de administração e aos comandos de gestão."
              : "O usuário terá acesso total ao painel administrativo e comandos restritos do bot."
          }
          confirmText={adminSelectedUser.is_admin ? "Remover Admin" : "Tornar Admin"}
          danger={adminSelectedUser.is_admin}
        />
      </div>
    );
  };

  // ── Users Tab ──

  const renderUsersTab = () => {
    const totalWithChannels = usersList.filter((user) => (user.channels?.length || 0) > 0).length;
    const totalAdmins = usersList.filter((user) => user.is_admin).length;
    const totalBlocked = usersList.filter((user) => user.is_blacklisted).length;
    const sortLabel = userSortOption === 'name' ? 'Nome A–Z' : userSortOption === 'channels' ? 'Mais canais' : 'Mais recentes';
    const copyUserId = async (user: User) => {
      await navigator.clipboard?.writeText(String(user.id));
      toast('ID copiado', 'success');
      setOpenUserMenu(null);
    };
    const formatCreatedAt = (value?: string) => {
      if (!value) return '';
      const date = new Date(value);
      if (!Number.isFinite(date.getTime())) return '';
      const now = new Date();
      const isToday = date.toDateString() === now.toDateString();
      return isToday
        ? `Hoje ${date.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })}`
        : date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
    };

    return (
      <div className="admin-users-page">
        <section className="admin-users-metrics" aria-label="Resumo dos usuários">
          <div className="metric-total"><Users size={15} /><strong>{usersList.length}</strong><span>Total de usuários<small>cadastrados</small></span></div>
          <div className="metric-linked"><Link2 size={15} /><strong>{totalWithChannels}</strong><span>Com canais<small>vinculados</small></span></div>
          <div className="metric-admin"><ShieldCheck size={15} /><strong>{totalAdmins}</strong><span>Administradores</span></div>
          <div className="metric-blocked"><UserX size={15} /><strong>{totalBlocked}</strong><span>Bloqueados</span></div>
        </section>

        <section className="admin-users-filters" aria-labelledby="users-filter-title">
          <span id="users-filter-title">Filtros</span>
          <div>
            <Select value={['all', 'with-channels', 'without-channels'].includes(userFilterOption) ? userFilterOption : 'all'} onValueChange={(val) => setUserFilterOption(val)}>
              <SelectTrigger aria-label="Escopo dos usuários"><Users size={13} /><SelectValue /></SelectTrigger>
              <SelectContent><SelectItem value="all">Todos os Usuários</SelectItem><SelectItem value="with-channels">Com canais</SelectItem><SelectItem value="without-channels">Sem canais</SelectItem></SelectContent>
            </Select>
            <Select value={['admins', 'blacklisted'].includes(userFilterOption) ? userFilterOption : 'all'} onValueChange={(val) => setUserFilterOption(val)}>
              <SelectTrigger aria-label="Status do usuário"><ShieldCheck size={13} /><span>{userFilterOption === 'admins' ? 'Admins' : userFilterOption === 'blacklisted' ? 'Bloqueados' : 'Status'}</span></SelectTrigger>
              <SelectContent><SelectItem value="all">Todos os status</SelectItem><SelectItem value="admins">Administradores</SelectItem><SelectItem value="blacklisted">Bloqueados</SelectItem></SelectContent>
            </Select>
            <Select value={userSortOption} onValueChange={(val) => setUserSortOption(val as AdminCrmSort)}>
              <SelectTrigger aria-label="Ordenação"><SlidersHorizontal size={13} /><span>Mais filtros</span></SelectTrigger>
              <SelectContent><SelectItem value="recent">Mais recentes</SelectItem><SelectItem value="name">Nome A–Z</SelectItem><SelectItem value="channels">Mais canais</SelectItem></SelectContent>
            </Select>
          </div>
        </section>

        <div className="admin-users-results"><span>{visibleUsers.length} {visibleUsers.length === 1 ? 'resultado encontrado' : 'resultados encontrados'}</span><button type="button" onClick={() => setUserSortOption(userSortOption === 'name' ? 'channels' : userSortOption === 'channels' ? 'recent' : 'name')}>Ordenar: <b>{sortLabel}</b>⌄</button></div>

        {visibleUsers.length === 0 ? (
          <div className="admin-users-empty"><Users size={22} /><strong>Nenhum usuário encontrado</strong><span>Tente ajustar a busca ou os filtros.</span></div>
        ) : (
          <div className="admin-users-list">
            {openUserMenu !== null && (
              <button
                type="button"
                className="admin-user-menu-backdrop"
                aria-label="Fechar menu de ações"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  setOpenUserMenu(null);
                }}
              />
            )}
            {paginatedUsers.map(user => (
              <article key={user.id} className="admin-user-card" onClick={() => setAdminSelectedUser(user)}>
                <div className="admin-user-avatar">{Array.from((user.first_name || user.username || '?').trim())[0]?.toLocaleUpperCase('pt-BR') || '?'}{!user.is_blacklisted && <i />}</div>
                <div className="admin-user-card-copy">
                  <strong>{user.first_name || user.username || 'Sem nome'}</strong>
                  <div className="admin-user-id"><span>ID: {user.id}</span><button type="button" aria-label={`Copiar ID de ${user.first_name || 'usuário'}`} onClick={(event) => { event.stopPropagation(); copyUserId(user); }}><Copy size={12} /></button></div>
                  <div className="admin-user-badges">
                    <span className={user.is_blacklisted ? 'is-blocked' : 'is-active'}>{user.is_blacklisted ? 'Bloqueado' : '● Ativo'}</span>
                    {user.is_admin && <span className="is-admin">Admin</span>}
                    <span className="has-channels"><Link2 size={10} /> {user.channels?.length || 0} {(user.channels?.length || 0) === 1 ? 'canal' : 'canais'}</span>
                  </div>
                </div>
                <div className="admin-user-card-side"><time>{formatCreatedAt(user.created_at)}</time><div data-user-actions>
                  <button type="button" className="admin-user-kebab" aria-label="Abrir ações do usuário" aria-expanded={openUserMenu === user.id} onClick={(event) => { event.stopPropagation(); setOpenUserMenu(openUserMenu === user.id ? null : user.id); }}><MoreVertical size={17} /></button>
                  {openUserMenu === user.id && <div className="admin-user-actions-menu" onClick={(event) => event.stopPropagation()}>
                    <button type="button" onClick={() => { setOpenUserMenu(null); setAdminSelectedUser(user); }}><UserIcon size={15} />Ver detalhes</button>
                    <button type="button" onClick={() => { setOpenUserMenu(null); onMessageUser(user.id); navigateToTab('notice'); }}><MessageSquare size={15} />Mensagem de suporte</button>
                    <button type="button" onClick={() => { setOpenUserMenu(null); setListConfirmation({ user, action: 'admin' }); }}><ShieldCheck size={15} />{user.is_admin ? 'Remover administrador' : 'Tornar administrador'}</button>
                    <button type="button" disabled={!user.channels?.length} onClick={() => { setOpenUserMenu(null); setAdminSelectedUser(user); }}><Link2 size={15} />Ver canais vinculados</button>
                    <button type="button" onClick={() => copyUserId(user)}><Copy size={15} />Copiar ID</button>
                    <button type="button" className="is-destructive" onClick={() => { setOpenUserMenu(null); setListConfirmation({ user, action: 'blacklist' }); }}><UserX size={15} />{user.is_blacklisted ? 'Remover da blacklist' : 'Adicionar à blacklist'}</button>
                  </div>}
                </div></div>
              </article>
            ))}
          </div>
        )}

        {visibleUsers.length > 0 && <nav className="admin-users-pagination" aria-label="Paginação de usuários"><button type="button" disabled={usersPage === 1} onClick={() => setUsersPage((page) => page - 1)}>‹</button><span>{usersPage}</span><button type="button" disabled={usersPage === usersPageCount} onClick={() => setUsersPage((page) => page + 1)}>›</button></nav>}

        <ConfirmModal open={listConfirmation !== null} onClose={() => setListConfirmation(null)} onConfirm={() => {
          if (!listConfirmation) return;
          if (listConfirmation.action === 'admin') handleToggleAdmin(listConfirmation.user.id);
          else handleToggleBlacklist(listConfirmation.user.id);
          setListConfirmation(null);
        }} title={listConfirmation?.action === 'admin' ? (listConfirmation.user.is_admin ? 'Remover privilégios de Admin?' : `Tornar ${listConfirmation.user.first_name || 'usuário'} administrador?`) : (listConfirmation?.user.is_blacklisted ? 'Remover da blacklist?' : `Adicionar ${listConfirmation?.user.first_name || 'usuário'} à blacklist?`)} message={listConfirmation?.action === 'admin' ? 'Esta ação altera o acesso administrativo do usuário ao FreddyBot.' : 'Esta ação altera o acesso do usuário aos recursos do FreddyBot.'} confirmText="Confirmar" danger={listConfirmation?.action === 'blacklist' && !listConfirmation.user.is_blacklisted} />
      </div>
    );
  };

  // ── Channels Tab ──

  const renderChannelsTab = () => {
    const channelsWithMembers = channelsList.filter((channel) => Number((channel as any).subscriberCount || 0) > 0).length;
    const channelsWithData = channelsList.filter((channel) => channel.defaultCaption?.caption || channel.buttons?.length || channel.customCaptions?.length).length;
    const latestChannel = [...channelsList].filter((channel) => channel.created_at).sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())[0];
    const latestDate = latestChannel ? new Date(latestChannel.created_at).toLocaleDateString('pt-BR') : '—';
    const channelSortLabel = channelSort === 'name' ? 'Nome A–Z' : 'Mais recentes';
    const copyChannelValue = async (value: string | number, message: string) => {
      await navigator.clipboard?.writeText(String(value));
      toast(message, 'success');
    };

    return (
      <div className="admin-channels-page">
        <section className="admin-channels-metrics" aria-label="Resumo dos canais">
          <div className="metric-connected"><Radio size={15} /><strong>{channelsList.length}</strong><span>Canais<small>conectados</small></span></div>
          <div className="metric-members"><Users size={15} /><strong>{channelsWithMembers}</strong><span>Com membros</span></div>
          <div className="metric-data"><ArrowUpDown size={15} /><strong>{channelsWithData}</strong><span>Com dados</span></div>
          <div className="metric-recent"><Calendar size={15} /><strong>{latestDate}</strong><span>Mais recente</span></div>
        </section>

        <section className="admin-channels-filters" aria-labelledby="channels-filter-title">
          <span id="channels-filter-title">Filtros</span>
          <div>
            <Select value={channelFilter === 'all' || channelFilter === 'with-owner' ? channelFilter : 'all'} onValueChange={(value) => setChannelFilter(value as typeof channelFilter)}>
              <SelectTrigger aria-label="Escopo dos canais"><Radio size={13} /><span>{channelFilter === 'with-owner' ? 'Com proprietário' : 'Todos os Canais'}</span></SelectTrigger>
              <SelectContent><SelectItem value="all">Todos os Canais</SelectItem><SelectItem value="with-owner">Com proprietário</SelectItem></SelectContent>
            </Select>
            <Select value={channelFilter === 'with-data' ? 'with-data' : 'all'} onValueChange={(value) => setChannelFilter(value as typeof channelFilter)}>
              <SelectTrigger aria-label="Status do canal"><Radio size={13} /><span>{channelFilter === 'with-data' ? 'Com dados' : 'Status'}</span></SelectTrigger>
              <SelectContent><SelectItem value="all">Todos os status</SelectItem><SelectItem value="with-data">Com dados</SelectItem></SelectContent>
            </Select>
            <Select value={channelFilter === 'recent' ? 'recent' : 'all'} onValueChange={(value) => setChannelFilter(value as typeof channelFilter)}>
              <SelectTrigger aria-label="Mais filtros"><SlidersHorizontal size={13} /><span>Mais filtros</span></SelectTrigger>
              <SelectContent><SelectItem value="all">Sem filtro adicional</SelectItem><SelectItem value="recent">Adicionados recentemente</SelectItem></SelectContent>
            </Select>
          </div>
        </section>

        <div className="admin-channels-results"><span>{visibleChannels.length} {visibleChannels.length === 1 ? 'canal encontrado' : 'canais encontrados'}</span><button type="button" onClick={() => setChannelSort(channelSort === 'name' ? 'recent' : 'name')}>Ordenar: <b>{channelSortLabel}</b>⌄</button></div>

        {visibleChannels.length === 0 ? (
          <div className="admin-channels-empty"><Radio size={23} /><strong>{channelsList.length ? 'Nenhum canal encontrado' : 'Nenhum canal conectado'}</strong><span>{channelsList.length ? 'Tente ajustar a busca ou os filtros.' : 'Quando um canal for vinculado ao FreddyBot, ele aparecerá aqui.'}</span></div>
        ) : (
          <div className="admin-channels-list">
            {paginatedChannels.map(channel => {
              const isExpanded = expandedChannelId === channel.id;
              const ownerUser = usersList.find(u => u.id === channel.ownerId);
              const addedAt = channel.created_at ? new Date(channel.created_at).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit', second: '2-digit' }) : 'Não informado';

              return (
                <article key={channel.id} className={`admin-channel-card ${isExpanded ? 'is-expanded' : ''}`}>
                  <button
                    type="button"
                    className="admin-channel-card-header"
                    onClick={() => setExpandedChannelId(isExpanded ? null : channel.id)}
                    aria-expanded={isExpanded}
                  >
                    <span className="admin-channel-icon"><Radio size={19} /><i /></span>
                    <span className="admin-channel-identity"><strong>{channel.title || 'Canal sem título'}</strong><small>ID: {channel.id}</small></span>
                    <ChevronRight size={18} className={isExpanded ? 'is-open' : ''} />
                  </button>

                  {isExpanded && (
                    <div className="admin-channel-expanded">
                      <div className="admin-channel-detail"><Radio size={15} /><span><small>ID DO CANAL</small><strong>{channel.id}</strong></span><button type="button" aria-label="Copiar ID do canal" onClick={() => copyChannelValue(channel.id, 'ID do canal copiado')}><Copy size={14} /></button></div>
                      <div className="admin-channel-detail"><UserIcon size={15} /><span><small>DONO DO CANAL</small><strong>{ownerUser?.first_name || ownerUser?.username || 'Desconhecido'} <em>(ID: {channel.ownerId || 'N/A'})</em></strong></span><button type="button" aria-label="Copiar ID do proprietário" disabled={!channel.ownerId} onClick={() => copyChannelValue(channel.ownerId, 'ID do proprietário copiado')}><Copy size={14} /></button></div>
                      <div className="admin-channel-detail"><Calendar size={15} /><span><small>ADICIONADO EM</small><strong>{addedAt}</strong></span><button type="button" aria-label="Copiar data de adição" onClick={() => copyChannelValue(addedAt, 'Data copiada')}><Copy size={14} /></button></div>
                      <Button type="button" className="admin-channel-dashboard-button" onClick={() => navigateToChannel(channel.id)}><ExternalLink size={14} />Ir para Dashboard do Canal</Button>
                    </div>
                  )}
                </article>
              );
            })}
          </div>
        )}

        {visibleChannels.length > 0 && <nav className="admin-channels-pagination" aria-label="Paginação de canais"><button type="button" disabled={channelsPage === 1} onClick={() => setChannelsPage((page) => page - 1)}>‹</button><span>{channelsPage}</span><button type="button" disabled={channelsPage === channelsPageCount} onClick={() => setChannelsPage((page) => page + 1)}>›</button></nav>}
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
          noticeMediaType={noticeMediaType}
          setNoticeMediaType={setNoticeMediaType}
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
          eyebrow="PAINEL ADMINISTRATIVO"
          title={tabCopy[localActiveTab].title}
          badge={
            localActiveTab === 'users' ? (
              <Badge variant="secondary" className="text-xs px-2.5 py-0.5 rounded-full font-bold bg-slate-800/90 text-slate-300 border-slate-700/60 shadow-2xs">
                {usersList.length}
              </Badge>
            ) : undefined
          }
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
      {localActiveTab === 'notice' && renderNoticeTab()}
      {localActiveTab === 'logs' && (
        <div className="space-y-4">
          <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando logs...</div>}>
            <AdminLogsTab navigateToChannel={navigateToChannel} initialChannelId={initialLogsChannelId} />
          </Suspense>
        </div>
      )}
      {localActiveTab === 'config' && (
        <div className="space-y-4">
          <AdminConfigTab />
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
