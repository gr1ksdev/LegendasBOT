import React, { useEffect } from 'react';
import { Radio, FileText, Calendar, Headphones, ChevronRight, Sun, Moon, Send } from 'lucide-react';
import { TelegramUser } from '../types';

interface SideMenuProps {
  isOpen: boolean;
  onClose: () => void;
  tgUser?: TelegramUser | null;
  displayName: string;
  userId: number;
  channelsCount: number;
  onOpenTemplates: () => void;
  onOpenConta: () => void;
  onOpenSchedules?: () => void;
  onNavigateChannels: () => void;
  theme: string;
  toggleTheme: () => void;
  connectedAccountEnabled?: boolean;
}

export const SideMenu: React.FC<SideMenuProps> = ({
  isOpen,
  onClose,
  tgUser,
  displayName,
  userId,
  channelsCount,
  onOpenTemplates,
  onOpenConta,
  onOpenSchedules,
  onNavigateChannels,
  theme,
  toggleTheme,
}) => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  const initials = displayName[0]?.toUpperCase() || '?';

  return (
    <div className="fixed inset-0 z-[20000] flex">
      {/* Backdrop overlay */}
      <div 
        className="fixed inset-0 bg-black/60 backdrop-blur-sm transition-opacity animate-in fade-in duration-200"
        onClick={onClose}
      />

      {/* Drawer content sliding from left - Montserrat font, clean borderless */}
      <div 
        className="relative z-10 w-64 max-w-[75vw] h-full bg-background p-5 flex flex-col justify-between shadow-2xl animate-in slide-in-from-left duration-300"
        style={{ fontFamily: "'Montserrat', sans-serif" }}
      >
        <div className="space-y-4">
          {/* User Profile Header (Top Left): Circular photo + Name & ID centered */}
          <div className="flex items-center gap-3 pt-1 pb-2">
            <div className="w-12 h-12 rounded-full overflow-hidden shrink-0 bg-accent text-accent-foreground flex items-center justify-center font-bold text-lg shadow-sm">
              {tgUser?.photo_url ? (
                <img src={tgUser.photo_url} alt="" className="w-full h-full object-cover" />
              ) : (
                initials
              )}
            </div>
            <div className="min-w-0 flex-1 flex flex-col justify-center">
              <h3 className="font-bold text-base text-foreground truncate leading-tight">{displayName}</h3>
              <p className="text-xs text-muted-foreground truncate mt-0.5">ID: {userId || 'N/A'}</p>
            </div>
          </div>

          {/* "Minha Conta" Div: Soft white background, borderless, title small top-left, count BIG in center */}
          <div 
            className="p-4 rounded-2xl bg-white/5 cursor-pointer hover:bg-white/10 transition-colors"
            onClick={() => {
              onNavigateChannels();
              onClose();
            }}
          >
            <div className="text-xs font-bold uppercase tracking-wider text-muted-foreground/80 mb-1">
              Minha Conta
            </div>
            <div className="flex flex-col items-center justify-center py-2">
              <span className="text-5xl font-black text-foreground tracking-tight">{channelsCount}</span>
              <span className="text-xs text-muted-foreground font-medium mt-1">
                {channelsCount === 1 ? 'canal cadastrado' : 'canais cadastrados'}
              </span>
            </div>
          </div>

          {/* Navigation Card Div: Soft white background, rounded corners, borderless */}
          <div className="rounded-2xl bg-white/5 overflow-hidden">
            {/* Item 1: Meus Canais */}
            <div className="relative">
              <button
                onClick={() => {
                  onNavigateChannels();
                  onClose();
                }}
                className="w-full flex items-center justify-between px-4 py-3.5 text-sm font-semibold text-foreground hover:bg-white/10 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <Radio size={18} className="text-foreground/80 shrink-0" />
                  <span>Meus Canais</span>
                </div>
                <ChevronRight size={16} className="text-muted-foreground/70 shrink-0" />
              </button>
              <div className="absolute bottom-0 left-[46px] right-4 h-px bg-white/10 pointer-events-none" />
            </div>

            {/* Item 2: Templates */}
            <div className="relative">
              <button
                onClick={() => {
                  onOpenTemplates();
                  onClose();
                }}
                className="w-full flex items-center justify-between px-4 py-3.5 text-sm font-semibold text-foreground hover:bg-white/10 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <FileText size={18} className="text-foreground/80 shrink-0" />
                  <span>Templates</span>
                </div>
                <ChevronRight size={16} className="text-muted-foreground/70 shrink-0" />
              </button>
              <div className="absolute bottom-0 left-[46px] right-4 h-px bg-white/10 pointer-events-none" />
            </div>

            {/* Item 3: Agendamentos */}
            <div className="relative">
              <button
                onClick={() => {
                  if (onOpenSchedules) {
                    onOpenSchedules();
                  } else {
                    onNavigateChannels();
                  }
                  onClose();
                }}
                className="w-full flex items-center justify-between px-4 py-3.5 text-sm font-semibold text-foreground hover:bg-white/10 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <Calendar size={18} className="text-foreground/80 shrink-0" />
                  <span>Agendamentos</span>
                </div>
                <ChevronRight size={16} className="text-muted-foreground/70 shrink-0" />
              </button>
              <div className="absolute bottom-0 left-[46px] right-4 h-px bg-white/10 pointer-events-none" />
            </div>

            {/* Item 4: Suporte */}
            <a
              href="https://t.me/LegendasBOTTopic"
              target="_blank"
              rel="noopener noreferrer"
              onClick={onClose}
              className="w-full flex items-center justify-between px-4 py-3.5 text-sm font-semibold text-foreground hover:bg-white/10 transition-colors"
            >
              <div className="flex items-center gap-3">
                <Headphones size={18} className="text-foreground/80 shrink-0" />
                <span>Suporte</span>
              </div>
              <ChevronRight size={16} className="text-muted-foreground/70 shrink-0" />
            </a>

            {/* Item 5: Voltar ao Painel Admin */}
            {(sessionStorage.getItem('navSource') === 'admin' || window.location.pathname.includes('/admin/')) && (
              <div className="relative">
                <button
                  onClick={() => {
                    sessionStorage.removeItem('navSource');
                    window.location.href = '/admin/dash?tab=channels';
                  }}
                  className="w-full flex items-center justify-between px-4 py-3.5 text-sm font-semibold text-foreground hover:bg-white/10 transition-colors"
                >
                  <div className="flex items-center gap-3">
                    <Radio size={18} className="text-foreground/80 shrink-0" />
                    <span>Painel Admin</span>
                  </div>
                  <ChevronRight size={16} className="text-muted-foreground/70 shrink-0" />
                </button>
                <div className="absolute bottom-0 left-[46px] right-4 h-px bg-white/10 pointer-events-none" />
              </div>
            )}
          </div>

          {/* Theme switcher button */}
          <button
            onClick={toggleTheme}
            className="w-full flex items-center justify-between px-4 py-3 rounded-2xl bg-white/5 hover:bg-white/10 text-xs font-semibold text-foreground transition-colors"
          >
            <div className="flex items-center gap-3">
              {theme === 'telegram' ? (
                <Send size={16} className="text-foreground/80 shrink-0" />
              ) : theme === 'dark' ? (
                <Sun size={16} className="text-foreground/80 shrink-0" />
              ) : (
                <Moon size={16} className="text-foreground/80 shrink-0" />
              )}
              <span>Alternar Tema</span>
            </div>
            <span className="text-[11px] text-muted-foreground capitalize">
              {theme === 'telegram' ? 'Telegram' : theme === 'dark' ? 'Escuro' : 'Claro'}
            </span>
          </button>
        </div>
      </div>
    </div>
  );
};
