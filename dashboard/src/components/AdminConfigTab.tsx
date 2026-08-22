import { useState, useEffect, useRef, type ReactNode } from 'react';
import { Save, Settings, FileText, Wrench, RefreshCw, Clock3, Database } from 'lucide-react';
import { fetchServerConfig, updateServerConfig, getCachedServerConfig } from '../api';
import { ServerConfig } from '../types';
import { useToast } from './Toast';
import { RichTextEditor } from './RichTextEditor';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Switch } from './ui/switch';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter, CardAction } from './ui/card';

const REFRESH_INTERVAL = 30 * 60 * 1000; // 30 minutes

function formatJsonForEditor(value: string): string {
    if (!value.trim()) return '';
    try {
        return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
        return value;
    }
}

const JSON_TOKEN_PATTERN = /("(?:\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"\s*:)|("(?:\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*")|\b(true|false)\b|\b(null)\b|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g;

function highlightJson(value: string): ReactNode[] {
    const nodes: ReactNode[] = [];
    let cursor = 0;

    for (const match of value.matchAll(JSON_TOKEN_PATTERN)) {
        const index = match.index ?? 0;
        if (index > cursor) nodes.push(value.slice(cursor, index));

        const token = match[0];
        const kind = match[1] ? 'key' : match[2] ? 'string' : match[3] ? 'boolean' : match[4] ? 'null' : 'number';
        nodes.push(<span className={`json-token json-${kind}`} key={`${index}-${kind}`}>{token}</span>);
        cursor = index + token.length;
    }

    if (cursor < value.length) nodes.push(value.slice(cursor));
    return nodes;
}

function JsonPayloadEditor({ value, onChange, disabled }: { value: string; onChange: (value: string) => void; disabled: boolean }) {
    const highlightRef = useRef<HTMLPreElement>(null);

    return (
        <div className="admin-config-json-editor">
            <pre ref={highlightRef} aria-hidden="true" className={!value ? 'is-placeholder' : undefined}>
                <code>{value ? highlightJson(value) : '{ "media_type": "photo", "media_file_id": "..." }'}</code>
            </pre>
            <textarea
                value={value}
                onChange={(event) => onChange(event.target.value)}
                onScroll={(event) => {
                    if (!highlightRef.current) return;
                    highlightRef.current.scrollTop = event.currentTarget.scrollTop;
                    highlightRef.current.scrollLeft = event.currentTarget.scrollLeft;
                }}
                disabled={disabled}
                spellCheck={false}
                aria-label="Payload JSON do PostBuilder"
            />
        </div>
    );
}

