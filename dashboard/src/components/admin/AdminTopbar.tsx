import { useEffect, useRef, useState, memo } from 'react';
import { Menu, Search, User, ArrowUpDown, SlidersHorizontal, Plus, ArrowLeft } from 'lucide-react';
import { AdminTabId } from '../../App';
import { AdminCrmFilter, AdminCrmSort, useAdminCrmControls } from './AdminCrmContext';
import { Channel, User as UserData } from '../../types';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';

interface AdminTopbarProps {
  activeTab: AdminTabId;
  onMenuToggle: () => void;
  onNavigate: (tab: AdminTabId) => void;
  canGoBack?: boolean;
  onGoBack?: () => void;
  adminName?: string;
  adminAvatar?: string;
  users: UserData[];
  channels: Channel[];
}

export const AdminTopbar = memo(function AdminTopbar({
  activeTab, onMenuToggle, onNavigate, canGoBack, onGoBack, adminName, adminAvatar
}: AdminTopbarProps) {
  const { searchQuery, setSearchQuery, sortBy, setSortBy, filterBy, setFilterBy } = useAdminCrmControls();
  const [searchFocused, setSearchFocused] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);
  const supportsCustomerControls = activeTab === 'overview' || activeTab === 'users';

  useEffect(() => {
    const handleShortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        searchRef.current?.focus();
      }
    };
    window.addEventListener('keydown', handleShortcut);
    return () => window.removeEventListener('keydown', handleShortcut);
  }, []);

  return (
    <header className="admin-topbar">
      <div className="admin-topbar-left flex items-center gap-2">
        <Button variant="ghost" size="icon" type="button" className="admin-topbar-menu-btn" onClick={onMenuToggle} aria-label="Abrir menu administrativo">
          <Menu size={18} />
        </Button>

        {canGoBack && onGoBack && (
          <Button
            variant="outline"
            size="sm"
            type="button"
            onClick={onGoBack}
            className="rounded-xl h-9 px-2.5 text-xs font-semibold text-foreground border-border flex items-center gap-1.5 cursor-pointer hover:bg-muted/50 shrink-0"
            title="Voltar para a aba anterior"
          >
            <ArrowLeft size={15} />
            <span className="hidden sm:inline">Voltar</span>
          </Button>
        )}

        <div className="relative w-64 max-w-[65vw]">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none" aria-hidden="true" />
          <Input
            ref={searchRef}
            type="text"
            placeholder={activeTab === 'channels' ? 'Buscar canal...' : activeTab === 'users' ? 'Buscar usuário...' : 'Buscar no sistema...'}
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="pl-9 h-9 text-xs rounded-xl bg-card border-border focus-visible:ring-1 focus-visible:ring-accent shadow-xs"
            aria-label="Buscar no sistema"
          />
        </div>
      </div>

      <div className="admin-topbar-right">
        <div className="admin-topbar-profile" aria-label={`Administrador ${adminName || 'Admin'}`} title={adminName || 'Admin'}>
          {adminAvatar ? (
            <img src={adminAvatar} alt="" className="admin-topbar-avatar-img" referrerPolicy="no-referrer" />
          ) : (
            <User size={15} aria-hidden="true" />
          )}
          <span>Eu</span>
        </div>

        <Button type="button" className="admin-topbar-primary" onClick={() => onNavigate('notice')}>
          <Plus size={15} aria-hidden="true" />
          <span>Novo broadcast</span>
        </Button>
      </div>
    </header>
  );
});
