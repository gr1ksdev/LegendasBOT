import { useState, useEffect, useCallback } from 'react';
import { UserCaptionTemplate, Button as TemplateButton } from '../types';
import {
  listUserCaptionTemplates,
  createUserCaptionTemplate,
  updateUserCaptionTemplate,
  deleteUserCaptionTemplate,
  createUserCaptionTemplateButton,
  updateUserCaptionTemplateButton,
  deleteUserCaptionTemplateButton,
  updateUserCaptionTemplateLayout
} from '../api';
import { ButtonGrid } from './ButtonGrid';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Input } from './ui/input';
import { ConfirmModal } from './ConfirmModal';
import {
  Plus, Trash2, Loader2, Check, ChevronRight, ChevronUp,
  Hash, Layers, LayoutGrid, Lightbulb, AlertCircle
} from 'lucide-react';

interface Props {
  toast: (msg: string, type: 'success' | 'error' | 'info') => void;
}

export function UserTemplatesManager({ toast }: Props) {
  const [templates, setTemplates] = useState<UserCaptionTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  
  // Compact create template action state
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [savingNew, setSavingNew] = useState(false);

  // Confirm delete modal state
  const [confirmDeleteTemplate, setConfirmDeleteTemplate] = useState<UserCaptionTemplate | null>(null);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const captions = await listUserCaptionTemplates();
      setTemplates(captions || []);
    } catch {
      toast('Erro ao carregar templates', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    load();
  }, [load]);

  const handleCreate = async () => {
    if (!newName.trim()) {
      toast('Digite um nome para o template', 'error');
      return;
    }
    setSavingNew(true);
    try {
      const tpl = await createUserCaptionTemplate(newName.trim(), '');
      if (tpl) {
        tpl.buttons = tpl.buttons || [];
        setTemplates(prev => [tpl, ...prev]);
        setExpandedId(tpl.id); // Automatically expand the newly created template
      }
      setNewName('');
      setShowCreate(false);
      toast('Template criado com sucesso', 'success');
    } catch {
      toast('Erro ao criar template', 'error');
    } finally {
      setSavingNew(false);
    }
  };

  const handleUpdateCaption = async (id: string, code: string, caption: string) => {
    try {
      await updateUserCaptionTemplate(id, code, caption);
      setTemplates(prev =>
        prev.map(t => (t.id === id ? { ...t, code, caption } : t))
      );
      toast('Template salvo', 'success');
    } catch {
      toast('Erro ao atualizar template', 'error');
    }
  };

  const executeDelete = async (id: string) => {
    setDeletingId(id);
    try {
      await deleteUserCaptionTemplate(id);
      setTemplates(prev => prev.filter(t => t.id !== id));
      if (expandedId === id) setExpandedId(null);
      toast('Template excluído com sucesso', 'success');
    } catch {
      toast('Erro ao excluir template', 'error');
    } finally {
      setDeletingId(null);
      setConfirmDeleteTemplate(null);
    }
  };

  const handleAddButton = async (templateId: string, btn: TemplateButton) => {
    try {
      await createUserCaptionTemplateButton(templateId, btn.nameButton, btn.buttonUrl, btn.style);
      load();
      toast('Botão adicionado', 'success');
    } catch {
      toast('Erro ao adicionar botão', 'error');
    }
  };

  const handleEditButton = async (templateId: string, buttonId: string, updates: any) => {
    try {
      await updateUserCaptionTemplateButton(templateId, buttonId, updates.nameButton, updates.buttonUrl || '', updates.style);
      setTemplates(prev =>
        prev.map(t => {
          if (t.id === templateId) {
            return {
              ...t,
              buttons: t.buttons.map(b => (b.buttonId === buttonId ? { ...b, ...updates } : b))
            };
          }
          return t;
        })
      );
    } catch {
      toast('Erro ao atualizar botão', 'error');
    }
  };

  const handleDeleteButton = async (templateId: string, buttonId: string) => {
    try {
      await deleteUserCaptionTemplateButton(templateId, buttonId);
      setTemplates(prev =>
        prev.map(t => {
          if (t.id === templateId) {
            return { ...t, buttons: t.buttons.filter(b => b.buttonId !== buttonId) };
          }
          return t;
        })
      );
    } catch {
      toast('Erro ao remover botão', 'error');
    }
  };

  const handleMoveButton = async (templateId: string, layout: { buttonId: string }[][]) => {
    try {
      await updateUserCaptionTemplateLayout(templateId, layout);

      setTemplates(prev =>
        prev.map(t => {
          if (t.id !== templateId) return t;

          const positionMap = new Map<string, { x: number; y: number }>();
          layout.forEach((row, y) => {
            row.forEach((col, x) => {
              if (col && col.buttonId) {
                positionMap.set(col.buttonId, { x, y });
              }
            });
          });

          const newButtons = t.buttons.map(b => {
            const pos = positionMap.get(b.buttonId);
            if (pos) {
              return { ...b, positionX: pos.x, positionY: pos.y };
            }
            return b;
          });

          return { ...t, buttons: newButtons };
        })
      );

      toast('Posição salva', 'success');
    } catch {
      toast('Erro ao atualizar layout', 'error');
    }
  };

  return (
    <div className="tab-content-wrapper space-y-4 relative pb-10">
      {/* Retractable Create Template Action */}
      <div className="rounded-2xl border border-border/80 bg-card p-3.5 shadow-sm transition-all hover:border-border">
        {!showCreate ? (
          <button
            type="button"
            onClick={() => setShowCreate(true)}
            className="w-full flex items-center justify-between gap-3 text-left group cursor-pointer"
          >
            <div className="flex items-center gap-3 min-w-0">
              <div className="flex items-center justify-center size-10 rounded-xl bg-blue-500/15 text-blue-400 border border-blue-500/25 shrink-0 group-hover:scale-105 transition-transform">
                <Plus size={19} />
              </div>
              <div className="min-w-0">
                <h4 className="text-sm font-bold text-foreground leading-tight flex items-center gap-1.5">
                  Novo template
                </h4>
                <p className="text-xs text-muted-foreground/80 font-medium truncate mt-0.5">
                  Crie um novo template de legenda e botões.
                </p>
              </div>
            </div>
            <div className="flex items-center justify-center size-8 rounded-lg bg-muted/30 text-muted-foreground shrink-0 group-hover:text-foreground">
              <ChevronRight size={16} />
            </div>
          </button>
        ) : (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <div className="flex items-center justify-center size-7 rounded-lg bg-blue-500/15 text-blue-400 border border-blue-500/25 shrink-0">
                  <Plus size={15} />
                </div>
                <h4 className="text-xs font-bold text-foreground">Novo template</h4>
              </div>
              <Button
                variant="ghost"
                size="sm"
                className="h-7 w-7 p-0 rounded-lg text-muted-foreground hover:text-foreground"
                onClick={() => { setShowCreate(false); setNewName(''); }}
              >
                <ChevronUp size={16} />
              </Button>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-medium text-muted-foreground block">
                Nome do template
              </label>
              <Input
                type="text"
                placeholder="Ex: promo_blackfriday"
                value={newName}
                onChange={e => setNewName(e.target.value)}
                className="h-10 text-xs rounded-xl bg-background border-border/80"
                autoFocus
                onKeyDown={e => {
                  if (e.key === 'Enter' && newName.trim() && !savingNew) {
                    e.preventDefault();
                    handleCreate();
                  }
                }}
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-1">
              <Button
                variant="outline"
                size="sm"
                className="rounded-xl h-9 text-xs font-semibold border-border"
                onClick={() => { setShowCreate(false); setNewName(''); }}
                disabled={savingNew}
              >
                Cancelar
              </Button>
              <Button
                variant="default"
                size="sm"
                onClick={handleCreate}
                disabled={savingNew || !newName.trim()}
                className="rounded-xl h-9 px-4 text-xs font-bold bg-accent hover:bg-accent/90 text-accent-foreground"
              >
                {savingNew ? (
                  <>
                    <Loader2 size={14} className="mr-1.5 animate-spin" /> Criando...
                  </>
                ) : (
                  <>
                    <Plus size={14} className="mr-1.5" /> Adicionar
                  </>
                )}
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* Section Header */}
      <div className="flex items-center justify-between px-1 pt-1">
        <h3 className="text-sm font-bold text-foreground">Meus Templates</h3>
        <Badge variant="secondary" className="rounded-full px-2.5 py-0.5 text-xs font-semibold bg-muted/60 text-muted-foreground">
          {templates.length}
        </Badge>
      </div>

      {/* Loading State */}
      {loading && (
        <div className="flex justify-center py-8">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      )}

      {/* Empty State */}
      {!loading && templates.length === 0 && (
        <div className="rounded-2xl border border-border/80 bg-card p-8 text-center space-y-3 shadow-sm">
          <div className="flex items-center justify-center size-14 rounded-2xl bg-muted/30 text-muted-foreground mx-auto">
            <Layers size={28} className="opacity-50" />
          </div>
          <div>
            <p className="text-sm font-bold text-foreground">Nenhum template encontrado</p>
            <p className="text-xs text-muted-foreground mt-1 max-w-xs mx-auto">
              Crie um novo template de legenda e botões usando o botão acima.
            </p>
          </div>
        </div>
      )}

      {/* Templates List */}
      {!loading && templates.length > 0 && (
        <div className="space-y-3">
          {templates.map(tpl => {
            const title = tpl.code || 'Sem Nome';
            const subTitle = tpl.caption || 'Sem legenda configurada';
            const buttonsCount = tpl.buttons.length;
            const isExpanded = expandedId === tpl.id;

            return (
              <div
                key={tpl.id}
                className="rounded-2xl border border-border/80 bg-card p-3.5 space-y-3 shadow-sm transition-all hover:border-border overflow-hidden"
              >
                {/* Header Row (Entire row clickable to toggle expand/collapse) */}
                <div
                  className="flex items-center justify-between gap-3 cursor-pointer select-none"
                  onClick={() => setExpandedId(isExpanded ? null : tpl.id)}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="flex items-center justify-center size-10 rounded-xl bg-blue-500/15 text-blue-400 border border-blue-500/25 shrink-0">
                      <Hash size={18} />
                    </div>
                    <div className="min-w-0">
                      <h4 className="text-sm font-bold text-foreground truncate leading-tight">
                        {title}
                      </h4>
                      <p className="text-xs text-muted-foreground/80 font-medium truncate mt-0.5">
                        {subTitle}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <Badge variant="secondary" className="rounded-md px-2 py-0.5 text-[10px] font-bold bg-muted/60 text-muted-foreground">
                      {buttonsCount} {buttonsCount === 1 ? 'botão' : 'botões'}
                    </Badge>
                    
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        setConfirmDeleteTemplate(tpl);
                      }}
                      className="size-8 rounded-lg flex items-center justify-center text-rose-400 hover:bg-rose-500/10 active:bg-rose-500/15 transition-colors cursor-pointer"
                      title="Excluir template"
                    >
                      {deletingId === tpl.id ? (
                        <Loader2 size={14} className="animate-spin text-rose-400" />
                      ) : (
                        <Trash2 size={14} />
                      )}
                    </button>

                    <div className="size-6 flex items-center justify-center text-muted-foreground">
                      {isExpanded ? <ChevronUp size={18} /> : <ChevronRight size={18} />}
                    </div>
                  </div>
                </div>

                {/* Expanded Template Editor */}
                {isExpanded && (
                  <TemplateEditor
                    template={tpl}
                    onUpdateCaption={handleUpdateCaption}
                    onAddButton={btn => handleAddButton(tpl.id, btn)}
                    onEditButton={(buttonId, updates) => handleEditButton(tpl.id, buttonId, updates)}
                    onDeleteButton={buttonId => handleDeleteButton(tpl.id, buttonId)}
                    onMoveButton={layout => handleMoveButton(tpl.id, layout)}
                    onRequestDelete={templateToDel => setConfirmDeleteTemplate(templateToDel)}
                    toast={toast}
                  />
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Confirmation Modal for Destructive Delete */}
      <ConfirmModal
        open={confirmDeleteTemplate !== null}
        onClose={() => setConfirmDeleteTemplate(null)}
        onConfirm={() => {
          if (confirmDeleteTemplate) executeDelete(confirmDeleteTemplate.id);
        }}
        title={`Excluir template "${confirmDeleteTemplate?.code}"?`}
        message="Essa ação não poderá ser desfeita e removerá permanentemente a legenda e os botões configurados neste template."
        danger={true}
        confirmText="Excluir"
      />
    </div>
  );
}

function TemplateEditor({
  template,
  onUpdateCaption,
  onAddButton,
  onEditButton,
  onDeleteButton,
  onMoveButton,
  onRequestDelete,
  toast,
}: {
  template: UserCaptionTemplate;
  onUpdateCaption: (id: string, code: string, caption: string) => void;
  onAddButton: (btn: TemplateButton) => void;
  onEditButton: (buttonId: string, updates: any) => void;
  onDeleteButton: (buttonId: string) => void;
  onMoveButton: (layout: { buttonId: string }[][]) => void;
  onRequestDelete: (template: UserCaptionTemplate) => void;
  toast: (msg: string, type: 'success' | 'error' | 'info') => void;
}) {
  const [code, setCode] = useState(template.code);
  const [caption, setCaption] = useState(template.caption);
  const [saving, setSaving] = useState(false);
  const [savedNotice, setSavedNotice] = useState(false);

  useEffect(() => {
    setCode(template.code);
    setCaption(template.caption);
    setSavedNotice(false);
  }, [template.id]);

  const handleSaveAll = async () => {
    if (!code.trim()) {
      toast('Nome curto é obrigatório', 'error');
      return;
    }
    setSaving(true);
    try {
      await onUpdateCaption(template.id, code.trim(), caption);
      setSavedNotice(true);
      setTimeout(() => setSavedNotice(false), 2500);
    } catch {
      toast('Erro ao salvar template', 'error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="pt-3 border-t border-border/60 space-y-4">
      {/* Informações do Template */}
      <div className="space-y-3">
        <h5 className="text-xs font-bold text-foreground flex items-center gap-1.5">
          Informações do template
        </h5>

        <div className="space-y-1">
          <label className="text-[11px] font-medium text-muted-foreground block">
            Nome curto
          </label>
          <Input
            type="text"
            value={code}
            onChange={e => setCode(e.target.value)}
            className="h-10 text-xs rounded-xl bg-background border-border/80"
            placeholder="Ex: promo"
          />
        </div>

        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <label className="text-[11px] font-medium text-muted-foreground">
              Legenda (opcional)
            </label>
            <span className="text-[10px] text-muted-foreground/70 font-normal">
              Use variáveis como &#123;nome&#125;
            </span>
          </div>
          <textarea
            value={caption}
            onChange={e => setCaption(e.target.value)}
            rows={3}
            placeholder="Digite a legenda do template..."
            className="w-full px-3 py-2 text-xs border border-border/80 rounded-xl bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-accent resize-y min-h-[80px]"
          />
        </div>
      </div>

      {/* Divisor */}
      <div className="h-px bg-border/60 w-full shrink-0" />

      {/* Botões Inline Header */}
      <div className="space-y-3">
        <div className="flex items-center gap-2.5">
          <div className="flex items-center justify-center size-7 rounded-lg bg-blue-500/15 text-blue-400 border border-blue-500/25 shrink-0">
            <LayoutGrid size={15} />
          </div>
          <div>
            <h5 className="text-xs font-bold text-foreground leading-none">
              Botões Inline
            </h5>
            <p className="text-[11px] text-muted-foreground/80 font-medium mt-0.5">
              Configure a grade de botões da mensagem.
            </p>
          </div>
        </div>

        {/* Button Grid Editor Component */}
        <ButtonGrid
          buttons={template.buttons}
          reactions=""
          reactionPosition={-1}
          channelId={0}
          onAdd={onAddButton}
          onEdit={onEditButton}
          onDelete={onDeleteButton}
          onMove={(buttonId, x, y) => {
            const moved = template.buttons.map(b =>
              b.buttonId === buttonId ? { ...b, positionX: x, positionY: y } : b
            );
            const rows = Array.from(
              { length: Math.max(...moved.map(b => b.positionY), 0) + 1 },
              (_, row) =>
                moved
                  .filter(b => b.positionY === row)
                  .sort((a, b) => a.positionX - b.positionX)
                  .map(b => ({ buttonId: b.buttonId }))
            );
            onMoveButton(rows);
          }}
          onMoveReactions={() => {}}
          hideReactions={true}
        />
      </div>

      {/* Divisor */}
      <div className="h-px bg-border/60 w-full shrink-0" />

      {/* Save Button */}
      <Button
        type="button"
        onClick={handleSaveAll}
        disabled={saving}
        className="w-full h-11 bg-accent hover:bg-accent/90 text-accent-foreground font-bold text-xs rounded-xl shadow-sm flex items-center justify-center gap-2 cursor-pointer transition-all"
      >
        {saving ? (
          <>
            <Loader2 size={16} className="animate-spin" /> Salvando...
          </>
        ) : savedNotice ? (
          <>
            <Check size={16} className="text-emerald-400" /> Alterações salvas
          </>
        ) : (
          <>
            <Check size={16} /> Salvar alterações
          </>
        )}
      </Button>

      {/* Destructive Delete Button */}
      <div className="text-center pt-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => onRequestDelete(template)}
          className="text-xs font-semibold text-rose-400 hover:text-rose-300 hover:bg-rose-500/10 rounded-xl cursor-pointer"
        >
          <Trash2 size={14} className="mr-1.5" /> Excluir template
        </Button>
      </div>
    </div>
  );
}