export function AdminConfigTab() {
    const initialConfig = getCachedServerConfig();
    const [config, setConfig] = useState<ServerConfig | null>(initialConfig);
    const [loading, setLoading] = useState(!initialConfig);
    const [saving, setSaving] = useState(false);
    const [savingSection, setSavingSection] = useState<'system' | 'captions' | 'postbuilder' | null>(null);

    const [globalDefault, setGlobalDefault] = useState(initialConfig?.globalDefaultCaption || '');
    const [globalNewPack, setGlobalNewPack] = useState(initialConfig?.globalNewPackCaption || '');
    const [fixedPostEnabled, setFixedPostEnabled] = useState(initialConfig ? Boolean(initialConfig.fixedPostBuilderEnabled) : true);
    const [fixedPostKey, setFixedPostKey] = useState(initialConfig?.fixedPostBuilderKey || 'legendasbot');
    const [fixedPostPayload, setFixedPostPayload] = useState(() => formatJsonForEditor(initialConfig?.fixedPostBuilderPayload || ''));
    const [logRetentionDays, setLogRetentionDays] = useState(initialConfig?.logRetentionDays || 30);
    const [logsEnabled, setLogsEnabled] = useState(initialConfig ? initialConfig.logsEnabled !== false : true);

    const toast = useToast();

    // ── Load / revalidate config on mount ──
    useEffect(() => {
        let isMounted = true;
        const loadConfig = async () => {
            try {
                const res = await fetchServerConfig();
                if (res.success && isMounted) {
                    const serverData = res.data || res.config;
                    if (serverData) {
                        setConfig(serverData);
                        setGlobalDefault(serverData.globalDefaultCaption || '');
                        setGlobalNewPack(serverData.globalNewPackCaption || '');
                        setFixedPostEnabled(Boolean(serverData.fixedPostBuilderEnabled));
                        setFixedPostKey(serverData.fixedPostBuilderKey || 'legendasbot');
                        setFixedPostPayload(formatJsonForEditor(serverData.fixedPostBuilderPayload || ''));
                        setLogRetentionDays(serverData.logRetentionDays || 30);
                        setLogsEnabled(serverData.logsEnabled !== false);
                    }
                }
            } catch {
                if (!initialConfig && isMounted) {
                    toast('Erro ao carregar configurações', 'error');
                }
            } finally {
                if (isMounted) setLoading(false);
            }
        };
        loadConfig();
        return () => { isMounted = false; };
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
                    logsEnabled: logsEnabled,
                });
            } catch {
                // silent — keep old cache alive
            }
        };

        const interval = setInterval(refresh, REFRESH_INTERVAL);
        return () => clearInterval(interval);
    }, [loading, !!config, fixedPostEnabled, logRetentionDays, logsEnabled]);

    const handleSave = async (
        overrides: Partial<ServerConfig> = {},
        section: 'system' | 'captions' | 'postbuilder' = 'system'
    ) => {
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
            logsEnabled: overrides.logsEnabled ?? logsEnabled,
        };

        setSaving(true);
        setSavingSection(section);
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
                    setFixedPostPayload(formatJsonForEditor(serverData.fixedPostBuilderPayload || ''));
                    setLogRetentionDays(serverData.logRetentionDays || 30);
                    setLogsEnabled(serverData.logsEnabled !== false);
                }
                toast('Configurações atualizadas com sucesso', 'success');
            }
        } catch (err) {
            toast('Erro ao atualizar configurações', 'error');
        } finally {
            setSaving(false);
            setSavingSection(null);
        }
    };

    const handleToggle = (field: 'maintence' | 'forceJoin' | 'fixedPostBuilderEnabled' | 'logsEnabled') => {
        if (!config) return;
        if (field === 'fixedPostBuilderEnabled') {
            const next = !fixedPostEnabled;
            setFixedPostEnabled(next);
            handleSave({ fixedPostBuilderEnabled: next }, 'postbuilder');
            return;
        }
        if (field === 'logsEnabled') {
            const next = !logsEnabled;
            setLogsEnabled(next);
            handleSave({ logsEnabled: next });
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
        }, 'postbuilder');
        toast('Cache do PostBuilder renovado', 'success');
    };

    if (loading) return (
        <Card className="rounded-2xl border border-white/10 bg-white/5">
            <CardContent className="flex flex-col items-center py-12 gap-3">
                <div className="auth-spinner" />
                <p className="text-sm text-muted-foreground">Carregando configurações...</p>
            </CardContent>
        </Card>
    );

    return (
        <div className="admin-config-page grid gap-5">
            {/* ── Sistema ── */}
            <Card className="admin-config-card admin-config-system rounded-2xl border border-white/10 bg-white/5">
                <CardHeader>
                    <div className="admin-config-heading">
                        <span className="admin-config-heading-icon"><Settings size={17} /></span>
                        <div>
                            <CardTitle>Sistema</CardTitle>
                            <CardDescription>Estado do bot e comportamento geral</CardDescription>
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-3">
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-xl border border-white/10 bg-white/5 p-3.5">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Manutenção</p>
                            <p className="text-xs text-muted-foreground">
                                Operação do bot em manutenção
                            </p>
                        </div>
                        <Switch
                            checked={!!config?.maintence}
                            onCheckedChange={() => !saving && handleToggle('maintence')}
                            disabled={saving}
                        />
                    </div>
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-xl border border-white/10 bg-white/5 p-3.5">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Force Join</p>
                            <p className="text-xs text-muted-foreground">
                                Inscrição obrigatória
                            </p>
                        </div>
                        <Switch
                            checked={!!config?.forceJoin}
                            onCheckedChange={() => !saving && handleToggle('forceJoin')}
                            disabled={saving}
                        />
                    </div>
                    <div className="admin-config-row flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-xl border border-white/10 bg-white/5 p-3.5">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Retenção de Logs (dias)</p>
                            <p className="text-xs text-muted-foreground">Limpeza automática de logs antigos</p>
                        </div>
                        <div className="admin-config-number-control">
                            <Clock3 size={14} />
                            <Input
                                type="number"
                                min={1}
                                max={365}
                                value={logRetentionDays}
                                onChange={(e) => setLogRetentionDays(Math.max(1, parseInt(e.target.value) || 1))}
                                disabled={saving}
                                className="w-20 h-9 text-xs font-semibold text-center rounded-xl bg-card border-border"
                            />
                            <span className="text-xs text-muted-foreground font-medium">dias</span>
                        </div>
                    </div>
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-xl border border-white/10 bg-white/5 p-3.5">
                        <div className="min-w-0 flex-1">
                            <p className="text-sm font-medium">Registro de Logs no Banco</p>
                            <p className="text-xs text-muted-foreground">Desativado economiza operações no banco de dados</p>
                        </div>
                        <Switch
                            checked={logsEnabled}
                            onCheckedChange={() => !saving && handleToggle('logsEnabled')}
                            disabled={saving}
                        />
                    </div>
                </CardContent>
                <CardFooter className="admin-config-footer justify-end gap-2">
                    <Button
                        variant="default"
                        onClick={() => !saving && handleSave()}
                        disabled={saving}
                        className="admin-config-save-button bg-accent hover:bg-accent/90 text-accent-foreground font-bold rounded-xl"
                    >
                        <Save size={15} className="mr-1.5" />
                        {savingSection === 'system' ? 'Salvando...' : 'Salvar'}
                    </Button>
                </CardFooter>
            </Card>

            {/* ── Legendas ── */}
            <Card className="admin-config-card admin-config-captions rounded-2xl border border-white/10 bg-white/5">
                <CardHeader>
                    <div className="admin-config-heading">
                        <span className="admin-config-heading-icon"><FileText size={17} /></span>
                        <div>
                            <CardTitle>Legendas</CardTitle>
                            <CardDescription>Conteúdo padrão aplicado a canais e packs</CardDescription>
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="admin-config-editor">
                        <p className="text-sm font-medium mb-1">Legenda Padrão Global</p>
                        <p className="text-xs text-muted-foreground mb-2">Preenche novos canais vinculados ao bot</p>
                        <RichTextEditor
                            value={globalDefault}
                            onChange={setGlobalDefault}
                            placeholder="Ex: @legendasbot [t.me/legendasbot](https://t.me/botusername)"
                            rows={4}
                        />
                    </div>
                    <div className="admin-config-editor">
                        <p className="text-sm font-medium mb-1">Legenda de Novo Pack</p>
                        <p className="text-xs text-muted-foreground mb-2">Valor inicial para mensagem de pack padrão</p>
                        <RichTextEditor
                            value={globalNewPack}
                            onChange={setGlobalNewPack}
                            placeholder="Texto inicial para novos packs..."
                            rows={7}
                        />
                    </div>
                </CardContent>
                <CardFooter className="admin-config-footer justify-end gap-2">
                    <Button
                        variant="default"
                        onClick={() => !saving && handleSave({}, 'captions')}
                        disabled={saving}
                        className="admin-config-save-button bg-accent hover:bg-accent/90 text-accent-foreground font-bold rounded-xl"
                    >
                        <Save size={15} className="mr-1.5" />
                        {savingSection === 'captions' ? 'Salvando...' : 'Salvar'}
                    </Button>
                </CardFooter>
            </Card>

            {/* ── PostBuilder ── */}
            <Card className="admin-config-card admin-config-postbuilder rounded-2xl border border-white/10 bg-white/5">
                <CardHeader>
                    <div className="admin-config-heading">
                        <span className="admin-config-heading-icon"><Wrench size={17} /></span>
                        <div>
                            <div className="admin-config-title-line">
                                <CardTitle>PostBuilder</CardTitle>
                                <span className="admin-config-beta">Beta</span>
                            </div>
                            <CardDescription>Post permanente usado em consultas inline</CardDescription>
                        </div>
                    </div>
                    <CardAction className="admin-config-card-action">
                        <Button variant="ghost" size="sm" onClick={refreshPostBuilderCache} disabled={saving || !config} title="Renovar cache do Redis">
                            <RefreshCw size={14} className={saving ? 'animate-spin' : ''} />
                        </Button>
                    </CardAction>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="admin-config-row flex items-center justify-between gap-4 rounded-xl border border-white/10 bg-white/5 p-3.5">
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
                        <div className="admin-config-postbuilder-fields space-y-3">
                            <div className="admin-config-field">
                                <label className="text-sm font-medium">Chave fixa</label>
                                <Input
                                    value={fixedPostKey}
                                    onChange={(e) => setFixedPostKey(e.target.value)}
                                    placeholder="legendasbot"
                                    disabled={saving}
                                    className="mt-1 rounded-xl bg-card border-border"
                                />
                            </div>
                            <div className="admin-config-field admin-config-json-field">
                                <div className="admin-config-field-label">
                                    <label className="text-sm font-medium">Payload JSON</label>
                                    <Database size={14} />
                                </div>
                                <JsonPayloadEditor
                                    value={fixedPostPayload}
                                    onChange={setFixedPostPayload}
                                    disabled={saving}
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
                        onClick={() => !saving && handleSave({}, 'postbuilder')}
                        disabled={saving}
                        className="admin-config-save-button bg-accent hover:bg-accent/90 text-accent-foreground font-bold rounded-xl"
                    >
                        <Save size={15} className="mr-1.5" />
                        {savingSection === 'postbuilder' ? 'Salvando...' : 'Salvar'}
                    </Button>
                </CardFooter>
            </Card>
        </div>
    );
}
