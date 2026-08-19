import { useRef, useState, useCallback, useEffect } from 'react';
import {
  Bold, Italic, Underline, Strikethrough, Code, Terminal,
  EyeOff, Link2, List, Quote, Undo2, Redo2, Copy, Eraser,
  AlignLeft, ChevronDown
} from 'lucide-react';
import { Button } from './ui/button';
import { EmojiRenderer } from './EmojiRenderer';

interface Props {
  value: string;
  onChange: (val: string) => void;
  rows?: number;
  placeholder?: string;
}

interface HistoryEntry {
  text: string;
  selStart: number;
  selEnd: number;
}

const FORMATS = [
  { key: 'bold', icon: Bold, label: 'Negrito', wrap: ['*', '*'], placeholder: 'negrito' },
  { key: 'italic', icon: Italic, label: 'Itálico', wrap: ['_', '_'], placeholder: 'itálico' },
  { key: 'underline', icon: Underline, label: 'Sublinhado', wrap: ['<u>', '</u>'], placeholder: 'sublinhado' },
  { key: 'strike', icon: Strikethrough, label: 'Tachado', wrap: ['~~', '~~'], placeholder: 'tachado' },
  { key: 'mono', icon: Code, label: 'Monoespaço', wrap: ['`', '`'], placeholder: 'código' },
  { key: 'spoiler', icon: EyeOff, label: 'Spoiler', wrap: ['||', '||'], placeholder: 'spoiler' },
] as const;

