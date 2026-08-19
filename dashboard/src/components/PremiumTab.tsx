import { useState, useEffect, useCallback } from 'react';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import {
    Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription,
} from './ui/dialog';
import { Channel, SubscriptionStatus } from '../types';
import {
    fetchSubscriptionStatus, createSubscriptionInvoice, cancelSubscription,
    removeExtraChannel, createExtraChannelInvoice
} from '../api';
import {
    Star, Crown, FlaskConical, Hash, ChevronRight, Check, Loader2, XCircle
} from 'lucide-react';

const BENEFITS = [
    'Conta Telegram Gerenciada',
    'Emojis Customizados',
    'Prioridade no executor',
];

interface PremiumTabProps {
    toast: (message: string, type: 'success' | 'error' | 'info') => void;
    channels?: Channel[];
    onStatusChange?: (status: SubscriptionStatus | null) => void;
}

export function PremiumTab({ toast, channels, onStatusChange }: PremiumTabProps) {
    const [status, setStatus] = useState<SubscriptionStatus | null>(null);
    const [loading, setLoading] = useState(true);
    const [actionLoading, setActionLoading] = useState<string | null>(null);
    const [dialogOpen, setDialogOpen] = useState(false);
    const [step, setStep] = useState<'info' | 'select'>('info');
    const [selectedChannels, setSelectedChannels] = useState<number[]>([]);

    // Derived values BEFORE any useEffect that references them
    const sub = status?.subscription;

    const loadStatus = useCallback(async () => {
        try {
            const res = await fetchSubscriptionStatus();
            const s = res?.data || null;
            setStatus(s);
            onStatusChange?.(s);
        } catch (err: any) {
            // silently fail on background reload
        } finally {
            setLoading(false);
        }
    }, [onStatusChange]);

    useEffect(() => {
        loadStatus();
    }, [loadStatus]);

    // Reset step + selection when dialog opens
    useEffect(() => {
        if (dialogOpen) {
            setStep('info');
            if (channels && channels.length > 0) {
                setSelectedChannels(channels.map(c => c.id));
            }
        }
    }, [dialogOpen, channels]);

    // Sincronizar extraChannelIds com sub.extraChannels quando status muda
    useEffect(() => {
        if (channels && channels.length > 0 && sub) {
            const newExtraIds = channels.slice(1, 1 + (sub.extraChannels || 0)).map(c => c.id);
            setExtraChannelIds(newExtraIds);
        }
    }, [sub?.extraChannels, channels]);

    const toggleChannel = (channelId: number) => {
        setSelectedChannels(prev =>
            prev.includes(channelId)
                ? prev.filter(id => id !== channelId)
                : [...prev, channelId]
        );
    };

    const openInvoice = useCallback((invoiceUrl: string) => {
        const tg = window.Telegram?.WebApp;
        if (tg?.openInvoice) {
            tg.openInvoice(invoiceUrl, (result) => {
                if (result === 'paid') {
                    toast('✅ Pagamento confirmado! Assinatura ativada.', 'success');
                } else if (result === 'cancelled') {
                    toast('Pagamento cancelado.', 'info');
                } else if (result === 'failed') {
                    toast('Falha no pagamento.', 'error');
                }
                loadStatus();
                setDialogOpen(false);
            });
        } else {
            window.open(invoiceUrl, '_blank', 'noopener,noreferrer');
            toast('Invoice aberta em nova aba.', 'info');
        }
    }, [toast, loadStatus]);

    const handleSubscribe = async () => {
        const isTestMode = status?.starsTestMode === true;
        const channelCount = selectedChannels.length || 1;
        setActionLoading('subscribe');
        try {
            const res = await createSubscriptionInvoice(isTestMode, channelCount);
            const invoiceUrl = res?.data?.invoiceUrl;
            if (invoiceUrl) {
                toast(
                    isTestMode
                        ? `🧪 Teste: invoice de ${totalStars} Star(s) aberta — pague para ativar.`
                        : `💳 Invoice de ${totalStars} Stars aberta.`,
                    'info'
                );
                openInvoice(invoiceUrl);
            } else {
                toast('Erro ao criar invoice', 'error');
            }
        } catch (err: any) {
            toast(err?.message || 'Erro ao criar assinatura', 'error');
        } finally {
            setActionLoading(null);
        }
    };

    const handleCancel = async () => {
        setActionLoading('cancel');
        try {
            await cancelSubscription();
            toast('Assinatura cancelada para o fim do período.', 'info');
            loadStatus();
        } catch (err: any) {
            toast(err?.message || 'Erro ao cancelar', 'error');
        } finally {
            setActionLoading(null);
        }
    };

    const handleAddExtra = async () => {
        const isTestMode = status?.starsTestMode === true;
        setActionLoading('addChannel');
        try {
            const res = await createExtraChannelInvoice(isTestMode);
            const invoiceUrl = res?.data?.invoiceUrl;
            if (invoiceUrl) {
                const tg = window.Telegram?.WebApp;
                if (tg?.openInvoice) {
                    tg.openInvoice(invoiceUrl, (result) => {
                        if (result === 'paid') {
                            toast('✅ Canal extra ativado!', 'success');
                        } else if (result === 'cancelled') {
                            toast('Pagamento cancelado.', 'info');
                        } else if (result === 'failed') {
                            toast('Falha no pagamento.', 'error');
                        }
                        loadStatus();
                    });
                } else {
                    window.open(invoiceUrl, '_blank', 'noopener,noreferrer');
                    toast('Invoice aberta em nova aba.', 'info');
                }
            } else {
                toast('Erro ao criar invoice', 'error');
            }
        } catch (err: any) {
            toast(err?.message || 'Erro ao adicionar canal extra', 'error');
        } finally {
            setActionLoading(null);
        }
    };

    const handleRemoveExtra = async () => {
        setActionLoading('removeChannel');
        try {
            await removeExtraChannel();
            toast('Canal extra removido.', 'info');
            loadStatus();
        } catch (err: any) {
            toast(err?.message || 'Erro ao remover canal extra', 'error');
        } finally {
            setActionLoading(null);
        }
    };

    const isActive = status?.hasSubscription && status.subscription?.status === 'active';
    const isCancelling = sub?.cancelAtPeriodEnd;
    const periodEnd = sub?.currentPeriodEnd ? new Date(sub.currentPeriodEnd).toLocaleDateString('pt-BR') : '';
    const periodStart = sub?.currentPeriodStart ? new Date(sub.currentPeriodStart).toLocaleDateString('pt-BR') : '';
    const totalBasePrice = status?.basePrice || 80;
    const totalExtraPrice = status?.extraChannelPrice || 35;
    const selectedCount = selectedChannels.length || 1;
    const extras = Math.max(0, selectedCount - 1);
    const totalStars = totalBasePrice + extras * totalExtraPrice;
    const activeChannels = 1 + (sub?.extraChannels || 0);
    const starsTestMode = status?.starsTestMode === true;

    // Track which channels the user marked as "extra" during this session.
    // Backend only stores a count, so this is purely local/visual.
    const [extraChannelIds, setExtraChannelIds] = useState<number[]>(() => {
        if (!channels || channels.length === 0 || !sub) return [];
        // Seed with the first N-1 channels as extras based on the stored count
        return channels.slice(1, 1 + (sub.extraChannels || 0)).map(c => c.id);
    });

    // ── Compact Trigger Card (like Conta Telegram) ──
    const triggerCard = (
        <Card
            className="cursor-pointer hover:bg-muted/50 transition-colors animate-stagger-in"
            style={{ animationDelay: '0.15s' }}
            onClick={() => setDialogOpen(true)}
        >
            <CardContent className="flex items-center gap-3 py-4">
                <div className="flex items-center justify-center size-11 rounded-xl shrink-0" style={{ background: 'var(--accent-soft)' }}>
                    <Crown size={22} style={{ color: 'var(--accent)' }} />
                </div>
                <div className="min-w-0 flex-1">
                    <span className="text-sm font-semibold">
                        {isActive ? 'Premium' : 'Premium'}
                    </span>
                    {isActive ? (
                        <p className="text-[11px] text-green-500 mt-0.5">
                            Ativo — {activeChannels} canal(is)
                        </p>
                    ) : (
                        <p className="text-[11px] text-muted-foreground mt-0.5">
                            Recursos exclusivos para seus canais
                        </p>
                    )}
                </div>
                <ChevronRight size={18} className="shrink-0 text-muted-foreground/30" />
            </CardContent>
        </Card>
    );

    if (loading) return triggerCard;

    // Se premium foi desativado pelo admin, não mostrar nada
    if (status?.premiumEnabled === false) return null;

    // ── Modal Content ──

    // Step 1: Plan info + benefits + price
    const stepPlanInfo = (
        <div className="flex flex-col gap-5">
            <DialogHeader>
                <div className="flex items-center justify-center size-14 rounded-2xl mx-auto" style={{ background: 'var(--accent-soft)' }}>
                    <Crown size={28} style={{ color: 'var(--accent)' }} />
                </div>
                <DialogTitle className="text-center text-lg">LegendasBr Premium</DialogTitle>
                <DialogDescription className="text-center text-xs">
                    Desbloqueie recursos exclusivos para seus canais no Telegram.
                </DialogDescription>
            </DialogHeader>

            <div className="space-y-2">
                {BENEFITS.map((b) => (
                    <div key={b} className="flex items-center gap-2.5 px-3 py-2 rounded-lg bg-muted/40">
                        <Check size={14} className="text-green-500 shrink-0" />
                        <span className="text-xs font-medium">{b}</span>
                    </div>
                ))}
            </div>

            <div className="flex items-center justify-center gap-2 py-2">
                <Star size={18} className="text-yellow-500 fill-yellow-500" />
                <span className="text-xl font-bold">{totalBasePrice}</span>
                <span className="text-xs text-muted-foreground">Stars / mês</span>
            </div>

            {starsTestMode && (
                <div className="flex justify-center">
                    <Badge variant="secondary" className="text-[10px] gap-1 px-2 py-0.5">
                        <FlaskConical size={10} /> Modo Teste
                    </Badge>
                </div>
            )}

            <Button className="w-full" onClick={() => setStep('select')}>
                Avançar <ChevronRight size={16} className="ml-1" />
            </Button>
        </div>
    );

    // Step 2: Channel selection
    const stepChannelSelect = (
        <div className="flex flex-col gap-4">
            <DialogHeader>
                <DialogTitle className="text-base">Selecione os canais</DialogTitle>
                <DialogDescription className="text-xs">
                    Escolha quais canais receberão os benefícios premium.
                </DialogDescription>
            </DialogHeader>

            {channels && channels.length > 0 && (
                <div className="space-y-1.5 max-h-56 overflow-y-auto">
                    {channels.map((ch) => {
                        const sel = selectedChannels.includes(ch.id);
                        return (
                            <button
                                key={ch.id}
                                onClick={() => toggleChannel(ch.id)}
                                className={`flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-colors w-full text-left ${
                                    sel ? 'bg-accent/10 border border-accent/25' : 'bg-muted/30 hover:bg-muted/50'
                                }`}
                            >
                                <div className={`size-4 rounded border-2 flex items-center justify-center transition-colors shrink-0 ${
                                    sel ? 'bg-accent border-accent' : 'border-muted-foreground/30'
                                }`}>
                                    {sel && <Check size={10} className="text-accent-foreground" />}
                                </div>
                                <Hash size={14} className="shrink-0 text-muted-foreground/40" />
                                <span className="text-sm font-medium truncate flex-1">{ch.title}</span>
                            </button>
                        );
                    })}
                </div>
            )}

            {/* Price summary */}
            <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-muted/40">
                <span className="text-xs text-muted-foreground">
                    {selectedCount} canal(is) • base {totalBasePrice}
                    {extras > 0 && ` + ${extras}×${totalExtraPrice}`}
                </span>
                <span className="text-sm font-bold flex items-center gap-1">
                    <Star size={14} className="text-yellow-500 fill-yellow-500" />
                    {totalStars}
                </span>
            </div>

            <div className="flex gap-2">
                <Button variant="outline" className="flex-1" onClick={() => setStep('info')}>
                    Voltar
                </Button>
                <Button
                    className="flex-1"
                    onClick={handleSubscribe}
                    disabled={actionLoading === 'subscribe' || selectedChannels.length === 0}
                >
                    {actionLoading === 'subscribe' ? (
                        <Loader2 size={14} className="animate-spin mr-1.5" />
                    ) : (
                        <Star size={14} className="mr-1.5" />
                    )}
                    Assinar {selectedCount} canal(is)
                </Button>
            </div>
        </div>
    );

    // Subscribed: management view
    const subscriptionManagement = (
        <div className="flex flex-col gap-4">
            <DialogHeader>
                <div className="flex items-center gap-3">
                    <div className="flex items-center justify-center size-10 rounded-xl" style={{ background: isCancelling ? 'rgba(234,179,8,0.1)' : 'rgba(34,197,94,0.1)' }}>
                        <Crown size={20} style={{ color: isCancelling ? '#eab308' : '#22c55e' }} />
                    </div>
                    <div>
                        <DialogTitle className="text-base">Premium Ativo</DialogTitle>
                        <DialogDescription className="text-xs">
                            {isCancelling ? `Expira em ${periodEnd}` : `Válida até ${periodEnd}`}
                        </DialogDescription>
                    </div>
                    <Badge className="ml-auto" variant={isCancelling ? 'secondary' : 'default'}>
                        {isCancelling ? 'Cancelando' : 'Ativa'}
                    </Badge>
                </div>
            </DialogHeader>

            <div className="space-y-2">
                <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-muted/30">
                    <span className="text-xs text-muted-foreground">Período</span>
                    <span className="text-xs font-medium">{periodStart} — {periodEnd}</span>
                </div>
                <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-muted/30">
                    <span className="text-xs text-muted-foreground">Canais inclusos</span>
                    <span className="text-xs font-semibold">{activeChannels}</span>
                </div>
                <div className="flex items-center justify-between px-3 py-2 rounded-lg bg-muted/30">
                    <span className="text-xs text-muted-foreground">Próxima renovação</span>
                    <span className="text-xs font-semibold flex items-center gap-1">
                        <Star size={12} className="text-yellow-500 fill-yellow-500" />
                        {totalStars}
                    </span>
                </div>
                <div className="pt-1">
                    <p className="text-[10px] text-muted-foreground uppercase tracking-wider font-semibold mb-2 px-1">
                        Canais ({activeChannels} ativos)
                    </p>
                    {channels && channels.length > 0 && (
                        <div className="space-y-1">
                            {channels.map((ch, idx) => {
                                const isBase = idx === 0;
                                const isExtra = extraChannelIds.includes(ch.id);
                                const isActiveChannel = isBase || isExtra;
                                return (
                                    <button
                                        key={ch.id}
                                        onClick={() => {
                                            if (isBase || actionLoading) return;
                                            if (isExtra) {
                                                setExtraChannelIds(prev => prev.filter(id => id !== ch.id));
                                                handleRemoveExtra();
                                            } else {
                                                setExtraChannelIds(prev => [...prev, ch.id]);
                                                handleAddExtra();
                                            }
                                        }}
                                        disabled={actionLoading !== null}
                                        className={`flex items-center gap-2.5 w-full px-3 py-2 rounded-lg transition-colors text-left ${
                                            isActiveChannel
                                                ? 'bg-accent/8 border border-accent/20'
                                                : 'bg-muted/20 hover:bg-muted/40 border border-transparent'
                                        } ${isBase ? 'cursor-default' : 'cursor-pointer'}`}
                                    >
                                        <Hash size={13} className="shrink-0 text-muted-foreground/40" />
                                        <span className="text-xs font-medium truncate flex-1">{ch.title}</span>
                                        {isBase && (
                                            <Badge variant="outline" className="text-[9px] px-1.5 py-0 h-4 leading-none">Base</Badge>
                                        )}
                                        {isExtra && (
                                            <Badge variant="default" className="text-[9px] px-1.5 py-0 h-4 leading-none">Extra</Badge>
                                        )}
                                        {!isActiveChannel && (
                                            <span className="text-[9px] text-muted-foreground/50">Inativo</span>
                                        )}
                                    </button>
                                );
                            })}
                        </div>
                    )}
                </div>
            </div>

            {!isCancelling && (
                <Button
                    variant="outline"
                    className="w-full border-red-500/30 text-red-500 hover:bg-red-500/10 text-xs h-8"
                    onClick={handleCancel}
                    disabled={actionLoading === 'cancel'}
                >
                    {actionLoading === 'cancel' ? (
                        <Loader2 size={14} className="animate-spin mr-1.5" />
                    ) : (
                        <XCircle size={14} className="mr-1.5" />
                    )}
                    Cancelar assinatura
                </Button>
            )}
        </div>
    );

    return (
        <>
            {triggerCard}
            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent showCloseButton={false} className="sm:max-w-sm bg-popover p-5">
                    {isActive ? subscriptionManagement : (step === 'info' ? stepPlanInfo : stepChannelSelect)}
                </DialogContent>
            </Dialog>
        </>
    );
}
