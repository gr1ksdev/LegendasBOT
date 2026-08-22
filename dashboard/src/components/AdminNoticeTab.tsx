import { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';
import {
    Users, Hash, Globe, MousePointerClick,
    Trash2, Link2, MessageSquare, Plus, Image as ImageIcon,
    Send, Info, AlertTriangle, ChevronRight, ArrowLeft,
    ExternalLink, Loader2, Headphones, ShieldAlert, Video
} from 'lucide-react';
import { RichTextEditor } from './RichTextEditor';
import { NoticeButton, NoticeTarget } from '../api';
import { Channel, User } from '../types';
import { ConfirmModal } from './ConfirmModal';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Input } from './ui/input';
import { Textarea } from './ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

interface AdminNoticeTabProps {
    noticeMessage: string;
    setNoticeMessage: Dispatch<SetStateAction<string>>;
    noticeImageUrl: string;
    setNoticeImageUrl: Dispatch<SetStateAction<string>>;
    noticeMediaType: 'photo' | 'video' | 'animation';
    setNoticeMediaType: Dispatch<SetStateAction<'photo' | 'video' | 'animation'>>;
    noticeTarget: NoticeTarget;
    setNoticeTarget: Dispatch<SetStateAction<NoticeTarget>>;
    noticeTargetId: string;
    setNoticeTargetId: Dispatch<SetStateAction<string>>;
    noticeButtons: NoticeButton[];
    handleAddNoticeButton: () => void;
    updateNoticeButton: (index: number, field: keyof NoticeButton, value: string) => void;
    removeNoticeButton: (index: number) => void;
    handleSendNotice: () => void;
    isSendingNotice: boolean;
    users: User[];
    channels: Channel[];
}

const targets: { id: NoticeTarget; label: string; icon: React.ReactNode; desc: string }[] = [
    { id: 'all', label: 'Todos', icon: <Globe size={16} />, desc: 'Todos os usuários e canais' },
    { id: 'channels', label: 'Canais', icon: <Hash size={16} />, desc: 'Apenas canais vinculados' },
    { id: 'users', label: 'Usuários', icon: <Users size={16} />, desc: 'Apenas usuários do bot' },
    { id: 'single', label: 'Suporte', icon: <Headphones size={16} />, desc: 'Mensagens com suporte' },
    { id: 'user_ids', label: 'IDs Usuários', icon: <Users size={16} />, desc: 'Informar IDs de usuários' },
    { id: 'channel_ids', label: 'IDs Canais', icon: <Hash size={16} />, desc: 'Informar IDs de canais' },
];

