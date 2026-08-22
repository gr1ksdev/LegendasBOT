import { ReactNode } from 'react';

interface AdminPageHeaderProps {
  eyebrow?: string;
  title: string;
  badge?: ReactNode;
  description?: string;
  actions?: ReactNode;
}

export function AdminPageHeader({ eyebrow = 'PAINEL ADMINISTRATIVO', title, badge, description, actions }: AdminPageHeaderProps) {
  return (
    <header className="minimal-page-header mb-6">
      <div className="minimal-page-header-copy">
        <span className="minimal-page-eyebrow text-xs font-semibold text-muted-foreground uppercase tracking-wider">{eyebrow}</span>
        <div className="minimal-page-title text-2xl font-extrabold text-foreground text-slate-100 tracking-tight mt-1 flex items-center gap-2.5">
          <span>{title}</span>
          {badge}
        </div>
        {description && <p className="text-sm text-muted-foreground mt-1">{description}</p>}
      </div>
      {actions && <div className="minimal-page-header-actions">{actions}</div>}
    </header>
  );
}
