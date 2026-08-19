import { useRef, useEffect, useState } from 'react';
import lottie from 'lottie-web';

interface Props {
  emojiId: string;
  size?: number;
  className?: string;
}

type EmojiStatus = 'loading' | 'lottie' | 'video' | 'image' | 'fallback';

// EmojiRenderer: loads custom emoji from our API and renders with correct player.
// - .tgs → Lottie (decompress gzip, play with lottie-web)
// - .webm → <video> with blob URL
// - .webp → <img> with blob URL
export function EmojiRenderer({ emojiId, size = 22, className = '' }: Props) {
  const [status, setStatus] = useState<EmojiStatus>('loading');
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [animData, setAnimData] = useState<Record<string, unknown> | null>(null);
  const lottieRef = useRef<HTMLDivElement>(null);
  const animInst = useRef<object | null>(null);

  useEffect(() => {
    if (!emojiId) { setStatus('fallback'); return; }

    let cancelled = false;
    const apiUrl = `/api/emoji/${emojiId}`;

    fetch(apiUrl, { credentials: 'same-origin' })
      .then(async res => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const contentType = res.headers.get('Content-Type') || '';
        const buf = await res.arrayBuffer();

        if (cancelled) return;

        if (contentType.includes('tgsticker')) {
          // TGS: Lottie gzipado
          try {
            const blob = new Blob([buf]);
            const ds = new DecompressionStream('gzip');
            const decompressed = await new Response(
              blob.stream().pipeThrough(ds)
            ).json();
            if (!cancelled) {
              setAnimData(decompressed as Record<string, unknown>);
              setStatus('lottie');
            }
          } catch {
            if (!cancelled) setStatus('fallback');
          }
        } else if (contentType.startsWith('video/')) {
          const url = URL.createObjectURL(new Blob([buf], { type: contentType }));
          if (!cancelled) { setBlobUrl(url); setStatus('video'); }
        } else if (contentType.startsWith('image/')) {
          const url = URL.createObjectURL(new Blob([buf], { type: contentType }));
          if (!cancelled) { setBlobUrl(url); setStatus('image'); }
        } else {
          const url = URL.createObjectURL(new Blob([buf], { type: 'video/webm' }));
          if (!cancelled) { setBlobUrl(url); setStatus('video'); }
        }
      })
      .catch(() => {
        if (!cancelled) setStatus('fallback');
      });

    return () => { cancelled = true; };
  }, [emojiId]);

  // Init / destroy Lottie
  useEffect(() => {
    if (status === 'lottie' && lottieRef.current && animData) {
      animInst.current = lottie.loadAnimation({
        container: lottieRef.current,
        animationData: animData,
        loop: true,
        autoplay: true,
      });
    }
    return () => {
      if (animInst.current) {
        (animInst.current as { destroy: () => void }).destroy();
        animInst.current = null;
      }
    };
  }, [status, animData]);

  // Cleanup blob URLs
  useEffect(() => {
    return () => { if (blobUrl) URL.revokeObjectURL(blobUrl); };
  }, [blobUrl]);

  const wrapStyle: React.CSSProperties = {
    width: size,
    height: size,
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
    verticalAlign: 'middle',
    margin: '0 1px',
  };

  if (status === 'fallback' || !emojiId) {
    return <span style={wrapStyle} className={className} />;
  }

  return (
    <span className={className} style={wrapStyle}>
      {status === 'loading' && (
        <span style={{ fontSize: size * 0.5, opacity: 0.3 }}>⋯</span>
      )}

      {status === 'lottie' && (
        <div ref={lottieRef} style={{ width: size, height: size }} />
      )}

      {status === 'video' && blobUrl && (
        <video
          src={blobUrl}
          autoPlay
          loop
          muted
          playsInline
          style={{ width: size, height: size }}
          onError={() => setStatus('fallback')}
        />
      )}

      {status === 'image' && blobUrl && (
        <img
          src={blobUrl}
          alt=""
          style={{ width: size, height: size }}
          onError={() => setStatus('fallback')}
        />
      )}
    </span>
  );
}
