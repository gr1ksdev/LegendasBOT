import { useState, useEffect, memo } from 'react';
import { Package, Pencil, X, Check, Info } from 'lucide-react';
import { RichTextEditor } from './RichTextEditor';
import { CaptionPreview } from './CaptionPreview';
import { Button } from './ui/button';
import { Switch } from './ui/switch';

interface Props {
  caption: string;
  messageButtons: boolean;
  stickerButtons: boolean;
  messagePosition: 'above' | 'below';
  replyToSticker: boolean;
  onUpdate?: (settings: { caption: string; messageButtons: boolean; stickerButtons: boolean; messagePosition: 'above' | 'below'; replyToSticker: boolean }) => void;
}

export const NewPackCaptionCard = memo(({ caption, messageButtons, stickerButtons, messagePosition, replyToSticker, onUpdate }: Props) => {
  const [editing, setEditing] = useState(false);
  const [text, setText] = useState(caption);
  const [messageBtn, setMessageBtn] = useState(messageButtons);
  const [stickerBtn, setStickerBtn] = useState(stickerButtons);
  const [position, setPosition] = useState<'above' | 'below'>(messagePosition);
  const [replySticker, setReplySticker] = useState(replyToSticker);
  const [showHelp, setShowHelp] = useState(false);

  useEffect(() => { setText(caption); }, [caption]);
  useEffect(() => { setMessageBtn(messageButtons); }, [messageButtons]);
  useEffect(() => { setStickerBtn(stickerButtons); }, [stickerButtons]);
  useEffect(() => { setPosition(messagePosition); }, [messagePosition]);
  useEffect(() => { setReplySticker(replyToSticker); }, [replyToSticker]);

  const save = () => {
    if (text.trim()) {
      onUpdate?.({ caption: text, messageButtons: messageBtn, stickerButtons: stickerBtn, messagePosition: position, replyToSticker: position === 'below' && replySticker });
      setShowHelp(false);
      setEditing(false);
    }
  };
  const cancel = () => {
    setText(caption);
    setMessageBtn(messageButtons);
    setStickerBtn(stickerButtons);
    setPosition(messagePosition);
    setReplySticker(replyToSticker);
    setShowHelp(false);
    setEditing(false);
  };

  return (
    <div className="rounded-2xl border border-border/80 bg-card p-3.5 space-y-3 shadow-sm transition-all hover:border-border">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex items-center justify-center size-10 rounded-xl shrink-0 bg-amber-500/15 text-amber-400 border border-amber-500/25">
            <Package size={19} />
          </div>
          <div className="min-w-0">
            <h3 className="text-sm font-bold text-foreground leading-tight">New Pack Caption</h3>
            <p className="text-[11px] text-muted-foreground mt-0.5">Template para novo pack</p>
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
            rows={7}
            placeholder="Template..."
          />
          <div className="space-y-2 pt-1">
            <div
              className={`flex items-center justify-between p-3 rounded-xl border border-border/60 gap-3 min-h-[48px] cursor-pointer transition-all ${messageBtn ? 'bg-accent/10 border-accent/30' : 'bg-muted/30'}`}
              onClick={() => setMessageBtn(v => !v)}
            >
              <div>
                <span className="text-xs font-bold text-foreground">Botão na mensagem do bot</span>
                <p className="text-[11px] text-muted-foreground mt-0.5">Mostra o botão do pack na mensagem editada.</p>
              </div>
              <Switch
                checked={messageBtn}
                onCheckedChange={(c) => setMessageBtn(c)}
                onClick={(e: React.MouseEvent) => e.stopPropagation()}
              />
            </div>

            <div
              className={`flex items-center justify-between p-3 rounded-xl border border-border/60 gap-3 min-h-[48px] cursor-pointer transition-all ${stickerBtn ? 'bg-accent/10 border-accent/30' : 'bg-muted/30'}`}
              onClick={() => setStickerBtn(v => !v)}
            >
              <div>
                <span className="text-xs font-bold text-foreground">Botão no sticker do pack</span>
                <p className="text-[11px] text-muted-foreground mt-0.5">Mostra o botão abaixo do sticker enviado.</p>
              </div>
              <Switch
                checked={stickerBtn}
                onCheckedChange={(c) => setStickerBtn(c)}
                onClick={(e: React.MouseEvent) => e.stopPropagation()}
              />
            </div>

            <div className="grid grid-cols-2 gap-2 pt-1">
              <Button
                size="sm"
                variant={position === 'above' ? 'default' : 'outline'}
                className="h-9 rounded-xl text-xs font-semibold"
                onClick={() => setPosition('above')}
              >
                Mensagem acima
              </Button>
              <Button
                size="sm"
                variant={position === 'below' ? 'default' : 'outline'}
                className="h-9 rounded-xl text-xs font-semibold"
                onClick={() => setPosition('below')}
              >
                Mensagem abaixo
              </Button>
            </div>

            {position === 'below' && (
              <div
                className={`flex items-center justify-between p-3 rounded-xl border border-border/60 gap-3 min-h-[48px] cursor-pointer transition-all ${replySticker ? 'bg-accent/10 border-accent/30' : 'bg-muted/30'}`}
                onClick={() => setReplySticker(v => !v)}
              >
                <div>
                  <span className="text-xs font-bold text-foreground">Marcar Sticker</span>
                  <p className="text-[11px] text-muted-foreground mt-0.5">Envia a mensagem respondendo ao sticker do pack.</p>
                </div>
                <Switch
                  checked={replySticker}
                  onCheckedChange={(c) => setReplySticker(c)}
                  onClick={(e: React.MouseEvent) => e.stopPropagation()}
                />
              </div>
            )}
          </div>

          <div className="flex items-start gap-2 py-1 text-muted-foreground">
            <Button
              variant="ghost"
              size="icon-xs"
              className={showHelp ? 'text-accent' : ''}
              onClick={() => setShowHelp(v => !v)}
              title="Variáveis disponíveis"
            >
              <Info size={14} />
            </Button>
            {showHelp && (
              <span className="text-[11px] pt-0.5 text-muted-foreground">
                Use <strong className="text-accent font-mono">$name</strong>, <strong className="text-accent font-mono">$title</strong>, <strong className="text-accent font-mono">$link</strong> e <strong className="text-accent font-mono">$count</strong>. Ex: [abrir pack]($link)
              </span>
            )}
          </div>
          <div className="flex items-center justify-end gap-2 pt-1">
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
          aria-label="Editar template de novo pack"
        >
          <CaptionPreview text={caption} />
        </div>
      )}
    </div>
  );
});
