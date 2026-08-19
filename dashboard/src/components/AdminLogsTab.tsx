import { useEffect, useMemo, useState } from 'react';
import {
  Calendar, ChevronDown, ChevronRight, Hash, RefreshCcw, Search, ChevronLeft, ChevronRight as ChevronRightIcon,
  Inbox, CheckCircle2, SkipForward, XCircle, ShieldAlert, RefreshCw, Link, FileText, LayoutGrid, Wrench, Edit3,
  PlusCircle, Trash2, Eye, Save, Send, AlertTriangle, User, ExternalLink, Code
} from 'lucide-react';
import { fetchAdminLogs } from '../api';
import { AdminLogsFilters, ChannelEvent } from '../types';
import { useToast } from './Toast';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

interface AdminLogsTabProps {
  navigateToChannel: (id: number) => void;
  initialChannelId?: string;
}

// Configuração completa de eventos com nomes em português, descrições e ícones
const eventConfig: Record<string, { label: string; desc: string; icon: any; colorClass: string }> = {
  post_received: { label: 'Post Recebido', desc: 'Novo post detectado no canal', icon: Inbox, colorClass: 'text-blue-400 bg-blue-500/15 border-blue-500/30' },
  post_processed: { label: 'Post Processado', desc: 'Edição e legendas aplicadas com sucesso', icon: CheckCircle2, colorClass: 'text-emerald-400 bg-emerald-500/15 border-emerald-500/30' },
  post_skipped: { label: 'Post Ignorado', desc: 'Post ignorado por regra de filtragem', icon: SkipForward, colorClass: 'text-amber-400 bg-amber-500/15 border-amber-500/30' },
  post_failed: { label: 'Falha no Post', desc: 'Erro ao processar postagem no canal', icon: XCircle, colorClass: 'text-rose-400 bg-rose-500/15 border-rose-500/30' },
  permission_missing: { label: 'Permissão Ausente', desc: 'Bot sem permissão necessária no canal', icon: ShieldAlert, colorClass: 'text-rose-400 bg-rose-500/15 border-rose-500/30' },
  metadata_updated: { label: 'Canal Atualizado', desc: 'Título ou username do canal alterado', icon: RefreshCw, colorClass: 'text-cyan-400 bg-cyan-500/15 border-cyan-500/30' },
  dynamic_links_extracted: { label: 'Links Extraídos', desc: 'Links dinâmicos convertidos em botões', icon: Link, colorClass: 'text-indigo-400 bg-indigo-500/15 border-indigo-500/30' },
  caption_applied: { label: 'Legenda Aplicada', desc: 'Template de legenda formatado no post', icon: FileText, colorClass: 'text-purple-400 bg-purple-500/15 border-purple-500/30' },
  buttons_applied: { label: 'Botões Aplicados', desc: 'Teclado de botões anexado à mensagem', icon: LayoutGrid, colorClass: 'text-blue-400 bg-blue-500/15 border-blue-500/30' },
  
  postbuilder_started: { label: 'PostBuilder Iniciado', desc: 'Usuário abriu o fluxo do PostBuilder', icon: Wrench, colorClass: 'text-purple-400 bg-purple-500/15 border-purple-500/30' },
  postbuilder_field_updated: { label: 'Campo Editado', desc: 'Título ou corpo da postagem atualizado', icon: Edit3, colorClass: 'text-indigo-400 bg-indigo-500/15 border-indigo-500/30' },
  postbuilder_button_added: { label: 'Botão Adicionado', desc: 'Novo botão adicionado no PostBuilder', icon: PlusCircle, colorClass: 'text-emerald-400 bg-emerald-500/15 border-emerald-500/30' },
  postbuilder_button_deleted: { label: 'Botão Removido', desc: 'Botão removido no PostBuilder', icon: Trash2, colorClass: 'text-amber-400 bg-amber-500/15 border-amber-500/30' },
  postbuilder_preview_sent: { label: 'Prévia Enviada', desc: 'Prévia da mensagem enviada ao usuário', icon: Eye, colorClass: 'text-cyan-400 bg-cyan-500/15 border-cyan-500/30' },
  postbuilder_saved: { label: 'Post Salvo', desc: 'Postagem salva para envio ou agendamento', icon: Save, colorClass: 'text-emerald-400 bg-emerald-500/15 border-emerald-500/30' },
  postbuilder_sent_to_channel: { label: 'Enviado ao Canal', desc: 'Postagem enviada com sucesso pelo PostBuilder', icon: Send, colorClass: 'text-emerald-400 bg-emerald-500/15 border-emerald-500/30' },
  postbuilder_failed: { label: 'Falha no PostBuilder', desc: 'Erro durante o fluxo do PostBuilder', icon: AlertTriangle, colorClass: 'text-rose-400 bg-rose-500/15 border-rose-500/30' },
  template_deleted: { label: 'Template Excluído', desc: 'Template de postagem removido', icon: Trash2, colorClass: 'text-rose-400 bg-rose-500/15 border-rose-500/30' },
};

