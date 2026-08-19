import { useState, useEffect, memo } from 'react';
import { Caption } from '../types';
import { FileText, Pencil, X, Check } from 'lucide-react';
import { RichTextEditor } from './RichTextEditor';
import { CaptionPreview } from './CaptionPreview';
import { Button } from './ui/button';

interface Props {
  caption: Caption;
  onUpdate?: (text: string) => void;
}

export const CaptionCard = memo(({ caption, onUpdate }: Props) => {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(caption.caption);

  useEffect(() => { setText(caption.caption); }, [caption.caption]);

  const save = () => { if (text.trim()) { onUpdate?.(text); setEditing(false); } };
  const cancel = () => { setText(caption.caption); setEditing(false); };

  return (
    <div className="rounded-2xl border border-border/80 bg-card p-3.5 space-y-3 shadow-sm transition-all hover:border-border">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex items-center justify-center size-10 rounded-xl shrink-0 bg-blue-500/15 text-blue-400 border border-blue-500/25">
            <FileText size={19} />
          </div>
          <div className="min-w-0">
            <h3 className="text-sm font-bold text-foreground leading-tight">Caption Padrão</h3>
            <p className="text-[11px] text-muted-foreground mt-0.5">Aplicada em todas as mensagens</p>
          </div>
        </div>
        {!editing && (
          <Button
            variant="ghost"
            size="sm"
            className="h-8 px-2.5 text-xs font-semibold text-accent hover:bg-accent/10 rounded-lg shrink-0 active:scale-95 transition-all"
            onClick={() => setEditing(true)}
          >
            <Pencil size={13} className="mr-1.5" /> Editar
          </Button>
        )}
      </div>

      {editing ? (
        <div className="space-y-3 pt-1">
          <RichTextEditor
            value={text}
            onChange={setText}
            rows={5}
            placeholder="Caption padrão..."
          />
          <div className="flex items-center justify-end gap-2">
            <Button variant="outline" size="sm" className="h-9 px-3 rounded-xl text-xs font-medium" onClick={cancel}>
              <X size={13} className="mr-1" /> Cancelar
            </Button>
            <Button variant="default" size="sm" className="h-9 px-4 rounded-xl text-xs font-bold shadow-sm" onClick={save}>
              <Check size={13} className="mr-1" /> Salvar
            </Button>
          </div>
        </div>
      ) : (
        <div
          role="button"
          tabIndex={0}
          onClick={() => setEditing(true)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              setEditing(true);
            }
          }}
          className="caption-preview-action w-full rounded-xl border border-border/60 bg-muted/20 px-3 py-2.5 text-left hover:border-accent/40 cursor-pointer transition-all active:scale-[0.99] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60"
          aria-label="Editar caption padrão"
        >
          <CaptionPreview text={caption.caption} />
        </div>
      )}
    </div>
  );
});
