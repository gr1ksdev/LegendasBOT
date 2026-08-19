import { useState, useEffect } from 'react';
import { CustomCaption } from '../types';
import { createCustomCaption, updateCustomCaption, deleteCustomCaption } from '../api';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { Plus, Pencil, Trash2, Hash, Link, X, Check, Loader2 } from 'lucide-react';

interface CustomCaptionsCardProps {
  channelId: number;
  captions: CustomCaption[];
  toast: (msg: string, type: 'success' | 'error' | 'info') => void;
}

export function CustomCaptionsCard({ channelId, captions, toast }: CustomCaptionsCardProps) {
  const [items, setItems] = useState<CustomCaption[]>(captions);
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [code, setCode] = useState('');
  const [caption, setCaption] = useState('');
  const [linkPreview, setLinkPreview] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setItems(captions);
  }, [captions]);

  const resetForm = () => {
    setCode('');
    setCaption('');
    setLinkPreview(false);
    setShowForm(false);
    setEditingId(null);
  };

  const openEdit = (item: CustomCaption) => {
    setEditingId(item.captionId);
    setCode(item.code);
    setCaption(item.caption);
    setLinkPreview(item.linkPreview);
    setShowForm(true);
  };

  const handleSave = async () => {
    if (!code.trim() || !caption.trim()) {
      toast('Código e legenda são obrigatórios', 'error');
      return;
    }
    setSaving(true);
    try {
      if (editingId) {
        await updateCustomCaption(channelId, editingId, { code: code.trim(), caption: caption.trim(), linkPreview });
        setItems(prev => prev.map(i => i.captionId === editingId ? { ...i, code: code.trim(), caption: caption.trim(), linkPreview } : i));
        toast('Legenda customizada atualizada', 'success');
      } else {
        const data = await createCustomCaption(channelId, { code: code.trim(), caption: caption.trim(), linkPreview });
        if (data) {
          setItems(prev => [...prev, data as CustomCaption]);
        }
        toast('Legenda customizada criada', 'success');
      }
      resetForm();
    } catch {
      toast('Erro ao salvar legenda customizada', 'error');
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteCustomCaption(channelId, id);
      setItems(prev => prev.filter(i => i.captionId !== id));
      toast('Legenda customizada excluída', 'success');
    } catch {
      toast('Erro ao excluir legenda customizada', 'error');
    }
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between py-3 px-4">
        <CardTitle className="text-sm font-semibold">Legendas Customizadas</CardTitle>
        <div className="flex items-center gap-2">
          <Badge variant="secondary" className="text-[10px]">{items.length}</Badge>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => { resetForm(); setShowForm(true); }} title="Nova legenda customizada">
            <Plus className="h-4 w-4" />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="px-4 pb-4 space-y-2">
        {showForm && (
          <div className="rounded-lg border border-border p-3 space-y-2 bg-muted/30">
            <input
              type="text"
              placeholder="Código (ex: promo)"
              value={code}
              onChange={e => setCode(e.target.value)}
              className="w-full px-3 py-1.5 text-sm border rounded-md bg-background text-foreground"
            />
            <textarea
              placeholder="Texto da legenda"
              value={caption}
              onChange={e => setCaption(e.target.value)}
              rows={3}
              className="w-full px-3 py-1.5 text-sm border rounded-md bg-background text-foreground resize-none"
            />
            <label className="flex items-center gap-2 text-xs text-muted-foreground cursor-pointer">
              <input type="checkbox" checked={linkPreview} onChange={e => setLinkPreview(e.target.checked)} className="rounded" />
              Link Preview
            </label>
            <div className="flex justify-end gap-2 pt-1">
              <Button variant="ghost" size="sm" onClick={resetForm} disabled={saving}>
                <X className="h-3 w-3 mr-1" /> Cancelar
              </Button>
              <Button variant="default" size="sm" onClick={handleSave} disabled={saving}>
                {saving ? <Loader2 className="h-3 w-3 mr-1 animate-spin" /> : <Check className="h-3 w-3 mr-1" />}
                {editingId ? 'Atualizar' : 'Criar'}
              </Button>
            </div>
          </div>
        )}

        {items.length === 0 && !showForm && (
          <p className="text-xs text-muted-foreground text-center py-4">
            Nenhuma legenda customizada. Clique em <strong>+</strong> para criar.
          </p>
        )}

        {items.map(item => (
          <div key={item.captionId} className="flex items-start gap-2 rounded-lg border border-border p-3">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-1">
                <Hash className="h-3 w-3 text-muted-foreground shrink-0" />
                <span className="text-xs font-mono font-semibold text-accent">{item.code}</span>
                {item.linkPreview && (
                  <Link className="h-3 w-3 text-muted-foreground shrink-0" />
                )}
              </div>
              <p className="text-xs text-foreground line-clamp-2">{item.caption}</p>
              {item.buttons && item.buttons.length > 0 && (
                <p className="text-[10px] text-muted-foreground mt-1">{item.buttons.length} botão(ões)</p>
              )}
            </div>
            <div className="flex items-center gap-1 shrink-0">
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => openEdit(item)} title="Editar">
                <Pencil className="h-3 w-3 text-blue-500" />
              </Button>
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => handleDelete(item.captionId)} title="Excluir">
                <Trash2 className="h-3 w-3 text-red-500" />
              </Button>
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