// Traduções dos motivos de skip
const skipReasonLabels: Record<string, string> = {
  via_bot: 'Mensagem enviada via bot inline',
  bot_own_message: 'Mensagem gerada pelo próprio bot',
  bot_separator_message: 'Sticker separador enviado pelo bot',
  maintenance: 'Modo de manutenção ativo',
  channel_not_found: 'Canal não cadastrado na dashboard',
  owner_blacklisted: 'Dono do canal está na blacklist',
  unsupported_message_type: 'Tipo de mídia não suportado',
};

const sourceConfig: Record<string, { label: string; badgeClass: string }> = {
  channel_post: { label: 'Postagem de Canal', badgeClass: 'bg-blue-500/15 text-blue-400 border-blue-500/25' },
  post_builder: { label: 'PostBuilder', badgeClass: 'bg-purple-500/15 text-purple-400 border-purple-500/25' },
};

const statusConfig: Record<string, { label: string; badgeClass: string }> = {
  success: { label: 'Sucesso', badgeClass: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/25' },
  error: { label: 'Erro', badgeClass: 'bg-rose-500/15 text-rose-400 border-rose-500/25' },
  info: { label: 'Informação', badgeClass: 'bg-blue-500/15 text-blue-400 border-blue-500/25' },
  skipped: { label: 'Ignorado', badgeClass: 'bg-amber-500/15 text-amber-400 border-amber-500/25' },
};

function formatDate(value: string): string {
  if (!value) return '-';
  return new Intl.DateTimeFormat('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(value));
}

function formatMetadataReadable(event: ChannelEvent): Array<{ label: string; value: string }> {
  if (!event.metadata) return [];
  try {
    const parsed = JSON.parse(event.metadata);
    const items: Array<{ label: string; value: string }> = [];

    for (const [key, val] of Object.entries(parsed)) {
      let formattedKey = key.replace(/_/g, ' ');
      formattedKey = formattedKey.charAt(0).toUpperCase() + formattedKey.slice(1);

      let formattedVal = String(val);
      if (key === 'reason' && typeof val === 'string') {
        formattedVal = skipReasonLabels[val] || val;
      } else if (typeof val === 'boolean') {
        formattedVal = val ? 'Sim' : 'Não';
      } else if (typeof val === 'object' && val !== null) {
        formattedVal = JSON.stringify(val);
      }

      items.push({ label: formattedKey, value: formattedVal });
    }
    return items;
  } catch {
    return [{ label: 'Conteúdo', value: event.metadata }];
  }
}

export function AdminLogsTab({ navigateToChannel, initialChannelId = '' }: AdminLogsTabProps) {
  const toast = useToast();
  const [filters, setFilters] = useState<AdminLogsFilters>({ limit: 50, offset: 0, channelId: initialChannelId });
  const [events, setEvents] = useState<ChannelEvent[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const page = useMemo(() => Math.floor((filters.offset || 0) / (filters.limit || 50)) + 1, [filters.offset, filters.limit]);
  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / (filters.limit || 50))), [total, filters.limit]);

  const loadLogs = async (nextFilters = filters) => {
    setLoading(true);
    try {
      const data = await fetchAdminLogs(nextFilters);
      setEvents(data.events || []);
      setTotal(data.total || 0);
    } catch (err: any) {
      toast(err.message || 'Erro ao carregar logs', 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadLogs(filters);
  }, []);

  const updateFilter = (key: keyof AdminLogsFilters, value: string) => {
    setFilters(prev => ({ ...prev, [key]: value, offset: 0 }));
  };

  const applyFilters = () => {
    const next = { ...filters, offset: 0 };
    setFilters(next);
    loadLogs(next);
  };

  const goToPage = (direction: 'prev' | 'next') => {
    const limit = filters.limit || 50;
    const offset = Math.max(0, (filters.offset || 0) + (direction === 'next' ? limit : -limit));
    const next = { ...filters, offset };
    setFilters(next);
    loadLogs(next);
  };

  return (
    <div className="admin-logs-page space-y-4">
      {/* Filter bar */}
      <div className="rounded-2xl border border-border/80 bg-card p-4 space-y-3 shadow-sm">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-2.5">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={15} />
            <Input
              className="h-10 pl-9 rounded-xl text-xs"
              placeholder="Buscar em texto/mensagem..."
              value={filters.q || ''}
              onChange={e => updateFilter('q', e.target.value)}
            />
          </div>
          <div className="relative">
            <Hash className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={15} />
            <Input
              className="h-10 pl-9 rounded-xl text-xs"
              placeholder="ID do canal (ex: -100...)"
              value={filters.channelId || ''}
              onChange={e => updateFilter('channelId', e.target.value)}
            />
          </div>
          <Select value={filters.source || ''} onValueChange={v => updateFilter('source', v ?? '')}>
            <SelectTrigger className="w-full h-10 rounded-xl text-xs">
              <SelectValue placeholder="Todas origens" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">Todas origens</SelectItem>
              <SelectItem value="channel_post">Postagens de Canal</SelectItem>
              <SelectItem value="post_builder">PostBuilder</SelectItem>
            </SelectContent>
          </Select>
          <Select value={filters.status || ''} onValueChange={v => updateFilter('status', v ?? '')}>
            <SelectTrigger className="w-full h-10 rounded-xl text-xs">
              <SelectValue placeholder="Todos status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">Todos status</SelectItem>
              <SelectItem value="success">Sucesso</SelectItem>
              <SelectItem value="error">Erro</SelectItem>
              <SelectItem value="skipped">Ignorado</SelectItem>
              <SelectItem value="info">Informação</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5 pt-1">
          <div className="relative">
            <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={15} />
            <Input
              className="h-10 pl-9 rounded-xl text-xs"
              type="date"
              value={filters.dateFrom || ''}
              onChange={e => updateFilter('dateFrom', e.target.value)}
            />
          </div>
          <div className="relative">
            <Calendar className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" size={15} />
            <Input
              className="h-10 pl-9 rounded-xl text-xs"
              type="date"
              value={filters.dateTo || ''}
              onChange={e => updateFilter('dateTo', e.target.value)}
            />
          </div>
          <div className="flex items-center gap-2">
            <Button variant="default" className="flex-1 h-10 rounded-xl text-xs font-bold shadow-sm" onClick={applyFilters} disabled={loading}>
              <Search size={15} className="mr-1.5" /> Filtrar
            </Button>
            <Button variant="outline" className="h-10 px-3 rounded-xl text-xs font-semibold" onClick={() => loadLogs(filters)} disabled={loading} title="Atualizar logs">
              <RefreshCcw size={15} className={loading ? 'animate-spin' : ''} />
            </Button>
          </div>
        </div>
      </div>

      {/* Status bar */}
      <div className="flex items-center justify-between px-1 text-xs text-muted-foreground">
        <span className="font-semibold">{total} evento(s) encontrado(s)</span>
        <span>Página {page} de {pageCount}</span>
      </div>

      {/* Events list */}
      <div className="space-y-2.5">
        {events.length === 0 && !loading ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground rounded-2xl border border-border/80 bg-card p-6 text-center">
            <Search size={32} className="opacity-30 mb-2.5" />
            <p className="text-sm font-bold text-foreground">Nenhum evento registrado</p>
            <p className="text-xs text-muted-foreground mt-1">Tente ajustar os filtros acima para visualizar mais resultados.</p>
          </div>
        ) : events.map(event => {
          const expanded = expandedId === event.id;
          const cfg = eventConfig[event.eventType] || {
            label: event.eventType.replace(/_/g, ' '),
            desc: 'Evento operacional do sistema',
            icon: Inbox,
            colorClass: 'text-muted-foreground bg-muted/20 border-border/60',
          };
          const IconComp = cfg.icon;
          const srcCfg = sourceConfig[event.source] || { label: event.source, badgeClass: 'bg-muted/40 text-muted-foreground border-border/60' };
          const stCfg = statusConfig[event.status] || { label: event.status, badgeClass: 'bg-muted/40 text-muted-foreground border-border/60' };
          const metadataItems = formatMetadataReadable(event);

          return (
            <div
              key={event.id}
              className={`rounded-2xl border transition-all duration-200 overflow-hidden bg-card ${
                expanded ? 'border-accent/40 shadow-sm' : 'border-border/80 hover:border-border'
              }`}
            >
              {/* Header clickable row */}
              <button
                type="button"
                className="w-full text-left p-3.5 flex items-start gap-3.5 hover:bg-muted/15 transition-colors cursor-pointer"
                onClick={() => setExpandedId(expanded ? null : event.id)}
              >
                {/* Event Type Icon */}
                <div className={`flex items-center justify-center size-10 rounded-xl shrink-0 border ${cfg.colorClass}`}>
                  <IconComp size={19} />
                </div>

                {/* Main info */}
                <div className="min-w-0 flex-1 space-y-1">
                  <div className="flex items-center justify-between gap-2 flex-wrap">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-bold text-foreground leading-tight">{cfg.label}</span>
                      <Badge variant="outline" className={`text-[10px] font-semibold px-2 py-0.5 rounded-lg border ${stCfg.badgeClass}`}>
                        {stCfg.label}
                      </Badge>
                      <Badge variant="outline" className={`text-[10px] font-semibold px-2 py-0.5 rounded-lg border ${srcCfg.badgeClass}`}>
                        {srcCfg.label}
                      </Badge>
                    </div>
                    <span className="text-[11px] font-medium text-muted-foreground/80 shrink-0">
                      {formatDate(event.created_at)}
                    </span>
                  </div>

                  {/* Description & Target/Actor Context */}
                  <div className="flex items-center justify-between gap-2 text-xs flex-wrap">
                    <span className="text-muted-foreground truncate">{cfg.desc}</span>
                    <div className="flex items-center gap-2 text-[11px] text-muted-foreground shrink-0">
                      {event.channelTitle ? (
                        <span className="font-semibold text-foreground truncate max-w-[200px]">
                          📢 {event.channelTitle}
                        </span>
                      ) : (
                        <span className="italic text-muted-foreground/70">Sem canal específico</span>
                      )}

                      {event.actorId > 0 && (
                        <span className="flex items-center gap-1 font-mono text-[10px] bg-muted/30 px-1.5 py-0.5 rounded-md">
                          <User size={10} /> User #{event.actorId}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Error highlight box */}
                  {event.errorMessage && (
                    <div className="mt-2 text-xs font-medium text-rose-400 bg-rose-500/10 border border-rose-500/20 p-2.5 rounded-xl flex items-start gap-2">
                      <AlertTriangle size={15} className="shrink-0 mt-0.5" />
                      <span className="break-all">{event.errorMessage}</span>
                    </div>
                  )}
                </div>

                {/* Expand toggle */}
                <div className="shrink-0 pt-2 text-muted-foreground">
                  {expanded ? <ChevronDown size={18} className="text-accent" /> : <ChevronRight size={18} />}
                </div>
              </button>

              {/* Expanded details */}
              {expanded && (
                <div className="border-t border-border/60 p-4 space-y-4 bg-muted/10">
                  {/* Identifiers Grid */}
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5 text-xs">
                    <div className="rounded-xl border border-border/60 bg-muted/20 p-2.5">
                      <span className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Canal ID</span>
                      <span className="font-mono font-bold text-foreground text-xs">{event.channelId || '-'}</span>
                    </div>
                    <div className="rounded-xl border border-border/60 bg-muted/20 p-2.5">
                      <span className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Dono (Owner ID)</span>
                      <span className="font-mono font-bold text-foreground text-xs">{event.ownerId || '-'}</span>
                    </div>
                    <div className="rounded-xl border border-border/60 bg-muted/20 p-2.5">
                      <span className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Usuário (Actor ID)</span>
                      <span className="font-mono font-bold text-foreground text-xs">{event.actorId || '-'}</span>
                    </div>
                    <div className="rounded-xl border border-border/60 bg-muted/20 p-2.5">
                      <span className="block text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Mensagem ID</span>
                      <span className="font-mono font-bold text-foreground text-xs">{event.telegramMessageId || '-'}</span>
                    </div>
                  </div>

                  {/* Actions & Links */}
                  {event.channelId !== 0 && (
                    <div className="flex items-center gap-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-8 px-3 rounded-lg text-xs font-semibold text-accent border-accent/30 hover:bg-accent/10"
                        onClick={() => navigateToChannel(event.channelId)}
                      >
                        <ExternalLink size={13} className="mr-1.5" /> Ir para configurações deste canal
                      </Button>
                    </div>
                  )}

                  {/* Human-readable metadata parameters */}
                  {metadataItems.length > 0 && (
                    <div className="space-y-2 pt-1">
                      <span className="text-[11px] font-bold uppercase tracking-wider text-muted-foreground">Parâmetros do Evento</span>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                        {metadataItems.map((item, idx) => (
                          <div key={idx} className="flex items-center justify-between p-2.5 rounded-xl border border-border/60 bg-card text-xs">
                            <span className="font-medium text-muted-foreground">{item.label}</span>
                            <span className="font-mono font-semibold text-foreground max-w-[220px] truncate" title={item.value}>
                              {item.value}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Raw JSON Debug View */}
                  {event.metadata && (
                    <div className="space-y-1.5 pt-1">
                      <div className="flex items-center justify-between text-[11px] font-semibold text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <Code size={13} /> JSON Bruto (Metadata)
                        </span>
                      </div>
                      <pre className="text-[11px] font-mono overflow-auto rounded-xl p-3 max-h-[180px] border border-border/60 bg-muted/40 text-muted-foreground leading-relaxed select-all">
                        {event.metadata}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>

      {/* Pagination */}
      {pageCount > 1 && (
        <div className="flex items-center justify-between pt-2 text-xs">
          <Button
            variant="outline"
            size="sm"
            className="h-9 px-3.5 rounded-xl font-semibold"
            disabled={loading || (filters.offset || 0) === 0}
            onClick={() => goToPage('prev')}
          >
            <ChevronLeft size={16} className="mr-1" /> Anterior
          </Button>

          <div className="flex items-center gap-1 font-medium text-muted-foreground">
            {Array.from({ length: Math.min(pageCount, 5) }, (_, i) => {
              let pageNum: number;
              if (pageCount <= 5) {
                pageNum = i + 1;
              } else if (page <= 3) {
                pageNum = i + 1;
              } else if (page >= pageCount - 2) {
                pageNum = pageCount - 4 + i;
              } else {
                pageNum = page - 2 + i;
              }
              return (
                <button
                  key={pageNum}
                  type="button"
                  onClick={() => {
                    const offset = (pageNum - 1) * (filters.limit || 50);
                    const next = { ...filters, offset };
                    setFilters(next);
                    loadLogs(next);
                  }}
                  className={`size-8 rounded-lg text-xs font-bold transition-all ${
                    pageNum === page
                      ? 'bg-accent text-accent-foreground shadow-sm scale-105'
                      : 'hover:bg-muted/30 text-muted-foreground'
                  }`}
                >
                  {pageNum}
                </button>
              );
            })}
          </div>

          <Button
            variant="outline"
            size="sm"
            className="h-9 px-3.5 rounded-xl font-semibold"
            disabled={loading || page >= pageCount}
            onClick={() => goToPage('next')}
          >
            Próxima <ChevronRightIcon size={16} className="ml-1" />
          </Button>
        </div>
      )}
    </div>
  );
}
