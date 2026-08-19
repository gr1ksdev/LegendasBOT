import { Channel, User } from '../../types';
import { AdminCrmFilter, AdminCrmSort } from './AdminCrmContext';

export type CrmStage = 'new' | 'activation' | 'active' | 'attention';

export const CRM_STAGE_ORDER: CrmStage[] = ['new', 'activation', 'active', 'attention'];

export const CRM_STAGE_LABELS: Record<CrmStage, string> = {
  new: 'Novos',
  activation: 'Em ativação',
  active: 'Ativos',
  attention: 'Atenção',
};

const DAY_MS = 24 * 60 * 60 * 1000;

function getTimestamp(value?: string): number {
  if (!value) return 0;
  const timestamp = new Date(value).getTime();
  return Number.isFinite(timestamp) ? timestamp : 0;
}

export function getUserStage(user: User, now = new Date()): CrmStage {
  if (user.is_blacklisted) return 'attention';
  if ((user.channels?.length || 0) > 0) return 'active';

  const createdAt = getTimestamp(user.created_at);
  const isRecent = createdAt > 0 && now.getTime() - createdAt <= 7 * DAY_MS;
  return isRecent ? 'new' : 'activation';
}

export function filterAndSortUsers(
  users: User[],
  searchQuery: string,
  filterBy: AdminCrmFilter,
  sortBy: AdminCrmSort,
): User[] {
  const query = searchQuery.trim().toLocaleLowerCase('pt-BR');

  const filtered = users.filter((user) => {
    const channelCount = user.channels?.length || 0;
    const matchesQuery = !query || [user.first_name, user.username, String(user.id)]
      .some((value) => (value || '').toLocaleLowerCase('pt-BR').includes(query));

    if (!matchesQuery) return false;
    if (filterBy === 'admins') return user.is_admin === true || (user as any).is_admin === 'true';
    if (filterBy === 'blacklisted') return user.is_blacklisted === true || (user as any).is_blacklisted === 'true';
    if (filterBy === 'with-channels') return channelCount > 0;
    if (filterBy === 'without-channels') return channelCount === 0;
    return true;
  });

  return [...filtered].sort((a, b) => {
    if (sortBy === 'name') {
      return (a.first_name || '').localeCompare(b.first_name || '', 'pt-BR', { sensitivity: 'base' });
    }
    if (sortBy === 'channels') {
      return (b.channels?.length || 0) - (a.channels?.length || 0);
    }
    const tA = getTimestamp(a.created_at) || (typeof a.id === 'number' ? a.id : 0);
    const tB = getTimestamp(b.created_at) || (typeof b.id === 'number' ? b.id : 0);
    return tB - tA;
  });
}

export function filterAndSortChannels(
  channels: Channel[],
  searchQuery: string,
  sortBy: AdminCrmSort,
): Channel[] {
  const query = searchQuery.trim().toLocaleLowerCase('pt-BR');
  const filtered = channels.filter((channel) => !query || [channel.title, String(channel.id), String(channel.ownerId)]
    .some((value) => (value || '').toLocaleLowerCase('pt-BR').includes(query)));

  return [...filtered].sort((a, b) => {
    if (sortBy === 'recent') return getTimestamp(b.created_at) - getTimestamp(a.created_at);
    return (a.title || '').localeCompare(b.title || '', 'pt-BR', { sensitivity: 'base' });
  });
}

export function groupUsersByStage(users: User[], now = new Date()): Record<CrmStage, User[]> {
  const groups: Record<CrmStage, User[]> = {
    new: [],
    activation: [],
    active: [],
    attention: [],
  };

  users.forEach((user) => groups[getUserStage(user, now)].push(user));
  return groups;
}

export interface AdminOverviewMetrics {
  totalUsers: number;
  totalChannels: number;
  admins: number;
  blacklisted: number;
  withoutChannels: number;
  recentUsers: number;
  activatedUsers: number;
  activationRate: number;
}

