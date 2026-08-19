import { createContext, ReactNode, useCallback, useContext, useMemo, useState } from 'react';
import { AdminTabId } from '../../App';

export type AdminCrmSort = 'recent' | 'name' | 'channels';
export type AdminCrmFilter = 'all' | 'admins' | 'blacklisted' | 'with-channels' | 'without-channels';

interface AdminCrmContextValue {
  searchQuery: string;
  setSearchQuery: (value: string) => void;
  sortBy: AdminCrmSort;
  setSortBy: (value: AdminCrmSort) => void;
  filterBy: AdminCrmFilter;
  setFilterBy: (value: AdminCrmFilter) => void;
  navigateToTab: (tab: AdminTabId) => void;
}

const AdminCrmContext = createContext<AdminCrmContextValue | null>(null);

interface AdminCrmProviderProps {
  children: ReactNode;
  onNavigate: (tab: AdminTabId) => void;
}

export function AdminCrmProvider({ children, onNavigate }: AdminCrmProviderProps) {
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState<AdminCrmSort>('recent');
  const [filterBy, setFilterBy] = useState<AdminCrmFilter>('all');

  const navigateToTab = useCallback((tab: AdminTabId) => {
    onNavigate(tab);
  }, [onNavigate]);

  const value = useMemo<AdminCrmContextValue>(() => ({
    searchQuery,
    setSearchQuery,
    sortBy,
    setSortBy,
    filterBy,
    setFilterBy,
    navigateToTab,
  }), [filterBy, navigateToTab, searchQuery, sortBy]);

  return (
    <AdminCrmContext.Provider value={value}>
      {children}
    </AdminCrmContext.Provider>
  );
}

export function useAdminCrmControls(): AdminCrmContextValue {
  const context = useContext(AdminCrmContext);
  if (!context) {
    throw new Error('useAdminCrmControls deve ser usado dentro de AdminCrmProvider');
  }
  return context;
}
