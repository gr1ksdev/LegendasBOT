import { useState, useEffect, useCallback } from 'react';

type Theme = 'light' | 'dark' | 'telegram';
type BaseTheme = 'light' | 'dark';

const TELEGRAM_CSS_VARS = [
  '--bg', '--card', '--card-elevated', '--text', '--text-secondary', '--hint',
  '--accent', '--accent-rgb', '--accent-soft', '--accent-hover', '--accent-text',
  '--link', '--border', '--border-active', '--surface-rgb', '--surface',
  '--surface-hover', '--nav-bg', '--input-bg-rgb', '--input-bg', '--overlay',
  '--toggle-bg', '--toggle-knob',
];

const isTheme = (value: string | null): value is Theme => {
  return value === 'light' || value === 'dark' || value === 'telegram';
};

const getAutoTheme = (): BaseTheme => {
  const hour = new Date().getHours();
  if (hour >= 18 || hour < 6) return 'dark';

  const tgScheme = window.Telegram?.WebApp?.colorScheme;
  if (tgScheme === 'light' || tgScheme === 'dark') return tgScheme;

  if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light';
  return 'dark';
};

const hexToRgb = (hex: string, fallback: string) => {
  const normalized = hex.trim().replace('#', '');
  if (!/^[0-9a-fA-F]{6}$/.test(normalized)) return fallback;
  const value = parseInt(normalized, 16);
  return `${(value >> 16) & 255}, ${(value >> 8) & 255}, ${value & 255}`;
};

const applyTelegramVars = () => {
  const root = document.documentElement;
  const tg = window.Telegram?.WebApp;
  const params = tg?.themeParams || {};
  const scheme: BaseTheme = tg?.colorScheme === 'light' || tg?.colorScheme === 'dark'
    ? tg.colorScheme
    : getAutoTheme();

  const fallback = scheme === 'light'
    ? {
        bg: '#0f1115', card: 'rgba(255, 255, 255, 0.05)', text: '#e8edf5', hint: '#94a3b8',
        secondary: '#64748b', accent: '#6366f1', border: 'transparent', surfaceRgb: '255, 255, 255',
        nav: 'rgba(15, 17, 21, 0.88)', overlay: 'rgba(0, 0, 0, 0.6)', inputRgb: '255, 255, 255',
      }
    : {
        bg: '#0a0c10', card: 'rgba(255, 255, 255, 0.05)', text: '#f1f5f9', hint: '#475569',
        secondary: '#94a3b8', accent: '#818cf8', border: 'transparent', surfaceRgb: '255, 255, 255',
        nav: 'rgba(10, 12, 16, 0.92)', overlay: 'rgba(0, 0, 0, 0.7)', inputRgb: '255, 255, 255',
      };

  const bg = params.bg_color || fallback.bg;
  const secondaryBg = params.secondary_bg_color || bg;
  const text = params.text_color || fallback.text;
  const hint = params.hint_color || fallback.hint;
  const link = params.link_color || fallback.accent;
  const button = params.button_color || link;
  const buttonText = params.button_text_color || '#ffffff';
  const surfaceRgb = hexToRgb(secondaryBg, fallback.surfaceRgb);
  const inputRgb = hexToRgb(secondaryBg, fallback.inputRgb);
  const accentRgb = hexToRgb(button, scheme === 'light' ? '99, 102, 241' : '129, 140, 248');

  root.style.setProperty('--bg', bg);
  root.style.setProperty('--card', 'rgba(255, 255, 255, 0.05)');
  root.style.setProperty('--card-elevated', 'rgba(255, 255, 255, 0.05)');
  root.style.setProperty('--text', text);
  root.style.setProperty('--text-secondary', params.subtitle_text_color || fallback.secondary);
  root.style.setProperty('--hint', hint);
  root.style.setProperty('--accent', button);
  root.style.setProperty('--accent-rgb', accentRgb);
  root.style.setProperty('--accent-soft', `rgba(${accentRgb}, 0.12)`);
  root.style.setProperty('--accent-hover', `rgba(${accentRgb}, 0.18)`);
  root.style.setProperty('--accent-text', buttonText);
  root.style.setProperty('--link', link);
  root.style.setProperty('--border', 'transparent');
  root.style.setProperty('--border-active', button);
  root.style.setProperty('--surface-rgb', surfaceRgb);
  root.style.setProperty('--surface', 'rgba(255, 255, 255, 0.05)');
  root.style.setProperty('--surface-hover', 'rgba(255, 255, 255, 0.08)');
  root.style.setProperty('--nav-bg', params.header_bg_color || fallback.nav);
  root.style.setProperty('--input-bg-rgb', inputRgb);
  root.style.setProperty('--input-bg', 'rgba(255, 255, 255, 0.05)');
  root.style.setProperty('--overlay', fallback.overlay);
  root.style.setProperty('--toggle-bg', params.secondary_bg_color || (scheme === 'light' ? '#e2e8f0' : '#1e293b'));
  root.style.setProperty('--toggle-knob', params.button_text_color || '#ffffff');

  return { bg, scheme };
};

export function useTheme() {
  const [theme] = useState<Theme>('telegram');

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', 'telegram');
    const { bg } = applyTelegramVars();

    const tg = window.Telegram?.WebApp;
    if (tg) {
      try {
        tg.setHeaderColor(bg);
        tg.setBackgroundColor(bg);
      } catch {}
    }

    let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement;
    if (!meta) {
      meta = document.createElement('meta');
      meta.name = 'theme-color';
      document.head.appendChild(meta);
    }
    meta.content = bg;
  }, []);

  const toggleTheme = useCallback(() => {}, []);

  return { theme, toggleTheme };
}
