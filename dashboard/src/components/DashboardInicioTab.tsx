import { useState, useEffect, memo } from 'react';
import {
    Users, LogOut, ShieldCheck, Send, Crown
} from 'lucide-react';
import { Channel } from '../types';
import { ConfirmModal } from './ConfirmModal';
import { fetchUserInfo, transferChannel, fetchSubscriptionStatus, fetchAccountStatus } from '../api';
import { useToast } from './Toast';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Input } from './ui/input';
import { PerfLine } from './WaveDivider';

interface DashboardInicioTabProps {
    channel: Channel;
    getGreeting: () => string;
    getGreetingIcon: () => React.ReactNode;
    handleDisconnect: () => void;
    showDisconnect: boolean;
    setShowDisconnect: (open: boolean) => void;
    isDisconnecting: boolean;
    confirmDisconnect: () => void;
    showDisconnectSuccess: boolean;
    hasPremium?: boolean;
}

export const DashboardInicioTab = memo(({
    channel, getGreeting, getGreetingIcon,
    handleDisconnect, showDisconnect, setShowDisconnect, isDisconnecting, confirmDisconnect,
    showDisconnectSuccess, hasPremium = false,
}: DashboardInicioTabProps) => {
    const [transferInput, setTransferInput] = useState('');
    const [isTransferring, setIsTransferring] = useState(false);
    const [showTransferConfirm, setShowTransferConfirm] = useState(false);
    const [transferNewOwnerName, setTransferNewOwnerName] = useState('');
    const [transferNewOwnerId, setTransferNewOwnerId] = useState<number | null>(null);
    const [showTransferError, setShowTransferError] = useState(false);
    const [transferErrorMessage, setTransferErrorMessage] = useState('');
    const [showTransferSuccess, setShowTransferSuccess] = useState(false);
    const toast = useToast();

    const handleTransferClick = async () => {
        const newOwner = transferInput.trim();
        if (!newOwner) {
            toast('Digite o ID ou Username do novo dono', 'error');
            return;
        }

        setIsTransferring(true);
        setTransferErrorMessage('');
        setShowTransferError(false);

        try {
            const resp = await fetchUserInfo(newOwner);
            const isSuccess = resp && (resp.success || resp.succes) && resp.user;

            if (isSuccess) {
                setTransferNewOwnerName(resp.user.first_name);
                setTransferNewOwnerId(resp.user.id);
                setShowTransferConfirm(true);
            } else {
                setTransferErrorMessage(`Não foi possível encontrar nenhum usuário com o ID ou Username "${newOwner}". Por favor, verifique e tente novamente.`);
                setShowTransferError(true);
            }
        } catch {
            setTransferErrorMessage(`Ocorreu um erro ao buscar as informações do usuário. Tente novamente.`);
            setShowTransferError(true);
        } finally {
            setIsTransferring(false);
        }
    };

    const confirmTransfer = async () => {
        try {
            if (!transferNewOwnerId) throw new Error("New owner ID not found");

            await transferChannel(transferNewOwnerId, channel.id);
            setShowTransferSuccess(true);
            setTransferInput('');
            setShowTransferConfirm(false);
            setTransferNewOwnerName('');
            setTransferNewOwnerId(null);
        } catch (err: any) {
            if (err instanceof Error) {
                try {
                    const parsedErr = JSON.parse(err.message);
                    setTransferErrorMessage(parsedErr.message || 'Erro ao passar a posse para o novo usuário.');
                } catch {
                    setTransferErrorMessage(err.message || 'Erro ao passar a posse para o novo usuário.');
                }
            } else {
                setTransferErrorMessage('Erro desconhecido ao transferir o canal');
            }
            setShowTransferConfirm(false);
            setShowTransferError(true);
        }
    };

    return (
        <div className="space-y-3 tab-content-wrapper">
            
            {/* Unified Identity Header (Transparente) */}
            <div className="animate-stagger-in bg-transparent py-1">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                        <div className="flex items-center justify-center size-9 rounded-xl shrink-0 bg-accent/15 text-accent">
                            {getGreetingIcon()}
                        </div>
                        <div>
                            <h2 className="text-[15px] font-bold leading-none text-foreground">{getGreeting()}</h2>
                            <p className="text-[10px] text-muted-foreground mt-1 uppercase tracking-wider font-semibold">Painel de Controle</p>
                        </div>
                    </div>
                    <div className="flex items-center gap-1.5">
                        {hasPremium && (
                            <Badge variant="default" className="text-[10px] gap-1 px-2 py-0.5 h-5">
                                <Crown size={10} /> Premium
                            </Badge>
                        )}
                        <div className="flex items-center gap-1.5 bg-accent/15 px-2.5 py-1 rounded-xl border border-accent/20">
                            <ShieldCheck size={12} className="text-accent" />
                            <span className="text-[11px] font-mono font-bold text-accent">{channel.ownerId}</span>
                        </div>
                    </div>
                </div>
            </div>

            {/* Canal */}
            <div className="bg-muted/20 rounded-2xl px-5 py-5">
                <div className="flex items-center gap-5">
                    <img
                        src={`/api/channel/${channel.id}/photo`}
                        alt={channel.title}
                        className="w-16 h-16 rounded-2xl shrink-0 object-cover bg-accent/10"
                        onError={(e) => {
                            (e.target as HTMLImageElement).style.display = 'none';
                            const fallback = (e.target as HTMLImageElement).nextElementSibling as HTMLElement;
                            if (fallback) fallback.style.display = 'flex';
                        }}
                    />
                    <div className="flex items-center justify-center w-16 h-16 rounded-2xl shrink-0 hidden" style={{ background: 'var(--accent-soft)' }}>
                        <span className="text-3xl font-bold" style={{ color: 'var(--accent)' }}>
                            {channel.title?.charAt(0).toUpperCase() || '?'}
                        </span>
                    </div>
                    <div className="min-w-0 flex-1">
                        <h3 className="text-[19px] font-bold text-foreground truncate leading-tight">{channel.title}</h3>
                        <p className="text-[12px] text-muted-foreground font-mono mt-1">ID {channel.id}</p>
                    </div>
                </div>
            </div>

            <PerfLine accent />

            {/* Transferir Posse */}
            <div className="rounded-xl border border-border p-4 space-y-3">
                <div className="flex items-center gap-3">
                    <div className="section-icon" style={{ background: 'var(--warning-soft)', color: 'var(--warning)' }}>
                        <Users size={18} />
                    </div>
                    <div className="min-w-0 flex-1">
                        <h3 className="text-[15px] font-semibold truncate">Transferir Posse</h3>
                        <p className="text-xs truncate text-muted-foreground">Passe a administração para outro usuário</p>
                    </div>
                </div>
                <div className="relative">
                    <Users size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground pointer-events-none" />
                    <Input
                        className="h-10 pl-9 rounded-xl"
                        placeholder="ID ou Username do novo dono"
                        value={transferInput}
                        onChange={(e) => setTransferInput(e.target.value)}
                    />
                </div>
                <Button
                    variant="default"
                    className="w-full h-10"
                    onClick={handleTransferClick}
                    disabled={!transferInput.trim() || isTransferring}
                >
                    {isTransferring ? 'Aguarde...' : <><Send size={16} /> Transferir</>}
                </Button>
            </div>

            {/* Desconectar */}
            <Button 
                variant="destructive"
                className="w-full h-12 text-[15px] font-semibold rounded-xl"
                onClick={handleDisconnect}
            >
                <LogOut size={18} />
                Desconectar Bot
            </Button>

            {/* Disconnect Confirm Modal */}
            <ConfirmModal
                open={showDisconnect}
                onClose={() => !isDisconnecting && setShowDisconnect(false)}
                onConfirm={confirmDisconnect}
                title="Desconectar Bot"
                message="Tem certeza que deseja desconectar o bot deste canal? Todas as configurações serão perdidas."
                confirmText={isDisconnecting ? "Desconectando..." : "Desconectar"}
                danger
            />

            {/* Disconnect Success Modal */}
            <ConfirmModal
                open={showDisconnectSuccess}
                onClose={() => { }}
                onConfirm={() => {
                    const tg = window.Telegram?.WebApp;
                    if (tg) {
                        tg.close();
                    }
                }}
                title="Desconectado"
                message="O bot foi desconectado com sucesso. Esta janela será fechada."
                confirmText="Fechar"
            />

            {/* Transfer Confirm Modal */}
            <ConfirmModal
                open={showTransferConfirm}
                onClose={() => !isTransferring && setShowTransferConfirm(false)}
                onConfirm={confirmTransfer}
                title="Confirmar Transferência"
                message={`Você tem certeza que deseja transferir a posse para ${transferNewOwnerName}? Você perderá o acesso de dono.`}
                confirmText={isTransferring ? "Transferindo..." : "Confirmar"}
                danger
            />

            {/* Transfer Success Modal */}
            <ConfirmModal
                open={showTransferSuccess}
                onClose={() => { }}
                onConfirm={() => {
                    const tg = window.Telegram?.WebApp;
                    if (tg) {
                        tg.close();
                    }
                }}
                title="Sucesso"
                message="Posse transferida com sucesso. O bot foi reiniciado e esta janela será fechada."
                confirmText="Fechar"
            />

            {/* Transfer Error Modal */}
            <ConfirmModal
                open={showTransferError}
                onClose={() => setShowTransferError(false)}
                onConfirm={() => setShowTransferError(false)}
                title="Erro na Transferência"
                message={transferErrorMessage}
                confirmText="Ok"
                danger
            />
        </div>
    );
});
