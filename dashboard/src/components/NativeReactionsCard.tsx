import { memo, useState, useCallback } from 'react';
import { Card, CardContent } from './ui/card';
import { Switch } from './ui/switch';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { Heart, Shuffle, Check, Loader2, Sparkles } from 'lucide-react';
import { updateNativeReactions, updateNativeReactionMode, updateNativeReactionsEnabled } from '../api';

interface NativeReactionsCardProps {
    channelId: number;
    enabled: boolean;
    emojis: string;
    mode: 'fixed' | 'random';
    toast: (msg: string, type: 'success' | 'error' | 'info') => void;
}

const EMOJI_SLOTS = ['👍', '❤️', '🔥', '😍', '🎉', '💯', '🤩', '👏', '⭐', '🥰'];

export const NativeReactionsCard = memo(function NativeReactionsCard({
    channelId,
    enabled,
    emojis,
    mode,
    toast,
}: NativeReactionsCardProps) {
    const [localEnabled, setLocalEnabled] = useState(enabled);
    const [localMode, setLocalMode] = useState(mode);
    const [localEmojis, setLocalEmojis] = useState(emojis);
    const [saving, setSaving] = useState<string | null>(null);

    const handleToggle = useCallback(async () => {
        const newVal = !localEnabled;
        setSaving('toggle');
        try {
            await updateNativeReactionsEnabled(channelId, newVal);
            setLocalEnabled(newVal);
            toast(newVal ? 'Reações nativas ativadas' : 'Reações nativas desativadas', 'success');
        } catch (err: any) {
            toast(err?.message || 'Erro ao alternar', 'error');
        } finally {
            setSaving(null);
        }
    }, [channelId, localEnabled, toast]);

    const handleModeChange = useCallback(async (newMode: 'fixed' | 'random') => {
        setSaving('mode');
        try {
            await updateNativeReactionMode(channelId, newMode);
            setLocalMode(newMode);
            toast(`Modo ${newMode === 'fixed' ? 'fixo' : 'aleatório'} ativado`, 'success');
        } catch (err: any) {
            toast(err?.message || 'Erro ao alterar modo', 'error');
        } finally {
            setSaving(null);
        }
    }, [channelId, toast]);

    const handleEmojiToggle = useCallback(async (emoji: string) => {
        const currentList = localEmojis ? localEmojis.split(',').map(e => e.trim()).filter(Boolean) : [];
        const exists = currentList.includes(emoji);
        const newList = exists
            ? currentList.filter(e => e !== emoji)
            : [...currentList, emoji];
        const newStr = newList.join(',');

        setSaving('emoji');
        try {
            await updateNativeReactions(channelId, newStr);
            setLocalEmojis(newStr);
            toast(exists ? 'Emoji removido' : 'Emoji adicionado', 'success');
        } catch (err: any) {
            toast(err?.message || 'Erro ao salvar emoji', 'error');
        } finally {
            setSaving(null);
        }
    }, [channelId, localEmojis, toast]);

    const selectedList = localEmojis ? localEmojis.split(',').map(e => e.trim()).filter(Boolean) : [];

    return (
        <div className="rounded-2xl border border-border/80 bg-card p-4 space-y-3.5 shadow-sm transition-all hover:border-border">
            {/* Header */}
            <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3 min-w-0">
                    <div className="flex items-center justify-center size-10 rounded-xl shrink-0 bg-emerald-500/15 text-emerald-400 border border-emerald-500/25">
                        <Heart size={19} />
                    </div>
                    <div className="min-w-0">
                        <h3 className="text-sm font-bold text-foreground leading-tight">Reações Nativas do Telegram</h3>
                        <p className="text-[11px] text-muted-foreground mt-0.5">
                            Use as reações nativas do Telegram em vez do grid.
                        </p>
                    </div>
                </div>
                <Switch
                    checked={localEnabled}
                    onCheckedChange={handleToggle}
                    disabled={saving === 'toggle'}
                    aria-label="Ativar reações nativas do Telegram"
                />
            </div>

                {localEnabled && (
                    <div className="space-y-4 pt-3 border-t border-border">
                        {/* Mode selector */}
                        <div>
                            <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">Modo</p>
                            <div className="flex gap-2">
                                <Button
                                    variant={localMode === 'fixed' ? 'default' : 'outline'}
                                    size="sm"
                                    className="flex-1 text-xs h-8"
                                    onClick={() => handleModeChange('fixed')}
                                    disabled={saving === 'mode'}
                                >
                                    {saving === 'mode' && localMode !== 'fixed' ? (
                                        <Loader2 size={12} className="animate-spin mr-1.5" />
                                    ) : (
                                        <Check size={12} className="mr-1.5" />
                                    )}
                                    Fixo
                                </Button>
                                <Button
                                    variant={localMode === 'random' ? 'default' : 'outline'}
                                    size="sm"
                                    className="flex-1 text-xs h-8"
                                    onClick={() => handleModeChange('random')}
                                    disabled={saving === 'mode'}
                                >
                                    {saving === 'mode' && localMode !== 'random' ? (
                                        <Loader2 size={12} className="animate-spin mr-1.5" />
                                    ) : (
                                        <Shuffle size={12} className="mr-1.5" />
                                    )}
                                    Aleatório
                                </Button>
                            </div>
                        </div>

                        {/* Emoji grid (only in fixed mode) */}
                        {localMode === 'fixed' && (
                            <div>
                                <p className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider mb-2">
                                    Escolha os emojis
                                    <span className="font-normal lowercase ml-1">(toque para adicionar/remover)</span>
                                </p>
                                <div className="flex flex-wrap gap-1.5">
                                    {EMOJI_SLOTS.map(emoji => {
                                        const selected = selectedList.includes(emoji);
                                        return (
                                            <button
                                                type="button"
                                                key={emoji}
                                                onClick={() => handleEmojiToggle(emoji)}
                                                disabled={saving === 'emoji'}
                                                className={`size-9 rounded-lg flex items-center justify-center text-lg transition-all ${
                                                    selected
                                                        ? 'bg-accent/15 border border-accent/30 scale-110'
                                                        : 'bg-muted/30 hover:bg-muted/50 border border-transparent'
                                                }`}
                                                aria-pressed={selected}
                                                aria-label={`${selected ? 'Remover' : 'Adicionar'} reação ${emoji}`}
                                            >
                                                {emoji}
                                            </button>
                                        );
                                    })}
                                </div>

                                {/* Selected preview */}
                                {selectedList.length > 0 && (
                                    <div className="mt-3 flex items-center gap-2 px-3 py-2 rounded-lg bg-muted/40">
                                        <Sparkles size={12} className="text-accent shrink-0" />
                                        <span className="text-[11px] text-muted-foreground">
                                            O bot reagirá com:
                                        </span>
                                        <span className="text-base">{selectedList.join(' ')}</span>
                                    </div>
                                )}
                            </div>
                        )}

                        {/* Random mode info */}
                        {localMode === 'random' && (
                            <div className="px-3 py-2 rounded-lg bg-muted/40 flex items-center gap-2">
                                <Shuffle size={14} className="text-accent shrink-0" />
                                <p className="text-[11px] text-muted-foreground">
                                    O bot escolherá um emoji aleatório da lista oficial do Telegram a cada postagem.
                                </p>
                            </div>
                        )}

                        {/* Badge */}
                        <div className="flex items-center gap-2">
                            <Badge variant="default" className="text-[9px] px-2 py-0 gap-1">
                                <Heart size={10} />
                                Nativo
                            </Badge>
                            <Badge variant="secondary" className="text-[9px] px-2 py-0">
                                {localMode === 'fixed' ? `${selectedList.length} emoji(s)` : 'Aleatório'}
                            </Badge>
                        </div>
                    </div>
                )}
        </div>
    );
});
