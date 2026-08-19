import { useState, useRef, useCallback, useEffect } from 'react';
import { Button as ButtonType } from '../types';
import {
  Plus, Minus, GripVertical, Pencil, Trash2, ExternalLink,
  X, Check, AlertTriangle, Grid3X3
} from 'lucide-react';
import { Card } from './ui/card';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Badge } from './ui/badge';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter, DialogClose,
} from './ui/dialog';

interface Props {
  buttons: ButtonType[];
  reactions: string;
  reactionPosition: number;
  channelId: number;
  onAdd: (button: ButtonType) => void;
  onDelete: (buttonId: string) => void;
  onEdit: (buttonId: string, updates: Partial<ButtonType>) => void;
  onMove: (buttonId: string, x: number, y: number) => void;
  onMoveReactions: (y: number) => void;
  hideReactions?: boolean;
}

export function ButtonGrid({ buttons, reactions, reactionPosition, channelId, hideReactions, onAdd, onDelete, onEdit, onMove, onMoveReactions }: Props) {
  const [cols, setCols] = useState(() => Math.max(4, buttons.reduce((m, b) => Math.max(m, b.positionX), 0) + 1));
  const [rows, setRows] = useState(() => {
    const maxBtnY = buttons.reduce((m, b) => Math.max(m, b.positionY), -1);
    const activeY = hideReactions ? maxBtnY : Math.max(maxBtnY, reactionPosition);
    return Math.max(1, activeY + 1);
  });

  // Sync grid dimensions when props change (e.g. after move or external update)
  useEffect(() => {
    const maxBRow = buttons.reduce((m, b) => Math.max(m, b.positionY), -1);
    const activeY = hideReactions ? maxBRow : Math.max(maxBRow, reactionPosition);
    const neededRows = Math.max(1, activeY + 1);
    setRows(prev => Math.max(prev, neededRows));

    const maxBCol = buttons.reduce((m, b) => Math.max(m, b.positionX), -1);
    const neededCols = Math.max(1, maxBCol + 1);
    setCols(prev => Math.max(prev, neededCols));
  }, [buttons, reactionPosition, hideReactions]);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editName, setEditName] = useState('');
  const [editUrl, setEditUrl] = useState('');
  const [editStyle, setEditStyle] = useState('');
  const [addingAt, setAddingAt] = useState<{ x: number; y: number } | null>(null);
  const [newName, setNewName] = useState('');
  const [newUrl, setNewUrl] = useState('');
  const [newStyle, setNewStyle] = useState('');
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const [dragBtnId, setDragBtnId] = useState<string | null>(null);
  const [dragOverKey, setDragOverKey] = useState<string | null>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const cloneRef = useRef<HTMLDivElement | null>(null);
  const cellRectsRef = useRef<Map<string, DOMRect>>(new Map());
  const longPressRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isDragging = useRef(false);
  const touchStart = useRef({ x: 0, y: 0 });
  const raf = useRef(0);

  const btnAt = (x: number, y: number) => buttons.find(b => b.positionX === x && b.positionY === y);
  const cellKey = (x: number, y: number) => `${x},${y}`;

  const getButtonStyle = (style?: string) => {
    switch (style) {
      case 'primary':
        return {
          background: 'rgba(36, 129, 204, 0.25)',
          borderColor: 'rgba(36, 129, 204, 0.6)',
          color: '#60a5fa',
        };
      case 'success':
        return {
          background: 'rgba(14, 165, 115, 0.25)',
          borderColor: 'rgba(14, 165, 115, 0.6)',
          color: '#34d399',
        };
      case 'danger':
        return {
          background: 'rgba(232, 62, 62, 0.25)',
          borderColor: 'rgba(232, 62, 62, 0.6)',
          color: '#f87171',
        };
      default:
        return {};
    }
  };

  const processUrl = (url: string) => {
    const u = url.trim();
    const lower = u.toLowerCase();
    if (u.startsWith('@')) return `https://t.me/${u.slice(1).replace(/^\/+/, '')}`;
    if (lower.startsWith('t.me/')) return `https://t.me/${u.slice(5).replace(/^\/+/, '')}`;
    if (lower.startsWith('telegram.me/')) return `https://t.me/${u.slice('telegram.me/'.length).replace(/^\/+/, '')}`;
    if (!u.includes('://') && u.includes('.')) return `https://${u}`;
    return u;
  };

  const validateUrl = (url: string) => {
    const u = processUrl(url);
    if (!u) return false;
    try {
      const parsed = new URL(u);
      if (parsed.protocol === 'tg:') return true;
      if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return false;
      if (!parsed.hostname) return false;
      if ((parsed.hostname === 't.me' || parsed.hostname === 'telegram.me') && parsed.pathname.replace(/\//g, '') === '') return false;
      return true;
    } catch {
      return false;
    }
  };

  const cacheCells = useCallback(() => {
    cellRectsRef.current.clear();
    gridRef.current?.querySelectorAll('[data-cell]').forEach(el => {
      const k = el.getAttribute('data-cell');
      if (k) cellRectsRef.current.set(k, el.getBoundingClientRect());
    });
  }, []);

  const findCell = (px: number, py: number): string | null => {
    for (const [k, r] of cellRectsRef.current) {
      if (px >= r.left && px <= r.right && py >= r.top && py <= r.bottom) {
        return k;
      }
    }
    return null;
  };

  const onDragStart = useCallback((e: React.DragEvent, id: string) => {
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', id);
    setDragBtnId(id);
    
    // For Desktop: Ensure the drag ghost is the full element
    if (id === 'REACTIONS_ROW' && e.currentTarget) {
      // Some browsers need this to correctly render the ghost of a spanned element
      // const target = e.currentTarget as HTMLElement;
      // e.dataTransfer.setDragImage(target, target.offsetWidth / 2, target.offsetHeight / 2);
    }
  }, []);

  const onDragEnd = useCallback(() => { setDragBtnId(null); setDragOverKey(null); }, []);

  const onCellDragOver = useCallback((e: React.DragEvent, x: number, y: number) => {
    e.preventDefault();
    setDragOverKey(cellKey(x, y));
  }, []);

  const onCellDrop = useCallback((e: React.DragEvent, x: number, y: number) => {
    e.preventDefault();
    const id = e.dataTransfer.getData('text/plain');
    if (id === 'REACTIONS_ROW') {
      onMoveReactions(y);
    } else if (id) {
      if (!btnAt(x, y)) {
        onMove(id, x, y);
      }
    }
    setDragBtnId(null);
    setDragOverKey(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buttons, onMove, onMoveReactions]);

  const cleanTouch = useCallback(() => {
    if (longPressRef.current) { clearTimeout(longPressRef.current); longPressRef.current = null; }
    if (cloneRef.current) { cloneRef.current.remove(); cloneRef.current = null; }
    cancelAnimationFrame(raf.current);
    isDragging.current = false;
    setDragBtnId(null);
    setDragOverKey(null);
  }, []);

  const onTouchStart = useCallback((e: React.TouchEvent, id: string) => {
    const t = e.touches[0];
    const target = (e.target as HTMLElement).closest('[draggable]') as HTMLElement;
    if (!target) return;

    touchStart.current = { x: t.clientX, y: t.clientY };
    isDragging.current = false;
    
    longPressRef.current = setTimeout(() => {
      isDragging.current = true;
      setDragBtnId(id);
      setSelectedId(null);
      cacheCells();
      
      const c = target.cloneNode(true) as HTMLDivElement;
      c.className = `${target.className} drag-clone`;
      c.style.position = 'fixed';
      c.style.margin = '0';
      c.style.zIndex = '25000';
      c.style.width = `${target.offsetWidth}px`;
      c.style.height = `${target.offsetHeight}px`;
      c.style.pointerEvents = 'none';
      c.style.opacity = '0.8';
      c.style.background = id === 'REACTIONS_ROW' ? 'var(--accent-soft)' : 'var(--bg-secondary)';
      c.style.boxShadow = '0 10px 30px rgba(0,0,0,0.4)';
      c.style.borderRadius = '8px';
      c.style.transform = `translate3d(${t.clientX - target.offsetWidth / 2}px, ${t.clientY - target.offsetHeight / 2}px, 0)`;
      document.body.appendChild(c);
      cloneRef.current = c;

      if (navigator.vibrate) navigator.vibrate(15);
    }, 250);
  }, [cacheCells]);

  const onTouchMove = useCallback((e: React.TouchEvent) => {
    const t = e.touches[0];
    if (!isDragging.current) {
      const dx = Math.abs(t.clientX - touchStart.current.x);
      const dy = Math.abs(t.clientY - touchStart.current.y);
      if (dx > 8 || dy > 8) {
        if (longPressRef.current) { clearTimeout(longPressRef.current); longPressRef.current = null; }
      }
      return;
    }
    e.preventDefault();
    cancelAnimationFrame(raf.current);
    raf.current = requestAnimationFrame(() => {
      if (cloneRef.current) {
        cloneRef.current.style.transform = `translate3d(${t.clientX - cloneRef.current.offsetWidth / 2}px, ${t.clientY - cloneRef.current.offsetHeight / 2}px, 0)`;
      }
      setDragOverKey(findCell(t.clientX, t.clientY));
    });
  }, []);

  const onTouchEnd = useCallback(() => {
    if (isDragging.current && dragBtnId && dragOverKey) {
      const [xs, ys] = dragOverKey.split(',');
      const x = +xs, y = +ys;
      if (dragBtnId === 'REACTIONS_ROW') {
        onMoveReactions(y);
      } else if (!btnAt(x, y)) {
        onMove(dragBtnId, x, y);
      }
    }
    cleanTouch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dragBtnId, dragOverKey, buttons, onMove, onMoveReactions, cleanTouch]);

  useEffect(() => () => cleanTouch(), [cleanTouch]);

  const onCellClick = (x: number, y: number) => {
    if (isDragging.current) return;
    const b = btnAt(x, y);
    if (b) {
      setSelectedId(prev => prev === b.buttonId ? null : b.buttonId);
      setAddingAt(null); setEditingId(null);
    } else {
      setAddingAt({ x, y });
      setNewName(''); setNewUrl(''); setNewStyle('');
      setSelectedId(null); setEditingId(null);
    }
  };

  const doAdd = () => {
    const finalUrl = processUrl(newUrl);
    if (!addingAt || !newName.trim() || !validateUrl(finalUrl)) return;
    onAdd({
      buttonId: '', // Temporarily empty, will be assigned real ID from DB via App.tsx
      nameButton: newName.trim(), buttonUrl: finalUrl,
      style: newStyle || undefined,
      positionX: addingAt.x, positionY: addingAt.y,
      ownerChannelId: channelId,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    });
    setAddingAt(null);
  };

  const startEdit = (b: ButtonType) => {
    setEditingId(b.buttonId);
    setEditName(b.nameButton);
    setEditUrl(b.buttonUrl);
    setEditStyle(b.style || '');
    setSelectedId(null); setAddingAt(null);
  };

  const doEdit = () => {
    const finalUrl = processUrl(editUrl);
    if (!editingId || !editName.trim() || !validateUrl(finalUrl)) return;
    onEdit(editingId, { nameButton: editName.trim(), buttonUrl: finalUrl, style: editStyle || undefined });
    setEditingId(null);
  };

  const adjustGrid = (axis: 'col' | 'row', dir: 1 | -1) => {
    if (axis === 'col') {
      const mx = buttons.reduce((m, b) => Math.max(m, b.positionX), -1);
      setCols(c => Math.max(Math.min(c + dir, 8), mx + 1, 1));
    } else {
      const mx = buttons.reduce((m, b) => Math.max(m, b.positionY), -1);
      const mxUsed = Math.max(mx, reactionPosition);
      setRows(r => Math.max(Math.min(r + dir, 10), mxUsed + 1, 1));
    }
  };

  const selBtn = selectedId ? buttons.find(b => b.buttonId === selectedId) : null;

  return (
    <div className="button-grid-content">
      <Card className="animate-stagger-in" style={{ animationDelay: '0s' }}>
        <div className="px-5 pt-3">
          {/* Header + grid controls merged */}
          <div className="flex items-center gap-2 mb-2">
            <div className="section-icon purple" style={{ width: 28, height: 28, borderRadius: 8 }}><Grid3X3 size={15} /></div>
            <div className="flex-1 min-w-0">
              <h3 className="text-sm font-semibold">Botões Inline</h3>
            </div>
            <Badge variant="secondary" className="text-[10px]">{buttons.length}</Badge>
            <div className="flex items-center gap-1 ml-1 border-l border-border pl-2">
              <button className="size-6 flex items-center justify-center rounded-md hover:bg-muted transition-colors" onClick={() => adjustGrid('col', -1)} title="Diminuir colunas"><Minus size={12} /></button>
              <span className="text-[11px] font-bold min-w-[14px] text-center text-muted-foreground">{cols}</span>
              <button className="size-6 flex items-center justify-center rounded-md hover:bg-muted transition-colors" onClick={() => adjustGrid('col', 1)} title="Aumentar colunas"><Plus size={12} /></button>
              <span className="text-[10px] text-muted-foreground/50 mx-1">×</span>
              <button className="size-6 flex items-center justify-center rounded-md hover:bg-muted transition-colors" onClick={() => adjustGrid('row', -1)} title="Diminuir linhas"><Minus size={12} /></button>
              <span className="text-[11px] font-bold min-w-[14px] text-center text-muted-foreground">{rows}</span>
              <button className="size-6 flex items-center justify-center rounded-md hover:bg-muted transition-colors" onClick={() => adjustGrid('row', 1)} title="Aumentar linhas"><Plus size={12} /></button>
            </div>
          </div>
        </div>

        {/* Grid */}
        <div className="w-full overflow-x-auto px-4" ref={gridRef} style={{ WebkitOverflowScrolling: 'touch' }}>
          <div
            className="grid gap-1.5"
            style={{
              gridTemplateColumns: `repeat(${cols}, 1fr)`,
              minWidth: cols > 4 ? cols * 72 : undefined,
              position: 'relative'
            }}
          >
            {/* 1. Render empty cells for drop targets */}
            {Array.from({ length: rows }, (_, y) =>
              Array.from({ length: cols }, (_, x) => {
                const k = cellKey(x, y);
                const hasBtn = !!btnAt(x, y);
                const isReac = y === reactionPosition;

                // When dragging reactions, highlight the WHOLE row
                const isOver = dragOverKey && (
                  dragBtnId === 'REACTIONS_ROW' 
                    ? dragOverKey.split(',')[1] === String(y)
                    : dragOverKey === k
                );
                
                return (
                  <div
                    key={`cell-${k}`}
                    data-cell={k}
                    className={`grid-cell ${isOver ? 'drag-over' : ''}`}
                    style={{
                      gridColumn: x + 1,
                      gridRow: y + 1,
                      height: '56px',
                      zIndex: 1,
                      pointerEvents: 'auto'
                    }}
                    onClick={() => {
                      if (isDragging.current || y === reactionPosition) return;
                      onCellClick(x, y);
                    }}
                    onDragOver={e => onCellDragOver(e, x, y)}
                    onDrop={e => onCellDrop(e, x, y)}
                    onDragLeave={() => setDragOverKey(null)}
                  >
                    {!hasBtn && !isReac && (
                      <div className="flex items-center justify-center size-7 rounded-lg bg-muted/20 border border-border/40 text-muted-foreground/60 group-hover:text-foreground group-hover:bg-muted/40 transition-all active:scale-95">
                        <Plus size={14} />
                      </div>
                    )}
                  </div>
                );
              })
            )}

            {/* 2. Render buttons */}
            {buttons.map(b => {
              const isSel = selectedId === b.buttonId;
              const isSource = dragBtnId === b.buttonId;
              const customStyle = getButtonStyle(b.style);
              
              return (
                <div
                  key={`btn-${b.buttonId}`}
                  draggable
                  onDragStart={e => onDragStart(e, b.buttonId)}
                  onDragEnd={onDragEnd}
                  onTouchStart={e => onTouchStart(e, b.buttonId)}
                  onTouchMove={e => onTouchMove(e)}
                  onTouchEnd={onTouchEnd}
                  onClick={(e) => { 
                    e.stopPropagation(); 
                    if (!isDragging.current) onCellClick(b.positionX, b.positionY); 
                  }}
                  className={`grid-cell occupied ${isSource ? 'drag-source' : ''} ${isSel ? 'selected' : ''}`}
                  style={{
                    gridColumn: b.positionX + 1,
                    gridRow: b.positionY + 1,
                    zIndex: 10,
                    cursor: 'grab',
                    ...customStyle,
                  }}
                >
                  <div className="flex flex-col items-center justify-center w-full h-full select-none min-w-0 gap-1 px-1 py-1 group" style={{ pointerEvents: 'none' }}>
                    <GripVertical size={13} className="text-muted-foreground/45 group-hover:text-muted-foreground/80 transition-colors" />
                    <span className="text-[12px] font-bold truncate max-w-full leading-tight text-center text-foreground">{b.nameButton}</span>
                  </div>
                </div>
              );
            })}

            {/* 3. Render Reactions Plate */}
            {!hideReactions && (() => {
              const reactionsList = (reactions || '').split(',').filter(r => r.trim() !== '');
              const isSource = dragBtnId === 'REACTIONS_ROW';
              const plateHeight = '56px';
              return (
                <div
                  key="reactions-plate"
                  draggable
                  onDragStart={e => onDragStart(e, 'REACTIONS_ROW')}
                  onDragEnd={onDragEnd}
                  onTouchStart={e => onTouchStart(e, 'REACTIONS_ROW')}
                  onTouchMove={e => onTouchMove(e)}
                  onTouchEnd={onTouchEnd}
                  className={`grid-cell reactions-plate ${isSource ? 'drag-source' : ''}`}
                  style={{
                    gridColumn: `1 / span ${cols}`,
                    gridRow: reactionPosition + 1,
                    background: 'rgba(var(--accent-rgb), 0.08)',
                    border: '1.5px dashed rgba(var(--accent-rgb), 0.25)',
                    borderRadius: '10px',
                    zIndex: 20,
                    height: plateHeight,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    cursor: 'grab',
                    touchAction: 'none',
                    userSelect: 'none'
                  }}
                >
                  <div className="flex flex-col items-center justify-center pointer-events-none">
                    <GripVertical size={14} className="text-accent/40 mb-1" />
                    {reactionsList.length === 0 ? (
                      <span className="text-[11px] font-semibold text-accent/80 tracking-wide uppercase">Bloco de Reações (Vazio)</span>
                    ) : (
                      <div className="flex gap-1.5 items-center bg-background/40 px-3 py-1 rounded-full shadow-sm border border-border/50">
                        {reactionsList.slice(0, 4).map((r, i) => (
                          <span key={i} className="text-[14px] leading-none drop-shadow-sm">{r}</span>
                        ))}
                        {reactionsList.length > 4 && <span className="text-[10px] font-bold text-muted-foreground ml-0.5">+{reactionsList.length - 4}</span>}
                      </div>
                    )}
                  </div>
                </div>
              );
            })()}
          </div>
        </div>

        {/* Selected detail */}
        {selBtn && !editingId && (
          <div className="mt-3 mx-4 p-4 rounded-xl bg-muted border border-border">
              <div className="flex items-center justify-between gap-3 mb-2">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <h4 className="text-sm font-semibold truncate">{selBtn.nameButton}</h4>
                    {selBtn.style && (
                      <Badge variant="outline" className="text-[10px] capitalize" style={getButtonStyle(selBtn.style)}>
                        {selBtn.style}
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs mt-0.5 truncate font-mono text-muted-foreground">{selBtn.buttonUrl}</p>
                </div>
              </div>
              <div className="flex gap-1.5 flex-wrap">
                <Button variant="secondary" size="sm" className="flex-1" onClick={() => startEdit(selBtn)}>
                  <Pencil size={12} /> Editar
                </Button>
                <Button variant="secondary" size="sm" className="flex-1" onClick={() => window.open(selBtn.buttonUrl, '_blank', 'noopener,noreferrer')}>
                  <ExternalLink size={12} /> Abrir
                </Button>
                <Button variant="destructive" size="sm" className="flex-1" onClick={() => setConfirmDeleteId(selBtn.buttonId)}>
                  <Trash2 size={12} /> Excluir
                </Button>
              </div>
            </div>
        )}

        {/* Edit form */}
        {editingId && (() => {
          const b = buttons.find(bt => bt.buttonId === editingId);
          if (!b) return null;
          const isValidBtn = editName.trim().length > 0 && validateUrl(editUrl);
          return (
            <div className="mt-3 mx-4 p-4 rounded-xl bg-muted border border-border space-y-3">
              <div className="flex items-center gap-2">
                <Pencil size={12} className="text-accent" />
                <span className="text-sm font-semibold">Editando "{b.nameButton}"</span>
              </div>
              <Input className="h-10" value={editName} onChange={e => setEditName(e.target.value)} placeholder="Nome" autoFocus />
              <Input
                className="h-10 font-mono"
                value={editUrl}
                onChange={e => setEditUrl(e.target.value)}
                onBlur={e => setEditUrl(processUrl(e.target.value))}
                placeholder="https://t.me/username..."
              />
              {!validateUrl(editUrl) && editUrl.trim().length > 0 && (
                <p className="text-xs text-destructive">Username do Telegram deve ter no mínimo 5 caracteres.</p>
              )}

              {/* Color style selector */}
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">Cor do Botão (Telegram)</label>
                <div className="flex items-center gap-3 pt-0.5">
                  <button
                    type="button"
                    onClick={() => setEditStyle('')}
                    title="Padrão (Neutro)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-white/10 ${editStyle === '' ? 'border-primary ring-2 ring-primary/40 scale-110' : 'border-border/60 hover:scale-105'}`}
                  >
                    <span className="text-[9px] font-bold text-muted-foreground">STD</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditStyle('primary')}
                    title="Primary (Azul)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-[#2481cc] ${editStyle === 'primary' ? 'border-white ring-2 ring-[#2481cc]/80 scale-110' : 'border-transparent hover:scale-105'}`}
                  >
                    {editStyle === 'primary' && <Check size={14} className="text-white" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditStyle('success')}
                    title="Success (Verde)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-[#0ea573] ${editStyle === 'success' ? 'border-white ring-2 ring-[#0ea573]/80 scale-110' : 'border-transparent hover:scale-105'}`}
                  >
                    {editStyle === 'success' && <Check size={14} className="text-white" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => setEditStyle('danger')}
                    title="Danger (Vermelho)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-[#e83e3e] ${editStyle === 'danger' ? 'border-white ring-2 ring-[#e83e3e]/80 scale-110' : 'border-transparent hover:scale-105'}`}
                  >
                    {editStyle === 'danger' && <Check size={14} className="text-white" />}
                  </button>
                </div>
              </div>

              <div className="flex gap-2 justify-end pt-1">
                <Button variant="secondary" size="sm" onClick={() => setEditingId(null)}>
                  <X size={12} /> Cancelar
                </Button>
                <Button variant="default" size="sm" onClick={doEdit} disabled={!isValidBtn}>
                  <Check size={12} /> Salvar
                </Button>
              </div>
            </div>
          );
        })()}

        {/* Add form */}
        {addingAt && (() => {
          const isValidBtn = newName.trim().length > 0 && validateUrl(newUrl);
          return (
            <div className="mt-3 mx-4 p-4 rounded-xl bg-muted border border-border space-y-3">
              <div className="flex items-center gap-2">
                <Plus size={12} className="text-green-500" />
                <span className="text-sm font-semibold">Novo botão</span>
              </div>
              <Input className="h-10" value={newName} onChange={e => setNewName(e.target.value)} placeholder="Nome do botão" autoFocus />
              <Input
                className="h-10 font-mono"
                value={newUrl}
                onChange={e => setNewUrl(e.target.value)}
                onBlur={e => setNewUrl(processUrl(e.target.value))}
                placeholder="https://t.me/username..."
              />
              {!validateUrl(newUrl) && newUrl.trim().length > 0 && (
                <p className="text-xs text-destructive">Username do Telegram deve ter no mínimo 5 caracteres.</p>
              )}

              {/* Color style selector */}
              <div className="space-y-1.5">
                <label className="text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">Cor do Botão (Telegram)</label>
                <div className="flex items-center gap-3 pt-0.5">
                  <button
                    type="button"
                    onClick={() => setNewStyle('')}
                    title="Padrão (Neutro)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-white/10 ${newStyle === '' ? 'border-primary ring-2 ring-primary/40 scale-110' : 'border-border/60 hover:scale-105'}`}
                  >
                    <span className="text-[9px] font-bold text-muted-foreground">STD</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setNewStyle('primary')}
                    title="Primary (Azul)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-[#2481cc] ${newStyle === 'primary' ? 'border-white ring-2 ring-[#2481cc]/80 scale-110' : 'border-transparent hover:scale-105'}`}
                  >
                    {newStyle === 'primary' && <Check size={14} className="text-white" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => setNewStyle('success')}
                    title="Success (Verde)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-[#0ea573] ${newStyle === 'success' ? 'border-white ring-2 ring-[#0ea573]/80 scale-110' : 'border-transparent hover:scale-105'}`}
                  >
                    {newStyle === 'success' && <Check size={14} className="text-white" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => setNewStyle('danger')}
                    title="Danger (Vermelho)"
                    className={`size-8 rounded-full border-2 transition-all flex items-center justify-center bg-[#e83e3e] ${newStyle === 'danger' ? 'border-white ring-2 ring-[#e83e3e]/80 scale-110' : 'border-transparent hover:scale-105'}`}
                  >
                    {newStyle === 'danger' && <Check size={14} className="text-white" />}
                  </button>
                </div>
              </div>

              <div className="flex gap-2 justify-end pt-1">
                <Button variant="secondary" size="sm" onClick={() => setAddingAt(null)}>
                  <X size={12} /> Cancelar
                </Button>
                <Button variant="default" size="sm" onClick={doAdd} disabled={!isValidBtn}>
                  <Check size={12} /> Adicionar
                </Button>
              </div>
            </div>
          );
        })()}
      </Card>

      {/* Delete confirm */}
      <Dialog open={!!confirmDeleteId} onOpenChange={(open) => { if (!open) setConfirmDeleteId(null); }}>
        <DialogContent className="sm:max-w-md p-6 text-center bg-card text-card-foreground border border-border/80 shadow-2xl rounded-2xl" showCloseButton={false}>
          <DialogHeader className="items-center gap-3">
            <div className="flex size-14 items-center justify-center rounded-2xl bg-destructive/15 text-destructive border border-destructive/30 shadow-sm">
              <AlertTriangle size={26} />
            </div>
            <DialogTitle className="text-lg font-bold text-foreground tracking-tight mt-1">Excluir Botão</DialogTitle>
            <DialogDescription className="text-xs leading-relaxed px-2 text-muted-foreground font-medium">
              Tem certeza que deseja excluir o botão <strong className="text-foreground">"{buttons.find(b => b.buttonId === confirmDeleteId)?.nameButton}"</strong>?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex flex-row items-center justify-center gap-3 mt-4 pt-2 border-t-0 p-0">
            <DialogClose render={<Button variant="outline" className="flex-1 rounded-xl h-11 text-xs font-semibold border-border bg-surface text-foreground hover:bg-muted">Cancelar</Button>} />
            <Button
              variant="destructive"
              className="flex-1 rounded-xl h-11 text-xs font-bold transition-all shadow-md"
              onClick={() => { onDelete(confirmDeleteId!); setConfirmDeleteId(null); setSelectedId(null); }}
            >
              <Trash2 size={15} className="mr-1.5" /> Excluir
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
