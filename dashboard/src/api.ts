import { DashboardData, Button, Permission, Channel, AdminDashboardData, AdminLogsFilters, AdminLogsResponse, AccountStatus, AuthStatus, CaptionTemplate, UserCaptionTemplate } from './types';

export interface AuthRequestBody {
    channelID: number;
    user: {
        id: number;
        first_name: string;
        last_name: string;
        username: string;
        photo_url: string;
        auth_date: string;
        hash: string;
    };
}

export class ApiError extends Error {
    status: number;
    constructor(message: string, status: number) {
        super(message);
        this.name = 'ApiError';
        this.status = status;
    }
}

const apiFetch = async (url: string, options: RequestInit = {}) => {
    const response = await fetch(url, {
        ...options,
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
            ...(options.headers || {}),
        },
    });

    if (!response.ok) {
        const errBody = await response.text().catch(() => '');
        throw new ApiError(errBody || `API Error (${response.status})`, response.status);
    }

    if (response.status !== 204) {
        try {
            return await response.json();
        } catch {
            return null;
        }
    }
    return null;
};

export const login = async (initData: string, userID: number) => {
    return apiFetch('/api/login', {
        method: 'POST',
        headers: {
            'x-telegram-init-data': initData,
        },
        body: JSON.stringify({ userID }),
    });
};

export const fetchDashboardData = async (channelId: string): Promise<DashboardData> => {
    const response = await apiFetch(`/api/channel/${channelId}`, {
        method: 'GET',
    });
    return response?.data;
};

export const fetchUserChannels = async (): Promise<Channel[]> => {
    const response = await apiFetch(`/api/me/channels`, {
        method: 'GET',
    });
    return response?.data || [];
};

export const fetchAdminDashboard = async (): Promise<AdminDashboardData> => {
    const response = await apiFetch(`/api/admin/overview`);
    const data = response?.data || {};
    return { 
        success: true, // Assuming success if apiFetch didn't throw
        users: data.users || [], 
        channels: data.channels || [] 
    };
};

export const updateDefaultCaption = async (channelId: number, caption: string) => {
    return apiFetch(`/api/channel/${channelId}/caption`, {
        method: 'PUT',
        body: JSON.stringify({ caption }),
    });
};

export const updateNewPackCaption = async (channelId: number, payload: {
    newPackCaption: string;
    newPackMessageButtons?: boolean;
    newPackStickerButtons?: boolean;
    newPackMessagePosition?: 'above' | 'below';
    newPackReplyToSticker?: boolean;
}) => {
    return apiFetch(`/api/channel/${channelId}/newpackcaption`, {
        method: 'PUT',
        body: JSON.stringify(payload),
    });
};

export const updateReactions = async (channelId: number, reactions: string) => {
    return apiFetch(`/api/channel/${channelId}/reactions`, {
        method: 'PUT',
        body: JSON.stringify({ reactions }),
    });
};

export const updateReactionsActive = async (channelId: number, active: boolean) => {
    return apiFetch(`/api/channel/${channelId}/reactions/active`, {
        method: 'PUT',
        body: JSON.stringify({ active }),
    });
};

export const updateReactionPosition = async (channelId: number, reactionPosition: number) => {
    return apiFetch(`/api/channel/${channelId}/reactions/position`, {
        method: 'PUT',
        body: JSON.stringify({ reactionPosition }),
    });
};

export const updateNativeReactions = async (channelId: number, nativeReactions: string) => {
    return apiFetch(`/api/channel/${channelId}/native-reactions`, {
        method: 'PUT',
        body: JSON.stringify({ nativeReactions }),
    });
};

export const updateNativeReactionMode = async (channelId: number, mode: string) => {
    return apiFetch(`/api/channel/${channelId}/native-reactions/mode`, {
        method: 'PUT',
        body: JSON.stringify({ mode }),
    });
};

export const updateNativeReactionsEnabled = async (channelId: number, enabled: boolean) => {
    return apiFetch(`/api/channel/${channelId}/native-reactions/enabled`, {
        method: 'PUT',
        body: JSON.stringify({ enabled }),
    });
};

