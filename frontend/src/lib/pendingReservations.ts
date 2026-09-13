import { listReservations } from '@/lib/adminReservation';
import { formatDate } from '@/lib/calendarUtils';
import type { Reservation } from '@/types/reservation';

/** バッジに出す上限。これ以上は「9+」にまとめて桁あふれでナビが崩れないようにする */
const BADGE_MAX = 9;

/**
 * 承認・拒否の対応が必要な pending 件数。
 * 過去日の pending は空き枠を圧迫しないため対象外（today は YYYY-MM-DD のローカル日付）。
 */
export function countActionablePending(reservations: Reservation[], today: string): number {
  return reservations.filter((r) => r.status === 'pending' && r.visit_date >= today).length;
}

/** バッジ表示用の文字列（10 件以上は「9+」） */
export function formatBadgeCount(n: number): string {
  return n > BADGE_MAX ? `${BADGE_MAX}+` : String(n);
}

/** バッジ用の aria-label */
export function pendingBadgeLabel(n: number): string {
  return `未対応の予約 ${n} 件`;
}

/** GET /admin/reservations?status=pending から今日以降の未対応件数を求める */
export async function fetchActionablePendingCount(): Promise<number> {
  const reservations = await listReservations(undefined, 'pending');
  return countActionablePending(reservations, formatDate(new Date()));
}
