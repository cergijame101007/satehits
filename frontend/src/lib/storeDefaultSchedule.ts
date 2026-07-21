/**
 * 店舗定例スケジュールのプレビュー（暫定・フロント定数）
 *
 * NOTE:
 * - 現状は backend の schedule_resolver.synthesizeFromStoreCalendar と同じ曜日ルールを FE で再実装している
 * - 今後オーナーが店舗定例を DB 登録できる機能追加時は、API（Resolver 経由）取得に差し替える
 * - DELETE /admin/schedules/{date} 自体は override 行削除のため変更不要
 */

import type { ScheduleType } from '@/types/reservation';
import { scheduleTypeLabels } from '@/lib/calendarTheme';

const DEFAULT_CAPACITY = 10;

export interface StoreDefaultSchedulePreview {
  type: ScheduleType;
  capacity: number;
  typeLabel: string;
  capacityLabel: string;
  summaryLines: string[];
  isClosed: boolean;
}

/** YYYY-MM-DD から店舗定例のタイプ・提供数を算出する */
export function getStoreDefaultSchedule(dateStr: string): { type: ScheduleType; capacity: number } {
  const [y, m, d] = dateStr.split('-').map(Number);
  const weekday = new Date(y, m - 1, d).getDay(); // 0=日 … 6=土

  // 木(4)・金(5) = closed、日(0) = morning、それ以外 = normal
  if (weekday === 4 || weekday === 5) {
    return { type: 'closed', capacity: 0 };
  }
  if (weekday === 0) {
    return { type: 'morning', capacity: DEFAULT_CAPACITY };
  }
  return { type: 'normal', capacity: DEFAULT_CAPACITY };
}

/** 確認モーダル用の店舗定例プレビュー */
export function formatStoreDefaultPreview(dateStr: string): StoreDefaultSchedulePreview {
  const { type, capacity } = getStoreDefaultSchedule(dateStr);
  const typeLabel = scheduleTypeLabels[type];
  const isClosed = type === 'closed';
  const capacityLabel = isClosed ? '予約不可' : `${capacity}食`;

  const summaryLines = [`タイプ: ${typeLabel}`, `提供可能数: ${capacityLabel}`];
  if (isClosed) {
    summaryLines.push('この日は定休日として扱われます');
  }

  return {
    type,
    capacity,
    typeLabel,
    capacityLabel,
    summaryLines,
    isClosed,
  };
}