export type OperationalAlertKind = 'blacklisted' | 'without-channels' | 'new-users';

export interface OperationalAlert {
  id: OperationalAlertKind;
  count: number;
  title: string;
  description: string;
  severity: 'attention' | 'info';
}

export function getOverviewMetrics(users: User[], channels: Channel[]): AdminOverviewMetrics {
  const activatedUsers = users.filter((user) => (user.channels?.length || 0) > 0 && !user.is_blacklisted).length;
  const withoutChannels = users.filter((user) => !user.is_blacklisted && (user.channels?.length || 0) === 0).length;
  const now = Date.now();
  const recentUsers = users.filter((user) => {
    const createdAt = getTimestamp(user.created_at);
    return createdAt > 0 && now - createdAt <= 7 * DAY_MS;
  }).length;
  return {
    totalUsers: users.length,
    totalChannels: channels.length,
    admins: users.filter((user) => user.is_admin).length,
    blacklisted: users.filter((user) => user.is_blacklisted).length,
    withoutChannels,
    recentUsers,
    activatedUsers,
    activationRate: users.length > 0 ? Math.round((activatedUsers / users.length) * 100) : 0,
  };
}

export function getOperationalAlerts(users: User[], channels: Channel[]): OperationalAlert[] {
  const metrics = getOverviewMetrics(users, channels);
  const alerts: OperationalAlert[] = [];

  if (metrics.blacklisted > 0) {
    alerts.push({
      id: 'blacklisted',
      count: metrics.blacklisted,
      title: 'Usuários bloqueados',
      description: 'Revise bloqueios ativos e confirme se ainda são necessários.',
      severity: 'attention',
    });
  }

  if (metrics.withoutChannels > 0) {
    alerts.push({
      id: 'without-channels',
      count: metrics.withoutChannels,
      title: 'Ativação pendente',
      description: 'Usuários ainda não possuem um canal vinculado.',
      severity: 'attention',
    });
  }

  if (metrics.recentUsers > 0) {
    alerts.push({
      id: 'new-users',
      count: metrics.recentUsers,
      title: 'Novos cadastros',
      description: 'Entraram na base nos últimos sete dias.',
      severity: 'info',
    });
  }

  return alerts;
}

export interface RecentUserPoint {
  key: string;
  label: string;
  count: number;
  previousCount: number;
}

export function getRecentUserSeries(users: User[], days = 5, now = new Date()): RecentUserPoint[] {
  const formatter = new Intl.DateTimeFormat('pt-BR', { weekday: 'short' });
  const points: RecentUserPoint[] = [];

  const getDayKey = (date: Date) => [
    date.getFullYear(),
    String(date.getMonth() + 1).padStart(2, '0'),
    String(date.getDate()).padStart(2, '0'),
  ].join('-');

  for (let offset = days - 1; offset >= 0; offset -= 1) {
    const date = new Date(now);
    date.setHours(0, 0, 0, 0);
    date.setDate(date.getDate() - offset);
    const key = getDayKey(date);
    points.push({
      key,
      label: formatter.format(date).replace('.', ''),
      count: 0,
      previousCount: 0,
    });
  }

  const currentByKey = new Map(points.map((point) => [point.key, point]));
  const previousByKey = new Map<string, RecentUserPoint>();

  points.forEach((point) => {
    const currentDate = new Date(`${point.key}T12:00:00`);
    currentDate.setDate(currentDate.getDate() - days);
    previousByKey.set(getDayKey(currentDate), point);
  });

  users.forEach((user) => {
    const timestamp = getTimestamp(user.created_at);
    if (!timestamp) return;
    const key = getDayKey(new Date(timestamp));
    const currentPoint = currentByKey.get(key);
    const previousPoint = previousByKey.get(key);
    if (currentPoint) currentPoint.count += 1;
    if (previousPoint) previousPoint.previousCount += 1;
  });

  return points;
}
