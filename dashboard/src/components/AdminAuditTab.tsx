import { Dispatch, SetStateAction, useState } from 'react';
import { Hash, ShieldAlert, ChevronRight, User as UserIcon, Zap, Trash2, ShieldCheck, CheckCircle2, Sparkles } from 'lucide-react';
import { bulkDeleteChannels } from '../api';
import { AuditResult, Channel } from '../types';
import { useToast } from './Toast';
import { ConfirmModal } from './ConfirmModal';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';

interface AdminAuditTabProps {
    navigateToChannel: (id: number) => void;
    onOpenUser: (id: number) => void;
    results: AuditResult[] | null;
    setResults: Dispatch<SetStateAction<AuditResult[] | null>>;
    loading: boolean;
    onRunAudit: () => void;
}

export function AdminAuditTab({ navigateToChannel, onOpenUser, results, setResults, loading, onRunAudit }: AdminAuditTabProps) {
    const [deletingId, setDeletingId] = useState<number | null>(null);
    const [confirmDelete, setConfirmDelete] = useState<{ userId: number, channels: number[], name: string } | null>(null);
    const toast = useToast();

    const handleRunAudit = () => {
        onRunAudit();
    };

    const handleBulkDelete = async () => {
        if (!confirmDelete) return;
        setDeletingId(confirmDelete.userId);
        const { userId, channels } = confirmDelete;
        setConfirmDelete(null);

        try {
            const res = await bulkDeleteChannels(userId, channels);
            if (res.success) {
                toast(`Remoção concluída: ${res.data.deletedCount} canais limpos`, "success");
                setResults(prev => prev ? prev.filter(r => r.userId !== userId) : null);
            } else {
                throw new Error(res.message || "Erro ao excluir canais");
            }
        } catch (err: any) {
            toast(err.message || "Erro na exclusão em massa", "error");
        } finally {
            setDeletingId(null);
        }
    };

    return (
        <div className="admin-audit-page space-y-5">
            {/* Audit CTA Card */}
            <Card>
                <CardContent className="flex flex-col items-center text-center py-8">
                    <div className="section-icon red mb-4" style={{ width: 64, height: 64 }}>
                        <ShieldAlert size={32} />
                    </div>
                    <h2 className="text-xl font-bold mb-2">Auditoria Ativa</h2>
                    <p className="text-sm text-muted-foreground mb-6 max-w-sm">
                        Esta ferramenta realiza uma varredura em tempo real em todos os canais do banco para identificar onde o bot legado <b>@XavolaBot</b> ainda possui permissões de administrador.
                    </p>
                    
                    <Button
                        variant={loading ? "ghost" : "destructive"}
                        className="w-full max-w-xs"
                        onClick={handleRunAudit}
                        disabled={loading}
                    >
                        {loading ? (
                            <>
                                <div className="auth-spinner" style={{ width: 16, height: 16 }} />
                                Varrendo canais...
                            </>
                        ) : (
                            <>
                                <Zap size={18} />
                                Iniciar Varredura Agora
                            </>
                        )}
                    </Button>
                </CardContent>
            </Card>

            {/* Results */}
            {results && results.length > 0 && (
                <div className="space-y-6 mt-8">
                    <h3 className="text-sm font-bold text-muted-foreground px-1">
                        Usuários com XavolaBot detectado ({results.length})
                    </h3>
                    
                    {results.map((result) => (
                        <div key={result.userId} className="space-y-3">
                            {/* User header — Cloudflare clean */}
                            <div className="flex items-center justify-between rounded-xl border border-border bg-card p-3">
                                <button
                                    className="flex min-w-0 flex-1 items-center gap-3 text-left transition-opacity hover:opacity-80"
                                    onClick={() => onOpenUser(result.userId)}
                                >
                                    <div className="section-icon purple sm shrink-0">
                                        <UserIcon size={14} />
                                    </div>
                                    <div className="min-w-0 flex-1">
                                        <span className="truncate text-sm font-bold block">{result.firstName}</span>
                                        <span className="truncate text-[10px] text-muted-foreground block">ID: {result.userId} • {result.channels.length} canais</span>
                                    </div>
                                    <ChevronRight size={16} className="shrink-0 text-muted-foreground/40" />
                                </button>
                                <Button
                                    variant="destructive"
                                    size="sm"
                                    disabled={deletingId !== null}
                                    onClick={() => setConfirmDelete({
                                        userId: result.userId,
                                        channels: result.channels.map(c => c.id),
                                        name: result.firstName
                                    })}
                                    className="shrink-0 ml-3"
                                >
                                    {deletingId === result.userId ? (
                                        <div className="auth-spinner" style={{ width: 14, height: 14 }} />
                                    ) : (
                                        <>
                                            <Trash2 size={14} />
                                            Limpar Tudo
                                        </>
                                    )}
                                </Button>
                            </div>
                            
                            {/* Channel list */}
                            <div className="grid gap-1.5 pl-5 border-l-2 border-border ml-4">
                                {result.channels.map((c: Channel) => (
                                    <button
                                        key={c.id}
                                        className="flex items-center w-full text-left gap-3 rounded-xl border border-border bg-card/50 p-3 hover:bg-muted/50 transition-colors"
                                        onClick={() => navigateToChannel(c.id)}
                                    >
                                        <div className="section-icon purple shrink-0" style={{ transform: 'scale(0.8)' }}>
                                            <Hash size={18} />
                                        </div>
                                        <div className="min-w-0 flex-1">
                                            <h3 className="text-[13px] font-semibold truncate">{c.title}</h3>
                                            <p className="text-[10px] text-muted-foreground truncate">ID: {c.id}</p>
                                        </div>
                                        <ChevronRight size={14} className="shrink-0 text-muted-foreground/30" />
                                    </button>
                                ))}
                            </div>
                        </div>
                    ))}
                </div>
            )}

            {/* Empty result */}
            {results && results.length === 0 && !loading && (
                <Card className="animate-in fade-in slide-in-from-bottom-2 duration-500">
                    <CardContent className="flex flex-col items-center text-center py-8">
                        <div className="section-icon green mb-4" style={{ width: 68, height: 68, borderRadius: 20 }}>
                            <ShieldCheck size={34} />
                        </div>

                        <div className="inline-flex items-center gap-1.5 mb-3 px-3 py-1 rounded-full border border-border bg-card text-[11px] font-bold text-green-500">
                            <CheckCircle2 size={13} />
                            Auditoria concluída
                        </div>

                        <h3 className="text-lg font-bold mb-2">Nenhum XavolaBot encontrado</h3>
                        <p className="text-sm leading-relaxed max-w-sm mb-5 text-muted-foreground">
                            A varredura terminou e nenhum canal do banco possui o bot legado com permissões de administrador.
                        </p>

                        <div className="grid gap-2 w-full max-w-sm">
                            <div className="flex items-center gap-3 rounded-xl border border-border bg-card/50 px-4 py-3 text-left">
                                <div className="section-icon green sm shrink-0"><CheckCircle2 size={14} /></div>
                                <div className="min-w-0">
                                    <p className="text-xs font-bold">Canais verificados</p>
                                    <p className="text-[10px] text-muted-foreground">Nenhuma permissão legada detectada.</p>
                                </div>
                            </div>
                            <div className="flex items-center gap-3 rounded-xl border border-border bg-card/50 px-4 py-3 text-left">
                                <div className="section-icon purple sm shrink-0"><Sparkles size={14} /></div>
                                <div className="min-w-0">
                                    <p className="text-xs font-bold">Nenhuma ação necessária</p>
                                    <p className="text-[10px] text-muted-foreground">A lista de limpeza permanece vazia.</p>
                                </div>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            )}

            <ConfirmModal
                open={confirmDelete !== null}
                onClose={() => setConfirmDelete(null)}
                onConfirm={handleBulkDelete}
                title="Confirmar Exclusão em Massa"
                message={`Você está prestes a remover permanentemente os ${confirmDelete?.channels.length} canais de "${confirmDelete?.name}". O bot sairá dos canais e todos os dados serão apagados. Deseja continuar?`}
                confirmText="Sim, Excluir Tudo"
                danger={true}
            />
        </div>
    );
}
