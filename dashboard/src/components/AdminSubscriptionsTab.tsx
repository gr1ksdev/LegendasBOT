import { useState, useEffect, useMemo } from 'react';
import { fetchAdminSubscriptions, adminCancelSubscriptions, adminRefundPayment } from '../api';
import { Subscription } from '../types';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import {
    Dialog, DialogContent, DialogHeader,
    DialogTitle, DialogDescription,
} from './ui/dialog';
import {
    Loader2, Crown, Search, AlertTriangle, XCircle, Hash,
    Undo2, CheckCircle, Copy, CalendarDays, User, Star,
    Clock, DollarSign, RefreshCw, Check
} from 'lucide-react';

interface AdminSubscriptionsTabProps {
    toast: (message: string, type: 'success' | 'error' | 'info') => void;
}

// ── Helpers ──

function daysRemaining(endDate: string): number {
    const end = new Date(endDate);
    const now = new Date();
    return Math.max(0, Math.ceil((end.getTime() - now.getTime()) / 86400000));
}

function formatDate(dateStr: string): string {
    if (!dateStr) return '—';
    return new Date(dateStr).toLocaleDateString('pt-BR');
}

function truncateId(id: string, chars = 12): string {
    if (!id) return '';
    return id.length > chars ? id.slice(0, chars) + '…' : id;
}

async function copyToClipboard(text: string, toast: (msg: string, type: 'success' | 'error' | 'info') => void) {
    try {
        await navigator.clipboard.writeText(text);
        toast('Copiado!', 'success');
    } catch {
        toast('Erro ao copiar', 'error');
    }
}

// ── Component ──