export const updateDynamicLinks = async (channelId: number, settings: {
    dynamicLinks: boolean;
    dlBotButtons: boolean;
    dlBotCaptions: boolean;
    dlBotReactions: boolean;
}) => {
    return apiFetch(`/api/channel/${channelId}/dynamic-links`, {
        method: 'PUT',
        body: JSON.stringify(settings),
    });
};

export const updateMessagePermission = async (channelId: number, perms: Permission) => {
    const payload = {
        linkPreview: Boolean(perms.linkPreview),
        message: Boolean(perms.message),
        audio: Boolean(perms.audio),
        video: Boolean(perms.video),
        photo: Boolean(perms.photo),
        document: Boolean(perms.document),
        sticker: Boolean(perms.sticker),
        gif: Boolean(perms.gif),
        reactions: Boolean(perms.reactions),
    };
    return apiFetch(`/api/channel/${channelId}/caption/permissions`, {
        method: 'PUT',
        body: JSON.stringify(payload),
    });
};

export const updateButtonsPermission = async (channelId: number, perms: Permission) => {
    const payload = {
        message: Boolean(perms.message),
        audio: Boolean(perms.audio),
        video: Boolean(perms.video),
        photo: Boolean(perms.photo),
        document: Boolean(perms.document),
        sticker: Boolean(perms.sticker),
        gif: Boolean(perms.gif),
    };
    return apiFetch(`/api/channel/${channelId}/buttons/permissions`, {
        method: 'PUT',
        body: JSON.stringify(payload),
    });
};

export const createButton = async (channelId: number, button: Partial<Button>) => {
    return apiFetch(`/api/channel/${channelId}/buttons`, {
        method: 'POST',
        body: JSON.stringify({
            nameButton: button.nameButton,
            buttonUrl: button.buttonUrl || undefined,
        }),
    });
};

export const updateButton = async (channelId: number, buttonId: string, button: Partial<Button>) => {
    return apiFetch(`/api/channel/${channelId}/buttons/${buttonId}`, {
        method: 'PUT',
        body: JSON.stringify(button),
    });
};

export const deleteButton = async (channelId: number, buttonId: string) => {
    return apiFetch(`/api/channel/${channelId}/buttons/${buttonId}`, {
        method: 'DELETE',
    });
};

export const updateLayoutButtons = async (channelId: number, layout: any[][]) => {
    return apiFetch(`/api/channel/${channelId}/buttons/layout`, {
        method: 'PUT',
        body: JSON.stringify({ layout }),
    });
};

export const transferChannel = async (newOwnerId: number, channelId: number) => {
	return apiFetch(`/api/channel/transfer`, {
		method: 'POST',
		body: JSON.stringify({ newOwnerId, channelId }),
	});
};

export const fetchUserInfo = async (usernameOrId: string) => {
    return apiFetch(`/api/user/info/${usernameOrId}`, {
        method: 'GET',
    });
};

export interface NoticeButton {
    text: string;
    type: string;
    value: string;
}

export type NoticeTarget = 'channels' | 'users' | 'all' | 'single' | 'user_ids' | 'channel_ids';

export interface NoticeRequest {
    message: string;
    target: NoticeTarget;
    targetId?: number;
    targetIds?: number[];
    imageUrl: string;
    mediaType: 'photo' | 'video' | 'animation';
    buttons: NoticeButton[];
}

export const sendAdminNotice = async (initData: string, payload: NoticeRequest) => {
    return apiFetch(`/api/admin/notice`, {
        method: 'POST',
        headers: {
            'Authorization': `tma ${initData}`,
            'x-telegram-init-data': initData,
        },
        body: JSON.stringify(payload),
    });
};

let cachedServerConfigData: any = null;

export const getCachedServerConfig = () => cachedServerConfigData;

export const fetchServerConfig = async () => {
    const response = await apiFetch(`/api/admin/config`, {
        method: 'GET',
    });
    if (response?.success) {
        cachedServerConfigData = response.data || response.config;
    }
    return response;
};

export const updateServerConfig = async (payload: {
    maintence: boolean;
    forceJoin: boolean;
    globalDefaultCaption: string;
    globalNewPackCaption: string;
    fixedPostBuilderEnabled: boolean;
    fixedPostBuilderKey: string;
    fixedPostBuilderPayload: string;
    logRetentionDays?: number;
    logsEnabled?: boolean;
}) => {
    const response = await apiFetch(`/api/admin/config`, {
        method: 'PUT',
        body: JSON.stringify(payload),
    });
    if (response?.success) {
        cachedServerConfigData = response.data || response.config;
    }
    return response;
};