function escapeHtml(value: string): string {
    return value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

export function AdminNoticeTab({
    noticeMessage, setNoticeMessage,
    noticeImageUrl, setNoticeImageUrl,
    noticeMediaType, setNoticeMediaType,
    noticeTarget, setNoticeTarget,
    noticeTargetId, setNoticeTargetId,
    noticeButtons, handleAddNoticeButton,
    updateNoticeButton, removeNoticeButton,
    handleSendNotice, isSendingNotice, users, channels
}: AdminNoticeTabProps) {
    const [step, setStep] = useState<'compose' | 'review'>('compose');
    const [isConfirmOpen, setIsConfirmOpen] = useState(false);
    const [imageUnavailable, setImageUnavailable] = useState(false);
    const [showMediaInput, setShowMediaInput] = useState(Boolean(noticeImageUrl));

    useEffect(() => setImageUnavailable(false), [noticeImageUrl, noticeMediaType]);

    const targetIds = useMemo(() => noticeTargetId
        .split(/[\s,;]+/)
        .map((value) => Number.parseInt(value.trim(), 10))
        .filter((value) => Number.isFinite(value) && value !== 0), [noticeTargetId]);

    const recipientCount = useMemo(() => {
        if (noticeTarget === 'all') return new Set([...users.map((user) => user.id), ...channels.map((channel) => channel.id)]).size;
        if (noticeTarget === 'users') return users.length;
        if (noticeTarget === 'channels') return channels.length;
        if (noticeTarget === 'single') return targetIds.length ? 1 : 0;
        return new Set(targetIds).size;
    }, [channels, noticeTarget, targetIds, users]);

    const maxChars = noticeImageUrl.trim() ? 1024 : 4096;
    const isOverLimit = noticeMessage.length > maxChars;
    const hasEmptyButtons = noticeButtons.some(b => !b.text.trim() || !b.value.trim());
    const specificTarget = noticeTarget === 'single' || noticeTarget === 'user_ids' || noticeTarget === 'channel_ids';
    const isReady = noticeMessage.trim().length > 0 && !isOverLimit && !hasEmptyButtons && (!specificTarget || recipientCount > 0);
    const selectedTarget = targets.find((item) => item.id === noticeTarget) || targets[0];

    const previewUrl = useMemo(() => {
        if (noticeImageUrl && !noticeImageUrl.startsWith('http') && noticeImageUrl.length > 20) {
            return `/api/admin/media-proxy/${noticeImageUrl}`;
        }
        return noticeImageUrl;
    }, [noticeImageUrl]);

    const headerHtml = noticeTarget === 'single' || noticeTarget === 'user_ids' ? '<b>MENSAGEM DO SUPORTE</b><br/><br/>' : '';

    const formattedHtml = useMemo(() => {
        return escapeHtml(noticeMessage)
            .replace(/\*\*(.*?)\*\*/g, '<b>$1</b>')
            .replace(/\*([^*\n]+)\*/g, '<b>$1</b>')
            .replace(/__([^_\n]+)__/g, '<u>$1</u>')
            .replace(/_([^_\n]+)_/g, '<i>$1</i>')
            .replace(/~~(.*?)~~/g, '<s>$1</s>')
            .replace(/\|\|(.*?)\|\|/g, '<span class="spoiler">$1</span>')
            .replace(/`([^`]+)`/g, '<code>$1</code>')
            .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>')
            .replace(/\n/g, '<br/>');
    }, [noticeMessage]);

    const currentTime = useMemo(() => {
        const d = new Date();
        return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
    }, []);

    // ──────────────────────────────────────────────
    // SCREEN 2: REVIEW BROADCAST
    // ──────────────────────────────────────────────
    if (step === 'review') {
        return (
            <div className="broadcast-screen broadcast-review">
                <header className="broadcast-review-header">
                    <button
                        type="button"
                        onClick={() => setStep('compose')}
                        className="broadcast-review-title"
                    >
                        <ArrowLeft size={18} />
                        <span>Revisar broadcast</span>
                    </button>
                </header>

                <section className="broadcast-review-card broadcast-info-card">
                    <Info size={18} className="shrink-0 mt-0.5 text-accent" />
                    <div>
                        <h2>Revise antes de enviar</h2>
                        <p>
                            Confira todos os detalhes do broadcast antes de enviar para <b>{recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}</b>.
                        </p>
                    </div>
                </section>

                <section className="broadcast-review-card">
                    <h2 className="broadcast-review-card-title"><Globe size={16} /> Público</h2>
                    <div className="broadcast-public-row">
                        <div>
                            <strong>{selectedTarget.desc}</strong>
                        </div>
                        <span className="broadcast-count-pill">
                            {recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}
                        </span>
                    </div>
                </section>

                <section className="broadcast-review-card broadcast-message-card">
                    <h2 className="broadcast-review-card-title"><MessageSquare size={16} /> Mensagem</h2>
                    <div className="broadcast-chat-stage">
                        <div className="broadcast-message-bubble">
                            {noticeImageUrl.trim() && !imageUnavailable && (noticeMediaType === 'video' ? (
                                <video src={previewUrl} className="broadcast-message-photo" controls playsInline preload="metadata" onError={() => setImageUnavailable(true)} />
                            ) : (
                                <img src={previewUrl} alt="Prévia da mídia no Telegram" className="broadcast-message-photo" onError={() => setImageUnavailable(true)} />
                            ))}
                            {noticeImageUrl.trim() && imageUnavailable && (
                                <div className="broadcast-message-media-fallback">
                                    {noticeMediaType === 'video' ? <Video size={20} /> : <ImageIcon size={20} />}
                                    <span>Prévia indisponível</span>
                                    <small>O arquivo será enviado pelo Telegram.</small>
                                </div>
                            )}
                            <div
                                dangerouslySetInnerHTML={{ __html: headerHtml + (formattedHtml || '<span class="text-slate-500 italic">Mensagem vazia</span>') }}
                                className="broadcast-message-copy"
                            />
                            <div className="broadcast-message-time">
                                <span>{currentTime}</span>
                                <span>✓✓</span>
                            </div>
                            {noticeButtons.length > 0 && (
                                <div className="broadcast-message-actions">
                                    {noticeButtons.map((btn, i) => (
                                        <div key={i}>
                                            {btn.type === 'url' ? <ExternalLink size={12} /> : <MessageSquare size={12} />}
                                            <span>{btn.text || `Botão ${i + 1}`}</span>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </section>

                {noticeButtons.length > 0 && (
                    <section className="broadcast-review-card">
                        <h2 className="broadcast-review-card-title"><Hash size={16} /> Botões</h2>
                        <div className="broadcast-telegram-buttons">
                            {noticeButtons.map((btn, i) => (
                                <div key={i}>
                                    {btn.type === 'url' ? <ExternalLink size={13} /> : <MessageSquare size={13} />}
                                    <span>{btn.text || `Botão ${i + 1}`}</span>
                                </div>
                            ))}
                        </div>
                    </section>
                )}

                <section className="broadcast-review-card broadcast-review-summary">
                    <h2>Resumo</h2>
                    <div>
                        <div>
                            <span className="text-muted-foreground">Público</span>
                            <strong>{selectedTarget.label}</strong>
                        </div>
                        <div>
                            <span className="text-muted-foreground">Destinatários</span>
                            <strong>{recipientCount.toLocaleString('pt-BR')}</strong>
                        </div>
                        <div>
                            <span className="text-muted-foreground">Mídia</span>
                            <strong>{noticeImageUrl.trim() ? noticeMediaType === 'video' ? '1 vídeo' : noticeMediaType === 'animation' ? '1 GIF' : '1 imagem' : 'Nenhuma'}</strong>
                        </div>
                        <div>
                            <span className="text-muted-foreground">Botões</span>
                            <strong>{noticeButtons.length === 1 ? '1 botão' : `${noticeButtons.length} botões`}</strong>
                        </div>
                    </div>
                </section>

                <section className="broadcast-review-card broadcast-warning-card">
                    <AlertTriangle size={18} />
                    <div>
                        <h2>Atenção</h2>
                        <p>
                            Este broadcast será enviado imediatamente para <b>{recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}</b>.
                        </p>
                    </div>
                </section>

                <div className="broadcast-final-actions">
                    <Button
                        type="button"
                        variant="destructive"
                        disabled={isSendingNotice || !isReady}
                        onClick={() => setIsConfirmOpen(true)}
                        className="broadcast-send-button"
                    >
                        {isSendingNotice ? (
                            <>
                                <Loader2 size={16} className="animate-spin" />
                                <span>Iniciando envio do broadcast...</span>
                            </>
                        ) : (
                            <>
                                <Send size={16} />
                                <span>Enviar para {recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}</span>
                            </>
                        )}
                    </Button>

                    <Button
                        type="button"
                        variant="outline"
                        disabled={isSendingNotice}
                        onClick={() => setStep('compose')}
                        className="broadcast-edit-button"
                    >
                        <ArrowLeft size={14} className="mr-1.5" />
                        <span>Voltar e editar</span>
                    </Button>
                </div>

                {/* Confirmation Modal */}
                <ConfirmModal
                    open={isConfirmOpen}
                    onClose={() => setIsConfirmOpen(false)}
                    onConfirm={() => {
                        setIsConfirmOpen(false);
                        handleSendNotice();
                    }}
                    title="Confirmar envio do broadcast?"
                    message={`O broadcast será enviado para aproximadamente ${recipientCount.toLocaleString('pt-BR')} destinatário${recipientCount === 1 ? '' : 's'} em "${selectedTarget.label}". Esta ação não pode ser desfeita.`}
                    confirmText={`Confirmar e Enviar (${recipientCount})`}
                    danger={true}
                />
            </div>
        );
    }

    // ──────────────────────────────────────────────
    // SCREEN 1: BROADCAST COMPOSER
    // ──────────────────────────────────────────────
    return (
        <div className="broadcast-screen broadcast-compose">
            {/* 01 — Alcance */}
            <section className="broadcast-step-card" aria-labelledby="broadcast-audience-title">
                <div className="broadcast-step-heading">
                    <div className="flex items-center gap-2">
                        <span className="text-xs font-black text-accent bg-accent/15 px-2 py-0.5 rounded-md">01</span>
                        <h3 id="broadcast-audience-title" className="text-sm font-bold text-foreground">Alcance</h3>
                    </div>
                    <span className="text-xs font-bold text-accent">
                        {recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}
                    </span>
                </div>

                <div className="grid grid-cols-2 gap-2" role="radiogroup" aria-label="Público-alvo">
                    {targets.map((item) => {
                        const isSelected = noticeTarget === item.id;
                        return (
                            <button
                                type="button"
                                key={item.id}
                                onClick={() => setNoticeTarget(item.id)}
                                role="radio"
                                aria-checked={isSelected}
                                className={`flex flex-col text-left p-3 rounded-2xl border transition-all cursor-pointer active:scale-[0.98] ${
                                    isSelected
                                        ? 'bg-accent/15 border-accent text-foreground shadow-2xs ring-1 ring-accent/30'
                                        : 'bg-white/5 border-white/10 hover:bg-white/10 text-foreground'
                                }`}
                            >
                                <div className="flex items-center gap-2 mb-1">
                                    <span className={isSelected ? 'text-accent' : 'text-muted-foreground'}>
                                        {item.icon}
                                    </span>
                                    <span className="text-xs font-bold truncate">{item.label}</span>
                                </div>
                                <span className="text-[11px] text-muted-foreground leading-tight line-clamp-2">
                                    {item.desc}
                                </span>
                            </button>
                        );
                    })}
                </div>

                {specificTarget && (
                    <div className="p-3.5 rounded-2xl border border-white/10 bg-white/5 space-y-1.5">
                        <label htmlFor="broadcast-target-ids" className="text-xs font-semibold text-foreground block">
                            {noticeTarget === 'channel_ids' ? 'IDs dos canais' : noticeTarget === 'user_ids' ? 'IDs dos usuários' : 'ID do usuário'}
                        </label>
                        <Textarea
                            id="broadcast-target-ids"
                            placeholder={noticeTarget === 'channel_ids' ? 'Ex.: -1001234567890, -1009876543210' : 'Ex.: 12345678, 987654321'}
                            value={noticeTargetId}
                            onChange={(e) => setNoticeTargetId(e.target.value)}
                            rows={noticeTarget === 'single' ? 1 : 3}
                            className="resize-none text-xs rounded-xl bg-card border-border shadow-xs"
                        />
                        <p className={`text-[11px] ${noticeTargetId.trim() && recipientCount === 0 ? 'text-destructive font-medium' : 'text-muted-foreground'}`}>
                            {recipientCount ? `${recipientCount} ID${recipientCount === 1 ? '' : 's'} válido${recipientCount === 1 ? '' : 's'} para este envio.` : 'Separe IDs por vírgula, espaço ou quebra de linha.'}
                        </p>
                    </div>
                )}
            </section>

            {/* 02 — Mensagem */}
            <section className="broadcast-step-card broadcast-editor-card" aria-labelledby="broadcast-message-title">
                <div className="broadcast-step-heading">
                    <div className="flex items-center gap-2">
                        <span className="text-xs font-black text-accent bg-accent/15 px-2 py-0.5 rounded-md">02</span>
                        <h3 id="broadcast-message-title" className="text-sm font-bold text-foreground">Mensagem</h3>
                    </div>
                    <span className={`text-xs font-bold ${isOverLimit ? 'text-destructive font-black' : 'text-muted-foreground'}`}>
                        {noticeMessage.length} / {maxChars}
                    </span>
                </div>

                <RichTextEditor
                    value={noticeMessage}
                    onChange={setNoticeMessage}
                    placeholder="Escreva sua mensagem..."
                    rows={6}
                />
                <p className="text-[11px] text-muted-foreground px-0.5">
                    Use Markdown para destacar, criar links e formatar sua mensagem. {noticeImageUrl.trim() && `Com mídia, o Telegram limita a legenda a ${maxChars} caracteres.`}
                </p>
            </section>

            {/* 03 — Mídia */}
            <section className="broadcast-step-card" aria-labelledby="broadcast-media-title">
                <div className="broadcast-step-heading">
                    <div className="flex items-center gap-2">
                        <span className="text-xs font-black text-accent bg-accent/15 px-2 py-0.5 rounded-md">03</span>
                        <h3 id="broadcast-media-title" className="text-sm font-bold text-foreground">Mídia</h3>
                    </div>
                    <Badge variant="secondary" className="text-[10px] font-semibold">
                        Opcional
                    </Badge>
                </div>

                <div className="broadcast-media-control">
                    <div className="broadcast-media-types" role="radiogroup" aria-label="Tipo de mídia">
                        <button type="button" className={noticeMediaType === 'photo' ? 'is-active' : ''} onClick={() => setNoticeMediaType('photo')}><ImageIcon size={14} /> Foto</button>
                        <button type="button" className={noticeMediaType === 'video' ? 'is-active' : ''} onClick={() => setNoticeMediaType('video')}><Video size={14} /> Vídeo</button>
                        <button type="button" className={noticeMediaType === 'animation' ? 'is-active' : ''} onClick={() => setNoticeMediaType('animation')}><ImageIcon size={14} /> GIF</button>
                    </div>
                    {noticeImageUrl.trim() && !imageUnavailable ? (
                        <div className="space-y-2.5">
                            <div className="flex items-center justify-between">
                                <div className="flex items-center gap-2 text-xs font-semibold text-foreground">
                                    {noticeMediaType === 'video' ? <Video size={16} className="text-accent" /> : <ImageIcon size={16} className="text-accent" />}
                                    <span>Mídia anexada</span>
                                </div>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => { setNoticeImageUrl(''); setShowMediaInput(false); }}
                                    className="h-7 px-2 text-xs text-red-400 hover:text-red-300 hover:bg-red-500/10 cursor-pointer"
                                >
                                    <Trash2 size={13} className="mr-1" /> Remover
                                </Button>
                            </div>
                            <div className="broadcast-media-preview">
                                {noticeMediaType === 'video' ? (
                                    <video src={previewUrl} controls playsInline preload="metadata" onError={() => setImageUnavailable(true)} />
                                ) : (
                                    <img src={previewUrl} alt="Prévia da mídia" onError={() => setImageUnavailable(true)} />
                                )}
                            </div>
                            <Input
                                type="text"
                                placeholder="https://exemplo.com/imagem.jpg"
                                value={noticeImageUrl}
                                onChange={(e) => setNoticeImageUrl(e.target.value)}
                                className="h-9 text-xs rounded-xl bg-card border-border shadow-xs"
                            />
                        </div>
                    ) : (
                        <div>
                            <button type="button" className="broadcast-media-dropzone" onClick={() => setShowMediaInput(true)}>
                                {noticeMediaType === 'video' ? <Video size={22} /> : <ImageIcon size={22} />}
                                <span><strong>Adicionar {noticeMediaType === 'video' ? 'vídeo' : noticeMediaType === 'animation' ? 'GIF' : 'imagem'}</strong><small>URL direta ou file_id do Telegram</small></span>
                            </button>
                            {(showMediaInput || imageUnavailable) && (
                                <div className="broadcast-media-url">
                                    <Input
                                        type="text"
                                        autoFocus
                                        placeholder="Cole a URL pública da imagem ou GIF"
                                        value={noticeImageUrl}
                                        onChange={(e) => setNoticeImageUrl(e.target.value)}
                                        className="h-10 text-xs rounded-xl bg-card border-border shadow-xs"
                                    />
                                    {imageUnavailable && <p>A prévia não carregou. Confira a URL e tente novamente.</p>}
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </section>

            {/* 04 — Botões */}
            <section className="broadcast-step-card" aria-labelledby="broadcast-buttons-title">
                <div className="broadcast-step-heading">
                    <div className="flex items-center gap-2">
                        <span className="text-xs font-black text-accent bg-accent/15 px-2 py-0.5 rounded-md">04</span>
                        <h3 id="broadcast-buttons-title" className="text-sm font-bold text-foreground">Botões</h3>
                    </div>
                    <Button
                        type="button"
                        size="sm"
                        variant="ghost"
                        onClick={handleAddNoticeButton}
                        disabled={noticeButtons.length >= 5}
                        className="broadcast-add-action"
                    >
                        <Plus size={13} className="mr-1" /> Adicionar
                    </Button>
                </div>

                {noticeButtons.length > 0 ? (
                    <div className="broadcast-button-editors">
                        {noticeButtons.map((btn, index) => (
                            <div key={index} className="broadcast-button-editor">
                                <div className="broadcast-button-editor-head">
                                    <div className="flex items-center gap-2">
                                        {btn.type === 'url' ? <Link2 size={15} className="text-accent" /> : <MessageSquare size={15} className="text-accent" />}
                                        <Select value={btn.type} onValueChange={(value) => updateNoticeButton(index, 'type', value ?? '')}>
                                            <SelectTrigger size="sm" className="h-8 text-xs font-semibold rounded-lg border-border bg-surface cursor-pointer">
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent className="bg-[#12141a] text-slate-100 border border-slate-800 rounded-xl shadow-2xl z-[99999]">
                                                <SelectItem value="url" className="text-xs font-medium cursor-pointer py-1.5">Link externo</SelectItem>
                                                <SelectItem value="callback" className="text-xs font-medium cursor-pointer py-1.5">Callback</SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>
                                    <button
                                        type="button"
                                        onClick={() => removeNoticeButton(index)}
                                        aria-label={`Remover botão ${index + 1}`}
                                        className="p-1.5 text-muted-foreground hover:text-red-400 transition-colors cursor-pointer"
                                    >
                                        <Trash2 size={15} />
                                    </button>
                                </div>
                                <div className="broadcast-button-fields">
                                    <Input
                                        placeholder="Texto do botão"
                                        value={btn.text}
                                        onChange={(e) => updateNoticeButton(index, 'text', e.target.value)}
                                        maxLength={30}
                                        className="h-9 text-xs rounded-xl bg-card border-border shadow-xs"
                                    />
                                    <Input
                                        placeholder={btn.type === 'url' ? 'https://...' : 'Comando'}
                                        value={btn.value}
                                        onChange={(e) => updateNoticeButton(index, 'value', e.target.value)}
                                        maxLength={100}
                                        className="h-9 text-xs rounded-xl bg-card border-border shadow-xs"
                                    />
                                </div>
                                <div className="broadcast-button-live-preview">
                                    {btn.type === 'url' ? <ExternalLink size={12} /> : <MessageSquare size={12} />}
                                    <span>{btn.text.trim() || `Botão ${index + 1}`}</span>
                                </div>
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="broadcast-buttons-empty">
                        <p className="text-xs font-semibold text-foreground">Nenhum botão adicionado.</p>
                        <p className="text-[11px] text-muted-foreground">Use botões para organizar ações e direcionar o usuário.</p>
                    </div>
                )}
            </section>

            {/* Compact Audience Summary & Primary CTA */}
            <section className="broadcast-audience-summary">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2.5">
                        <div className="flex items-center justify-center size-9 rounded-xl bg-accent/15 text-accent shrink-0">
                            {selectedTarget.icon}
                        </div>
                        <div>
                            <span className="text-xs font-bold text-foreground">{selectedTarget.label}</span>
                            <p className="text-[11px] text-muted-foreground">
                                {recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}
                            </p>
                        </div>
                    </div>
                    {isOverLimit && (
                        <Badge variant="destructive" className="text-[10px]">Limite excedido</Badge>
                    )}
                </div>

                {!isReady && (
                    <div className="flex items-center gap-1.5 text-[11px] text-amber-400 bg-amber-500/10 border border-amber-500/20 p-2.5 rounded-xl">
                        <ShieldAlert size={14} className="shrink-0" />
                        <span>Preencha a mensagem e selecione o público antes de continuar.</span>
                    </div>
                )}

            </section>

            <Button
                type="button"
                variant="default"
                disabled={!isReady}
                onClick={() => setStep('review')}
                className="broadcast-review-button"
            >
                <Send size={16} />
                <span>Revisar broadcast</span>
                <ChevronRight size={16} />
            </Button>
        </div>
    );
}
