import { useEffect, useState, useCallback } from 'react';
import { EmojiRenderer } from './EmojiRenderer';
import { fetchEmojiHistory, getCachedEmojiHistory } from '../api';

interface Props {
  onSelect: (emojiId: string) => void;
  onClose: () => void;
}

export function EmojiPicker({ onSelect, onClose }: Props) {
  const initial = getCachedEmojiHistory();
  const [ids, setIds] = useState<string[]>(initial);
  const [loading, setLoading] = useState(initial.length === 0);

  useEffect(() => {
    fetchEmojiHistory()
      .then(data => {
        setIds(data);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  }, []);

  const handleClick = useCallback((emojiId: string) => {
    onSelect(emojiId);
    onClose();
  }, [onSelect, onClose]);

  return (
    <div className="emoji-picker" onClick={e => e.stopPropagation()}>
      <div className="emoji-picker-header">
        <span className="emoji-picker-title">Emojis Recentes</span>
        <button
          type="button"
          className="emoji-picker-close"
          onClick={onClose}
          title="Fechar"
        >
          ✕
        </button>
      </div>

      <div className="emoji-picker-grid">
        {loading && (
          <span className="emoji-picker-loading">Carregando...</span>
        )}

        {!loading && ids.length === 0 && (
          <span className="emoji-picker-empty">
            Nenhum emoji personalizado usado ainda.
            <br />
            Envie um pelo Telegram para aparecer aqui.
          </span>
        )}

        {!loading && ids.map(id => (
          <button
            key={id}
            type="button"
            className="emoji-picker-item"
            onClick={() => handleClick(id)}
            title={`Emoji ID: ${id}`}
          >
            <EmojiRenderer emojiId={id} size={32} />
          </button>
        ))}
      </div>
    </div>
  );
}