export const disconnectChannel = async (channelId: number) => {
    return apiFetch(`/api/channel/${channelId}`, {
        method: 'DELETE',
    });
};

export const updateUserAdmin = async (userId: number) => {
    return apiFetch(`/api/admin/users/${userId}/admin`, {
        method: 'POST',
    });
};

export const updateUserBlacklist = async (userId: number) => {
    return apiFetch(`/api/admin/users/${userId}/blacklist`, {
        method: 'POST',
    });
};

export const fetchAuditCheckBot = async () => {
    return apiFetch('/api/admin/audit/checkbot');
};

export const bulkDeleteChannels = async (userId: number, channelIds: number[]) => {
    return apiFetch('/api/admin/audit/bulk-delete', {
        method: 'POST',
        body: JSON.stringify({ userId, channelIds }),
    });
};


export const fetchAdminLogs = async (filters: AdminLogsFilters = {}): Promise<AdminLogsResponse> => {
    const params = new URLSearchParams();
    Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined && value !== null && String(value).trim() !== '') {
            params.set(key, String(value));
        }
    });
    const response = await apiFetch(`/api/admin/logs?${params.toString()}`, {
        method: 'GET',
    });
    return response?.data || { events: [], total: 0, limit: filters.limit || 50, offset: filters.offset || 0 };
};

export const deleteAllAdminLogs = async (): Promise<{ deleted: number; message: string }> => {
    const response = await apiFetch('/api/admin/logs', {
        method: 'DELETE',
    });
    return response?.data;
};

// ===== Connected Account API =====

let pendingAccountStatusPromise: Promise<AccountStatus> | null = null;
export const fetchAccountStatus = async (): Promise<AccountStatus> => {
    if (pendingAccountStatusPromise) return pendingAccountStatusPromise;
    pendingAccountStatusPromise = (async () => {
        const response = await apiFetch('/api/account', { method: 'GET' });
        return response?.data || { status: 'disconnected' };
    })().finally(() => {
        pendingAccountStatusPromise = null;
    });
    return pendingAccountStatusPromise;
};

export const fetchAuthStatus = async (): Promise<AuthStatus> => {
    const response = await apiFetch('/api/account/status', { method: 'GET' });
    return response?.data || { step: 'phone' };
};

export const connectAccount = async (phoneNumber: string): Promise<AuthStatus> => {
    const response = await apiFetch('/api/account/connect', {
        method: 'POST',
        body: JSON.stringify({ phoneNumber }),
    });
    return response?.data || { step: 'error', error: 'Erro ao conectar' };
};

export const verifyCode = async (code: string): Promise<AuthStatus> => {
    const response = await apiFetch('/api/account/verify', {
        method: 'POST',
        body: JSON.stringify({ code }),
    });
    return response?.data || { step: 'error', error: 'Erro ao verificar código' };
};

export const sendPassword = async (password: string): Promise<AuthStatus> => {
    const response = await apiFetch('/api/account/password', {
        method: 'POST',
        body: JSON.stringify({ password }),
    });
    return response?.data || { step: 'error', error: 'Erro ao verificar senha' };
};

export const disconnectAccount = async (): Promise<void> => {
    await apiFetch('/api/account', { method: 'DELETE' });
};

// ===== Admin MTProto Accounts API =====

export const fetchAdminAccounts = async () => {
    const response = await apiFetch('/api/admin/accounts', { method: 'GET' });
    return response?.data || [];
};

export const adminConnectAccount = async (label: string, phoneNumber: string) => {
    const response = await apiFetch('/api/admin/accounts/connect', {
        method: 'POST',
        body: JSON.stringify({ label, phoneNumber }),
    });
    return response?.data || { step: 'error', error: 'Erro ao conectar' };
};

export const adminVerifyCode = async (sessionId: string, code: string) => {
    const response = await apiFetch('/api/admin/accounts/verify', {
        method: 'POST',
        body: JSON.stringify({ sessionId, code }),
    });
    return response?.data || { step: 'error', error: 'Erro ao verificar código' };
};

