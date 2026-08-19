import { useState, useEffect, useCallback, useMemo, memo, Suspense, lazy } from 'react';
import { DashboardData, Button, TelegramUser, AdminDashboardData, Channel, AuditResult } from './types';
import {
  login, fetchDashboardData, fetchUserChannels, fetchAdminDashboard,
  updateMessagePermission, updateButtonsPermission,
  createButton, deleteButton, updateButton, updateLayoutButtons,
  updateDefaultCaption, updateNewPackCaption, updateReactions, 
  updateReactionPosition, updateDynamicLinks,
  sendAdminNotice, NoticeButton, NoticeRequest, NoticeTarget, disconnectChannel, fetchAuditCheckBot,
  fetchSubscriptionStatus, fetchAccountStatus
} from './api';
import { ButtonGrid } from './components/ButtonGrid';
import { CaptionCard } from './components/CaptionCard';
import { NewPackCaptionCard } from './components/NewPackCaptionCard';
import { ReactionsCard } from './components/ReactionsCard';
import { DashboardInicioTab } from './components/DashboardInicioTab';
import { PremiumTab } from './components/PremiumTab';
import { NativeReactionsCard } from './components/NativeReactionsCard';
import { UserTemplatesManager } from './components/UserTemplatesManager';
import { PerfLine } from './components/WaveDivider';
import { ErrorBoundary } from './components/ErrorBoundary';
import { TabBar, Tab } from './components/TabBar';
import { AdminLayout } from './components/admin/AdminLayout';
import { ToastProvider, useToast } from './components/Toast';
import { useTheme } from './hooks/useTheme';

const AdminDashboard = lazy(() => import('./components/AdminDashboard').then(m => ({ default: m.AdminDashboard })));
const ContaTelegramTab = lazy(() => import('./components/ContaTelegramTab').then(m => ({ default: m.ContaTelegramTab })));
const PremiumConfigTab = lazy(() => import('./components/PremiumConfigTab').then(m => ({ default: m.PremiumConfigTab })));
const ScheduleTab = lazy(() => import('./components/ScheduleTab').then(m => ({ default: m.ScheduleTab })));
import { Button as ShadButton } from './components/ui/button';
import { Switch } from './components/ui/switch';
import { Badge } from './components/ui/badge';
import {
  Dialog, DialogContent,
} from './components/ui/dialog';
import { SideMenu } from './components/SideMenu';
import {
  Hash, Sun, Moon, Send, ExternalLink, MousePointerClick, Link2,
  LayoutDashboard, Type, Grid3X3, Shield, MessageCircle,
  AlertTriangle, ChevronRight, ArrowLeft, Zap, UserCheck,
  CloudMoon, Sunrise, Headphones, Video, Image, FileText, Smile, Film, SlidersHorizontal,
  Crown, Calendar, Menu, Layers
} from 'lucide-react';

const BASE_TABS: Tab[] = [
  { id: 'geral', label: 'Início', icon: <LayoutDashboard size={22} /> },
  { id: 'legendas', label: 'Legendas', icon: <Type size={22} /> },
  { id: 'botoes', label: 'Botões', icon: <Grid3X3 size={22} /> },
  { id: 'permissoes', label: 'Permissões', icon: <Shield size={22} /> },
  { id: 'conta', label: 'Conta Telegram', icon: <UserCheck size={22} /> },
];

const permLabels: Record<string, string> = {
  message: 'Mensagem', audio: 'Áudio', video: 'Vídeo',
  photo: 'Foto', document: 'Arquivo', sticker: 'Sticker', gif: 'GIF', linkPreview: 'Link Preview',
  reactions: 'Reações',
};

function getChannelIdFromUrl(): string | null {
  const match = window.location.pathname.match(/\/dashboard\/(-?\d+)/);
  return match ? match[1] : null;
}

function isChannelsRoute(): boolean {
  return window.location.pathname.startsWith('/me/channels');
}

function isRootRoute(): boolean {
  return window.location.pathname === '/' || window.location.pathname === '';
}

function isAdminDashRoute(): boolean {
  return window.location.pathname.startsWith('/admin/dash');
}

export type AdminTabId = 'overview' | 'users' | 'channels' | 'notice' | 'config' | 'audit' | 'logs' | 'accounts' | 'premium-features' | 'subscriptions';

function getInitialAdminTabFromUrl(): AdminTabId {
  const tab = new URLSearchParams(window.location.search).get('tab');
  if (tab === 'logs') return 'logs';
  if (tab === 'users' || tab === 'channels' || tab === 'notice' || tab === 'config' || tab === 'audit' || tab === 'accounts' || tab === 'premium-features' || tab === 'subscriptions') return tab as AdminTabId;
  return 'overview';
}

function getInitialLogsChannelIdFromUrl(): string {
  return new URLSearchParams(window.location.search).get('channelId') || '';
}

type AuthState = 'idle' | 'authenticating' | 'authenticated' | 'error';

const MemoizedAdminDashboard = memo(AdminDashboard);

