import { memo } from 'react';
import { Badge } from '@/components/ui/badge';

type BadgeVariant = 'success' | 'warning' | 'danger' | 'accent' | 'default';

interface StatusBadgeProps {
  label: string;
  variant?: BadgeVariant;
  dot?: boolean;
  size?: 'sm' | 'md';
}

const SHADCN_VARIANTS: Record<BadgeVariant, "default" | "secondary" | "destructive" | "outline"> = {
  success: 'secondary',
  warning: 'outline',
  danger: 'destructive',
  accent: 'default',
  default: 'secondary',
};

const VARIANT_CLASSES: Record<BadgeVariant, string> = {
  success: 'badge-success',
  warning: 'badge-warning',
  danger: 'badge-danger',
  accent: 'badge-accent',
  default: 'badge-default',
};

export const StatusBadge = memo(function StatusBadge({
  label,
  variant = 'default',
  dot = false,
  size = 'sm',
}: StatusBadgeProps) {
  return (
    <Badge variant={SHADCN_VARIANTS[variant]} className={`status-badge ${VARIANT_CLASSES[variant]} badge-${size}`}>
      {dot && <span className="status-badge-dot" />}
      {label}
    </Badge>
  );
});