export const adminSendPassword = async (sessionId: string, password: string) => {
    const response = await apiFetch('/api/admin/accounts/password', {
        method: 'POST',
        body: JSON.stringify({ sessionId, password }),
    });
    return response?.data || { step: 'error', error: 'Erro ao verificar senha' };
};



// User Post Templates
export async function listUserPostTemplates(): Promise<any[]> {
  return await apiFetch('/api/me/post-templates');
}

export async function deleteUserPostTemplate(id: string) {
  return await apiFetch(`/api/me/post-templates/${id}`, { method: 'DELETE' });
};

export const adminDeleteAccount = async (id: string) => {
    return apiFetch(`/api/admin/accounts/${id}`, { method: 'DELETE' });
};

/* ===== Subscription (Premium) ===== */

let pendingSubscriptionStatusPromise: Promise<any> | null = null;
export const fetchSubscriptionStatus = async (): Promise<any> => {
    if (pendingSubscriptionStatusPromise) return pendingSubscriptionStatusPromise;
    pendingSubscriptionStatusPromise = apiFetch('/api/subscription', { method: 'GET' }).finally(() => {
        pendingSubscriptionStatusPromise = null;
    });
    return pendingSubscriptionStatusPromise;
};

/** Cria uma invoice link para pagamento via WebApp.openInvoice().
 *  test: se true, usa 1 Star no ambiente com STARS_TEST_MODE=true.
 *  channels: numero de canais a incluir no premium (para calculo de preco) */
export const createSubscriptionInvoice = async (test?: boolean, channels?: number): Promise<any> => {
    const params = new URLSearchParams();
    if (test) params.set('test', 'true');
    if (channels && channels > 1) params.set('channels', String(channels));
    const query = params.toString();
    return apiFetch(`/api/subscription/create${query ? '?' + query : ''}`, { method: 'POST' });
};

export const cancelSubscription = async (): Promise<any> => {
    return apiFetch('/api/subscription/cancel', { method: 'POST' });
};

export const createExtraChannelInvoice = async (test?: boolean): Promise<any> => {
    const params = new URLSearchParams();
    if (test) params.set('test', 'true');
    return apiFetch(`/api/subscription/channels/add-invoice?${params.toString()}`, { method: 'POST' });
};

export const removeExtraChannel = async (): Promise<any> => {
    return apiFetch('/api/subscription/channels/remove', { method: 'POST' });
};

/* ===== Channel Separator (Premium) ===== */

export const getChannelSeparator = async (channelId: number): Promise<any> => {
    return apiFetch(`/api/channel/${channelId}/separator`, { method: 'GET' });
};

export const saveChannelSeparator = async (channelId: number, data: { type?: string; emojiText: string; emojiId: string; emojiEntitiesJSON: string }): Promise<any> => {
    return apiFetch(`/api/channel/${channelId}/separator`, {
        method: 'PUT',
        body: JSON.stringify(data),
    });
};

export const deleteChannelSeparator = async (channelId: number): Promise<any> => {
    return apiFetch(`/api/channel/${channelId}/separator`, { method: 'DELETE' });
};

/* ===== Admin Premium Features ===== */

export const fetchPremiumFeatures = async (): Promise<any> => {
    return apiFetch('/api/admin/premium/features', { method: 'GET' });
};

