import { memo, ReactNode } from 'react';
import {
  LayoutDashboard, Users, Radio, Megaphone, ScrollText,
  Settings, ShieldCheck, Sparkles, CreditCard, KeyRound, ChevronLeft,
  ChevronRight
} from 'lucide-react';
import { AdminTabId } from '../../App';

interface SidebarItem {
  id: AdminTabId;
  label: string;
  icon: ReactNode;
  section?: string;
}

const SIDEBAR_ITEMS: SidebarItem[] = [
  { id: 'overview', label: 'Visão Geral', icon: <LayoutDashboard size={18} />, section: 'Principal' },
  { id: 'users', label: 'Usuários', icon: <Users size={18} />, section: 'Principal' },
  { id: 'channels', label: 'Canais', icon: <Radio size={18} />, section: 'Principal' },
  { id: 'notice', label: 'Broadcast', icon: <Megaphone size={18} />, section: 'Operações' },
  { id: 'audit', label: 'Auditoria', icon: <ShieldCheck size={18} />, section: 'Operações' },
  { id: 'logs', label: 'Logs', icon: <ScrollText size={18} />, section: 'Operações' },
  { id: 'accounts', label: 'Contas MTProto', icon: <KeyRound size={18} />, section: 'Premium' },
  { id: 'premium-features', label: 'Features', icon: <Sparkles size={18} />, section: 'Premium' },
  { id: 'subscriptions', label: 'Assinaturas', icon: <CreditCard size={18} />, section: 'Premium' },
  { id: 'config', label: 'Configurações', icon: <Settings size={18} />, section: 'Sistema' },
];

interface AdminSidebarProps {
  activeTab: AdminTabId;
  onTabChange: (tab: AdminTabId) => void;
  isOpen: boolean;
  onToggle: () => void;
  mobileOpen: boolean;
  adminName?: string;
  adminAvatar?: string;
}

export const AdminSidebar = memo(function AdminSidebar({
  activeTab, onTabChange, isOpen, onToggle, mobileOpen, adminName, adminAvatar
}: AdminSidebarProps) {
  const initials = (adminName || 'Admin')[0]?.toUpperCase() || 'A';

  // Group items by section
  const sections = SIDEBAR_ITEMS.reduce((acc, item) => {
    const section = item.section || 'Outros';
    if (!acc[section]) acc[section] = [];
    acc[section].push(item);
    return acc;
  }, {} as Record<string, SidebarItem[]>);

  return (
    <aside className={`admin-sidebar ${isOpen ? 'open' : 'collapsed'} ${mobileOpen ? 'mobile-open' : ''}`}>
      {/* Profile Header on Top (identical to SideMenu.tsx) */}
      <div className="sidebar-profile-header flex items-center gap-3 p-4 border-b border-border/20">
        <div className="w-10 h-10 rounded-full overflow-hidden shrink-0 bg-accent text-accent-foreground flex items-center justify-center font-bold text-base shadow-sm">
          {adminAvatar ? (
            <img src={adminAvatar} alt="" className="w-full h-full object-cover" />
          ) : (
            initials
          )}
        </div>
        {isOpen && (
          <div className="min-w-0 flex-1 flex flex-col justify-center">
            <h3 className="font-bold text-sm text-foreground truncate leading-tight">{adminName || 'Administrador'}</h3>
            <p className="text-[11px] text-muted-foreground truncate mt-0.5">Painel Admin</p>
          </div>
        )}
      </div>

      {/* Navigation Sections (Grouped in soft rounded cards like SideMenu.tsx) */}
      <nav className="sidebar-nav p-3 space-y-3 overflow-y-auto flex-1">
        {Object.entries(sections).map(([sectionName, items]) => (
          <div key={sectionName} className="sidebar-section">
            {isOpen && (
              <div className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground/80 px-3 mb-1.5">
                {sectionName}
              </div>
            )}
            <div className="rounded-2xl bg-white/5 dark:bg-white/5 overflow-hidden border border-white/10 dark:border-white/5">
              {items.map((item, idx) => (
                <div key={item.id} className="relative">
                  <button
                    type="button"
                    onClick={() => onTabChange(item.id)}
                    className={`w-full flex items-center gap-3 px-3.5 py-3 text-xs font-semibold transition-colors text-left ${
                      activeTab === item.id
                        ? 'bg-accent/15 text-accent font-bold'
                        : 'text-foreground/80 hover:bg-white/10 hover:text-foreground'
                    }`}
                    title={!isOpen ? item.label : undefined}
                    aria-current={activeTab === item.id ? 'page' : undefined}
                  >
                    <span className="shrink-0 text-foreground/80">{item.icon}</span>
                    {isOpen && <span className="truncate">{item.label}</span>}
                  </button>
                  {idx < items.length - 1 && (
                    <div className="absolute bottom-0 left-10 right-3 h-px bg-white/10 pointer-events-none" />
                  )}
                </div>
              ))}
            </div>
          </div>
        ))}
      </nav>

      {/* Collapse Toggle Button */}
      <div className="p-3 border-t border-border/20">
        <button
          type="button"
          className="w-full flex items-center justify-center py-2.5 rounded-2xl bg-white/5 hover:bg-white/10 text-xs font-semibold text-foreground transition-colors"
          onClick={onToggle}
          aria-label={isOpen ? 'Recolher menu' : 'Expandir menu'}
        >
          {isOpen ? (
            <div className="flex items-center gap-2">
              <ChevronLeft size={16} />
              <span>Recolher menu</span>
            </div>
          ) : (
            <ChevronRight size={16} />
          )}
        </button>
      </div>
    </aside>
  );
});
