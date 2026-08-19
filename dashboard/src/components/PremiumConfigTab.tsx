import { useState, useEffect, useCallback } from 'react';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import {
    Crown, Sparkles, Hash, Trash2, Check, Loader2, X
} from 'lucide-react';
import { RichTextEditor } from './RichTextEditor';
import { CaptionPreview } from './CaptionPreview';
import { EmojiRenderer } from './EmojiRenderer';
import {
    getChannelSeparator, saveChannelSeparator, deleteChannelSeparator,
    fetchEmojiHistory
} from '../api';

interface PremiumConfigTabProps {
    channelId: number;
    caption: string;
    onUpdateCaption: (text: string) => void;
    toast: (message: string, type: 'success' | 'error' | 'info') => void;
    hasSubscription: boolean;
    hasAccount: boolean;
}

interface SeparatorData {
    id: string;
    type: string;
    emojiText: string;
    emojiId: string;
    emojiEntitiesJSON: string;
}

interface EmojiEntity {
    type: string;
    offset: number;
    length: number;
    emoji_id: string;
}

// ── Component ──

export function PremiumConfigTab({
    channelId,
    caption,
    onUpdateCaption,
    toast,
    hasSubscription,
    hasAccount,
}: PremiumConfigTabProps) {
    // ── Caption state ──
    const [editingCaption, setEditingCaption] = useState(false);
    const [captionText, setCaptionText] = useState(caption);

    useEffect(() => { setCaptionText(caption); }, [caption]);

    const saveCaption = () => {
        if (captionText.trim()) {
            onUpdateCaption(captionText);
            setEditingCaption(false);
            toast('Caption salva com sucesso', 'success');
        }
    };

    // ── Separator state ──
    const [separator, setSeparator] = useState<SeparatorData | null>(null);
    const [separatorEmojis, setSeparatorEmojis] = useState<{ text: string; id: string }[]>([]);
    const [loadingSep, setLoadingSep] = useState(true);
    const [savingSep, setSavingSep] = useState(false);

    const loadSeparator = useCallback(async () => {
        setLoadingSep(true);
        try {
            const res = await getChannelSeparator(channelId);
            const data = res?.data;
            if (data && data.emojiEntitiesJSON) {
                setSeparator(data);
                try {
                    const entities: EmojiEntity[] = JSON.parse(data.emojiEntitiesJSON);
                    const emojis = entities.map(e => ({
                        text: e.emoji_id,
                        id: e.emoji_id,
                    }));
                    setSeparatorEmojis(emojis);
                } catch {
                    // Fallback: usar emojiText e emojiId
                    if (data.emojiText && data.emojiId) {
                        setSeparatorEmojis([{ text: data.emojiText, id: data.emojiId }]);
                    }
                }
            } else if (data && data.emojiText && data.emojiId) {
                setSeparator(data);
                setSeparatorEmojis([{ text: data.emojiText, id: data.emojiId }]);
            } else {
                setSeparator(null);
                setSeparatorEmojis([]);
            }
        } catch {
            // Separator não existe ainda
            setSeparator(null);
            setSeparatorEmojis([]);
        } finally {
            setLoadingSep(false);
        }
    }, [channelId]);

    useEffect(() => { loadSeparator(); }, [loadSeparator]);

    const addEmojiToSeparator = useCallback((emojiId: string) => {
        setSeparatorEmojis(prev => [...prev, { text: emojiId, id: emojiId }]);
    }, []);

    const removeEmojiFromSeparator = useCallback((idx: number) => {
        setSeparatorEmojis(prev => prev.filter((_, i) => i !== idx));
    }, []);

    const saveSeparator = useCallback(async () => {
        if (separatorEmojis.length === 0) {
            toast('Adicione pelo menos um emoji', 'error');
            return;
        }
        setSavingSep(true);
        try {
            const entities: EmojiEntity[] = separatorEmojis.map((e, i) => ({
                type: 'custom_emoji',
                offset: i,
                length: 1,
                emoji_id: e.id,
            }));

            await saveChannelSeparator(channelId, {
                type: 'custom_emoji',
                emojiText: separatorEmojis.map(e => e.text).join(''),
                emojiId: separatorEmojis[0].id,
                emojiEntitiesJSON: JSON.stringify(entities),
            });
            toast('Separador salvo com sucesso', 'success');
            loadSeparator();
        } catch (err: any) {
            toast(err?.message || 'Erro ao salvar separador', 'error');
        } finally {
            setSavingSep(false);
        }
    }, [channelId, loadSeparator, separatorEmojis, toast]);

    const deleteSeparator = useCallback(async () => {
        setSavingSep(true);
        try {
            await deleteChannelSeparator(channelId);
            setSeparator(null);
            setSeparatorEmojis([]);
            toast('Separador removido', 'info');
        } catch (err: any) {
            toast(err?.message || 'Erro ao remover separador', 'error');
        } finally {
            setSavingSep(false);
        }
    }, [channelId, toast]);

    // ── Emoji Picker (recent emojis from API) ──
    const [recentEmojis, setRecentEmojis] = useState<string[]>([]);

    useEffect(() => {
        fetchEmojiHistory()
            .then(ids => setRecentEmojis(ids))
            .catch(err => console.warn('Falha ao buscar histórico de emojis:', err));
    }, []);

    // ── Render ──
    return (
        <div className="space-y-4">
            {/* ── Header ── */}
            <div className="flex items-center gap-3">
                <div className="flex items-center justify-center size-10 rounded-xl" style={{ background: 'rgba(168,85,247,0.1)' }}>
                    <Crown size={20} style={{ color: '#a855f7' }} />
                </div>
                <div>
                    <h3 className="text-sm font-bold">Configurações Premium</h3>
                    <p className="text-[11px] text-muted-foreground">Recursos exclusivos para assinantes</p>
                </div>
            </div>

            {/* ── Status Card ── */}
            <Card className="border-accent/20">
                <CardContent className="py-3 px-4">
                    <div className="flex items-center gap-2">
                        <Sparkles size={14} className="text-accent shrink-0" />
                        <span className="text-xs font-medium">Status Premium</span>
                        <div className="ml-auto flex gap-1.5">
                            {hasSubscription && (
                                <Badge variant="default" className="text-[9px] px-1.5 py-0 h-4">Assinatura</Badge>
                            )}
                            {hasAccount && (
                                <Badge variant="secondary" className="text-[9px] px-1.5 py-0 h-4">Conta MTProto</Badge>
                            )}
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* ── Caption Section ── */}
            <Card>
                <CardContent className="pt-4 space-y-3">
                    <div className="flex items-center gap-2">
                        <Hash size={14} className="text-muted-foreground" />
                        <span className="text-xs font-semibold">Legenda com Emojis Premium</span>
                    </div>
                    <p className="text-[10px] text-muted-foreground">
                        Use emojis customizados na legenda do bot. O bot vai enviar esses emojis junto com a mensagem.
                    </p>

                    {editingCaption ? (
                        <div className="space-y-2">
                            <RichTextEditor
                                value={captionText}
                                onChange={setCaptionText}
                                rows={5}
                                placeholder="Escreva sua legenda com emojis premium..."
                            />
                            <div className="flex items-center justify-end gap-2">
                                <Button variant="secondary" size="sm" onClick={() => { setCaptionText(caption); setEditingCaption(false); }}>
                                    <X size={12} className="mr-1" /> Cancelar
                                </Button>
                                <Button size="sm" onClick={saveCaption}>
                                    <Check size={12} className="mr-1" /> Salvar
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <button
                            onClick={() => setEditingCaption(true)}
                            className="w-full text-left transition-colors"
                        >
                            <CaptionPreview text={caption} />
                        </button>
                    )}
                </CardContent>
            </Card>

            {/* ── Separator Section ── */}
            <Card>
                <CardContent className="pt-4 space-y-3">
                    <div className="flex items-center gap-2">
                        <Sparkles size={14} className="text-muted-foreground" />
                        <span className="text-xs font-semibold">Separador com Emojis Premium</span>
                    </div>
                    <p className="text-[10px] text-muted-foreground">
                        Configure um separador que aparece entre a legenda e o post original. Use emojis customizados premium.
                    </p>

                    {loadingSep ? (
                        <div className="flex items-center justify-center py-4">
                            <Loader2 size={16} className="animate-spin text-muted-foreground" />
                        </div>
                    ) : (
                        <>
                            {/* Emojis selecionados */}
                            {separatorEmojis.length > 0 && (
                                <div className="flex items-center gap-1.5 flex-wrap">
                                    {separatorEmojis.map((e, idx) => (
                                        <div
                                            key={idx}
                                            className="flex items-center gap-1 bg-accent/10 border border-accent/20 rounded-lg px-2 py-1"
                                        >
                                            <EmojiRenderer emojiId={e.id} size={16} />
                                            <span className="text-[10px] text-muted-foreground max-w-[60px] truncate">{e.id}</span>
                                            <button
                                                onClick={() => removeEmojiFromSeparator(idx)}
                                                className="text-muted-foreground/40 hover:text-red-500 transition-colors"
                                            >
                                                <X size={10} />
                                            </button>
                                        </div>
                                    ))}
                                </div>
                            )}

                            {/* Recent emojis */}
                            {recentEmojis.length > 0 && (
                                <div>
                                    <p className="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold mb-1.5">
                                        Emojis recentes
                                    </p>
                                    <div className="flex gap-1 flex-wrap">
                                        {recentEmojis.slice(0, 12).map(emojiId => (
                                            <button
                                                key={emojiId}
                                                onClick={() => addEmojiToSeparator(emojiId)}
                                                className="size-8 rounded-lg bg-muted/30 hover:bg-muted/50 flex items-center justify-center transition-colors border border-transparent hover:border-accent/30"
                                                title={emojiId}
                                            >
                                                <EmojiRenderer emojiId={emojiId} size={18} />
                                            </button>
                                        ))}
                                    </div>
                                </div>
                            )}

                            {/* Actions */}
                            <div className="flex items-center gap-2 pt-1">
                                {separatorEmojis.length > 0 && (
                                    <Button
                                        size="sm"
                                        onClick={saveSeparator}
                                        disabled={savingSep}
                                        className="h-7 text-xs"
                                    >
                                        {savingSep ? <Loader2 size={12} className="animate-spin mr-1" /> : <Check size={12} className="mr-1" />}
                                        Salvar separador
                                    </Button>
                                )}
                                {separator && (
                                    <Button
                                        variant="outline"
                                        size="sm"
                                        onClick={deleteSeparator}
                                        disabled={savingSep}
                                        className="h-7 text-xs text-red-500 border-red-500/30 hover:bg-red-500/10"
                                    >
                                        <Trash2 size={12} className="mr-1" />
                                        Remover
                                    </Button>
                                )}
                            </div>

                            {separatorEmojis.length === 0 && (
                                <p className="text-[10px] text-muted-foreground text-center py-2 italic">
                                    Nenhum emoji selecionado. Toque em um emoji recente acima para adicionar.
                                </p>
                            )}
                        </>
                    )}
                </CardContent>
            </Card>
        </div>
    );
}
