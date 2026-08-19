import { useMemo } from 'react';
import { EmojiRenderer } from './EmojiRenderer';

interface Props {
  text: string;
}

// ── Regex to find emoji tags ──
const EMOJI_RE = /(<tg-emoji\s+emoji-id="(\d+)"[^>]*>)(.*?)(<\/tg-emoji>)/gi;

// ── URL sanitization ──
function sanitizeUrl(url: string): string {
  try {
    const parsed = new URL(url);
    if (['http:', 'https:'].includes(parsed.protocol)) {
      return parsed.href;
    }
    return '#';
  } catch {
    return '#';
  }
}

// ── Markdown → HTML (for non-emoji parts) ──
function mdToHtml(text: string): string {
  if (!text) return '';

  let h = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

    // Bold (*text*)
    .replace(/\*(.+?)\*/g, '<strong>$1</strong>')
    // Italic (_text_)
    .replace(/_(.+?)_/g, '<em>$1</em>')
    // Strikethrough
    .replace(/~~(.+?)~~/g, '<s>$1</s>')
    // Spoiler
    .replace(/\|\|(.+?)\|\|/g, '<span class="cp-spoiler">$1</span>')
    // Inline code
    .replace(/`([^`\n]+)`/g, '<code class="cp-icode">$1</code>')
    // Underline (raw HTML in input)
    .replace(/&lt;u&gt;(.+?)&lt;\/u&gt;/g, '<u>$1</u>')
    // Code block
    .replace(/```(.+?)```/gs, '<pre class="cp-pre"><code>$1</code></pre>')
    // Markdown links (with URL sanitization)
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_m, text, url) =>
      `<a class="cp-a" href="${sanitizeUrl(url)}" target="_blank" rel="noopener noreferrer">${text}</a>`
    )
    // Blockquote
    .replace(/^&gt;\s?(.+)$/gm, '<blockquote class="cp-bq">$1</blockquote>')
    // List items
    .replace(/^•\s?(.+)$/gm, '<li class="cp-li">$1</li>');

  // Highlight template variables ($name, $title, $link, $count)
  h = h.replace(/(\$(?:name|title|link|count))\b/g, '<span class="cp-var font-mono text-[11px] px-1.5 py-0.5 rounded bg-accent/15 text-accent font-bold">$1</span>');

  // Auto-link t.me URLs
  h = h.replace(/(^|[^"'])(https?:\/\/t\.me\/[a-zA-Z0-9_/-]+)/g, '$1<a class="cp-a text-accent font-medium hover:underline" href="$2" target="_blank" rel="noopener noreferrer">$2</a>');

  return h;
}

// ── Tokenize text into segments: HTML-text or Emoji ──
type Segment =
  | { type: 'html'; html: string }
  | { type: 'emoji'; id: string; char: string };

function tokenize(text: string): Segment[] {
  if (!text) return [];

  const segments: Segment[] = [];
  let last = 0;

  // Reset regex state
  EMOJI_RE.lastIndex = 0;

  for (;;) {
    const m = EMOJI_RE.exec(text);
    if (!m) break;

    // Text before this emoji
    if (m.index > last) {
      segments.push({ type: 'html', html: text.slice(last, m.index) });
    }

    segments.push({ type: 'emoji', id: m[2], char: m[3] });
    last = m.index + m[0].length;
  }

  // Remaining text after last emoji
  if (last < text.length) {
    segments.push({ type: 'html', html: text.slice(last) });
  }

  return segments;
}

// ── Preview Component ──
export function CaptionPreview({ text }: Props) {
  const segments = useMemo(() => tokenize(text), [text]);

  return (
    <div className="cp-bubble cursor-pointer hover:border-accent/50 transition-colors">
      {segments.length === 0 ? (
        <span className="cp-placeholder">Clique para adicionar legenda...</span>
      ) : (
        segments.map((seg, i) =>
          seg.type === 'emoji' ? (
            <EmojiRenderer key={`e${i}`} emojiId={seg.id} />
          ) : (
            <span
              key={`h${i}`}
              dangerouslySetInnerHTML={{
                __html: mdToHtml(seg.html),
              }}
            />
          ),
        )
      )}
    </div>
  );
}