export const togglePremiumFeature = async (key: string, enabled: boolean): Promise<any> => {
    return apiFetch(`/api/admin/premium/features/${key}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ enabled }),
    });
};

export const updatePremiumFeature = async (key: string, data: Partial<{ name: string; description: string; enabled: boolean; price: number }>): Promise<any> => {
    return apiFetch(`/api/admin/premium/features/${key}`, {
        method: 'PUT',
        body: JSON.stringify(data),
    });
};

/* ===== Admin Subscriptions ===== */

export const fetchAdminSubscriptions = async (): Promise<any> => {
    return apiFetch('/api/admin/subscriptions', { method: 'GET' });
};

export const adminCancelSubscriptions = async (userIds: number[], instant: boolean): Promise<any> => {
    return apiFetch('/api/admin/subscriptions/cancel', {
        method: 'POST',
        body: JSON.stringify({ userIds, instant }),
    });
};

export const adminRefundPayment = async (userId: number, telegramPaymentChargeId: string): Promise<any> => {
    return apiFetch('/api/admin/subscriptions/refund', {
        method: 'POST',
        body: JSON.stringify({ userId, telegramPaymentChargeId }),
    });
};

export const adminToggleAccount = async (id: string, enabled: boolean) => {
    const response = await apiFetch(`/api/admin/accounts/${id}/toggle`, {
        method: 'POST',
        body: JSON.stringify({ enabled }),
    });
    return response?.data;
};

/* ===== Emoji History ===== */

let cachedEmojiHistory: string[] | null = null;
let emojiHistoryPromise: Promise<string[]> | null = null;

export const getCachedEmojiHistory = (): string[] => cachedEmojiHistory || [];

export const fetchEmojiHistory = async (): Promise<string[]> => {
    if (cachedEmojiHistory) return cachedEmojiHistory;
    if (emojiHistoryPromise) return emojiHistoryPromise;

    emojiHistoryPromise = (async () => {
        try {
            const response = await apiFetch('/api/emoji/history', { method: 'GET' });
            cachedEmojiHistory = response?.ids || [];
            return cachedEmojiHistory;
        } catch {
            return [];
        } finally {
            emojiHistoryPromise = null;
        }
    })();

    return emojiHistoryPromise;
};

/* ===== Custom Captions API ===== */

export const createCustomCaption = async (channelId: number, data: { code: string; caption: string; linkPreview?: boolean }): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/custom-captions`, {
        method: 'POST',
        body: JSON.stringify(data),
    });
    return response?.data;
};

export const updateCustomCaption = async (channelId: number, captionId: string, data: { code: string; caption: string; linkPreview?: boolean }): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/custom-captions/${captionId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
    });
    return response?.data;
};

export const deleteCustomCaption = async (channelId: number, captionId: string): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/custom-captions/${captionId}`, {
        method: 'DELETE',
    });
    return response?.data;
};

export const createCustomCaptionButton = async (channelId: number, captionId: string, data: { nameButton: string; buttonUrl?: string }): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/custom-captions/${captionId}/buttons`, {
        method: 'POST',
        body: JSON.stringify(data),
    });
    return response?.data;
};

export const updateCustomCaptionButton = async (channelId: number, captionId: string, buttonId: string, data: { nameButton: string; buttonUrl?: string }): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/custom-captions/${captionId}/buttons/${buttonId}`, {
        method: 'PUT',
        body: JSON.stringify(data),
    });
    return response?.data;
};

export const deleteCustomCaptionButton = async (channelId: number, captionId: string, buttonId: string): Promise<any> => {
    return apiFetch(`/api/channel/${channelId}/custom-captions/${captionId}/buttons/${buttonId}`, {
        method: 'DELETE',
    });
};

export const updateCustomCaptionLayout = async (channelId: number, captionId: string, layout: { buttonId: string }[][]): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/custom-captions/${captionId}/layout`, {
        method: 'PUT',
        body: JSON.stringify({ layout }),
    });
    return response?.data;
};

/* ===== Caption Template API ===== */

export const listCaptionTemplates = async (channelId: number): Promise<CaptionTemplate[]> => {
    const response = await apiFetch(`/api/channel/${channelId}/caption-templates`, { method: 'GET' });
    return response?.data || [];
};

export const saveCaptionTemplate = async (channelId: number, name: string): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/caption-templates`, {
        method: 'POST',
        body: JSON.stringify({ name }),
    });
    return response?.data;
};

export const getCaptionTemplate = async (channelId: number, templateId: string): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/caption-templates/${templateId}`, { method: 'GET' });
    return response?.data;
};