const DashboardContent = memo(function DashboardContent() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState('geral');
  const [adminActiveTab, setAdminActiveTab] = useState<AdminTabId>(() => getInitialAdminTabFromUrl());
  const [adminSelectedUserId, setAdminSelectedUserId] = useState<number | null>(null);
  const [tgUser, setTgUser] = useState<TelegramUser | null>(null);
  const [authState, setAuthState] = useState<AuthState>('idle');
  const [authError, setAuthError] = useState<string>('');
  const toast = useToast();
  const { theme, toggleTheme } = useTheme();

  const [adminData, setAdminData] = useState<AdminDashboardData | null>(null);
  const [noticeMessage, setNoticeMessage] = useState('');
  const [noticeImageUrl, setNoticeImageUrl] = useState('');
  const [noticeTarget, setNoticeTarget] = useState<NoticeTarget>('all');
  const [noticeTargetId, setNoticeTargetId] = useState<string>('');
  const [noticeButtons, setNoticeButtons] = useState<NoticeButton[]>([]);
  const [isSendingNotice, setIsSendingNotice] = useState(false);
  const [auditResults, setAuditResults] = useState<AuditResult[] | null>(null);
  const [auditLoading, setAuditLoading] = useState(false);
  const [initialLogsChannelId] = useState(() => getInitialLogsChannelIdFromUrl());
  const [showContaModal, setShowContaModal] = useState(false);
  const [showTemplatesModal, setShowTemplatesModal] = useState(false);
  const [showSchedulesModal, setShowSchedulesModal] = useState(false);
  const [showSideMenu, setShowSideMenu] = useState(false);
  const [hasPremiumAccess, setHasPremiumAccess] = useState(false);
  const [hasSubscription, setHasSubscription] = useState(false);
  const [hasMtprotoAccount, setHasMtprotoAccount] = useState(false);
  const [premiumEnabled, setPremiumEnabled] = useState(false);
  const [connectedAccountEnabled, setConnectedAccountEnabled] = useState(false);

  // Travar scroll do body quando o modal estiver aberto (funciona no iOS tambem)
  useEffect(() => {
    if (showContaModal) {
      const scrollY = window.scrollY;
      document.body.style.position = 'fixed';
      document.body.style.top = `-${scrollY}px`;
      document.body.style.left = '0';
      document.body.style.right = '0';
      document.body.style.overflow = 'hidden';
    } else {
      const top = document.body.style.top;
      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.left = '';
      document.body.style.right = '';
      document.body.style.overflow = '';
      if (top) {
        window.scrollTo(0, parseInt(top, 10) * -1);
      }
    }
    return () => {
      document.body.style.position = '';
      document.body.style.top = '';
      document.body.style.left = '';
      document.body.style.right = '';
      document.body.style.overflow = '';
    };
  }, [showContaModal]);

  useEffect(() => {
    const savedUid = sessionStorage.getItem('lastAdminUserId');
    if (savedUid) {
      setAdminSelectedUserId(parseInt(savedUid, 10));
    }
  }, []);

  // Check premium access for channel-level tabs.
  // Só roda após autenticar (authState === 'authenticated') e é fail-closed:
  // se a busca falhar, as features ficam ocultas em vez de vazar.
  useEffect(() => {
    if (authState !== 'authenticated') return;
    let cancelled = false;

    Promise.all([
      fetchSubscriptionStatus().catch(() => ({ data: null })),
      fetchAccountStatus().catch(() => ({ status: 'disconnected' })),
    ]).then(([subRes, accStatus]) => {
      if (cancelled) return;

      const s = subRes?.data;
      const active = s?.hasSubscription && s?.subscription?.status === 'active';
      const account = accStatus?.status === 'connected';
      const pEnabled = s?.premiumEnabled === true; // fail-closed
      const caEnabled = s?.connectedAccountEnabled === true; // fail-closed

      setHasSubscription(active);
      setHasMtprotoAccount(account);
      setHasPremiumAccess(active || account);
      setPremiumEnabled(pEnabled);
      setConnectedAccountEnabled(caEnabled);
    });
    return () => { cancelled = true; };
  }, [authState]);

  const channelId = getChannelIdFromUrl();
  const isAdmin = isAdminDashRoute();
  const isRoot = isRootRoute();
  const isChannels = isChannelsRoute() || isRoot;
  const isSpecificChannel = !isAdmin && !isChannels && !!channelId;

  const handleBack = useCallback(() => {
    const source = sessionStorage.getItem('navSource');
    if (source === 'admin') {
      window.location.href = '/admin/dash';
    } else {
      window.location.href = '/me/channels';
    }
  }, []);

  useEffect(() => {
    const tg = window.Telegram?.WebApp;
    if (tg) {
      tg.ready();
      tg.expand();
      if (tg.initDataUnsafe?.user) setTgUser(tg.initDataUnsafe.user);

      if (isSpecificChannel) {
        tg.BackButton.show();
        tg.BackButton.onClick(handleBack);
      } else {
        tg.BackButton.hide();
      }
    }
    return () => {
      window.Telegram?.WebApp.BackButton.offClick(handleBack);
    };
  }, [isSpecificChannel, handleBack]);

  const handleBlacklist = useCallback(() => {
    const tg = window.Telegram?.WebApp;
    if (tg) {
      tg.showConfirm("Você está na blacklist e seu acesso ao painel foi bloqueado. Em caso de dúvidas, acione a /ouvidoria no bot.", () => {
        tg.close();
      });
    } else {
      alert("Você está na blacklist e seu acesso ao painel foi bloqueado.");
    }
  }, []);

  useEffect(() => {
    (async () => {
      setLoading(true);
      setAuthState('authenticating');

      const tg = window.Telegram?.WebApp;
      const initData = tg?.initData || '';
      const userID = tg?.initDataUnsafe?.user?.id || 0;

      try {
        if (isRootRoute()) {
          setAuthState('authenticated');
          setData({
            channel: null as any,
            user: {
              id: 0,
              first_name: 'Convidado',
              username: '',
              is_admin: false,
              is_blacklisted: false,
              isContribute: false,
              created_at: '',
              updated_at: '',
              channels: []
            }
          });
          return;
        }

        const authRes = await login(initData, userID);
        if (!authRes.success) throw new Error(authRes.message || 'Falha no login');

        if (authRes.data?.isBlacklisted) {
          handleBlacklist();
          return;
        }

        setAuthState('authenticated');

        // Re-loga uma única vez e refaz a chamada em caso de sessão expirada/401
        // (mantém autenticação cookie-only).
        const retryOn401 = async <T,>(fn: () => Promise<T>): Promise<T> => {
          try {
            return await fn();
          } catch (e: any) {
            if (e?.status !== 401) throw e;
            const tg = window.Telegram?.WebApp;
            const reInit = tg?.initData || '';
            const reUser = tg?.initDataUnsafe?.user?.id || 0;
            const re = await login(reInit, reUser);
            if (!re.success) throw e;
            return await fn();
          }
        };

        if (isAdminDashRoute()) {
          const response = await retryOn401(() => fetchAdminDashboard());
          setAdminData(response);
        } else if (isChannelsRoute()) {
          const response = await retryOn401(() => fetchUserChannels());
          const channelsData = Array.isArray(response) ? response : [];
          setData({
            channel: null as any,
            user: {
              id: userID,
              first_name: tg?.initDataUnsafe?.user?.first_name || 'Usuário',
              is_admin: false,
              is_blacklisted: false,
              isContribute: false,
              created_at: '',
              updated_at: '',
              channels: channelsData,
              username: tg?.initDataUnsafe?.user?.username || ''
            }
          });
        } else if (channelId) {
          const response = await retryOn401(() => fetchDashboardData(channelId));
          const dashRes = response;
          
          if (dashRes.user?.is_blacklisted) {
            handleBlacklist();
            return;
          }

          // Resolver conflito de posição entre botões e reações (se houver)
          if (dashRes.channel && dashRes.channel.buttons) {
            const { buttons, reactionPosition } = dashRes.channel;
            const hasConflict = buttons.some(b => b.positionY === reactionPosition);
            if (hasConflict) {
              const maxBtnY = buttons.reduce((max, b) => Math.max(max, b.positionY), -1);
              dashRes.channel.reactionPosition = maxBtnY + 1;
            }
          }

          setData(dashRes);
        }

        if (tg?.CloudStorage && initData) {
          tg.CloudStorage.setItem('initData', initData);
        }

      } catch (err: any) {
        console.warn('Auth/fetch failed, checking error:', err);
        const errMsg = err?.message || 'Erro na autenticação';

        // @ts-ignore
        if (import.meta.env.DEV) {
          const { fallbackData, mockAdminData } = await import('./mockData');
          setAuthState('authenticated');
          if (isAdminDashRoute()) setAdminData(mockAdminData);
          else if (isChannelsRoute()) setData({ channel: null as any, user: { ...fallbackData.user, channels: mockAdminData.channels } });
          else setData(fallbackData);
        } else {
          setAuthState('error');
          setAuthError(errMsg);
        }
      } finally {
        setLoading(false);
      }
    })();
  }, [channelId, handleBlacklist]);

  const handleMsgPerm = useCallback(async (field: string, value: boolean) => {
    if (!data?.channel?.defaultCaption) return;
    const cid = data.channel.id;
    
    setData(p => {
      if (!p?.channel?.defaultCaption) return p;
      return {
        ...p, channel: {
          ...p.channel, defaultCaption: {
            ...p.channel.defaultCaption,
            messagePermission: { ...(p.channel.defaultCaption.messagePermission || {}), [field]: value }
          }
        }
      };
    });

    try {
      const currentPerms = data.channel.defaultCaption.messagePermission || {};
      const newPerms = { ...currentPerms, [field]: value };
      await updateMessagePermission(cid, newPerms);
      toast(`${permLabels[field] || field} ${value ? 'ativado' : 'desativado'}`, value ? 'success' : 'info');
    } catch {
      setData(data); 
      toast(`Erro ao atualizar permissão`, 'error');
    }
  }, [toast, data]);

  const handleBtnPerm = useCallback(async (field: string, value: boolean) => {
    if (!data?.channel?.defaultCaption) return;
    const cid = data.channel.id;

    setData(p => {
      if (!p?.channel?.defaultCaption) return p;
      return {
        ...p, channel: {
          ...p.channel, defaultCaption: {
            ...p.channel.defaultCaption,
            buttonsPermission: { ...(p.channel.defaultCaption.buttonsPermission || {}), [field]: value }
          }
        }
      };
    });

    try {
      const currentPerms = data.channel.defaultCaption.buttonsPermission || {};
      const newPerms = { ...currentPerms, [field]: value };
      await updateButtonsPermission(cid, newPerms);
      toast(`${permLabels[field] || field} ${value ? 'ativado' : 'desativado'}`, value ? 'success' : 'info');
    } catch {
      setData(data);
      toast(`Erro ao atualizar permissão`, 'error');
    }
  }, [toast, data]);

  const handleDynamicLinks = useCallback(async (field: string, value: boolean) => {
    if (!data) return;
    const cid = parseInt(String(channelId), 10);

    const newSettings = {
      dynamicLinks: field === 'dynamicLinks' ? value : data.channel.dynamicLinks,
      dlBotButtons: field === 'dlBotButtons' ? value : data.channel.dlBotButtons,
      dlBotCaptions: field === 'dlBotCaptions' ? value : data.channel.dlBotCaptions,
      dlBotReactions: field === 'dlBotReactions' ? value : data.channel.dlBotReactions,
    };

    setData(p => {
      if (!p) return p;
      return { ...p, channel: { ...p.channel, ...newSettings } };
    });

    try {
      await updateDynamicLinks(cid, newSettings);
      const labels: Record<string, string> = {
        dynamicLinks: 'Links Dinâmicos',
        dlBotButtons: 'Botões do Bot',
        dlBotCaptions: 'Legendas do Bot',
        dlBotReactions: 'Reações do Bot'
      };
      toast(`${labels[field] || field} ${value ? 'ativados' : 'desativados'}`, value ? 'success' : 'info');
    } catch {
      setData(data);
      toast(`Erro ao atualizar configuração`, 'error');
    }
  }, [toast, channelId, data]);

  const handleAddButton = useCallback(async (button: Button) => {
    const cid = parseInt(String(channelId), 10);
    try {
      const resp = await createButton(cid, button);
      const newButtonData = resp?.data || resp?.Data || resp;
      const realId = newButtonData?.buttonId || newButtonData?.ButtonID || newButtonData?.id;

      if (!realId) throw new Error("ID not returned from API");

      const finalButton = { ...button, buttonId: realId };

      let nextButtons: Button[] = [];
      setData(p => {
        if (!p) return p;
        nextButtons = [...p.channel.buttons, finalButton];
        return { ...p, channel: { ...p.channel, buttons: nextButtons } };
      });
      toast(`"${button.nameButton}" adicionado`, 'success');

      if (nextButtons.length > 0) {
        const layout: any[][] = [];
        const maxRow = nextButtons.reduce((max, b) => Math.max(max, b.positionY), 0);
        for (let currentY = 0; currentY <= maxRow; currentY++) {
          const rowButtons = nextButtons
            .filter(b => b.positionY === currentY)
            .sort((a, b) => a.positionX - b.positionX)
            .map(b => ({
              buttonId: b.buttonId,
              nameButton: b.nameButton,
              buttonUrl: b.buttonUrl,
              positionX: b.positionX,
              positionY: b.positionY,
            }));
          layout.push(rowButtons);
        }
        updateLayoutButtons(cid, layout).catch(console.error);
      }
    } catch (err) {
      console.error(err);
      toast(`Erro ao adicionar botão`, 'error');
    }
  }, [toast, channelId]);

  const handleDeleteButton = useCallback(async (buttonId: string) => {
    const cid = parseInt(String(channelId), 10);
    try {
      await deleteButton(cid, buttonId);
      setData(p => {
        if (!p) return p;
        const btn = p.channel.buttons.find(b => b.buttonId === buttonId);
        if (btn) toast(`"${btn.nameButton}" excluído`, 'error');
        return { ...p, channel: { ...p.channel, buttons: p.channel.buttons.filter(b => b.buttonId !== buttonId) } };
      });
    } catch {
      toast(`Erro ao excluir botão`, 'error');
    }
  }, [toast, channelId]);

  const handleEditButton = useCallback(async (buttonId: string, updates: Partial<Button>) => {
    const cid = parseInt(String(channelId), 10);
    try {
      await updateButton(cid, buttonId, updates);
      setData(p => {
        if (!p) return p;
        return {
          ...p, channel: {
            ...p.channel, buttons: p.channel.buttons.map(b =>
              b.buttonId === buttonId ? { ...b, ...updates, updated_at: new Date().toISOString() } : b
            )
          }
        };
      });
      toast('Botão atualizado', 'success');
    } catch {
      toast('Erro ao atualizar botão', 'error');
    }
  }, [toast, channelId]);

  const handleMoveButton = useCallback(async (buttonId: string, x: number, y: number) => {
    const cid = parseInt(String(channelId), 10);
    if (!data) return;

    const updatedButtons = data.channel.buttons.map(b =>
      b.buttonId === buttonId ? { ...b, positionX: x, positionY: y, updated_at: new Date().toISOString() } : b
    );

    const currentReactionPos = data.channel.reactionPosition;
    let desiredReactionPos = currentReactionPos;
    
    const hasConflict = updatedButtons.some(b => b.positionY === currentReactionPos);
    if (hasConflict) {
      const maxBtnY = updatedButtons.reduce((max, b) => Math.max(max, b.positionY), -1);
      desiredReactionPos = maxBtnY + 1;
    }

    try {
      if (desiredReactionPos !== currentReactionPos) {
        await updateReactionPosition(cid, 99);
      }

      const layout: any[][] = [];
      const maxRow = Math.max(...updatedButtons.map(b => b.positionY));
      for (let currentY = 0; currentY <= maxRow; currentY++) {
        const rowButtons = updatedButtons
          .filter(b => b.positionY === currentY)
          .sort((a, b) => a.positionX - b.positionX)
          .map(b => ({
            buttonId: b.buttonId,
            nameButton: b.nameButton,
            buttonUrl: b.buttonUrl,
            positionX: b.positionX,
            positionY: b.positionY,
          }));
        layout.push(rowButtons);
      }

      await updateLayoutButtons(cid, layout);
      
      if (desiredReactionPos !== currentReactionPos) {
        await updateReactionPosition(cid, desiredReactionPos);
      }

      setData(p => {
        if (!p) return p;
        return {
          ...p, channel: {
            ...p.channel, 
            buttons: updatedButtons,
            reactionPosition: desiredReactionPos
          }
        };
      });
      toast('Botão movido com sucesso', 'info');
    } catch (err: any) {
      toast(err.message || 'Erro ao mover botão', 'error');
      fetchDashboardData(String(channelId)).then(setData);
    }
  }, [toast, channelId, data]);

  const handleMoveReactions = useCallback(async (y: number) => {
    const cid = parseInt(String(channelId), 10);
    if (!data) return;

    const conflictingButtons = data.channel.buttons.filter(b => b.positionY === y);
    if (conflictingButtons.length > 0) {
      const names = conflictingButtons.map(b => `"${b.nameButton}"`).join(', ');
      toast(`Não é possível mover: a linha ${y + 1} possui botões (${names})`, 'error');
      return;
    }

    try {
      await updateReactionPosition(cid, y);
      setData(p => {
        if (!p) return p;
        return { ...p, channel: { ...p.channel, reactionPosition: y } };
      });
      toast('Posição das reações atualizada', 'info');
    } catch (err: any) {
      toast(err.message || 'Erro ao mover reações', 'error');
    }
  }, [toast, channelId, data]);

  const handleUpdateCaption = useCallback(async (text: string) => {
    const cid = parseInt(String(channelId), 10);
    try {
      await updateDefaultCaption(cid, text);
      setData(p => {
        if (!p) return p;
        return { ...p, channel: { ...p.channel, defaultCaption: { ...p.channel.defaultCaption, caption: text } } };
      });
      toast('Caption atualizada', 'success');
    } catch {
      toast('Erro ao atualizar caption', 'error');
    }
  }, [toast, channelId]);

  const handleUpdateNewPack = useCallback(async (settings: {
    caption: string;
    messageButtons: boolean;
    stickerButtons: boolean;
    messagePosition: 'above' | 'below';
    replyToSticker: boolean;
  }) => {
    const cid = parseInt(String(channelId), 10);
    try {
      await updateNewPackCaption(cid, {
        newPackCaption: settings.caption,
        newPackMessageButtons: settings.messageButtons,
        newPackStickerButtons: settings.stickerButtons,
        newPackMessagePosition: settings.messagePosition,
        newPackReplyToSticker: settings.replyToSticker,
      });
      setData(p => {
        if (!p) return p;
        return {
          ...p,
          channel: {
            ...p.channel,
            newPackCaption: settings.caption,
            newPackMessageButtons: settings.messageButtons,
            newPackStickerButtons: settings.stickerButtons,
            newPackMessagePosition: settings.messagePosition,
            newPackReplyToSticker: settings.replyToSticker,
          }
        };
      });
      toast('New Pack atualizada', 'success');
    } catch {
      toast('Erro ao atualizar New Pack', 'error');
    }
  }, [toast, channelId]);

  const handleUpdateReactions = useCallback(async (text: string) => {
    const cid = parseInt(String(channelId), 10);
    if (!data) return;

    try {
      await updateReactions(cid, text);
      
      let newPos = data.channel.reactionPosition;
      const hasConflict = data.channel.buttons.some(b => b.positionY === newPos);
      
      if (text.trim() !== '' && (hasConflict || (newPos === 0 && data.channel.buttons.some(b => b.positionY === 0)))) {
        const maxBtnY = data.channel.buttons.reduce((max, b) => Math.max(max, b.positionY), -1);
        newPos = maxBtnY + 1;
        await updateReactionPosition(cid, newPos);
      }

      setData(p => {
        if (!p) return p;
        return { ...p, channel: { ...p.channel, reactions: text, reactionPosition: newPos } };
      });
      toast('Reações atualizadas', 'success');
    } catch (err: any) {
      toast(err.message || 'Erro ao atualizar reações', 'error');
    }
  }, [toast, channelId, data]);

  const getGreeting = useCallback(() => {
    const h = new Date().getHours();
    if (h < 12) return 'Bom dia';
    if (h < 18) return 'Boa tarde';
    return 'Boa noite';
  }, []);

  const getGreetingIcon = useCallback(() => {
    const h = new Date().getHours();
    if (h < 12) return <Sunrise size={24} />;
    if (h < 18) return <Sun size={24} />;
    return <CloudMoon size={24} />;
  }, []);

  const [showDisconnect, setShowDisconnect] = useState(false);
  const [showDisconnectSuccess, setShowDisconnectSuccess] = useState(false);

  const handleDisconnect = useCallback(() => {
    setShowDisconnect(true);
  }, []);

  const [isDisconnecting, setIsDisconnecting] = useState(false);

  const confirmDisconnect = useCallback(async () => {
    setIsDisconnecting(true);
    try {
      if (!channelId) throw new Error("ID do canal não encontrado");

      const cid = parseInt(String(channelId), 10);
      const res = await disconnectChannel(cid);

      if (res.status === 204) {
        setShowDisconnectSuccess(true);
      } else {
        const errText = await res.text().catch(() => '');
        throw new Error(errText || `Erro na API (${res.status})`);
      }
    } catch (err: any) {
      toast(err.message || 'Erro ao desconectar o bot', 'error');
    } finally {
      setIsDisconnecting(false);
      setShowDisconnect(false);
    }
  }, [channelId, toast]);

  const parseNoticeTargetIds = useCallback((raw: string) => {
    return raw
      .split(/[\s,;]+/)
      .map(v => v.trim())
      .filter(Boolean)
      .map(v => Number.parseInt(v, 10))
      .filter(v => Number.isFinite(v) && v !== 0);
  }, []);

  const handleSendNotice = useCallback(async () => {
    if (!noticeMessage.trim()) {
      toast('A mensagem não pode estar vazia', 'error');
      return;
    }

    const specificTarget = noticeTarget === 'single' || noticeTarget === 'user_ids' || noticeTarget === 'channel_ids';
    const targetIds = specificTarget ? parseNoticeTargetIds(noticeTargetId) : [];
    if (specificTarget && targetIds.length === 0) {
      toast('Informe pelo menos um ID válido', 'error');
      return;
    }

    setIsSendingNotice(true);
    try {
      const tg = window.Telegram?.WebApp;
      const initData = tg?.initData || '';

      const payload: NoticeRequest = {
        message: noticeMessage,
        imageUrl: noticeImageUrl,
        target: noticeTarget,
        targetId: noticeTarget === 'single' ? targetIds[0] : undefined,
        targetIds: noticeTarget === 'user_ids' || noticeTarget === 'channel_ids' ? targetIds : undefined,
        buttons: noticeButtons
      };

      await sendAdminNotice(initData, payload);
      toast('Broadcast iniciado. O envio será processado em segundo plano.', 'success');
      setNoticeMessage('');
      setNoticeImageUrl('');
      setNoticeTargetId('');
      setNoticeButtons([]);
    } catch (err: any) {
      toast(err.message || 'Erro ao enviar mensagem', 'error');
    } finally {
      setIsSendingNotice(false);
    }
  }, [noticeMessage, noticeImageUrl, noticeTarget, noticeTargetId, noticeButtons, parseNoticeTargetIds, toast]);

  const handleRunAudit = useCallback(async () => {
    setAuditLoading(true);
    try {
      const res = await fetchAuditCheckBot();
      if (res.success) {
        const auditData = Array.isArray(res.data) ? res.data : [];
        setAuditResults(auditData);
        if (auditData.length === 0) {
          toast("Varredura concluída: nenhum canal com @XavolaBot.", "success");
        } else {
          toast(`Auditoria concluída: ${auditData.length} usuários afetados`, "info");
        }
      } else {
        throw new Error(res.message || 'Erro na auditoria');
      }
    } catch (err: any) {
      toast(err.message || 'Erro ao realizar auditoria', 'error');
    } finally {
      setAuditLoading(false);
    }
  }, [toast]);

  const handleAddNoticeButton = useCallback(() => {
    setNoticeButtons(prev => [...prev, { text: '', type: 'url', value: '' }]);
  }, []);

  const updateNoticeButton = useCallback((index: number, field: keyof NoticeButton, value: string) => {
    setNoticeButtons(prev => {
      const newBtns = [...prev];
      newBtns[index] = { ...newBtns[index], [field]: value };
      return newBtns;
    });
  }, []);

  const removeNoticeButton = useCallback((index: number) => {
    setNoticeButtons(prev => prev.filter((_, i) => i !== index));
  }, []);

  const navigateToChannel = useCallback((id: number) => {
    if (isAdminDashRoute()) {
      sessionStorage.setItem('navSource', 'admin');
    } else {
      sessionStorage.removeItem('navSource');
    }
    window.location.href = `/dashboard/${id}`;
  }, []);

  const onSelectAdminUser = useCallback((id: number | null) => {
    setAdminSelectedUserId(id);
    if (id) sessionStorage.setItem('lastAdminUserId', id.toString());
    else sessionStorage.removeItem('lastAdminUserId');
  }, []);

  const openAdminUserDetail = useCallback((id: number) => {
    setAdminSelectedUserId(id);
    sessionStorage.setItem('lastAdminUserId', id.toString());
    setAdminActiveTab('users');
  }, []);

  const openSupportNoticeForUser = useCallback((id: number) => {
    setAdminSelectedUserId(id);
    sessionStorage.setItem('lastAdminUserId', id.toString());
    setNoticeTarget('single');
    setNoticeTargetId(id.toString());
    setAdminActiveTab('notice');
  }, []);

  useEffect(() => {
    // Tab switch effect handled via CSS entrance animations
  }, [activeTab, adminActiveTab, loading]);

  // derived state — must be defined before early returns to maintain hook order
  // (useMemo below is a hook and must run on every render per React's Rules of Hooks)
  const channel = data?.channel;

  // Dynamic tabs — premium tab only shows when premium is enabled AND user has subscription or MTProto account
  const userTabs: Tab[] = useMemo(() => {
    const list: Tab[] = [...BASE_TABS];
    // Filtrar aba "Conta" se a feature connected_account estiver desativada
    if (!connectedAccountEnabled) {
      const idx = list.findIndex(t => t.id === 'conta');
      if (idx !== -1) list.splice(idx, 1);
    }
    if (premiumEnabled && hasPremiumAccess && channel) {
      list.push({ id: 'premium', label: 'Premium', icon: <Crown size={22} /> });
    }
    return list;
  }, [connectedAccountEnabled, premiumEnabled, hasPremiumAccess, channel]);

  if (authState === 'error') {
    let displayMessage = authError || 'Não foi possível autenticar. Tente novamente pelo Telegram.';
    
    try {
      if (displayMessage.startsWith('{')) {
        const parsed = JSON.parse(displayMessage);
        displayMessage = parsed.message || parsed.error || displayMessage;
      }
    } catch (e) {}

    return (
      <div className="app-layout">
        <div className="main-content" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '60vh', gap: 16, textAlign: 'center' }}>
          <div style={{ width: 64, height: 64, borderRadius: 20, background: 'var(--danger-soft)', display: 'flex', alignItems: 'center', justifyContent: 'center', marginBottom: 8 }}>
            <AlertTriangle size={32} style={{ color: 'var(--danger)' }} />
          </div>
          <h2 style={{ fontSize: 20, fontWeight: 800 }}>Ops! Acesso negado</h2>
          <p style={{ fontSize: 15, color: 'var(--hint)', maxWidth: 320, lineHeight: 1.6 }}>{displayMessage}</p>
          <ShadButton variant="default" onClick={() => window.location.href = '/me/channels'} className="mt-3 min-w-[200px]">
            <ArrowLeft size={18} /> Voltar para Meus Canais
          </ShadButton>
        </div>
      </div>
    );
  }

  if (loading || (!data && !adminData)) {
    return (
      <div className="app-layout">
        <div className="main-content space-y-4" style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minHeight: '60vh', gap: 12 }}>
          <div className="auth-spinner" />
          <p style={{ fontSize: 14, color: 'var(--hint)', fontWeight: 500 }}>
            {authState === 'authenticating' ? 'Autenticando...' : 'Carregando dados...'}
          </p>
        </div>
      </div>
    );
  }

  const user = data?.user;
  const displayName = tgUser?.first_name || user?.first_name || 'Administrador';
  const initials = displayName[0]?.toUpperCase() || '?';

  if (!isAdmin && showTemplatesModal) {
    return (
      <div className="app-layout channel-dashboard templates-page flex flex-col min-h-dvh bg-background">
        <div className="sticky top-0 z-50 flex items-center gap-3 px-4 py-3 bg-card/90 backdrop-blur-md border-b border-border/80 shrink-0">
          <ShadButton variant="ghost" size="icon" onClick={() => setShowTemplatesModal(false)} className="-ml-1 text-muted-foreground hover:text-foreground">
            <ArrowLeft size={20} />
          </ShadButton>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <Layers className="size-4 text-purple-400" />
              <h1 className="text-base font-bold text-foreground leading-none">Meus Templates</h1>
            </div>
            <p className="text-xs text-muted-foreground mt-1 truncate">
              Crie e gerencie seus templates de legendas e botões.
            </p>
          </div>
        </div>
        <div className="main-content flex-1 p-4 overflow-y-auto pb-10">
          <UserTemplatesManager toast={toast} />
        </div>
      </div>
    );
  }

  if (!isAdmin && showSchedulesModal) {
    return (
      <div className="app-layout channel-dashboard schedule-page flex flex-col min-h-dvh bg-background">
        <div className="sticky top-0 z-50 flex items-center gap-3 px-4 py-3 bg-card/90 backdrop-blur-md border-b border-border/80 shrink-0">
          <ShadButton variant="ghost" size="icon" onClick={() => setShowSchedulesModal(false)} className="-ml-1 text-muted-foreground hover:text-foreground">
            <ArrowLeft size={20} />
          </ShadButton>
          <div className="min-w-0 flex-1">
            <h1 className="text-base font-bold text-foreground leading-tight">Todos os Agendamentos</h1>
            <p className="text-xs text-muted-foreground truncate">Visualize e gerencie todos os agendamentos de postagens.</p>
          </div>
        </div>
        <div className="main-content flex-1 p-4 overflow-y-auto pb-24">
          <ScheduleTab />
        </div>
      </div>
    );
  }

  // ── Admin: use new CRM layout ──
  if (isAdmin && adminData) {
    return (
      <AdminLayout
        activeTab={adminActiveTab}
        onTabChange={(id) => setAdminActiveTab(id)}
        adminName={displayName}
        adminAvatar={tgUser?.photo_url}
        users={adminData.users || []}
        channels={adminData.channels || []}
      >
        <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando painel administrativo...</div>}>
          <MemoizedAdminDashboard
            adminData={adminData}
            activeTab={adminActiveTab}
            navigateToChannel={navigateToChannel}
            selectedUserId={adminSelectedUserId}
            onSelectUser={onSelectAdminUser}
            onOpenUserDetail={openAdminUserDetail}
            onMessageUser={openSupportNoticeForUser}
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
            auditResults={auditResults}
            setAuditResults={setAuditResults}
            auditLoading={auditLoading}
            handleRunAudit={handleRunAudit}
            initialLogsChannelId={initialLogsChannelId}
            toast={toast}
          />
        </Suspense>
      </AdminLayout>
    );
  }

  return (
    <div className={`app-layout ${isAdmin ? 'admin-layout' : ''} ${!isChannels && !isAdmin ? 'channel-dashboard' : ''}`}>
      <div className={isAdmin ? 'app-main' : 'w-full flex flex-col min-h-screen'}>
        <div className="top-bar animate-stagger-in flex items-center justify-between">
          <div className="flex items-center">
            <button
              className="sidebar-trigger-btn mr-2"
              onClick={() => setShowSideMenu(true)}
              title="Abrir Menu"
            >
              <Menu size={22} />
            </button>
          </div>

          {(sessionStorage.getItem('navSource') === 'admin' || data?.user?.is_admin) && (
            <button
              type="button"
              onClick={() => {
                sessionStorage.removeItem('navSource');
                window.location.href = '/admin/dash?tab=channels';
              }}
              className="flex items-center gap-1.5 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors py-1.5 px-2.5 rounded-lg hover:bg-white/10 cursor-pointer ml-auto"
              title="Voltar ao Painel Admin"
            >
              <ArrowLeft size={14} />
              <span>Painel Admin</span>
            </button>
          )}
        </div>

        <div className="main-content">
          {isAdmin && adminData && (
            <div className="tab-content-wrapper">
              <MemoizedAdminDashboard
                adminData={adminData}
                activeTab={adminActiveTab}
                navigateToChannel={navigateToChannel}
                selectedUserId={adminSelectedUserId}
                onSelectUser={onSelectAdminUser}
                onOpenUserDetail={openAdminUserDetail}
                onMessageUser={openSupportNoticeForUser}
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
                auditResults={auditResults}
                setAuditResults={setAuditResults}
                auditLoading={auditLoading}
                handleRunAudit={handleRunAudit}
                initialLogsChannelId={initialLogsChannelId}
                toast={toast}
              />
            </div>
          )}

          {(!isAdmin && (!channel || isChannelsRoute())) && (
            <div className="space-y-3">
              {isChannels && (
                <>
                  <div className="action-card animate-stagger-in">
                    <div className="action-card-icon">
                      {getGreetingIcon()}
                    </div>
                    <div className="action-card-body">
                      <div className="action-card-title text-[15px] font-bold">{getGreeting()}, <span style={{ color: 'var(--accent)' }}>{displayName}</span></div>
                      <div className="action-card-desc">Selecione um canal para gerenciar suas configurações.</div>
                    </div>
                  </div>

                  {connectedAccountEnabled && (
                    <div className="action-card animate-stagger-in" onClick={() => setShowContaModal(true)}>
                      <div className="action-card-icon">
                        <UserCheck size={22} />
                      </div>
                      <div className="action-card-body">
                        <div className="action-card-title">Conta Telegram</div>
                        <div className="action-card-desc">Conecte sua conta pessoal para recursos exclusivos</div>
                      </div>
                      <ChevronRight size={18} className="action-card-chevron" />
                    </div>
                  )}

                  {premiumEnabled && (
                    <div className="animate-stagger-in">
                      <PremiumTab toast={toast} channels={user?.channels || undefined} />
                    </div>
                  )}

                  <div className="action-card animate-stagger-in" onClick={() => setShowTemplatesModal(true)}>
                    <div className="action-card-icon">
                      <FileText size={22} />
                    </div>
                    <div className="action-card-body">
                      <div className="action-card-title">Meus Templates</div>
                      <div className="action-card-desc">Gerencie legendas, botões e templates do Post Builder</div>
                    </div>
                    <ChevronRight size={18} className="action-card-chevron" />
                  </div>

                  <div className="action-card animate-stagger-in" onClick={() => setShowSchedulesModal(true)}>
                    <div className="action-card-icon">
                      <Calendar size={22} />
                    </div>
                    <div className="action-card-body">
                      <div className="action-card-title">Agendamentos</div>
                      <div className="action-card-desc">Visualize e gerencie os agendamentos de todos os seus canais</div>
                    </div>
                    <ChevronRight size={18} className="action-card-chevron" />
                  </div>
                </>
              )}

              <div className="space-y-2">
                {isChannels && (
                  <div className="section-heading">
                    <span className="section-heading-text">Canais Encontrados</span>
                    <span className="section-heading-line" />
                  </div>
                )}

                {user?.channels && user?.channels.length > 0 && isChannels ? (
                  user?.channels.map((c: Channel) => (
                    <div
                      key={c.id}
                      className="action-card animate-stagger-in"
                      onClick={() => navigateToChannel(c.id)}
                    >
                      <div className="action-card-icon" style={{ width: 40, height: 40 }}>
                        <Hash size={18} />
                      </div>
                      <div className="action-card-body">
                        <div className="action-card-title truncate">{c.title}</div>
                        <div className="action-card-desc">ID: {c.id}</div>
                      </div>
                      <ChevronRight size={16} className="action-card-chevron" />
                    </div>
                  ))
                ) : (!channel ? (
                  <div className="content-card py-8">
                    <div className="flex flex-col items-center text-center px-4">
                      <div className="flex items-center justify-center size-16 rounded-2xl mb-5" style={{ background: 'var(--accent-soft)' }}>
                        <Shield size={32} style={{ color: 'var(--accent)' }} />
                      </div>
                      <h3 className="text-lg font-bold mb-2">Mantenha seu canal organizado</h3>
                      <p className="text-sm text-muted-foreground mb-2 leading-relaxed max-w-sm">
                        O LegendasBOT ajuda a gerenciar botões, legendas automáticas e permissões de forma simples e rápida.
                      </p>
                      <p className="text-sm text-muted-foreground/70 mb-6 max-w-sm">
                        Para começar, adicione este bot como <strong className="text-foreground">administrador</strong> no seu canal do Telegram.
                      </p>
                      <div className="w-full h-px bg-border mb-6" />
                      <h4 className="text-sm font-semibold mb-3">Fique por dentro das novidades!</h4>
                      <p className="text-sm text-muted-foreground mb-4">
                        Entre no nosso canal oficial para acompanhar atualizações, dicas e novos recursos.
                      </p>
                      <ShadButton
                        variant="default"
                        className="w-full"
                        onClick={() => window.open('https://t.me/LegendasBOTTopic', '_blank', 'noopener,noreferrer')}
                      >
                        <span className="inline-flex items-center justify-center gap-2 text-[15px] font-semibold">
                          <ExternalLink size={20} className="shrink-0" />
                          Entrar no Canal de Atualizações
                        </span>
                      </ShadButton>
                    </div>
                  </div>
                ) : null)}
              </div>

            </div>
          )}

          {!isChannels && !isAdmin && activeTab === 'geral' && channel && (
            <div className="tab-content-wrapper">
              <DashboardInicioTab
                channel={channel}
                getGreeting={getGreeting}
                getGreetingIcon={getGreetingIcon}
                handleDisconnect={handleDisconnect}
                showDisconnect={showDisconnect}
                setShowDisconnect={setShowDisconnect}
                isDisconnecting={isDisconnecting}
                confirmDisconnect={confirmDisconnect}
                showDisconnectSuccess={showDisconnectSuccess}
                hasPremium={hasPremiumAccess}
              />
            </div>
          )}

          {!isChannels && !isAdmin && activeTab === 'legendas' && channel && (
            <div className="space-y-5 tab-content-wrapper">
              <CaptionCard caption={channel.defaultCaption} onUpdate={handleUpdateCaption} />
              <NewPackCaptionCard
                caption={channel.newPackCaption}
                messageButtons={channel.newPackMessageButtons ?? true}
                stickerButtons={channel.newPackStickerButtons ?? true}
                messagePosition={channel.newPackMessagePosition ?? 'above'}
                replyToSticker={channel.newPackReplyToSticker ?? false}
                onUpdate={handleUpdateNewPack}
              />
              <ReactionsCard reactions={channel.reactions} onUpdate={handleUpdateReactions} />

              <div className="flex items-center gap-3 my-6 pt-2">
                <div className="h-px bg-border/60 flex-1" />
                <span className="text-[10px] font-bold tracking-widest text-muted-foreground/80 uppercase">OUTRAS OPÇÕES</span>
                <div className="h-px bg-border/60 flex-1" />
              </div>

              <NativeReactionsCard
                channelId={channel.id}
                enabled={channel.nativeReactionsEnabled ?? false}
                emojis={channel.nativeReactions ?? ''}
                mode={channel.nativeReactionMode ?? 'fixed'}
                toast={toast}
              />
            </div>
          )}

          {!isChannels && !isAdmin && activeTab === 'botoes' && channel && (
            <ButtonGrid
              buttons={channel.buttons}
              reactions={channel.reactions}
              reactionPosition={channel.reactionPosition}
              channelId={channel.id}
              onAdd={handleAddButton}
              onDelete={handleDeleteButton}
              onEdit={handleEditButton}
              onMove={handleMoveButton}
              onMoveReactions={handleMoveReactions}
            />
          )}

          {!isChannels && !isAdmin && activeTab === 'conta' && (
            <div className="tab-content-wrapper">
              <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando conta...</div>}>
                <ContaTelegramTab />
              </Suspense>
            </div>
          )}

          {!isChannels && !isAdmin && activeTab === 'premium' && channel && premiumEnabled && (
            <div className="tab-content-wrapper">
              <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando premium...</div>}>
                <PremiumConfigTab
                  channelId={channel.id}
                  caption={channel.defaultCaption?.caption || ''}
                  onUpdateCaption={handleUpdateCaption}
                  toast={toast}
                  hasSubscription={hasSubscription}
                  hasAccount={hasMtprotoAccount}
                />
              </Suspense>
            </div>
          )}

          {!isChannels && !isAdmin && activeTab === 'agendamentos' && channel && (
            <div className="tab-content-wrapper">
              <Suspense fallback={<div className="p-8 text-center text-xs text-muted-foreground animate-pulse">Carregando agendamentos...</div>}>
                <ScheduleTab channelId={channel.id} />
              </Suspense>
            </div>
          )}

          {!isChannels && !isAdmin && activeTab === 'permissoes' && channel && (
            <div className="space-y-2 tab-content-wrapper">
              {/* Configurações de Reações */}
              <div className="content-card">
                <div className="content-card-header">
                  <div className="content-card-icon"><Zap size={18} /></div>
                  <div className="flex-1 min-w-0">
                    <div className="content-card-title">Configurações de Reações</div>
                    <div className="content-card-desc">
                      {channel.defaultCaption?.messagePermission?.reactions ? 'Ativadas' : 'Desativadas'}
                    </div>
                  </div>
                  <Badge variant={channel.defaultCaption?.messagePermission?.reactions ? "default" : "secondary"}>
                    {channel.defaultCaption?.messagePermission?.reactions ? 'ON' : 'OFF'}
                  </Badge>
                </div>
                <div className="flex items-center justify-between rounded-xl px-4 py-3 bg-muted/50 cursor-pointer transition-colors hover:bg-muted/80" onClick={() => handleMsgPerm('reactions', !channel.defaultCaption?.messagePermission?.reactions)}>
                  <div className="flex items-center gap-3">
                    <Zap size={16} className={channel.defaultCaption?.messagePermission?.reactions ? 'text-accent' : 'text-muted-foreground'} />
                    <span className="text-sm font-medium">Ativar Reações em Posts</span>
                  </div>
                  <Switch checked={!!channel.defaultCaption?.messagePermission?.reactions} onCheckedChange={() => handleMsgPerm('reactions', !channel.defaultCaption?.messagePermission?.reactions)} onClick={(e: React.MouseEvent) => e.stopPropagation()} />
                </div>
              </div>

              <PerfLine accent />

              {/* Links Dinâmicos */}
              <div className="content-card">
                <div className="content-card-header">
                  <div className="content-card-icon"><Link2 size={18} /></div>
                  <div className="flex-1 min-w-0">
                    <div className="content-card-title">Links Dinâmicos</div>
                    <div className="content-card-desc">Transforma links em botões automaticamente</div>
                  </div>
                  <Badge variant={channel.dynamicLinks ? "default" : "secondary"}>
                    {channel.dynamicLinks ? 'ON' : 'OFF'}
                  </Badge>
                </div>
                <div className="flex items-center justify-between rounded-xl px-4 py-3 bg-muted/50 cursor-pointer transition-colors hover:bg-muted/80" onClick={() => handleDynamicLinks('dynamicLinks', !channel.dynamicLinks)}>
                  <div className="flex items-center gap-3">
                    <ExternalLink size={16} className={channel.dynamicLinks ? 'text-accent' : 'text-muted-foreground'} />
                    <span className="text-sm font-medium">Ativar Links Dinâmicos</span>
                  </div>
                  <Switch checked={!!channel.dynamicLinks} onCheckedChange={() => handleDynamicLinks('dynamicLinks', !channel.dynamicLinks)} onClick={(e: React.MouseEvent) => e.stopPropagation()} />
                </div>

                {channel.dynamicLinks && (
                  <div className="pl-6 space-y-2 mt-3 ml-4">
                    <div className="text-[10px] font-bold text-muted-foreground/40 uppercase mb-2 tracking-wider">Regras de Exceção</div>
                    <div className="flex items-center justify-between rounded-xl px-4 py-3 bg-muted/50 cursor-pointer transition-colors hover:bg-muted/80" onClick={() => handleDynamicLinks('dlBotButtons', !channel.dlBotButtons)}>
                      <div className="flex items-center gap-3">
                        <MousePointerClick size={14} className={channel.dlBotButtons ? 'text-accent' : 'text-muted-foreground'} />
                        <span className="text-xs font-medium">Manter Botões do Bot</span>
                      </div>
                      <Switch checked={!!channel.dlBotButtons} onCheckedChange={() => handleDynamicLinks('dlBotButtons', !channel.dlBotButtons)} onClick={(e: React.MouseEvent) => e.stopPropagation()} />
                    </div>
                    <div className="flex items-center justify-between rounded-xl px-4 py-3 bg-muted/50 cursor-pointer transition-colors hover:bg-muted/80" onClick={() => handleDynamicLinks('dlBotCaptions', !channel.dlBotCaptions)}>
                      <div className="flex items-center gap-3">
                        <Type size={14} className={channel.dlBotCaptions ? 'text-accent' : 'text-muted-foreground'} />
                        <span className="text-xs font-medium">Manter Legendas do Bot</span>
                      </div>
                      <Switch checked={!!channel.dlBotCaptions} onCheckedChange={() => handleDynamicLinks('dlBotCaptions', !channel.dlBotCaptions)} onClick={(e: React.MouseEvent) => e.stopPropagation()} />
                    </div>
                    <div className="flex items-center justify-between rounded-xl px-4 py-3 bg-muted/50 cursor-pointer transition-colors hover:bg-muted/80" onClick={() => handleDynamicLinks('dlBotReactions', !channel.dlBotReactions)}>
                      <div className="flex items-center gap-3">
                        <Zap size={14} className={channel.dlBotReactions ? 'text-accent' : 'text-muted-foreground'} />
                        <span className="text-xs font-medium">Manter Reações do Bot</span>
                      </div>
                      <Switch checked={!!channel.dlBotReactions} onCheckedChange={() => handleDynamicLinks('dlBotReactions', !channel.dlBotReactions)} onClick={(e: React.MouseEvent) => e.stopPropagation()} />
                    </div>
                    <p className="text-[10px] text-muted-foreground/40 italic mt-2">
                      * Estas regras só se aplicam se um link dinâmico for detectado na postagem.
                    </p>
                  </div>
                )}
              </div>

              <PerfLine accent />

              {/* Permissões por Tipo — combinado: legenda + botões */}
              <div className="content-card">
                <div className="content-card-header">
                  <div className="content-card-icon"><SlidersHorizontal size={18} /></div>
                  <div className="flex-1 min-w-0">
                    <div className="content-card-title">Permissões por Tipo</div>
                    <div className="content-card-desc">Ative legendas e botões para cada tipo de conteúdo</div>
                  </div>
                </div>

                <div className="space-y-2">
                  {([
                    { key: 'message', label: 'Mensagem', desc: 'Texto simples', icon: <MessageCircle size={16} /> },
                    { key: 'audio', label: 'Áudio', desc: 'Arquivos de áudio', icon: <Headphones size={16} /> },
                    { key: 'video', label: 'Vídeo', desc: 'Arquivos de vídeo', icon: <Video size={16} /> },
                    { key: 'photo', label: 'Foto', desc: 'Imagens e fotos', icon: <Image size={16} /> },
                    { key: 'document', label: 'Arquivo', desc: 'Documentos em geral', icon: <FileText size={16} /> },
                    { key: 'sticker', label: 'Sticker', desc: 'Figurinhas', icon: <Smile size={16} /> },
                    { key: 'gif', label: 'GIF', desc: 'Animações', icon: <Film size={16} /> },
                    { key: 'linkPreview', label: 'Link Preview', desc: 'Visualização de links', icon: <Link2 size={16} /> },
                  ] as const).map((type) => {
                    const msgOn = !!channel.defaultCaption?.messagePermission?.[type.key as keyof typeof channel.defaultCaption.messagePermission];
                    const btnOn = !!channel.defaultCaption?.buttonsPermission?.[type.key as keyof typeof channel.defaultCaption.buttonsPermission];
                    return (
                      <div
                        key={type.key}
                        className="p-3 rounded-xl bg-muted/50 border-none animate-stagger-in"
                      >
                        <div className="flex items-center gap-3 mb-3">
                          <span className="shrink-0 text-accent/70">{type.icon}</span>
                          <div className="min-w-0">
                            <span className="text-[13px] font-semibold">{type.label}</span>
                            <p className="text-[10px] text-muted-foreground/70 truncate">{type.desc}</p>
                          </div>
                        </div>

                        <div className="flex items-center gap-2 ml-9">
                          <div
                            className={`flex items-center gap-1.5 px-3 py-2 rounded-lg cursor-pointer transition-colors flex-1 ${msgOn ? 'bg-accent/10' : 'bg-muted hover:bg-muted/80'}`}
                            onClick={() => handleMsgPerm(type.key, !msgOn)}
                          >
                            <span className={`text-[11px] font-semibold ${msgOn ? 'text-accent' : 'text-muted-foreground'}`}>Legenda</span>
                            <Switch
                              size="sm"
                              checked={msgOn}
                              onCheckedChange={(checked) => handleMsgPerm(type.key, checked)}
                              onClick={(e: React.MouseEvent) => e.stopPropagation()}
                            />
                          </div>

                          <div
                            className={`flex items-center gap-1.5 px-3 py-2 rounded-lg cursor-pointer transition-colors flex-1 ${btnOn ? 'bg-accent/10' : 'bg-muted hover:bg-muted/80'}`}
                            onClick={() => handleBtnPerm(type.key, !btnOn)}
                          >
                            <span className={`text-[11px] font-semibold ${btnOn ? 'text-accent' : 'text-muted-foreground'}`}>Botões</span>
                            <Switch
                              size="sm"
                              checked={btnOn}
                              onCheckedChange={(checked) => handleBtnPerm(type.key, checked)}
                              onClick={(e: React.MouseEvent) => e.stopPropagation()}
                            />
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
      
      {!isChannels && !isAdmin && (
        <TabBar tabs={userTabs} activeTab={activeTab} onTabChange={setActiveTab} />
      )}

      {connectedAccountEnabled && (
        <Dialog open={showContaModal} onOpenChange={(open) => setShowContaModal(open)}>
          <DialogContent
            className="sm:max-w-lg p-0 bg-background"
            style={{ maxHeight: '90vh', overflowY: 'auto' }}
            showCloseButton={true}
          >
            <div className="p-5">
              <ContaTelegramTab startConnecting />
            </div>
          </DialogContent>
        </Dialog>
      )}
      <SideMenu
        isOpen={showSideMenu}
        onClose={() => setShowSideMenu(false)}
        tgUser={tgUser}
        displayName={displayName}
        userId={tgUser?.id || user?.id || 0}
        channelsCount={user?.channels?.length || 0}
        onOpenTemplates={() => setShowTemplatesModal(true)}
        onOpenConta={() => setShowContaModal(true)}
        onOpenSchedules={() => setShowSchedulesModal(true)}
        onNavigateChannels={() => { window.location.href = '/me/channels'; }}
        theme={theme}
        toggleTheme={toggleTheme}
        connectedAccountEnabled={connectedAccountEnabled}
      />
    </div>
  );
});

export default function App() {
  return (
    <ToastProvider>
      <ErrorBoundary>
        <DashboardContent />
      </ErrorBoundary>
    </ToastProvider>
  );
}
