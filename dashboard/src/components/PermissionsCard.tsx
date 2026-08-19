import { Permission } from '../types';
import {
  MessageSquare, Headphones, Video, Image, Smile, Film, Link2, FileText
} from 'lucide-react';
import { type ReactNode } from 'react';
import { Card, CardContent } from './ui/card';
import { Badge } from './ui/badge';
import { Switch } from './ui/switch';

interface Props {
  title: string;
  icon: ReactNode;
  permission: Permission;
  onToggle?: (field: string, value: boolean) => void;
}

const fields: { key: string; label: string; icon: ReactNode }[] = [
  { key: 'message', label: 'Mensagem', icon: <MessageSquare size={16} /> },
  { key: 'audio', label: 'Áudio', icon: <Headphones size={16} /> },
  { key: 'video', label: 'Vídeo', icon: <Video size={16} /> },
  { key: 'photo', label: 'Foto', icon: <Image size={16} /> },
  { key: 'document', label: 'Arquivo', icon: <FileText size={16} /> },
  { key: 'sticker', label: 'Sticker', icon: <Smile size={16} /> },
  { key: 'gif', label: 'GIF', icon: <Film size={16} /> },
  { key: 'linkPreview', label: 'Link Preview', icon: <Link2 size={16} /> },
];

export function PermissionsCard({ title, icon, permission, onToggle }: Props) {
  const isMessagePerm = title.toLowerCase().includes('mensagem');
  const available = fields.filter(f => {
    if (f.key === 'linkPreview') return isMessagePerm;
    if (!permission) return false;
    return (f.key in permission);
  });
  const perm = (permission || {}) as unknown as Record<string, unknown>;
  const active = available.filter(f => perm[f.key] === true).length;

  return (
    <Card>
      <CardContent className="pt-4">
        <div className="flex items-center gap-3 mb-4">
          <div className="section-icon purple">{icon}</div>
          <div className="flex-1 min-w-0">
            <h3 className="text-[15px] font-semibold truncate">{title}</h3>
            <p className="text-xs mt-0.5 text-muted-foreground">
              {active} de {available.length} ativas
            </p>
          </div>
          <Badge variant="secondary">{active}/{available.length}</Badge>
        </div>

        <div className="space-y-2">
          {available.map((f, index) => {
            const isOn = perm[f.key] === true;
            return (
              <div
                key={f.key}
                className={`flex items-center justify-between px-[18px] py-3 rounded-[20px] gap-3 min-h-[52px] cursor-pointer transition-all animate-stagger-in ${isOn ? 'bg-accent/10' : 'bg-muted/50'}`}
                style={{ animationDelay: `${index * 0.05}s` }}
                onClick={() => onToggle?.(f.key, !isOn)}
              >
                <div className="flex items-center gap-3 min-w-0">
                  <span
                    className="flex-shrink-0"
                    style={{ color: isOn ? 'var(--accent)' : 'var(--hint)', opacity: isOn ? 1 : 0.4 }}
                  >
                    {f.icon}
                  </span>
                  <span className="text-[13px] font-medium">{f.label}</span>
                </div>
                <Switch
                  checked={isOn}
                  onCheckedChange={(checked) => onToggle?.(f.key, checked)}
                  onClick={(e: React.MouseEvent) => e.stopPropagation()}
                />
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
