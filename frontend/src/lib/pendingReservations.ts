import { listReservations } from '@/lib/adminReservation';
import { formatDate } from '@/lib/calendarUtils';
import type { Reservation } from '@/types/reservation';

/** バッジに出す上限。これ以上は「9+」にまとめて桁あふれでナビが崩れないようにする */
const BADGE_MAX = 9;

/** バッジ・注意帯から飛ばす先。全期間の申請中一覧を開く */
export const PENDING_LIST_HREF = '/admin/reservations?range=all&status=pending';

/** 未対応件数が変わったことを知らせる window イベント名 */
export const PENDING_COUNT_CHANGED_EVENT = 'pending-count-changed';

/** 赤い丸バッジの共通クラス（文字サイズは使う側で足す） */
export const pendingBadgeClassName =
  'ml-1 align-middle inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-full bg-red-600 text-white font-medium leading-none tabular-nums';

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

/** スクリーンリーダー向けの件数（バッジの数字は aria-hidden にしてこちらを読ませる） */
export function pendingBadgeLabel(n: number): string {
  return `未対応の予約 ${n} 件`;
}

let inFlight: Promise<number> | null = null;

/**
 * GET /admin/reservations?status=pending から今日以降の未対応件数を求める。
 * レイアウトのナビと Island が同時に呼んでも 1 リクエストにまとめる（in-flight Promise を共有）。
 */
export function fetchActionablePendingCount(): Promise<number> {
  if (inFlight) {
    return inFlight;
  }

  const request: Promise<number> = listReservations(undefined, 'pending')
    .then((reservations) => countActionablePending(reservations, formatDate(new Date())))
    .finally(() => {
      // 通知で捨てられた古い取得が、後から始まった取得の共有を消さないようにする
      if (inFlight === request) {
        inFlight = null;
      }
    });
  inFlight = request;
  return request;
}

/**
 * ステータス更新などで未対応件数が変わったことを通知する。
 * 変更前に始まった取得を使い回さないよう、共有中の in-flight は捨てる。
 */
export function notifyPendingCountChanged(): void {
  inFlight = null;
  window.dispatchEvent(new CustomEvent(PENDING_COUNT_CHANGED_EVENT));
}

/** 件数変更の通知を購読する。戻り値で購読解除 */
export function subscribePendingCountChanged(handler: () => void): () => void {
  window.addEventListener(PENDING_COUNT_CHANGED_EVENT, handler);
  return () => window.removeEventListener(PENDING_COUNT_CHANGED_EVENT, handler);
}

/**
 * 件数を取り直すべきタイミングを購読する: 件数変更の通知と、bfcache からの復帰（pageshow の persisted）。
 * 戻り値で両方の購読を解除する。
 */
export function subscribePendingCountRefresh(handler: () => void): () => void {
  const onPageShow = (event: PageTransitionEvent) => {
    if (event.persisted) handler();
  };
  const unsubscribeChanged = subscribePendingCountChanged(handler);
  window.addEventListener('pageshow', onPageShow);
  return () => {
    unsubscribeChanged();
    window.removeEventListener('pageshow', onPageShow);
  };
}
