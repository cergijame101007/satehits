import type { ScheduleType } from '@/types/reservation';

/** スケジュールタイプの背景色 */
export const typeColors: Record<ScheduleType, string> = {
  normal: '#E0F2FE',
  morning: '#FEF3C7',
  event: '#EDE9FE',
  external_event: '#F3E8FF',
  special: '#FCE7F3',
  closed: '#F3F4F6',
  temporary_closed: '#FEE2E2',
};

export const typeTextColors: Record<ScheduleType, string> = {
  normal: '#0369A1',
  morning: '#92400E',
  event: '#6D28D9',
  external_event: '#7C3AED',
  special: '#BE185D',
  closed: '#6B7280',
  temporary_closed: '#DC2626',
};

export const WEEKDAYS = ['日', '月', '火', '水', '木', '金', '土'] as const;

export function isClosedScheduleType(type: ScheduleType): boolean {
  return type === 'closed' || type === 'temporary_closed' || type === 'external_event';
}