export function RichTextEditor({ value, onChange, rows = 6, placeholder }: Props) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [history, setHistory] = useState<HistoryEntry[]>([{ text: value, selStart: 0, selEnd: 0 }]);
  const [historyIdx, setHistoryIdx] = useState(0);
  const [showMore, setShowMore] = useState(false);
  const [linkMode, setLinkMode] = useState(false);
  const [linkUrl, setLinkUrl] = useState('');
  const [linkText, setLinkText] = useState('');
  const [recentEmojiIds, setRecentEmojiIds] = useState<string[]>([]);

  // Carrega histórico de emojis do usuário
  useEffect(() => {
    fetch('/api/emoji/history', { credentials: 'same-origin' })
      .then(res => res.json())
      .then(data => setRecentEmojiIds(data.ids || []))
      .catch(() => {
        // Falha silenciosa — emojis recentes são um recurso opcional
      });
  }, []);

  const pushHistory = useCallback((text: string, selStart: number, selEnd: number) => {
    setHistory(prev => {
      const next = [...prev.slice(0, historyIdx + 1), { text, selStart, selEnd }];
      if (next.length > 50) next.shift();
      return next;
    });
    setHistoryIdx(prev => Math.min(prev + 1, 50));
  }, [historyIdx]);

  const applyFormat = useCallback((before: string, after: string, placeholderText: string) => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.focus();

    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const selected = value.substring(start, end);
    const text = selected || placeholderText;

    const newValue = value.substring(0, start) + before + text + after + value.substring(end);
    onChange(newValue);
    pushHistory(newValue, start + before.length, start + before.length + text.length);

    requestAnimationFrame(() => {
      ta.selectionStart = start + before.length;
      ta.selectionEnd = start + before.length + text.length;
      ta.focus();
    });
  }, [value, onChange, pushHistory]);

  const insertCodeBlock = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.focus();

    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const selected = value.substring(start, end) || 'código aqui';

    const block = '```\n' + selected + '\n```';
    const newValue = value.substring(0, start) + block + value.substring(end);
    onChange(newValue);
    pushHistory(newValue, start + 4, start + 4 + selected.length);

    requestAnimationFrame(() => {
      ta.selectionStart = start + 4;
      ta.selectionEnd = start + 4 + selected.length;
      ta.focus();
    });
  }, [value, onChange, pushHistory]);

  const insertLink = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;

    const start = ta.selectionStart;
    const selected = value.substring(start, ta.selectionEnd);

    if (linkMode) {
      // Apply link
      const displayText = linkText || selected || 'texto';
      const url = linkUrl || 'https://';
      const link = `[${displayText}](${url})`;
      const newValue = value.substring(0, start) + link + value.substring(ta.selectionEnd);
      onChange(newValue);
      pushHistory(newValue, start, start + link.length);
      setLinkMode(false);
      setLinkUrl('');
      setLinkText('');
      requestAnimationFrame(() => {
        ta.selectionStart = start;
        ta.selectionEnd = start + link.length;
        ta.focus();
      });
    } else {
      setLinkText(selected);
      setLinkMode(true);
    }
  }, [value, onChange, pushHistory, linkMode, linkUrl, linkText]);

  const insertEmoji = useCallback((emojiId: string) => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.focus();

    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const tag = `<tg-emoji emoji-id="${emojiId}"> </tg-emoji>`;
    const newValue = value.substring(0, start) + tag + value.substring(end);
    onChange(newValue);
    pushHistory(newValue, start, start + tag.length);
  }, [value, onChange, pushHistory]);

  const insertQuote = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.focus();

    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const selected = value.substring(start, end) || 'citação';
    const lines = selected.split('\n').map(l => '> ' + l).join('\n');
    const newValue = value.substring(0, start) + lines + value.substring(end);
    onChange(newValue);
    pushHistory(newValue, start, start + lines.length);
  }, [value, onChange, pushHistory]);

  const insertList = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.focus();

    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const selected = value.substring(start, end);
    const items = selected ? selected.split('\n').map(l => '• ' + l).join('\n') : '• item 1\n• item 2\n• item 3';
    const newValue = value.substring(0, start) + items + value.substring(end);
    onChange(newValue);
    pushHistory(newValue, start, start + items.length);
  }, [value, onChange, pushHistory]);

  const clearFormatting = useCallback(() => {
    const ta = textareaRef.current;
    if (!ta) return;
    ta.focus();

    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    if (start === end) return;

    let selected = value.substring(start, end);
    // Remove common formatting markers
    selected = selected.replace(/\*(.*?)\*/g, '$1');
    selected = selected.replace(/_(.*?)_/g, '$1');
    selected = selected.replace(/~~(.*?)~~/g, '$1');
    selected = selected.replace(/\|\|(.*?)\|\|/g, '$1');
    selected = selected.replace(/`([^`]+)`/g, '$1');
    selected = selected.replace(/<u>(.*?)<\/u>/g, '$1');
    selected = selected.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');
    selected = selected.replace(/^> /gm, '');
    selected = selected.replace(/^• /gm, '');

    const newValue = value.substring(0, start) + selected + value.substring(end);
    onChange(newValue);
    pushHistory(newValue, start, start + selected.length);
  }, [value, onChange, pushHistory]);

  const undo = useCallback(() => {
    if (historyIdx <= 0) return;
    const prev = history[historyIdx - 1];
    if (prev) {
      onChange(prev.text);
      setHistoryIdx(historyIdx - 1);
      const ta = textareaRef.current;
      if (ta) {
        requestAnimationFrame(() => {
          ta.selectionStart = prev.selStart;
          ta.selectionEnd = prev.selEnd;
          ta.focus();
        });
      }
    }
  }, [history, historyIdx, onChange]);

  const redo = useCallback(() => {
    if (historyIdx >= history.length - 1) return;
    const next = history[historyIdx + 1];
    if (next) {
      onChange(next.text);
      setHistoryIdx(historyIdx + 1);
      const ta = textareaRef.current;
      if (ta) {
        requestAnimationFrame(() => {
          ta.selectionStart = next.selStart;
          ta.selectionEnd = next.selEnd;
          ta.focus();
        });
      }
    }
  }, [history, historyIdx, onChange]);

  const copyAll = useCallback(() => {
    navigator.clipboard?.writeText(value);
  }, [value]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'z') {
      e.preventDefault();
      if (e.shiftKey) redo();
      else undo();
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'b') {
      e.preventDefault();
      applyFormat('*', '*', 'negrito');
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'i') {
      e.preventDefault();
      applyFormat('_', '_', 'itálico');
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'u') {
      e.preventDefault();
      applyFormat('<u>', '</u>', 'sublinhado');
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      insertLink();
    }
  }, [undo, redo, applyFormat, insertLink]);

  return (
    <div className="rte-container">
      {/* Main toolbar */}
      <div className="rte-toolbar">
        <div className="rte-toolbar-group">
          {FORMATS.map(f => (
            <button
              key={f.key}
              className="rte-btn"
              title={f.label}
              onMouseDown={e => e.preventDefault()}
              onClick={() => applyFormat(f.wrap[0], f.wrap[1], f.placeholder)}
            >
              <f.icon size={16} />
            </button>
          ))}
        </div>

        <div className="rte-divider" />

        <div className="rte-toolbar-group">
          <button className="rte-btn" title="Bloco de código" onMouseDown={e => e.preventDefault()} onClick={insertCodeBlock}>
            <Terminal size={16} />
          </button>
          <button className={`rte-btn ${linkMode ? 'rte-btn-active' : ''}`} title="Link [texto](url)" onMouseDown={e => e.preventDefault()} onClick={insertLink}>
            <Link2 size={16} />
          </button>
        </div>

        <div className="rte-divider" />

        <button
          className={`rte-btn rte-btn-more ${showMore ? 'rte-btn-active' : ''}`}
          title="Mais opções"
          onClick={() => setShowMore(!showMore)}
        >
          <ChevronDown size={16} className={`transition-transform ${showMore ? 'rotate-180' : ''}`} />
        </button>
      </div>

      {/* Extended toolbar */}
      {showMore && (
        <div className="rte-toolbar rte-toolbar-extended">
          <button className="rte-btn" title="Lista" onMouseDown={e => e.preventDefault()} onClick={insertList}>
            <List size={16} />
          </button>
          <button className="rte-btn" title="Citação" onMouseDown={e => e.preventDefault()} onClick={insertQuote}>
            <Quote size={16} />
          </button>

          <div className="rte-divider" />

          <button className="rte-btn" title="Desfazer (Ctrl+Z)" onMouseDown={e => e.preventDefault()} onClick={undo} disabled={historyIdx <= 0}>
            <Undo2 size={16} />
          </button>
          <button className="rte-btn" title="Refazer (Ctrl+Shift+Z)" onMouseDown={e => e.preventDefault()} onClick={redo} disabled={historyIdx >= history.length - 1}>
            <Redo2 size={16} />
          </button>

          <div className="rte-divider" />

          <button className="rte-btn" title="Limpar formatação" onMouseDown={e => e.preventDefault()} onClick={clearFormatting}>
            <Eraser size={16} />
          </button>
          <button className="rte-btn" title="Copiar tudo" onMouseDown={e => e.preventDefault()} onClick={copyAll}>
            <Copy size={16} />
          </button>
        </div>
      )}

      {/* Link insertion panel */}
      {linkMode && (
        <div className="rte-link-panel">
          <div className="flex items-center gap-2 mb-2">
            <Link2 size={14} style={{ color: 'var(--accent)', flexShrink: 0 }} />
            <span className="text-xs font-semibold" style={{ color: 'var(--accent)' }}>Inserir Link</span>
          </div>
          <div className="rte-link-fields">
            <div className="rte-link-field">
              <label className="text-[11px] font-medium text-muted-foreground">
                <AlignLeft size={11} className="inline mr-1" />
                Texto
              </label>
              <input
                className="rte-link-input"
                value={linkText}
                onChange={e => setLinkText(e.target.value)}
                placeholder="texto do link"
                autoFocus
              />
            </div>
            <div className="rte-link-field">
              <label className="text-[11px] font-medium text-muted-foreground">
                <Link2 size={11} className="inline mr-1" />
                URL
              </label>
              <input
                className="rte-link-input"
                value={linkUrl}
                onChange={e => setLinkUrl(e.target.value)}
                placeholder="https://"
              />
            </div>
          </div>
          <div className="flex gap-2 mt-3">
            <Button
              variant="secondary"
              size="sm"
              className="flex-1"
              onClick={() => { setLinkMode(false); setLinkUrl(''); setLinkText(''); }}
            >
              Cancelar
            </Button>
            <Button variant="default" size="sm" className="flex-1" onClick={insertLink}>
              Inserir
            </Button>
          </div>
        </div>
      )}

      {/* Textarea */}
      <textarea
        ref={textareaRef}
        className="rte-textarea"
        value={value}
        onChange={e => {
          onChange(e.target.value);
          pushHistory(e.target.value, e.target.selectionStart, e.target.selectionEnd);
        }}
        onKeyDown={handleKeyDown}
        rows={rows}
        placeholder={placeholder}
        spellCheck={false}
      />

      {/* Emoji bar — sempre visível abaixo do input */}
      {recentEmojiIds.length > 0 && (
        <div className="rte-emoji-bar">
          {recentEmojiIds.slice(-30).map(id => (
            <button
              key={id}
              type="button"
              className="rte-emoji-bar-item"
              title="Inserir emoji"
              onMouseDown={e => e.preventDefault()}
              onClick={() => insertEmoji(id)}
            >
              <EmojiRenderer emojiId={id} size={22} />
            </button>
          ))}
        </div>
      )}

    </div>
  );
}
