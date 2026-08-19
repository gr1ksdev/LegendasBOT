import { Dispatch, SetStateAction, useEffect, useMemo, useState } from 'react';
import {
    Users, Hash, Globe, MousePointerClick,
    Trash2, Link2, MessageSquare, Plus, Image as ImageIcon,
    Send, Eye, Radio, ShieldAlert
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
    { id: 'channels', label: 'Canais', icon: <Hash size={16} />, desc: 'Apenas canais cadastrados' },
    { id: 'users', label: 'Usuários', icon: <Users size={16} />, desc: 'Apenas usuários do bot' },
    { id: 'single', label: 'Suporte', icon: <MousePointerClick size={16} />, desc: 'Mensagem para 1 usuário' },
    { id: 'user_ids', label: 'IDs Usuários', icon: <Users size={16} />, desc: 'Lista personalizada' },
    { id: 'channel_ids', label: 'IDs Canais', icon: <Hash size={16} />, desc: 'Lista personalizada' },
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
    noticeTarget, setNoticeTarget,
    noticeTargetId, setNoticeTargetId,
    noticeButtons, handleAddNoticeButton,
    updateNoticeButton, removeNoticeButton,
    handleSendNotice, isSendingNotice, users, channels
}: AdminNoticeTabProps) {
    const [isConfirmOpen, setIsConfirmOpen] = useState(false);
    const [imageUnavailable, setImageUnavailable] = useState(false);

    useEffect(() => setImageUnavailable(false), [noticeImageUrl]);

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

    const renderPreview = () => {
        let previewUrl = noticeImageUrl;
        if (noticeImageUrl && !noticeImageUrl.startsWith('http') && noticeImageUrl.length > 20) {
            previewUrl = `/api/admin/media-proxy/${noticeImageUrl}`;
        }

        const headerHtml = noticeTarget === 'single' || noticeTarget === 'user_ids' ? '<b>MENSAGEM DO SUPORTE</b><br/><br/>' : '';

        let htmlContent = escapeHtml(noticeMessage)
            .replace(/\*\*(.*?)\*\*/g, '<b>$1</b>')
            .replace(/__(.*?)__/g, '<i>$1</i>')
            .replace(/~~(.*?)~~/g, '<s>$1</s>')
            .replace(/\|\|(.*?)\|\|/g, '<span class="spoiler">$1</span>')
            .replace(/`([^`]+)`/g, '<code>$1</code>')
            .replace(/\n/g, '<br/>');

        return (
            <div className="bg-muted/30 shadow-sm p-3 rounded-2xl rounded-bl-sm max-w-[320px] w-full mx-auto text-[14px] text-foreground leading-relaxed">
                {noticeImageUrl && !imageUnavailable && (
                    <img src={previewUrl} alt="Prévia da mídia do broadcast" className="w-full rounded-xl mb-2 object-contain max-h-[350px] bg-background" onError={() => setImageUnavailable(true)} />
                )}
                {noticeImageUrl && imageUnavailable && <p className="broadcast-preview-media-error">A imagem não pôde ser carregada na prévia.</p>}
                <div dangerouslySetInnerHTML={{ __html: headerHtml + (htmlContent || '<span class="text-muted-foreground/50 font-medium">Sua mensagem aparecerá aqui...</span>') }} className="mb-2 break-words" />
                {noticeButtons.length > 0 && (
                    <div className="flex flex-col gap-1.5 mt-3 pt-2 border-t border-border">
                        {noticeButtons.map((btn, i) => (
                            <div key={i} className="bg-background hover:bg-muted/50 transition-colors border border-border rounded-xl py-2 px-3 text-center text-accent font-semibold text-[13px] cursor-pointer">
                                {btn.text || 'Botão'}
                            </div>
                        ))}
                    </div>
                )}
            </div>
        );
    };

    return (
        <div className="admin-notice-page broadcast-workspace">
            <section className="broadcast-composer" aria-label="Composição do broadcast">
                <header className="broadcast-intro">
                    <div>
                        <span>Comunicação</span>
                        <h2>Novo broadcast</h2>
                        <p>Defina o público, escreva a mensagem e revise antes de iniciar o envio.</p>
                    </div>
                    <Radio size={19} aria-hidden="true" />
                </header>

                <section className="broadcast-section" aria-labelledby="broadcast-audience-title">
                    <div className="broadcast-section-heading">
                        <div><span>01</span><h3 id="broadcast-audience-title">Alcance</h3></div>
                        <strong>{recipientCount.toLocaleString('pt-BR')} destinatário{recipientCount === 1 ? '' : 's'}</strong>
                    </div>
                    <div className="broadcast-targets" role="radiogroup" aria-label="Público-alvo">
                        {targets.map((item) => (
                            <button type="button" key={item.id} onClick={() => setNoticeTarget(item.id)} role="radio" aria-checked={noticeTarget === item.id} className={noticeTarget === item.id ? 'is-selected' : ''}>
                                {item.icon}<span><b>{item.label}</b><small>{item.desc}</small></span>
                            </button>
                        ))}
                    </div>
                    {specificTarget && (
                        <div className="broadcast-id-field">
                            <label htmlFor="broadcast-target-ids">{noticeTarget === 'channel_ids' ? 'IDs dos canais' : noticeTarget === 'user_ids' ? 'IDs dos usuários' : 'ID do usuário'}</label>
                            <Textarea id="broadcast-target-ids" placeholder={noticeTarget === 'channel_ids' ? 'Ex.: -1001234567890, -1009876543210' : 'Ex.: 12345678, 987654321'} value={noticeTargetId} onChange={(event) => setNoticeTargetId(event.target.value)} rows={noticeTarget === 'single' ? 1 : 3} className="resize-none" />
                            <p className={noticeTargetId.trim() && recipientCount === 0 ? 'is-invalid' : ''}>{recipientCount ? `${recipientCount} ID${recipientCount === 1 ? '' : 's'} válido${recipientCount === 1 ? '' : 's'} para este envio.` : 'Separe IDs por vírgula, espaço ou quebra de linha.'}</p>
                        </div>
                    )}
                </section>

                <section className="broadcast-section" aria-labelledby="broadcast-message-title">
                    <div className="broadcast-section-heading">
                        <div><span>02</span><h3 id="broadcast-message-title">Mensagem</h3></div>
                        <strong className={isOverLimit ? 'is-invalid' : ''}>{noticeMessage.length} / {maxChars}</strong>
                    </div>
                    <RichTextEditor value={noticeMessage} onChange={setNoticeMessage} placeholder="Escreva a mensagem que será enviada..." rows={7} />
                    <p className="broadcast-help">Use Markdown para destaque. Com imagem, o Telegram limita a legenda a {maxChars.toLocaleString('pt-BR')} caracteres.</p>
                </section>

                <section className="broadcast-section broadcast-media" aria-labelledby="broadcast-media-title">
                    <div className="broadcast-section-heading">
                        <div><span>03</span><h3 id="broadcast-media-title">Mídia</h3></div>
                        <Badge variant="secondary">Opcional</Badge>
                    </div>
                    <label className="broadcast-media-input"><ImageIcon size={16} aria-hidden="true" /><span>URL da imagem ou GIF</span><Input placeholder="https://exemplo.com/imagem.jpg" value={noticeImageUrl} onChange={(event) => setNoticeImageUrl(event.target.value)} /></label>
                </section>

                <section className="broadcast-section" aria-labelledby="broadcast-buttons-title">
                    <div className="broadcast-section-heading">
                        <div><span>04</span><h3 id="broadcast-buttons-title">Botões</h3></div>
                        <button type="button" className="broadcast-add-button" onClick={handleAddNoticeButton} disabled={noticeButtons.length >= 5}><Plus size={15} /> Adicionar</button>
                    </div>
                    {noticeButtons.length ? <div className="broadcast-buttons-list">
                        {noticeButtons.map((btn, index) => (
                            <div className="broadcast-button-row" key={index}>
                                <div className="broadcast-button-row-top">
                                    {btn.type === 'url' ? <Link2 size={15} aria-hidden="true" /> : <MessageSquare size={15} aria-hidden="true" />}
                                    <Select value={btn.type} onValueChange={(value) => updateNoticeButton(index, 'type', value ?? '')}>
                                        <SelectTrigger aria-label={`Tipo do botão ${index + 1}`}><SelectValue /></SelectTrigger>
                                        <SelectContent><SelectItem value="url">Link externo</SelectItem><SelectItem value="callback">Callback</SelectItem></SelectContent>
                                    </Select>
                                    <button type="button" onClick={() => removeNoticeButton(index)} aria-label={`Remover botão ${index + 1}`}><Trash2 size={15} /></button>
                                </div>
                                <div><Input placeholder="Texto do botão" value={btn.text} onChange={(event) => updateNoticeButton(index, 'text', event.target.value)} maxLength={30} /><Input placeholder={btn.type === 'url' ? 'https://...' : 'Comando'} value={btn.value} onChange={(event) => updateNoticeButton(index, 'value', event.target.value)} maxLength={100} /></div>
                            </div>
                        ))}
                    </div> : <p className="broadcast-empty-buttons">Nenhum botão adicionado. Use botões apenas quando houver uma ação clara.</p>}
                </section>

                <div className="broadcast-submit-row">
                    {!isReady && <p><ShieldAlert size={15} aria-hidden="true" /> Complete a mensagem e revise o público antes de continuar.</p>}
                    <Button variant="default" onClick={() => setIsConfirmOpen(true)} disabled={isSendingNotice || !isReady}>{isSendingNotice ? 'Iniciando envio...' : <><Send size={16} /> Revisar e iniciar</>}</Button>
                </div>
            </section>

            <aside className="broadcast-preview-panel" aria-label="Resumo e pré-visualização">
                <header><div><span>Resumo do envio</span><h2>{selectedTarget.label}</h2></div><Eye size={18} aria-hidden="true" /></header>
                <dl className="broadcast-summary"><div><dt>Alcance estimado</dt><dd>{recipientCount.toLocaleString('pt-BR')}</dd></div><div><dt>Formato</dt><dd>{noticeImageUrl ? 'Imagem + legenda' : 'Mensagem'}</dd></div><div><dt>Botões</dt><dd>{noticeButtons.length}</dd></div></dl>
                <p className="broadcast-disclaimer">A estimativa usa a base carregada agora. O servidor confirma apenas o início do processamento, não a entrega individual.</p>
                <div className="broadcast-preview-heading"><span>Prévia no Telegram</span><Badge variant="secondary">Ao vivo</Badge></div>
                <div className="broadcast-preview-stage">{renderPreview()}</div>
            </aside>

            <ConfirmModal open={isConfirmOpen} onClose={() => setIsConfirmOpen(false)} onConfirm={handleSendNotice} title="Iniciar broadcast?" message={`Você iniciará o envio para aproximadamente ${recipientCount.toLocaleString('pt-BR')} destinatário${recipientCount === 1 ? '' : 's'} em “${selectedTarget.label}”. O processamento seguirá em segundo plano.`} confirmText="Iniciar envio" danger={false} />
        </div>
    );
}