export function AdminSubscriptionsTab({ toast }: AdminSubscriptionsTabProps) {
    const [subs, setSubs] = useState<Subscription[]>([]);
    const [loading, setLoading] = useState(true);
    const [selected, setSelected] = useState<Set<number>>(new Set());
    const [instantMode, setInstantMode] = useState(false);
    const [actionLoading, setActionLoading] = useState(false);
    const [search, setSearch] = useState('');
    const [statusFilter, setStatusFilter] = useState<string>('all');
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [confirmType, setConfirmType] = useState<'cancel' | 'refund'>('cancel');
    const [refundTarget, setRefundTarget] = useState<{ userId: number; chargeId: string } | null>(null);
    const [refundedSet, setRefundedSet] = useState<Set<string>>(new Set());
    const [copiedId, setCopiedId] = useState<string | null>(null);

    const load = async () => {
        setLoading(true);
        try {
            const res = await fetchAdminSubscriptions();
            const list = res?.data || [];
            setSubs(list);
        } catch (err: any) {
            toast(err?.message || 'Erro ao carregar assinaturas', 'error');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => { load(); }, []);

    // ── Stats ──
    const stats = useMemo(() => {
        const total = subs.length;
        const active = subs.filter(s => s.status === 'active' && !s.cancelAtPeriodEnd).length;
        const cancelling = subs.filter(s => s.status === 'active' && s.cancelAtPeriodEnd).length;
        const expired = subs.filter(s => s.status === 'expired').length;
        const withCharge = subs.filter(s => !!s.telegramPaymentId).length;
        return { total, active, cancelling, expired, withCharge };
    }, [subs]);

    // ── Filter + Search ──
    const filtered = useMemo(() => {
        return subs.filter(s => {
            // Status filter
            if (statusFilter === 'active' && !(s.status === 'active' && !s.cancelAtPeriodEnd)) return false;
            if (statusFilter === 'cancelling' && !(s.status === 'active' && s.cancelAtPeriodEnd)) return false;
            if (statusFilter === 'expired' && s.status !== 'expired') return false;
            if (statusFilter === 'with_charge' && !s.telegramPaymentId) return false;

            // Search
            if (!search) return true;
            const q = search.toLowerCase();
            return (
                String(s.userId).includes(q) ||
                s.id.toLowerCase().includes(q) ||
                (s.telegramPaymentId || '').toLowerCase().includes(q)
            );
        });
    }, [subs, search, statusFilter]);

    // ── Actions ──
    const toggleSelect = (userId: number) => {
        setSelected(prev => {
            const next = new Set(prev);
            if (next.has(userId)) next.delete(userId);
            else next.add(userId);
            return next;
        });
    };

    const handleCancel = async () => {
        const ids = Array.from(selected);
        if (ids.length === 0) return;
        setActionLoading(true);
        try {
            await adminCancelSubscriptions(ids, instantMode);
            toast(`Assinaturas de ${ids.length} usuario(s) ${instantMode ? 'expiradas' : 'marcadas p/ cancelamento'}`, 'success');
            setSelected(new Set());
            setConfirmOpen(false);
            load();
        } catch (err: any) {
            toast(err?.message || 'Erro ao cancelar', 'error');
        } finally {
            setActionLoading(false);
        }
    };

    const handleRefund = async () => {
        if (!refundTarget) return;
        setActionLoading(true);
        try {
            await adminRefundPayment(refundTarget.userId, refundTarget.chargeId);
            toast(`✅ Reembolso realizado — usuario ${refundTarget.userId}`, 'success');
            setRefundedSet(prev => new Set(prev).add(refundTarget.chargeId));
            setRefundTarget(null);
            setConfirmOpen(false);
            load();
        } catch (err: any) {
            toast(err?.message || 'Erro ao reembolsar', 'error');
        } finally {
            setActionLoading(false);
        }
    };

    const handleCopy = (text: string, id: string) => {
        copyToClipboard(text, toast);
        setCopiedId(id);
        setTimeout(() => setCopiedId(null), 2000);
    };

    const openRefundConfirm = (userId: number, chargeId: string) => {
        setRefundTarget({ userId, chargeId });
        setConfirmType('refund');
        setConfirmOpen(true);
    };

    const openCancelConfirm = () => {
        setConfirmType('cancel');
        setConfirmOpen(true);
    };

    // ── Status helpers ──
    const getStatusConfig = (s: Subscription) => {
        if (refundedSet.has(s.telegramPaymentId || '')) {
            return { label: 'Reembolsado', variant: 'outline' as const, className: 'border-green-500/40 text-green-600' };
        }
        if (s.status === 'active' && s.cancelAtPeriodEnd) return { label: 'Cancelando', variant: 'secondary' as const, className: '' };
        if (s.status === 'active') return { label: 'Ativa', variant: 'default' as const, className: '' };
        if (s.status === 'expired') return { label: 'Expirada', variant: 'outline' as const, className: '' };
        return { label: s.status, variant: 'outline' as const, className: '' };
    };

    const isRefunded = (s: Subscription) => refundedSet.has(s.telegramPaymentId || '');

    // ── Render ──
    return (
        <div className="admin-subscriptions-page space-y-5">
            {/* ── Stats Cards ── */}
            <div className="grid grid-cols-2 sm:grid-cols-5 gap-2.5">
                {[
                    { label: 'Total', value: stats.total, icon: Crown, color: 'var(--accent)' },
                    { label: 'Ativas', value: stats.active, icon: CheckCircle, color: 'var(--success)' },
                    { label: 'Cancelando', value: stats.cancelling, icon: Clock, color: 'var(--warning)' },
                    { label: 'Expiradas', value: stats.expired, icon: XCircle, color: 'var(--danger)' },
                    { label: 'Com charge', value: stats.withCharge, icon: DollarSign, color: '#595956' },
                ].map(stat => (
                    <Card key={stat.label} className="border-border/50">
                        <CardContent className="flex items-center gap-2.5 py-3 px-3.5">
                            <div
                                className="flex items-center justify-center size-8 rounded-lg shrink-0"
                                style={{ background: `${stat.color}15` }}
                            >
                                <stat.icon size={14} style={{ color: stat.color }} />
                            </div>
                            <div className="min-w-0">
                                <p className="text-lg font-bold leading-tight">{stat.value}</p>
                                <p className="text-[10px] text-muted-foreground leading-tight">{stat.label}</p>
                            </div>
                        </CardContent>
                    </Card>
                ))}
            </div>

            {/* ── Header ── */}
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                    <h2 className="text-base font-bold shrink-0">Assinaturas</h2>
                    {/* Filtros com scroll horizontal no mobile */}
                    <div className="flex gap-1.5 overflow-x-auto scrollbar-none [-ms-overflow-style:none] [scrollbar-width:none] py-0.5">
                        {(['all', 'active', 'cancelling', 'expired', 'with_charge'] as const).map(f => (
                            <button
                                key={f}
                                onClick={() => setStatusFilter(f)}
                                className={`text-xs px-2.5 py-1 rounded-md whitespace-nowrap transition-colors shrink-0 font-medium ${
                                    statusFilter === f
                                        ? 'bg-primary text-primary-foreground shadow-xs'
                                        : 'bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground'
                                }`}
                            >
                                {f === 'all' ? 'Todas' : f === 'active' ? 'Ativas' : f === 'cancelling' ? 'Cancelando' : f === 'expired' ? 'Expiradas' : 'Com charge'}
                            </button>
                        ))}
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <div className="relative flex-1 sm:flex-none">
                        <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
                        <input
                            className="h-8 rounded-lg border border-border bg-transparent pl-8 pr-3 text-xs outline-none focus:border-accent w-full sm:w-48"
                            placeholder="Buscar user ID ou charge..."
                            value={search}
                            onChange={e => setSearch(e.target.value)}
                        />
                    </div>
                    <Button variant="ghost" size="icon" className="size-8 shrink-0" onClick={load} disabled={loading} title="Atualizar">
                        <RefreshCw size={14} className={loading ? 'animate-spin' : ''} />
                    </Button>
                </div>
            </div>

            {/* ── Mass Actions Bar ── */}
            {selected.size > 0 && (
                <Card className="border-yellow-500/30 bg-yellow-500/5">
                    <CardContent className="flex items-center justify-between gap-3 py-2.5 px-4">
                        <div className="flex items-center gap-2 text-xs">
                            <AlertTriangle size={14} className="text-yellow-500 shrink-0" />
                            <span className="text-muted-foreground">
                                <strong className="text-foreground">{selected.size}</strong> selecionada(s)
                            </span>
                            <button
                                onClick={() => setSelected(new Set())}
                                className="text-muted-foreground/50 hover:text-muted-foreground underline text-[10px] ml-1"
                            >
                                limpar
                            </button>
                        </div>
                        <div className="flex items-center gap-2.5">
                            <button
                                onClick={() => setInstantMode(!instantMode)}
                                className="flex items-center gap-1.5 text-[10px] text-muted-foreground cursor-pointer select-none hover:text-foreground transition-colors"
                            >
                                <div className={`size-3.5 rounded border-2 flex items-center justify-center transition-all ${
                                    instantMode ? 'bg-accent border-accent' : 'border-muted-foreground/30'
                                }`}>
                                    {instantMode && <Check size={8} className="text-accent-foreground" />}
                                </div>
                                Instantâneo
                            </button>
                            <Button
                                variant="destructive"
                                size="sm"
                                className="h-7 text-xs px-2.5"
                                onClick={openCancelConfirm}
                                disabled={actionLoading}
                            >
                                {actionLoading ? <Loader2 size={12} className="animate-spin" /> : <XCircle size={12} className="mr-1" />}
                                Cancelar
                            </Button>
                        </div>
                    </CardContent>
                </Card>
            )}

            {/* ── List / Empty / Loading ── */}
            {loading ? (
                <div className="flex items-center justify-center py-16">
                    <Loader2 size={22} className="animate-spin text-muted-foreground" />
                </div>
            ) : filtered.length === 0 ? (
                <div className="text-center py-16">
                    <Crown size={32} className="mx-auto text-muted-foreground/20 mb-3" />
                    <p className="text-sm text-muted-foreground">
                        {subs.length === 0 ? 'Nenhuma assinatura encontrada' : 'Nenhum resultado para a busca'}
                    </p>
                </div>
            ) : (
                <>
                    {/* Select all */}
                    <label className="flex items-center gap-2.5 px-1 py-1.5 cursor-pointer hover:bg-muted/20 rounded-lg transition-colors group">
                        <div className={`size-4 rounded border-2 flex items-center justify-center transition-all shrink-0 ${
                            selected.size === filtered.length && filtered.length > 0
                                ? 'bg-accent border-accent'
                                : 'border-muted-foreground/30 group-hover:border-muted-foreground/50'
                        }`}>
                            {selected.size === filtered.length && filtered.length > 0 && (
                                <Check size={10} className="text-accent-foreground" />
                            )}
                        </div>
                        <span className="text-[11px] text-muted-foreground group-hover:text-foreground transition-colors">
                            Selecionar todos ({filtered.length})
                        </span>
                    </label>

                    {/* Subscription Cards */}
                    <div className="space-y-2">
                        {filtered.map((s) => {
                            const sc = getStatusConfig(s);
                            const sel = selected.has(s.userId);
                            const refunded = isRefunded(s);
                            const hasCharge = !!s.telegramPaymentId;
                            const daysLeft = s.status === 'active' ? daysRemaining(s.currentPeriodEnd) : 0;

                            return (
                                <Card
                                    key={s.id}
                                    className={`transition-all duration-150 border ${
                                        refunded
                                            ? 'border-green-500/20 bg-green-500/[0.02]'
                                            : sel
                                            ? 'border-accent/30 bg-accent/[0.03]'
                                            : 'border-border/50 hover:border-border'
                                    }`}
                                >
                                    <CardContent className="p-3.5">
                                        {/* Row 1: Checkbox + User ID + Status + Actions */}
                                        <div className="flex items-start gap-3">
                                            <button
                                                onClick={() => toggleSelect(s.userId)}
                                                className={`size-4 rounded border-2 flex items-center justify-center transition-all shrink-0 mt-0.5 ${
                                                    sel
                                                        ? 'bg-accent border-accent'
                                                        : 'border-muted-foreground/30 hover:border-muted-foreground/50'
                                                }`}
                                            >
                                                {sel && <Check size={10} className="text-accent-foreground" />}
                                            </button>
                                            <div className="min-w-0 flex-1 space-y-2.5">
                                                {/* User info row */}
                                                <div className="flex items-center justify-between gap-2">
                                                    <div className="flex items-center gap-2 min-w-0">
                                                        <div className="flex items-center justify-center size-7 rounded-lg shrink-0 bg-muted/60">
                                                            <User size={13} className="text-muted-foreground/60" />
                                                        </div>
                                                        <div className="min-w-0">
                                                            <span className="text-sm font-semibold">{s.userId}</span>
                                                            <span className="text-[10px] text-muted-foreground ml-2 hidden sm:inline">
                                                                ID: {truncateId(s.id, 10)}
                                                            </span>
                                                        </div>
                                                    </div>
                                                    <div className="flex items-center gap-1.5 shrink-0">
                                                        <Badge variant={sc.variant} className={`text-[10px] px-2 py-0 h-4.5 ${sc.className}`}>
                                                            {sc.label}
                                                        </Badge>
                                                    </div>
                                                </div>

                                                {/* Detail rows */}
                                                <div className="grid grid-cols-2 sm:grid-cols-4 gap-x-4 gap-y-1.5">
                                                    {/* Period */}
                                                    <div className="flex items-center gap-1.5 min-w-0">
                                                        <CalendarDays size={11} className="shrink-0 text-muted-foreground/40" />
                                                        <span className="text-[11px] text-muted-foreground truncate">
                                                            {formatDate(s.currentPeriodStart)} — {formatDate(s.currentPeriodEnd)}
                                                        </span>
                                                    </div>

                                                    {/* Days remaining */}
                                                    {s.status === 'active' && (
                                                        <div className="flex items-center gap-1.5">
                                                            <Clock size={11} className="shrink-0 text-muted-foreground/40" />
                                                            <span className={`text-[11px] ${daysLeft <= 3 ? 'text-red-500 font-medium' : 'text-muted-foreground'}`}>
                                                                {daysLeft} dia(s)
                                                            </span>
                                                        </div>
                                                    )}

                                                    {/* Channels */}
                                                    <div className="flex items-center gap-1.5">
                                                        <Hash size={11} className="shrink-0 text-muted-foreground/40" />
                                                        <span className="text-[11px] text-muted-foreground">
                                                            {s.extraChannels > 0 ? `1 + ${s.extraChannels} extra` : '1 canal'}
                                                        </span>
                                                    </div>

                                                    {/* Created */}
                                                    <div className="flex items-center gap-1.5">
                                                        <Star size={11} className="shrink-0 text-muted-foreground/40" />
                                                        <span className="text-[11px] text-muted-foreground">{formatDate(s.createdAt)}</span>
                                                    </div>
                                                </div>

                                                {/* Charge ID row */}
                                                {hasCharge && (
                                                    <div className="flex items-center gap-1.5 pt-0.5">
                                                        <DollarSign size={11} className="shrink-0 text-muted-foreground/30" />
                                                        <span className="text-[10px] text-muted-foreground/50 font-mono truncate">
                                                            {s.telegramPaymentId}
                                                        </span>
                                                        <button
                                                            onClick={() => handleCopy(s.telegramPaymentId!, 'charge-' + s.id)}
                                                            className="shrink-0 p-0.5 rounded hover:bg-muted/50 transition-colors"
                                                            title="Copiar charge ID"
                                                        >
                                                            {copiedId === 'charge-' + s.id ? (
                                                                <CheckCircle size={11} className="text-green-500" />
                                                            ) : (
                                                                <Copy size={11} className="text-muted-foreground/40" />
                                                            )}
                                                        </button>
                                                    </div>
                                                )}
                                            </div>

                                            {/* Action buttons */}
                                            <div className="flex flex-col gap-1 shrink-0">
                                                {!refunded && hasCharge && s.status === 'active' && (
                                                    <Button
                                                        variant="outline"
                                                        size="sm"
                                                        className="h-7 px-2 text-[10px] text-blue-600 border-blue-200 hover:bg-blue-50 dark:text-blue-400 dark:border-blue-800 dark:hover:bg-blue-950"
                                                        onClick={() => openRefundConfirm(s.userId, s.telegramPaymentId!)}
                                                    >
                                                        <Undo2 size={11} className="mr-1" /> Reembolsar
                                                    </Button>
                                                )}
                                                {refunded && (
                                                    <Badge variant="outline" className="text-[9px] border-green-500/40 text-green-600 px-2 py-0 h-5">
                                                        <CheckCircle size={9} className="mr-1" /> Reembolsado
                                                    </Badge>
                                                )}
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            );
                        })}
                    </div>
                </>
            )}

            {/* ── Confirm Dialog ── */}
            <Dialog open={confirmOpen} onOpenChange={(open) => { if (!open) { setConfirmOpen(false); setRefundTarget(null); } }}>
                <DialogContent showCloseButton={false} className="sm:max-w-sm bg-popover p-5">
                    <div className="flex flex-col gap-5">
                        <DialogHeader>
                            <div className="flex items-center justify-center size-10 rounded-xl mx-auto" style={{ background: confirmType === 'cancel' ? 'rgba(234,179,8,0.1)' : 'rgba(59,130,246,0.1)' }}>
                                {confirmType === 'cancel'
                                    ? <AlertTriangle size={20} style={{ color: '#eab308' }} />
                                    : <Undo2 size={20} style={{ color: '#3b82f6' }} />
                                }
                            </div>
                            <DialogTitle className="text-center text-base">
                                {confirmType === 'cancel' ? 'Cancelar assinatura(s)' : 'Reembolsar Stars'}
                            </DialogTitle>
                            <DialogDescription className="text-center text-xs">
                                {confirmType === 'cancel'
                                    ? `${selected.size} assinatura(s) selecionada(s)`
                                    : `Usuário ${refundTarget?.userId}`
                                }
                            </DialogDescription>
                        </DialogHeader>

                        {confirmType === 'cancel' ? (
                            <div className="space-y-2">
                                <p className="text-xs text-muted-foreground text-center px-2">
                                    {instantMode
                                        ? `${selected.size} assinatura(s) serão expiradas IMEDIATAMENTE. Os usuários perdem o acesso premium na hora.`
                                        : `${selected.size} assinatura(s) serão canceladas ao final do período atual.`
                                    }
                                </p>
                            </div>
                        ) : (
                            <div className="space-y-2">
                                <p className="text-xs text-muted-foreground text-center px-2">
                                    O usuário receberá o reembolso dos Stars. A assinatura será expirada imediatamente.
                                </p>
                                {refundTarget?.chargeId && (
                                    <div className="flex items-center gap-2 bg-muted/40 rounded-lg px-3 py-2">
                                        <DollarSign size={13} className="shrink-0 text-muted-foreground/40" />
                                        <span className="text-[10px] font-mono break-all text-muted-foreground/70 flex-1 min-w-0">
                                            {refundTarget.chargeId}
                                        </span>
                                        <button
                                            onClick={() => handleCopy(refundTarget.chargeId, 'refund-charge')}
                                            className="shrink-0 p-0.5 rounded hover:bg-muted/50 transition-colors"
                                        >
                                            {copiedId === 'refund-charge' ? (
                                                <CheckCircle size={12} className="text-green-500" />
                                            ) : (
                                                <Copy size={12} className="text-muted-foreground/40" />
                                            )}
                                        </button>
                                    </div>
                                )}
                                <div className="flex items-start gap-1.5 text-yellow-600 dark:text-yellow-400 bg-yellow-500/10 px-3 py-2 rounded-lg">
                                    <AlertTriangle size={12} className="shrink-0 mt-0.5" />
                                    <span className="text-[10px]">Reembolso instantâneo e irreversível</span>
                                </div>
                            </div>
                        )}

                        <div className="space-y-2">
                            <Button
                                variant={confirmType === 'cancel' ? 'destructive' : 'default'}
                                className="w-full text-xs h-8"
                                onClick={confirmType === 'cancel' ? handleCancel : handleRefund}
                                disabled={actionLoading}
                            >
                                {actionLoading ? (
                                    <Loader2 size={13} className="animate-spin mr-1.5" />
                                ) : confirmType === 'cancel' ? (
                                    <XCircle size={13} className="mr-1.5" />
                                ) : (
                                    <Undo2 size={13} className="mr-1.5" />
                                )}
                                {confirmType === 'cancel' ? 'Confirmar cancelamento' : 'Confirmar reembolso'}
                            </Button>
                            <Button
                                variant="outline"
                                className="w-full text-xs h-8"
                                onClick={() => { setConfirmOpen(false); setRefundTarget(null); }}
                            >
                                Voltar
                            </Button>
                        </div>
                    </div>
                </DialogContent>
            </Dialog>
        </div>
    );
}
