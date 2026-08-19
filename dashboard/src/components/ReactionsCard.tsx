import { useState, useEffect, memo } from 'react';
import { SmilePlus, X, Plus } from 'lucide-react';
import { Button } from './ui/button';

interface ReactionsCardProps {
    reactions: string;
    onUpdate: (reactions: string) => Promise<void>;
}

export const ReactionsCard = memo(({ reactions, onUpdate }: ReactionsCardProps) => {
    const [slots, setSlots] = useState<string[]>(['', '', '', '', '']);
    const [loading, setLoading] = useState(false);

    // Initialize slots from comma-separated string
    useEffect(() => {
        if (reactions) {
            const split = reactions.split(',').map(s => s.trim());
            const newSlots = ['', '', '', '', ''];
            for (let i = 0; i < 5; i++) {
                if (split[i]) newSlots[i] = split[i];
            }
            setSlots(newSlots);
        }
    }, [reactions]);

    const isEmoji = (str: string) => {
        const emojiRegex = /^(\u00a9|\u00ae|[\u2000-\u3300]|\ud83c[\ud000-\udfff]|\ud83d[\ud000-\udfff]|\ud83e[\ud000-\udfff])+$/;
        return emojiRegex.test(str);
    };

    const handleSlotChange = (index: number, value: string) => {
        const trimmed = value.trim();
        if (trimmed === '') {
            const newSlots = [...slots];
            newSlots[index] = '';
            setSlots(newSlots);
            return;
        }

        if (isEmoji(trimmed)) {
            const newSlots = [...slots];
            const emojis = Array.from(trimmed);
            newSlots[index] = emojis[0];
            setSlots(newSlots);
        }
    };

    const handleClearSlot = (index: number) => {
        const newSlots = [...slots];
        newSlots[index] = '';
        setSlots(newSlots);
    };

    const handleSave = async () => {
        setLoading(true);
        try {
            const reactionsString = slots.filter(s => s.trim() !== '').join(',');
            await onUpdate(reactionsString);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="rounded-2xl border border-border/80 bg-card p-4 space-y-4 shadow-sm transition-all hover:border-border">
            <div className="flex items-center gap-3 min-w-0">
                <div className="flex items-center justify-center size-10 rounded-xl shrink-0 bg-purple-500/15 text-purple-400 border border-purple-500/25">
                    <SmilePlus size={19} />
                </div>
                <div className="min-w-0">
                    <h3 className="text-sm font-bold text-foreground leading-tight">Reações / Votos (Grid)</h3>
                    <p className="text-[11px] text-muted-foreground mt-0.5">Adicione até 5 emojis para votação rápida.</p>
                </div>
            </div>

            <div className="space-y-4 pt-1">
                <div className="grid grid-cols-5 gap-2.5">
                    {slots.map((slot, index) => (
                        <div
                            key={index}
                            className={`emoji-slot relative flex min-h-12 items-center justify-center aspect-square rounded-xl border transition-all active:scale-[0.97] focus-within:ring-2 focus-within:ring-ring/60 ${
                                slot
                                    ? 'border-accent/70 bg-accent/15 text-2xl'
                                    : 'border-dashed border-border bg-muted/30 hover:border-accent/60 hover:bg-accent/10'
                            }`}
                        >
                            <input
                                type="text"
                                value={slot}
                                onChange={(e) => handleSlotChange(index, e.target.value)}
                                className="absolute inset-0 w-full h-full opacity-0 cursor-pointer z-10 text-center"
                                aria-label={`Slot de emoji ${index + 1}`}
                                autoComplete="off"
                            />
                            {slot ? (
                                <span className="pointer-events-none select-none text-2xl">{slot}</span>
                            ) : (
                                <Plus size={19} className="text-muted-foreground/80 pointer-events-none" />
                            )}

                            {slot && (
                                <button
                                    type="button"
                                    className="absolute -top-1.5 -right-1.5 z-20 size-5 bg-destructive text-destructive-foreground rounded-full flex items-center justify-center shadow-md hover:bg-destructive/90 transition-transform active:scale-90"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        handleClearSlot(index);
                                    }}
                                    title="Remover emoji"
                                    aria-label={`Remover emoji do slot ${index + 1}`}
                                >
                                    <X size={11} />
                                </button>
                            )}
                        </div>
                    ))}
                </div>

                <Button
                    type="button"
                    className="telegram-primary-action w-full h-12 text-white font-bold rounded-xl text-sm transition-all active:scale-[0.99] disabled:opacity-50"
                    onClick={handleSave}
                    disabled={loading}
                >
                    {loading ? 'Salvando...' : 'Salvar Reações'}
                </Button>
            </div>
        </div>
    );
});