export const applyCaptionTemplate = async (channelId: number, templateId: string): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/caption-templates/${templateId}/apply`, { method: 'POST' });
    return response?.data;
};

export const deleteCaptionTemplate = async (channelId: number, templateId: string): Promise<any> => {
    const response = await apiFetch(`/api/channel/${channelId}/caption-templates/${templateId}`, { method: 'DELETE' });
    return response?.data;
};

/* ===== Scheduler API ===== */

export const fetchMySchedules = async (): Promise<any[]> => {
    const response = await apiFetch('/api/schedule', { method: 'GET' });
    return response?.data || [];
};

export const createSchedule = async (data: {
    channelId: number;
    scheduleType: string;
    scheduleTime?: string;
    scheduledAt?: string;
    scheduleDays?: number[];
    repeatUntil?: string;
    loopQueue?: boolean;
}): Promise<any> => {
    const response = await apiFetch('/api/schedule', {
        method: 'POST',
        body: JSON.stringify(data),
    });
    return response?.data;
};

export const getScheduleById = async (id: string): Promise<any> => {
    const response = await apiFetch(`/api/schedule/${id}`, { method: 'GET' });
    return response?.data;
};

export const updateScheduleStatus = async (id: string, status: string): Promise<any> => {
    const response = await apiFetch(`/api/schedule/${id}/status`, {
        method: 'PUT',
        body: JSON.stringify({ status }),
    });
    return response?.data;
};

export const deleteSchedule = async (id: string): Promise<any> => {
    return apiFetch(`/api/schedule/${id}`, { method: 'DELETE' });
};

export const updateScheduleTime = async (
    id: string,
    data: {
        nextRunAt?: string;
        scheduleTime?: string;
        pinMessage?: boolean;
        intervalMin?: number;
        windowStart?: string;
        windowEnd?: string;
        autoDeleteMin?: number;
    }
): Promise<any> => {
    const response = await apiFetch(`/api/schedule/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
    });
    return response?.data;
};

/* ===== User Caption Template API ===== */

export const listUserCaptionTemplates = async (): Promise<UserCaptionTemplate[]> => {
    const response = await apiFetch('/api/me/templates', { method: 'GET' });
    return response?.data || [];
};

export const createUserCaptionTemplate = async (code: string, caption: string): Promise<any> => {
    const response = await apiFetch('/api/me/templates', {
        method: 'POST',
        body: JSON.stringify({ code, caption }),
    });
    return response?.data;
};

export const getUserCaptionTemplate = async (id: string): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${id}`, { method: 'GET' });
    return response?.data;
};

export const updateUserCaptionTemplate = async (id: string, code: string, caption: string): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${id}`, {
        method: 'PUT',
        body: JSON.stringify({ code, caption }),
    });
    return response?.data;
};

export const deleteUserCaptionTemplate = async (id: string): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${id}`, { method: 'DELETE' });
    return response?.data;
};

export const createUserCaptionTemplateButton = async (templateId: string, nameButton: string, buttonUrl: string, style?: string): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${templateId}/buttons`, {
        method: 'POST',
        body: JSON.stringify({ nameButton, buttonUrl, style }),
    });
    return response?.data;
};

export const updateUserCaptionTemplateButton = async (templateId: string, buttonId: string, nameButton: string, buttonUrl: string, style?: string): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${templateId}/buttons/${buttonId}`, {
        method: 'PUT',
        body: JSON.stringify({ nameButton, buttonUrl, style }),
    });
    return response?.data;
};

export const deleteUserCaptionTemplateButton = async (templateId: string, buttonId: string): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${templateId}/buttons/${buttonId}`, { method: 'DELETE' });
    return response?.data;
};

export const updateUserCaptionTemplateLayout = async (templateId: string, layout: { buttonId: string }[][]): Promise<any> => {
    const response = await apiFetch(`/api/me/templates/${templateId}/layout`, {
        method: 'PUT',
        body: JSON.stringify({ layout }),
    });
    return response?.data;
};

/* ===== Post Templates (Rascunhos) ===== */
export const getPostTemplates = async (): Promise<UserPostTemplate[]> => {
    const response = await apiFetch('/api/me/post-templates', { method: 'GET' });
    return response?.data || [];
};

export const savePostTemplate = async (name: string): Promise<UserPostTemplate> => {
    const response = await apiFetch('/api/me/post-templates', {
        method: 'POST',
        body: JSON.stringify({ name }),
    });
    return response?.data;
};

export const deletePostTemplate = async (id: string): Promise<void> => {
    await apiFetch(`/api/me/post-templates/${id}`, { method: 'DELETE' });
};

export const loadPostTemplate = async (id: string): Promise<any> => {
    const response = await apiFetch(`/api/me/post-templates/${id}/load`, { method: 'POST' });
    return response?.data;
};
