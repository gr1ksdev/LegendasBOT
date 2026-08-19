import { useState, useCallback, createContext, useContext, useRef } from 'react';
import { CheckCircle2, Info, AlertCircle, X } from 'lucide-react';

interface ToastItem {
  id: string;
  message: string;
  type: 'success' | 'info' | 'error';
}

type ToastFn = (message: string, type?: 'success' | 'info' | 'error') => void;
const ToastContext = createContext<ToastFn>(() => { });
export function useToast() { return useContext(ToastContext); }

const icons = {
  success: <CheckCircle2 size={18} />,
  info: <Info size={18} />,
  error: <AlertCircle size={18} />,
};

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const lastToastRef = useRef<{ message: string; time: number } | null>(null);

  const removeToast = useCallback((id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  const addToast: ToastFn = useCallback((message, type = 'success') => {
    const now = Date.now();
    // Dedup: se mesma mensagem foi chamada nos últimos 100ms, ignora
    if (lastToastRef.current && lastToastRef.current.message === message && now - lastToastRef.current.time < 100) {
      return;
    }
    lastToastRef.current = { message, time: now };

    const id = Math.random().toString(36).slice(2);
    setToasts(prev => [...prev, { id, message, type }]);
    setTimeout(() => removeToast(id), 4000);
  }, [removeToast]);

  return (
    <ToastContext.Provider value={addToast}>
      {children}
      <div className="toast-wrap">
        {toasts.map(t => (
          <div key={t.id} className={`toast-msg toast-${t.type}`}>
            <div className="toast-icon">{icons[t.type]}</div>
            <div className="toast-content">
              <span className="toast-text">{t.message}</span>
            </div>
            <button className="toast-close" onClick={() => removeToast(t.id)}>
              <X size={14} />
            </button>
            <div className="toast-progress" />
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
