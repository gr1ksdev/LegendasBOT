import { useState, useEffect, useCallback } from 'react';
import { CaptionTemplate } from '../types';
import { listCaptionTemplates, saveCaptionTemplate, applyCaptionTemplate, deleteCaptionTemplate } from '../api';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Save, Upload, Trash2, Loader2, X, FileText } from 'lucide-react';

interface CaptionTemplateCardProps {
  channelId: number;
  toast: (msg: string, type: 'success' | 'error' | 'info') => void;
}

export function CaptionTemplateCard({ channelId, toast }: CaptionTemplateCardProps) {
  const [templates, setTemplates] = useState<CaptionTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [applying, setApplying] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [showNameInput, setShowNameInput] = useState(false);
  const [name, setName] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await listCaptionTemplates(channelId);
      setTemplates(data || []);
    } catch {
      toast('Erro ao carregar templates', 'error');
    } finally {
      setLoading(false);
    }
  }, [channelId, toast]);

  useEffect(() => {
    load();
  }, [load]);

  const handleSave = async () => {
    if (!name.trim()) {
      toast('Digite um nome para o template', 'error');
      return;
    }
    setSaving(true);
    try {
      await saveCaptionTemplate(channelId, name.trim());
      toast('Template salvo', 'success');
      setName('');
      setShowNameInput(false);
      load();
    } catch {
      toast('Erro ao salvar template', 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleApply = async (id: string) => {
    setApplying(id);
    try {
      await applyCaptionTemplate(channelId, id);
      toast('Template aplicado com sucesso!', 'success');
    } catch {
      toast('Erro ao aplicar template', 'error');
    } finally {
      setApplying(null);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Excluir este template permanentemente?')) return;
    setDeleting(id);
    try {
      await deleteCaptionTemplate(channelId, id);
      setTemplates(prev => prev.filter(t => t.id !== id));
      toast('Template excluído', 'success');
    } catch {
      toast('Erro ao excluir template', 'error');
    } finally {
      setDeleting(null);
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-3 px-4">
        <CardTitle className="text-sm font-semibold flex items-center gap-2">
          <FileText className="h-4 w-4" />
          Templates de Legenda
        </CardTitle>
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="text-[10px]">{templates.length}</Badge>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => setShowNameInput(true)} title="Salvar como template">
            <Save className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="px-4 pb-4 space-y-2">
        {showNameInput && (
          <div className="rounded-lg border border-border p-3 space-y-2 bg-muted/30">
            <p className="text-xs text-muted-foreground">Salvar estado atual (legendas + custom captions) como template</p>
            <input
              type="text"
              placeholder="Nome do template (ex: Natal 2026)"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full px-3 py-1.5 text-sm border rounded-md bg-background text-foreground"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => { setShowNameInput(false); setName(''); }} disabled={saving}>
                <X className="h-3 w-3 mr-1" /> Cancelar
              </Button>
              <Button variant="default" size="sm" onClick={handleSave} disabled={saving || !name.trim()}>
                {saving ? <Loader2 className="h-3 w-3 mr-1 animate-spin" /> : <Save className="h-3 w-3 mr-1" />}
                Salvar
              </Button>
            </div>
          </div>
        )}

        {loading && (
          <div className="flex justify-center py-4">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        )}

        {!loading && templates.length === 0 && (
          <p className="text-xs text-muted-foreground text-center py-4">
            Nenhum template salvo. Clique em <strong>Salvar</strong> para criar.
          </p>
        )}

        {templates.map(tpl => {
          let captionCount = 0;
          try {
            const data = JSON.parse(tpl.templateData);
            if (data.customCaptions) captionCount = data.customCaptions.length;
          } catch {}
          return (
            <div key={tpl.id} className="flex items-center gap-2 rounded-lg border border-border p-3">
              <div className="flex-1 min-w-0">
                <p className="text-xs font-semibold text-foreground truncate">{tpl.name}</p>
                <p className="text-[10px] text-muted-foreground">
                  {captionCount} legenda(s) customizada(s)
                </p>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 px-2 text-[11px] text-accent"
                  onClick={() => handleApply(tpl.id)}
                  disabled={applying === tpl.id}
                  title="Aplicar template"
                >
                  {applying === tpl.id ? (
                    <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                  ) : (
                    <Upload className="h-3 w-3 mr-1" />
                  )}
                  Aplicar
                </Button>
                <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => handleDelete(tpl.id)} disabled={deleting === tpl.id} title="Excluir">
                  {deleting === tpl.id ? <Loader2 className="h-3 w-3 animate-spin text-red-500" /> : <Trash2 className="h-3 w-3 text-red-500" />}
                </Button>
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
