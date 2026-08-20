import { useState, useEffect, useRef } from 'react';
import { Save, Settings, FileText, Wrench, RefreshCw } from 'lucide-react';
import { fetchServerConfig, updateServerConfig } from '../api';
import { ServerConfig } from '../types';
import { useToast } from './Toast';
import { RichTextEditor } from './RichTextEditor';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Switch } from './ui/switch';
import { Textarea } from './ui/textarea';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter, CardAction } from './ui/card';

const REFRESH_INTERVAL = 30 * 60 * 1000; // 30 minutes

export function AdminConfigTab() {
    const [config, setConfig] = useState<ServerConfig | null>(null);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const autoRefreshed = useRef(false);

    const [globalDefault, setGlobalDefault] = useState('');
    const [globalNewPack, setGlobalNewPack] = useState('');
    const [fixedPostEnabled, setFixedPostEnabled] = useState(true);
    const [fixedPostKey, setFixedPostKey] = useState('legendasbot');
    const [fixedPostPayload, setFixedPostPayload] = useState('');
    const [logRetentionDays, setLogRetentionDays] = useState(30);

    const toast = useToast();

    // ── Load config on mount ──
    useEffect(() => {
        const loadConfig = async () => {
            try {
                const res = await fetchServerConfig();
                if (res.success) {
                    const serverData = res.data || res.config;
                    if (serverData) {
                        setConfig(serverData);
                        setGlobalDefault(serverData.globalDefaultCaption || '');
                        setGlobalNewPack(serverData.globalNewPackCaption || '');
                        setFixedPostEnabled(Boolean(serverData.fixedPostBuilderEnabled));
                        setFixedPostKey(serverData.fixedPostBuilderKey || 'legendasbot');
                        setFixedPostPayload(serverData.fixedPostBuilderPayload || '');
                        setLogRetentionDays(serverData.logRetentionDays || 30);
                    }
                }
            } catch {
                toast('Erro ao carregar configurações', 'error');
            } finally {
                setLoading(false);
            }
        };
        loadConfig();
    }, [toast]);

    // ── Auto-refresh PostBuilder cache on mount + periodic ──
    useEffect(() => {
        if (loading || !config || !fixedPostEnabled) return;

        const refresh = async () => {
            try {
                await updateServerConfig({
                    maintence: config.maintence,
                    forceJoin: config.forceJoin,
                    globalDefaultCaption: globalDefault,
                    globalNewPackCaption: globalNewPack,
                    fixedPostBuilderEnabled: fixedPostEnabled,
                    fixedPostBuilderKey: fixedPostKey,
                    fixedPostBuilderPayload: fixedPostPayload,
                    logRetentionDays: logRetentionDays,
                });
            } catch {
                // silent — keep old cache alive
            }
        };

        const interval = setInterval(refresh, REFRESH_INTERVAL);
        return () => clearInterval(interval);
    }, [loading, !!config, fixedPostEnabled, logRetentionDays]);

    const handleSave = async (overrides: Partial<ServerConfig> = {}) => {
        if (!config) return;

        const payload = {
            maintence: overrides.maintence ?? config.maintence,
            forceJoin: overrides.forceJoin ?? config.forceJoin,
            globalDefaultCaption: overrides.globalDefaultCaption ?? globalDefault,
            globalNewPackCaption: overrides.globalNewPackCaption ?? globalNewPack,
            fixedPostBuilderEnabled: overrides.fixedPostBuilderEnabled ?? fixedPostEnabled,
            fixedPostBuilderKey: overrides.fixedPostBuilderKey ?? fixedPostKey,
            fixedPostBuilderPayload: overrides.fixedPostBuilderPayload ?? fixedPostPayload,
            logRetentionDays: overrides.logRetentionDays ?? logRetentionDays,
        };

        setSaving(true);
        try {
            const res = await updateServerConfig(payload);
            if (res.success) {
                const serverData = res.data || res.config;
                if (serverData) {
                    setConfig(serverData);
                    setGlobalDefault(serverData.globalDefaultCaption || '');
                    setGlobalNewPack(serverData.globalNewPackCaption || '');
                    setFixedPostEnabled(Boolean(serverData.fixedPostBuilderEnabled));
                    setFixedPostKey(serverData.fixedPostBuilderKey || 'legendasbot');
                    setFixedPostPayload(serverData.fixedPostBuilderPayload || '');
                    setLogRetentionDays(serverData.logRetentionDays || 30);
                }
                toast('Configurações atualizadas com sucesso', 'success');
            }
        } catch (err) {
            toast('Erro ao atualizar configurações', 'error');
        } finally {
            setSaving(false);
        }
    };

    const handleToggle = (field: 'maintence' | 'forceJoin' | 'fixedPostBuilderEnabled') => {
        if (!config) return;
        if (field === 'fixedPostBuilderEnabled') {
            const next = !fixedPostEnabled;
            setFixedPostEnabled(next);
            handleSave({ fixedPostBuilderEnabled: next });
            return;
        }
        handleSave({ [field]: !config[field] });
    };

    const refreshPostBuilderCache = async () => {
        if (!config || saving) return;
        await handleSave({
            fixedPostBuilderEnabled: fixedPostEnabled,
            fixedPostBuilderKey: fixedPostKey,
            fixedPostBuilderPayload: fixedPostPayload,
        });
        toast('Cache do PostBuilder renovado', 'success');
    };

    if (loading) return (
        <Card>
            <CardContent className="flex flex-col items-center py-12 gap-3">
                <div className="auth-spinner" />
                <p className="text-sm text-muted-foreground">Carregando configurações...</p>
            </CardContent>
        </Card>
    );

    return (
        <div className="admin-config-page grid gap-5">
            {/* ── Sistema ── */}
            <Card className="admin-config-card admin-config-system">
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <Settings size={16} className="text-accent" />
                        <CardTitle>Sistema</CardTitle>
                    </div>
                    <CardDescription>Estado do bot e comportamento geral</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-lg border border-border p-3">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Manutenção</p>
                            <p className="text-xs text-muted-foreground">
                                {config?.maintence ? 'Bot offline para usuários' : 'Operando normalmente'}
                            </p>
                        </div>
                        <Switch
                            checked={!!config?.maintence}
                            onCheckedChange={() => !saving && handleToggle('maintence')}
                            disabled={saving}
                        />
                    </div>
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-lg border border-border p-3">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Force Join</p>
                            <p className="text-xs text-muted-foreground">
                                {config?.forceJoin ? 'Inscrição obrigatória' : 'Acesso livre'}
                            </p>
                        </div>
                        <Switch
                            checked={!!config?.forceJoin}
                            onCheckedChange={() => !saving && handleToggle('forceJoin')}
                            disabled={saving}
                        />
                    </div>
                    <div className="admin-config-row flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-lg border border-border p-3">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Retenção de Logs (dias)</p>
                            <p className="text-xs text-muted-foreground">
                                Limpeza automática a cada 1 hora para registros com mais de {logRetentionDays} dias
                            </p>
                        </div>
                        <div className="flex items-center gap-2">
                            <Input
                                type="number"
                                min={1}
                                max={365}
                                value={logRetentionDays}
                                onChange={(e) => setLogRetentionDays(Math.max(1, parseInt(e.target.value) || 1))}
                                disabled={saving}
                                className="w-20 h-9 text-xs font-semibold text-center"
                            />
                            <span className="text-xs text-muted-foreground font-medium">dias</span>
                        </div>
                    </div>
                </CardContent>
                <CardFooter className="admin-config-footer justify-end gap-2">
                    <Button
                        variant="default"
                        onClick={() => !saving && handleSave()}
                        disabled={saving}
                        className="admin-config-save-button"
                    >
                        <Save size={15} className="mr-1.5" />
                        {saving ? 'Salvando...' : 'Salvar'}
                    </Button>
                </CardFooter>
            </Card>

            {/* ── Legendas ── */}
            <Card className="admin-config-card admin-config-captions">
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <FileText size={16} className="text-accent" />
                        <CardTitle>Legendas</CardTitle>
                    </div>
                    <CardDescription>Conteúdo padrão aplicado a canais e packs</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="admin-config-editor">
                        <p className="text-sm font-medium mb-1">Legenda Padrão Global</p>
                        <p className="text-xs text-muted-foreground mb-2">Preenche novos canais vinculados ao bot</p>
                        <RichTextEditor
                            value={globalDefault}
                            onChange={setGlobalDefault}
                            placeholder="Ex: @legendasbot [t.me/legendasbot](https://t.me/botusername)"
                        />
                    </div>
                    <div className="admin-config-editor">
                        <p className="text-sm font-medium mb-1">Legenda de Novo Pack</p>
                        <p className="text-xs text-muted-foreground mb-2">Valor inicial para mensagem de pack padrão</p>
                        <RichTextEditor
                            value={globalNewPack}
                            onChange={setGlobalNewPack}
                            placeholder="Texto inicial para novos packs..."
                        />
                    </div>
                </CardContent>
                <CardFooter className="admin-config-footer justify-end gap-2">
                    <Button
                        variant="default"
                        onClick={() => !saving && handleSave()}
                        disabled={saving}
                        className="admin-config-save-button"
                    >
                        <Save size={15} className="mr-1.5" />
                        {saving ? 'Salvando...' : 'Salvar'}
                    </Button>
                </CardFooter>
            </Card>

            {/* ── PostBuilder ── */}
            <Card className="admin-config-card admin-config-postbuilder">
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <Wrench size={16} className="text-accent" />
                        <CardTitle>PostBuilder</CardTitle>
                    </div>
                    <CardDescription>Post permanente usado em consultas inline</CardDescription>
                    <CardAction className="admin-config-card-action">
                        <Button variant="ghost" size="sm" onClick={refreshPostBuilderCache} disabled={saving || !config} title="Renovar cache do Redis">
                            <RefreshCw size={14} className={saving ? 'animate-spin' : ''} />
                        </Button>
                    </CardAction>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-lg border border-border p-3">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Postagem fixa</p>
                            <p className="text-xs text-muted-foreground">Post permanente usado no inline com chave fixa</p>
                        </div>
                        <Switch
                            checked={fixedPostEnabled}
                            onCheckedChange={() => !saving && handleToggle('fixedPostBuilderEnabled')}
                            disabled={saving}
                        />
                    </div>
                    {fixedPostEnabled && (
                        <div className="admin-config-postbuilder-fields space-y-3 pl-1">
                            <div>
                                <label className="text-sm font-medium">Chave fixa</label>
                                <Input
                                    value={fixedPostKey}
                                    onChange={(e) => setFixedPostKey(e.target.value)}
                                    placeholder="legendasbot"
                                    disabled={saving}
                                    className="mt-1"
                                />
                            </div>
                            <div>
                                <label className="text-sm font-medium">Payload JSON</label>
                                <Textarea
                                    value={fixedPostPayload}
                                    onChange={(e) => setFixedPostPayload(e.target.value)}
                                    placeholder='{ "media_type": "photo", "media_file_id": "..." }'
                                    disabled={saving}
                                    className="admin-config-payload mt-1 min-h-[100px] font-mono text-xs"
                                />
                            </div>
                            <p className="text-xs text-muted-foreground">
                                Uso inline:{' '}
                                <code className="rounded bg-muted px-1 py-0.5 text-[11px] font-mono text-foreground">
                                    @FreddyCaptionBot pb {fixedPostKey || 'legendasbot'}
                                </code>
                                . Quando desativado, a chave é removida do Redis.
                            </p>
                        </div>
                    )}
                </CardContent>
                <CardFooter className="admin-config-footer justify-end gap-2">
                    <Button
                        variant="default"
                        onClick={() => !saving && handleSave()}
                        disabled={saving}
                        className="admin-config-save-button"
                    >
                        <Save size={15} className="mr-1.5" />
                        {saving ? 'Salvando...' : 'Salvar'}
                    </Button>
                </CardFooter>
            </Card>
        </div>
    );
}
