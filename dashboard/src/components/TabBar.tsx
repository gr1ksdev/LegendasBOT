import { memo, type ReactNode } from 'react';

export interface Tab {
  id: string;
  label: string;
  icon: ReactNode;
}

interface Props {
  tabs: Tab[];
  activeTab: string;
  onTabChange: (id: string) => void;
}

export const TabBar = memo(({ tabs, activeTab, onTabChange }: Props) => {
  return (
    <nav className="bottom-nav" aria-label="Navegação principal">
      <div className="bottom-nav-container">
        {tabs.map((t) => {
          const isActive = activeTab === t.id;
          return (
            <button
              key={t.id}
              type="button"
              className={`nav-item ${isActive ? 'active' : ''}`}
              onClick={() => onTabChange(t.id)}
              aria-current={isActive ? 'page' : undefined}
            >
              <span className="nav-item-content">
                <span className="nav-icon shrink-0">{t.icon}</span>
                <span className="nav-label truncate">{t.label}</span>
              </span>
            </button>
          );
        })}
      </div>
    </nav>
  );
});
