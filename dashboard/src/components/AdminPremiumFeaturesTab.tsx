import { useState, useEffect, useCallback } from 'react';
import { Card, CardContent } from './ui/card';
import { Button } from './ui/button';
import { Badge } from './ui/badge';
import { PremiumFeature } from '../types';
import { fetchPremiumFeatures, togglePremiumFeature, updatePremiumFeature } from '../api';
import { Crown, ToggleLeft, ToggleRight, Star, Loader2, Save } from 'lucide-react';

interface AdminPremiumFeaturesTabProps {
    toast: (message: string, type: 'success' | 'error' | 'info') => void;
}

export function AdminPremiumFeaturesTab({ toast }: AdminPremiumFeaturesTabProps) {
    const [features, setFeatures] = useState<PremiumFeature[]>([]);
    const [loading, setLoading] = useState(true);
    const [toggling, setToggling] = useState<string | null>(null);
    const [editingPrice, setEditingPrice] = useState<Record<string, number>>({});
    const [savingPrice, setSavingPrice] = useState<string | null>(null);

    const load = useCallback(async () => {
        try {
            const res = await fetchPremiumFeatures();
            const data = Array.isArray(res?.data) ? res.data : [];
            setFeatures(data);
            // Init editing prices
            const prices: Record<string, number> = {};
            data.forEach((f: PremiumFeature) => { prices[f.key] = f.price; });
            setEditingPrice(prices);
        } catch (err: any) {
            toast(err?.message || 'Erro ao carregar features', 'error');
        } finally {
            setLoading(false);
        }
    }, [toast]);

    useEffect(() => { load(); }, [load]);

    const handleToggle = async (key: string, currentEnabled: boolean) => {
        setToggling(key);
        try {
            await togglePremiumFeature(key, !currentEnabled);
            setFeatures(prev => prev.map(f =>
                f.key === key ? { ...f, enabled: !currentEnabled } : f
            ));
            toast(`Feature ${!currentEnabled ? 'ativada' : 'desativada'}`, 'success');
        } catch (err: any) {
            toast(err?.message || 'Erro ao alternar feature', 'error');
        } finally {
            setToggling(null);
        }
    };

    const handleSavePrice = async (key: string) => {
        setSavingPrice(key);
        try {
            const newPrice = editingPrice[key] ?? 0;
            await updatePremiumFeature(key, { price: newPrice });
            setFeatures(prev => prev.map(f =>
                f.key === key ? { ...f, price: newPrice } : f
            ));
            toast('Preço atualizado!', 'success');
        } catch (err: any) {
            toast(err?.message || 'Erro ao atualizar preço', 'error');
        } finally {
            setSavingPrice(null);
        }
    };

    const getFeatureIcon = (key: string): string => {
        switch (key) {
            case 'managed_premium_account': return '🤖';
            case 'connected_account': return '👤';
            case 'custom_emojis': return '✨';
            case 'extra_channels': return '📡';
            default: return '⚙️';
        }
    };

    if (loading) {
        return (
            <Card>
                <CardContent className="flex flex-col items-center justify-center py-12">
                    <Loader2 size={32} className="animate-spin mb-4 text-muted-foreground" />
                    <p className="text-sm text-muted-foreground">Carregando features...</p>
                </CardContent>
            </Card>
        );
    }

    return (
        <div className="admin-features-page space-y-5">
            {/* Header */}
            <Card className="admin-feature-intro">
                <CardContent className="pt-4">
                    <div className="flex items-center gap-3 mb-1">
                        <div className="flex items-center justify-center size-10 rounded-xl shrink-0" style={{ background: 'var(--accent-soft)' }}>
                            <Crown size={22} style={{ color: 'var(--accent)' }} />
                        </div>
                        <div>
                            <h2 className="text-base font-bold">Features Premium</h2>
                            <p className="text-xs text-muted-foreground">
                                Ative ou desative recursos premium globalmente
                            </p>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Feature Cards */}
            {features.map((feature, idx) => (
                <Card
                    key={feature.key}
                    className={`admin-feature-row transition-colors ${!feature.enabled ? 'opacity-70' : ''}`}
                    style={{ animationDelay: `${0.05 + idx * 0.04}s` }}
                >
                    <CardContent className="pt-4">
                        <div className="flex items-start justify-between gap-4">
                            <div className="flex items-start gap-3 min-w-0 flex-1">
                                <div className="flex items-center justify-center size-10 rounded-xl shrink-0 text-lg" style={{ background: 'var(--accent-soft)' }}>
                                    {getFeatureIcon(feature.key)}
                                </div>
                                <div className="min-w-0 flex-1">
                                    <div className="flex items-center gap-2 flex-wrap">
                                        <h3 className="text-sm font-bold">{feature.name}</h3>
                                        <Badge variant={feature.enabled ? "default" : "secondary"} className="text-[10px] px-2 py-0">
                                            {feature.enabled ? 'ATIVO' : 'INATIVO'}
                                        </Badge>
                                    </div>
                                    <p className="text-xs text-muted-foreground mt-1">{feature.description}</p>

                                    {/* Price editor */}
                                    <div className="flex items-center gap-2 mt-3">
                                        <div className="flex items-center gap-1 bg-muted/50 rounded-lg px-3 py-1.5">
                                            <Star size={14} className="text-yellow-500 fill-yellow-500 shrink-0" />
                                            <input
                                                type="number"
                                                min={0}
                                                max={999}
                                                className="w-16 bg-transparent text-sm font-semibold text-center outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                                                value={editingPrice[feature.key] ?? feature.price}
                                                onChange={e => setEditingPrice(prev => ({ ...prev, [feature.key]: parseInt(e.target.value) || 0 }))}
                                            />
                                            <span className="text-[10px] text-muted-foreground">Stars</span>
                                        </div>
                                        <Button
                                            variant="ghost"
                                            size="sm"
                                            className="h-7 px-2 text-xs"
                                            onClick={() => handleSavePrice(feature.key)}
                                            disabled={savingPrice === feature.key || editingPrice[feature.key] === feature.price}
                                        >
                                            {savingPrice === feature.key ? (
                                                <Loader2 size={12} className="animate-spin" />
                                            ) : (
                                                <Save size={12} />
                                            )}
                                            <span className="ml-1">Salvar</span>
                                        </Button>
                                    </div>
                                </div>
                            </div>

                            {/* Toggle */}
                            <button
                                className="shrink-0 mt-1"
                                onClick={() => handleToggle(feature.key, feature.enabled)}
                                disabled={toggling === feature.key}
                            >
                                {toggling === feature.key ? (
                                    <Loader2 size={28} className="animate-spin text-muted-foreground" />
                                ) : feature.enabled ? (
                                    <ToggleRight size={28} className="text-green-500" />
                                ) : (
                                    <ToggleLeft size={28} className="text-muted-foreground/40" />
                                )}
                            </button>
                        </div>
                    </CardContent>
                </Card>
            ))}
        </div>
    );
}
