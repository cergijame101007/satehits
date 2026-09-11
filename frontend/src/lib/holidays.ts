/**
 * 国民の祝日・休日の判定（内閣府 CSV 由来の同梱データ）
 *
 * - `src/data/holidays.json` は `make update-holidays` で backend の CSV と同一ソースから生成する（docs/holidays.md）
 * - backend の `internal/domain/holiday` と同じデータなので、店舗定例プレビューが API の合成結果とずれない
 * - データに無い年は「祝日なし」として扱う（曜日ルールにフォールバック）
 */

import holidayDates from '@/data/holidays.json';

const HOLIDAY_DATES: ReadonlySet<string> = new Set(holidayDates);

/** YYYY-MM-DD が国民の祝日・休日なら true */
export function isHoliday(dateStr: string): boolean {
  return HOLIDAY_DATES.has(dateStr);
}
