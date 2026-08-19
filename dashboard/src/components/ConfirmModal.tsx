import { AlertTriangle, Send } from 'lucide-react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter, DialogClose,
} from './ui/dialog';
import { Button } from './ui/button';

interface ConfirmModalProps {
    open: boolean;
    onClose: () => void;
    onConfirm: () => void;
    title: string;
    message: string;
    confirmText?: string;
    danger?: boolean;
    alertOnly?: boolean;
}

export function ConfirmModal({
    open, onClose, onConfirm, title, message, confirmText, danger, alertOnly
}: ConfirmModalProps) {
    return (
        <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose(); }}>
            <DialogContent className="sm:max-w-md p-6 text-center bg-card text-card-foreground border border-border/80 shadow-2xl rounded-2xl" showCloseButton={false}>
                <DialogHeader className="items-center gap-3">
                    <div
                        className={`flex size-14 items-center justify-center rounded-2xl transition-all shadow-sm ${
                            danger
                                ? 'bg-destructive/15 text-destructive border border-destructive/30'
                                : 'bg-accent/15 text-accent border border-accent/30'
                        }`}
                    >
                        {danger ? <AlertTriangle size={26} /> : <Send size={24} />}
                    </div>
                    <DialogTitle className="text-lg font-bold text-foreground tracking-tight mt-1">
                        {title}
                    </DialogTitle>
                    <DialogDescription className="text-xs leading-relaxed px-2 text-muted-foreground font-medium">
                        {message}
                    </DialogDescription>
                </DialogHeader>
                <DialogFooter className="flex flex-row items-center justify-center gap-3 mt-4 pt-2 border-t-0 p-0">
                    {!alertOnly && (
                        <DialogClose render={
                            <Button variant="outline" className="flex-1 rounded-xl h-11 px-4 text-xs font-semibold border-border bg-surface text-foreground hover:bg-muted transition-colors">
                                Cancelar
                            </Button>
                        } />
                    )}
                    <Button
                        variant={danger ? "destructive" : "default"}
                        className="flex-1 rounded-xl h-11 px-4 text-xs font-bold transition-all shadow-md"
                        onClick={() => { onConfirm(); onClose(); }}
                    >
                        {confirmText || 'Confirmar'}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
