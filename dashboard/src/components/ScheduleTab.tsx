import { useState, useEffect } from 'react';
import { ScheduledPost, UserPostTemplate } from '../types';
import { fetchMySchedules, updateScheduleStatus, deleteSchedule, updateScheduleTime, getPostTemplates, deletePostTemplate, loadPostTemplate } from '../api';
import { useToast } from './Toast';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { ConfirmModal } from './ConfirmModal';
import {
  Calendar, Clock, RotateCw, Edit3, Pin, PinOff, Pause, Play, Trash2, Plus, AlertCircle, Info, Sparkles, Folder
} from 'lucide-react';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from './ui/dialog';

interface ScheduleTabProps {
  channelId?: number;
}

const scheduleTypeLabels: Record<string, string> = {
  once: 'Único',
  daily: 'Diário',
  weekly: 'Semanal',
  interval: 'Intervalo',
  queue: 'Fila',
};

const statusConfig: Record<string, { label: string; badgeClass: string; dotClass: string }> = {
  pending: { label: 'Pendente', badgeClass: 'bg-blue-500/15 text-blue-400 border-blue-500/25', dotClass: 'bg-blue-400' },
  scheduled: { label: 'Pendente', badgeClass: 'bg-blue-500/15 text-blue-400 border-blue-500/25', dotClass: 'bg-blue-400' },
  active: { label: 'Ativo', badgeClass: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/25', dotClass: 'bg-emerald-400' },
  paused: { label: 'Pausado', badgeClass: 'bg-amber-500/15 text-amber-400 border-amber-500/25', dotClass: 'bg-amber-400' },
  completed: { label: 'Concluído', badgeClass: 'bg-slate-500/15 text-slate-300 border-slate-500/25', dotClass: 'bg-slate-400' },
  sent: { label: 'Concluído', badgeClass: 'bg-slate-500/15 text-slate-300 border-slate-500/25', dotClass: 'bg-slate-400' },
  cancelled: { label: 'Falhou', badgeClass: 'bg-rose-500/15 text-rose-400 border-rose-500/25', dotClass: 'bg-rose-400' },
  failed: { label: 'Falhou', badgeClass: 'bg-rose-500/15 text-rose-400 border-rose-500/25', dotClass: 'bg-rose-400' },
  error: { label: 'Falhou', badgeClass: 'bg-rose-500/15 text-rose-400 border-rose-500/25', dotClass: 'bg-rose-400' },
};

const inactiveStatuses = new Set(['completed', 'sent', 'cancelled', 'failed', 'error']);

function formatSmartNextRun(dateStr: string): string {
  if (!dateStr) return '-';
  try {
    const d = new Date(dateStr);
    const now = new Date();

    const isToday = d.getDate() === now.getDate() &&
      d.getMonth() === now.getMonth() &&
      d.getFullYear() === now.getFullYear();

    const tomorrow = new Date(now);
    tomorrow.setDate(now.getDate() + 1);
    const isTomorrow = d.getDate() === tomorrow.getDate() &&
      d.getMonth() === tomorrow.getMonth() &&
      d.getFullYear() === tomorrow.getFullYear();

    const timeStr = d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' });

    if (isToday) return `Hoje, ${timeStr}`;
    if (isTomorrow) return `Amanhã, ${timeStr}`;

    return `${d.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: '2-digit' })} às ${timeStr}`;
  } catch {
    return dateStr;
  }
}

export function ScheduleTab({ channelId }: ScheduleTabProps) {
  const [schedules, setSchedules] = useState<ScheduledPost[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingSchedule, setEditingSchedule] = useState<ScheduledPost | null>(null);
  const [confirmDeleteSchedule, setConfirmDeleteSchedule] = useState<ScheduledPost | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);

  const [editDate, setEditDate] = useState('');
  const [editTime, setEditTime] = useState('');
  const [editIntervalMin, setEditIntervalMin] = useState<number | ''>('');
  const [editWindowStart, setEditWindowStart] = useState('');
  const [editWindowEnd, setEditWindowEnd] = useState('');
  const [editAutoDeleteMin, setEditAutoDeleteMin] = useState<number>(0);
  const [showDraftsModal, setShowDraftsModal] = useState(false);
  const [drafts, setDrafts] = useState<UserPostTemplate[]>([]);
  const [loadingDrafts, setLoadingDrafts] = useState(false);
  const [saving, setSaving] = useState(false);
  const toast = useToast();
  const activeScheduleCount = schedules.filter(schedule => (
    !inactiveStatuses.has(schedule.status.toLowerCase())
  )).length;

  const openDraftsModal = async () => {
    setShowDraftsModal(true);
    setLoadingDrafts(true);
    try {
      const data = await getPostTemplates();
      setDrafts(data);
    } catch {
      toast('Erro ao buscar rascunhos', 'error');
    } finally {
      setLoadingDrafts(false);
    }
  };

  const handleDeleteDraft = async (id: string) => {
    try {
      await deletePostTemplate(id);
      setDrafts(prev => prev.filter(d => d.id !== id));
      toast('Rascunho excluído com sucesso', 'success');
    } catch {
      toast('Erro ao excluir rascunho', 'error');
    }
  };

  const handleLoadDraft = async (id: string) => {
    try {
      await loadPostTemplate(id);
      toast('Rascunho carregado no PostBuilder!', 'success');
      setShowDraftsModal(false);
    } catch {
      toast('Erro ao carregar rascunho', 'error');
    }
  };

  useEffect(() => {
    loadSchedules();
  }, [channelId]);

  const loadSchedules = async () => {
    setLoading(true);
    try {
      const all = await fetchMySchedules();
      if (channelId) {
        setSchedules(all.filter((s: ScheduledPost) => s.channelId === channelId));
      } else {
        setSchedules(all);
      }
    } catch {
      toast('Erro ao carregar agendamentos', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleTogglePause = async (schedule: ScheduledPost) => {
    const newStatus = schedule.status.toLowerCase() === 'paused' ? 'pending' : 'paused';
    try {
      await updateScheduleStatus(schedule.id, newStatus);
      setSchedules(prev =>
        prev.map(s => (s.id === schedule.id ? { ...s, status: newStatus } : s))
      );
      toast(newStatus === 'paused' ? 'Agendamento pausado' : 'Agendamento retomado', 'success');
    } catch {
      toast('Erro ao atualizar agendamento', 'error');
    }
  };

  const handleConfirmDelete = async () => {
    if (!confirmDeleteSchedule) return;
    try {
      await deleteSchedule(confirmDeleteSchedule.id);
      setSchedules(prev => prev.filter(s => s.id !== confirmDeleteSchedule.id));
      toast('Agendamento removido com sucesso', 'success');
    } catch {
      toast('Erro ao remover agendamento', 'error');
    } finally {
      setConfirmDeleteSchedule(null);
    }
  };

  const handleTogglePin = async (schedule: ScheduledPost) => {
    const newPin = !schedule.pinMessage;
    try {
      await updateScheduleTime(schedule.id, { pinMessage: newPin });
      setSchedules(prev =>
        prev.map(s => (s.id === schedule.id ? { ...s, pinMessage: newPin } : s))
      );
      toast(newPin ? 'Mensagem será fixada no canal' : 'Mensagem não será mais fixada', 'success');
    } catch {
      toast('Erro ao alterar fixação', 'error');
    }
  };

  const openEditModal = (schedule: ScheduledPost) => {
    setEditingSchedule(schedule);
    if (schedule.nextRunAt) {
      const d = new Date(schedule.nextRunAt);
      setEditDate(d.toISOString().split('T')[0]);
      setEditTime(d.toTimeString().slice(0, 5));
    }
    setEditIntervalMin(schedule.intervalMin || '');
    setEditWindowStart(schedule.windowStart || '');
    setEditWindowEnd(schedule.windowEnd || '');
    setEditAutoDeleteMin(schedule.autoDeleteMin || 0);
  };

  const closeEditModal = () => {
    setEditingSchedule(null);
    setEditDate('');
    setEditTime('');
    setEditIntervalMin('');
    setEditWindowStart('');
    setEditWindowEnd('');
    setEditAutoDeleteMin(0);
  };

  const handleSaveEdit = async () => {
    if (!editingSchedule) return;
    if (!editDate || !editTime) {
      toast('Preencha data e horário', 'error');
      return;
    }

    if (editingSchedule.scheduleType === 'interval') {
      const intervalVal = Number(editIntervalMin);
      if (!editIntervalMin || intervalVal < 5) {
        toast('O intervalo mínimo é de 5 minutos', 'error');
        return;
      }
      if ((editWindowStart && !editWindowEnd) || (!editWindowStart && editWindowEnd)) {
        toast('Informe início e fim da janela de horário', 'error');
        return;
      }
    }

    setSaving(true);
    try {
      const [year, month, day] = editDate.split('-').map(Number);
      const [hours, minutes] = editTime.split(':').map(Number);
      
      const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}T${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:00-03:00`;
      
      const patchData: {
        nextRunAt: string;
        intervalMin?: number;
        windowStart?: string;
        windowEnd?: string;
        autoDeleteMin?: number;
      } = { nextRunAt: dateStr, autoDeleteMin: editAutoDeleteMin };

      if (editingSchedule.scheduleType === 'interval') {
        patchData.intervalMin = Number(editIntervalMin);
        patchData.windowStart = editWindowStart;
        patchData.windowEnd = editWindowEnd;
      }

      await updateScheduleTime(editingSchedule.id, patchData);
      
      setSchedules(prev =>
        prev.map(s => {
          if (s.id === editingSchedule.id) {
            return {
              ...s,
              nextRunAt: new Date(dateStr).toISOString(),
              scheduleTime: editingSchedule.scheduleType !== 'once' ? editTime : s.scheduleTime,
              intervalMin: editingSchedule.scheduleType === 'interval' ? Number(editIntervalMin) : s.intervalMin,
              windowStart: editingSchedule.scheduleType === 'interval' ? editWindowStart : s.windowStart,
              windowEnd: editingSchedule.scheduleType === 'interval' ? editWindowEnd : s.windowEnd,
              autoDeleteMin: editAutoDeleteMin,
            };
          }
          return s;
        })
      );
      
      toast('Agendamento atualizado com sucesso', 'success');
      closeEditModal();
    } catch {
      toast('Erro ao atualizar agendamento', 'error');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="tab-content-wrapper flex flex-col items-center justify-center py-16 space-y-3">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent" />
        <span className="text-xs font-semibold text-muted-foreground">Carregando agendamentos...</span>
      </div>
    );
  }

  return (
    <div className="tab-content-wrapper space-y-4 relative pb-10">
      {/* Section Header */}
      <div className="flex items-center justify-between px-1 pt-1">
        <h3 className="text-sm font-bold text-foreground">
          Agendamentos Ativos
        </h3>
        <Badge variant="secondary" className="rounded-full px-2.5 py-0.5 text-xs font-semibold bg-muted/60 text-muted-foreground">
          {activeScheduleCount}
        </Badge>
      </div>

      {/* Empty State */}
      {schedules.length === 0 ? (
        <div className="rounded-2xl border border-border/80 bg-card p-8 text-center space-y-3 shadow-sm">
          <div className="flex items-center justify-center size-14 rounded-2xl bg-muted/30 text-muted-foreground mx-auto">
            <Calendar size={28} className="opacity-50" />
          </div>
          <div>
            <p className="text-sm font-bold text-foreground">Nenhum agendamento encontrado</p>
            <p className="text-xs text-muted-foreground mt-1 max-w-xs mx-auto">
              {channelId ? 'Nenhum agendamento ativo registrado para este canal.' : 'Nenhum agendamento ativo encontrado nos seus canais.'}
            </p>
          </div>
          <div className="pt-2">
            <Button variant="outline" size="sm" className="rounded-xl text-xs font-bold text-accent border-accent/30" onClick={() => setShowCreateModal(true)}>
              <Plus size={14} className="mr-1" /> Criar Novo Agendamento
            </Button>
          </div>
        </div>
      ) : (
        /* Schedule Cards List */
        <div className="space-y-3">
          {schedules.map(schedule => {
            const normalizedStatus = schedule.status.toLowerCase();
            const st = statusConfig[normalizedStatus] || { label: 'Desconhecido', badgeClass: 'bg-muted/40 text-muted-foreground border-border/60', dotClass: 'bg-muted-foreground' };
            const typeLabel = scheduleTypeLabels[schedule.scheduleType] || 'Outro';

            return (
              <div
                key={schedule.id}
                className="rounded-2xl border border-border/80 bg-card p-4 space-y-3.5 shadow-sm transition-all hover:border-border"
              >
                {/* Header Row */}
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="flex items-center justify-center size-10 rounded-xl bg-purple-500/15 text-purple-400 border border-purple-500/25 shrink-0">
                      <Calendar size={19} />
                    </div>
                    <div className="min-w-0">
                      <h4 className="text-sm font-bold text-foreground truncate leading-tight">
                        {schedule.channelTitle || 'Sem Canal'}
                      </h4>
                      <span className="text-xs text-muted-foreground/80 font-medium">
                        {typeLabel}
                      </span>
                    </div>
                  </div>

                  {/* Status Badge */}
                  <Badge variant="outline" className={`text-[10px] font-semibold px-2.5 py-1 rounded-lg border shrink-0 flex items-center gap-1.5 ${st.badgeClass}`}>
                    <span className={`size-1.5 rounded-full ${st.dotClass}`} />
                    {st.label}
                  </Badge>
                </div>

                {/* Error Banner */}
                {schedule.lastError && (
                  <div className="text-xs font-medium text-rose-400 bg-rose-500/10 border border-rose-500/20 p-2.5 rounded-xl flex items-start gap-2">
                    <AlertCircle size={15} className="shrink-0 mt-0.5" />
                    <span className="break-all">{schedule.lastError}</span>
                  </div>
                )}

                {/* Next Post Row */}
                <div className="flex items-start gap-3 pt-1 pb-0.5 text-xs">
                  <div className="flex size-8 items-center justify-center rounded-lg bg-blue-500/15 text-blue-500 border border-blue-500/25 shrink-0">
                    <Clock size={15} />
                  </div>
                  <div>
                    <span className="block text-[11px] text-muted-foreground font-medium">Próxima postagem</span>
                    <span className="text-sm font-semibold text-foreground leading-snug">
                      {formatSmartNextRun(schedule.nextRunAt)}
                    </span>
                  </div>
                </div>

                {/* Separador Intermediário: Alinhado com o início do texto "Próxima postagem" */}
                <div className="h-px bg-muted-foreground/20 ml-11 my-1 shrink-0" />

                {/* Interval / Window Row */}
                <div className="flex items-start gap-3 py-0.5 text-xs">
                  <div className="flex size-8 items-center justify-center rounded-lg bg-blue-500/15 text-blue-500 border border-blue-500/25 shrink-0">
                    <RotateCw size={15} />
                  </div>
                  <div>
                    <span className="block text-sm font-semibold text-foreground leading-snug">
                      {schedule.scheduleType === 'interval' && schedule.intervalMin
                        ? `A cada ${schedule.intervalMin} ${schedule.intervalMin === 1 ? 'minuto' : 'minutos'}`
                        : (schedule.scheduleTime ? `Horário: ${schedule.scheduleTime}` : 'Fila contínua')}
                    </span>
                    {schedule.scheduleType === 'interval' && schedule.windowStart && schedule.windowEnd && (
                      <span className="block mt-0.5 text-[11px] text-muted-foreground font-medium">
                        Janela: {schedule.windowStart} – {schedule.windowEnd}
                      </span>
                    )}
                    {schedule.scheduleTime && schedule.scheduleType !== 'interval' && schedule.scheduleDays && (
                      <span className="block mt-0.5 text-[11px] text-muted-foreground font-medium">
                        Dias: {schedule.scheduleDays}
                      </span>
                    )}
                  </div>
                </div>

                {/* Separador Rodapé: Antes do Menu de Ações */}
                <div className="h-px bg-muted-foreground/20 w-full mt-2 mb-0.5 shrink-0" />

                {/* Action Toolbar Footer */}
                <div className="pt-1 flex items-center justify-between">
                  <button
                    type="button"
                    onClick={() => openEditModal(schedule)}
                    className="flex-1 flex flex-col items-center justify-center gap-1 py-1.5 px-1 rounded-lg text-[10px] font-semibold text-blue-400 hover:bg-blue-500/10 active:bg-blue-500/15 transition-colors cursor-pointer"
                  >
                    <Edit3 size={15} />
                    <span>Editar</span>
                  </button>

                  <div className="w-[1.5px] h-7 bg-muted-foreground/35 shrink-0 my-auto" />

                  <button
                    type="button"
                    onClick={() => handleTogglePin(schedule)}
                    className="flex-1 flex flex-col items-center justify-center gap-1 py-1.5 px-1 rounded-lg text-[10px] font-semibold text-amber-400 hover:bg-amber-500/10 active:bg-amber-500/15 transition-colors cursor-pointer"
                  >
                    {schedule.pinMessage ? <PinOff size={15} /> : <Pin size={15} />}
                    <span>{schedule.pinMessage ? 'Desfixar' : 'Fixar'}</span>
                  </button>

                  <div className="w-[1.5px] h-7 bg-muted-foreground/35 shrink-0 my-auto" />

                  <button
                    type="button"
                    onClick={() => handleTogglePause(schedule)}
                    className="flex-1 flex flex-col items-center justify-center gap-1 py-1.5 px-1 rounded-lg text-[10px] font-semibold text-amber-400 hover:bg-amber-500/10 active:bg-amber-500/15 transition-colors cursor-pointer"
                  >
                    {normalizedStatus === 'paused' ? (
                      <>
                        <Play size={15} className="text-emerald-400" />
                        <span className="text-emerald-400">Retomar</span>
                      </>
                    ) : (
                      <>
                        <Pause size={15} />
                        <span>Pausar</span>
                      </>
                    )}
                  </button>

                  <div className="w-[1.5px] h-7 bg-muted-foreground/35 shrink-0 my-auto" />

                  <button
                    type="button"
                    onClick={() => setConfirmDeleteSchedule(schedule)}
                    className="flex-1 flex flex-col items-center justify-center gap-1 py-1.5 px-1 rounded-lg text-[10px] font-semibold text-rose-400 hover:bg-rose-500/10 active:bg-rose-500/15 transition-colors cursor-pointer"
                  >
                    <Trash2 size={15} />
                    <span>Excluir</span>
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Floating Action Button (FAB) */}
      <button
        type="button"
        onClick={() => setShowCreateModal(true)}
        className={`schedule-fab fixed size-14 rounded-full bg-accent hover:brightness-95 text-white shadow-lg border border-blue-300/25 flex items-center justify-center font-bold text-2xl cursor-pointer transition-all active:scale-95 z-40 ${channelId ? 'schedule-fab-above-nav' : ''}`}
        title="Novo Agendamento"
        aria-label="Criar novo agendamento"
      >
        <Plus size={26} />
      </button>

      {/* Modal de Criação / Instruções de Agendamento */}
      <Dialog open={showCreateModal} onOpenChange={setShowCreateModal}>
        <DialogContent className="sm:max-w-md p-6 bg-card text-card-foreground border border-border/80 shadow-2xl rounded-2xl">
          <DialogHeader className="items-center text-center gap-2">
            <div className="flex size-12 items-center justify-center rounded-2xl bg-accent/15 text-accent border border-accent/30">
              <Sparkles size={24} />
            </div>
            <DialogTitle className="text-lg font-bold text-foreground">Novo Agendamento</DialogTitle>
            <DialogDescription className="text-xs text-muted-foreground">
              Para agendar postagens automáticas com o FreddyBot, você pode usar o **PostBuilder** diretamente no chat do bot no Telegram.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-2 text-xs">
            <div className="p-3 rounded-xl bg-muted/20 border border-border/60 flex items-start gap-2.5">
              <Info size={16} className="text-accent shrink-0 mt-0.5" />
              <div className="space-y-1">
                <span className="font-bold text-foreground block">Como agendar no Telegram:</span>
                <ol className="list-decimal list-inside space-y-1 text-muted-foreground font-medium">
                  <li>Envie uma mídia ou mensagem ao bot no Telegram.</li>
                  <li>Clique em <strong>Agendar Post</strong> no menu interativo.</li>
                  <li>Escolha a data, o horário ou o intervalo de repetição.</li>
                </ol>
              </div>
            </div>
          </div>

          <div className="pt-2">
            <Button
              className="w-full rounded-xl h-10 text-xs font-bold bg-accent text-accent-foreground shadow-sm"
              onClick={() => setShowCreateModal(false)}
            >
              Entendido
            </Button>
          </div>
        </DialogContent>
      </Dialog>

      {/* Confirmation Modal for Deleting Schedule */}
      {confirmDeleteSchedule && (
        <ConfirmModal
          open={!!confirmDeleteSchedule}
          onClose={() => setConfirmDeleteSchedule(null)}
          onConfirm={handleConfirmDelete}
          title="Excluir Agendamento"
          message={`Tem certeza que deseja excluir o agendamento de "${confirmDeleteSchedule.channelTitle || 'Canal'}"? Esta ação não pode ser desfeita.`}
          confirmText="Sim, Excluir"
          danger
        />
      )}

      {/* Edit Schedule Modal */}
      {editingSchedule && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4">
          <div className="bg-card border border-border/80 rounded-2xl p-5 w-full max-w-md shadow-2xl space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-base font-bold text-foreground">Editar Agendamento</h3>
              <Badge variant="outline" className="text-[10px] font-semibold">
                {scheduleTypeLabels[editingSchedule.scheduleType] || 'Outro'}
              </Badge>
            </div>
            
            <div className="space-y-3 text-xs">
              <div>
                <label className="block font-semibold mb-1 text-muted-foreground">Data do Próximo Envio</label>
                <input
                  type="date"
                  value={editDate}
                  onChange={(e) => setEditDate(e.target.value)}
                  className="w-full px-3 py-2 border border-border/80 rounded-xl bg-background text-foreground font-medium"
                />
              </div>
              
              <div>
                <label className="block font-semibold mb-1 text-muted-foreground">Horário do Próximo Envio</label>
                <input
                  type="time"
                  value={editTime}
                  onChange={(e) => setEditTime(e.target.value)}
                  className="w-full px-3 py-2 border border-border/80 rounded-xl bg-background text-foreground font-medium"
                />
              </div>

              {editingSchedule.scheduleType === 'interval' && (
                <>
                  <div>
                    <label className="block font-semibold mb-1 text-muted-foreground">Intervalo (em minutos)</label>
                    <input
                      type="number"
                      min="5"
                      placeholder="Ex: 30"
                      value={editIntervalMin}
                      onChange={(e) => setEditIntervalMin(e.target.value === '' ? '' : Number(e.target.value))}
                      className="w-full px-3 py-2 border border-border/80 rounded-xl bg-background text-foreground font-medium"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block font-semibold mb-1 text-muted-foreground text-[11px]">Janela Início</label>
                      <input
                        type="time"
                        value={editWindowStart}
                        onChange={(e) => setEditWindowStart(e.target.value)}
                        className="w-full px-3 py-2 border border-border/80 rounded-xl bg-background text-foreground text-xs font-medium"
                      />
                    </div>
                    <div>
                      <label className="block font-semibold mb-1 text-muted-foreground text-[11px]">Janela Fim</label>
                      <input
                        type="time"
                        value={editWindowEnd}
                        onChange={(e) => setEditWindowEnd(e.target.value)}
                        className="w-full px-3 py-2 border border-border/80 rounded-xl bg-background text-foreground text-xs font-medium"
                      />
                    </div>
                  </div>
                </>
              )}

              <div>
                <label className="block font-semibold mb-1 text-muted-foreground">⏱️ Auto-Destruição da Mensagem</label>
                <select
                  value={editAutoDeleteMin}
                  onChange={(e) => setEditAutoDeleteMin(Number(e.target.value))}
                  className="w-full px-3 py-2 border border-border/80 rounded-xl bg-background text-foreground text-xs font-medium cursor-pointer"
                >
                  <option value={0}>Desativada (Mensagem permanente)</option>
                  <option value={60}>⏱️ Apagar 1 hora após envio</option>
                  <option value={360}>⏱️ Apagar 6 horas após envio</option>
                  <option value={720}>⏱️ Apagar 12 horas após envio</option>
                  <option value={1440}>⏱️ Apagar 24 horas após envio</option>
                  <option value={2880}>⏱️ Apagar 48 horas após envio</option>
                </select>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-2">
              <Button
                variant="outline"
                className="rounded-xl h-10 px-4 text-xs font-semibold border-border"
                onClick={closeEditModal}
                disabled={saving}
              >
                Cancelar
              </Button>
              <Button
                className="rounded-xl h-10 px-4 text-xs font-bold bg-accent text-accent-foreground"
                onClick={handleSaveEdit}
                disabled={saving}
              >
                {saving ? 'Salvando...' : 'Salvar Alterações'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
