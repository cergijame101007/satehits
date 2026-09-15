import type { ScheduleType } from '@/types/reservation';

/** スケジュールタイプの背景色 */
export const typeColors: Record<ScheduleType, string> = {
  normal: '#E0F2FE',
  morning: '#FEF3C7',
  event: '#EDE9FE',
  external_event: '#F3E8FF',
  special_menu: '#FCE7F3',
  closed: '#F3F4F6',
};

export const typeTextColors: Record<ScheduleType, string> = {
  normal: '#0369A1',
  morning: '#92400E',
  event: '#6D28D9',
  external_event: '#7C3AED',
  special_menu: '#BE185D',
  closed: '#6B7280',
};

/** スケジュールタイプの表示ラベル */
export const scheduleTypeLabels: Record<ScheduleType, string> = {
  normal: '通常',
  morning: '朝営業',
  event: 'イベント',
  external_event: '外部イベント（店休）',
  special_menu: '特別メニュー',
  closed: '臨時休',
};

/** 店舗定例の closed（木・金。is_default = true）の表示ラベル */
export const REGULAR_CLOSED_LABEL = '定休日';

/** 日ごとの表示ラベル。closed は店舗定例なら定休日、個別設定（daily_schedules 行あり）なら臨時休 */
export function getScheduleTypeLabel(type: ScheduleType, isDefault: boolean): string {
  return type === 'closed' && isDefault ? REGULAR_CLOSED_LABEL : scheduleTypeLabels[type];
}

/** 凡例ラベル。closed のバッジは定休日・臨時休で共通なので併記する */
export function getScheduleLegendLabel(type: ScheduleType): string {
  return type === 'closed'
    ? `${REGULAR_CLOSED_LABEL}・${scheduleTypeLabels.closed}`
    : scheduleTypeLabels[type];
}

/** スケジュールタイプの短縮表示 */
export const scheduleTypeShort: Record<ScheduleType, string> = {
  normal: '通',
  morning: '朝',
  event: 'イ',
  external_event: '外',
  special_menu: '特',
  closed: '休',
};

export const WEEKDAYS = ['日', '月', '火', '水', '木', '金', '土'] as const;

export function isClosedScheduleType(type: ScheduleType): boolean {
  return type === 'closed' || type === 'external_event';
}
