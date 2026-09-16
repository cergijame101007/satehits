import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Reservation } from '@/types/reservation';
import { listReservations, ReservationApiError } from '@/lib/adminReservation';
import {
  countActionablePending,
  fetchActionablePendingCount,
  formatBadgeCount,
  notifyPendingCountChanged,
  PENDING_COUNT_CHANGED_EVENT,
  PENDING_LIST_HREF,
  pendingBadgeLabel,
  subscribePendingCountChanged,
  subscribePendingCountRefresh,
} from '@/lib/pendingReservations';
import { createDeferred } from '@/test/deferred';

vi.mock('@/lib/adminReservation', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/adminReservation')>();
  return { ...actual, listReservations: vi.fn() };
});

const listReservationsMock = vi.mocked(listReservations);

const TODAY = '2026-09-14';

function reservation(partial: Partial<Reservation>): Reservation {
  return {
    id: 'r1',
    name: '山田',
    people: 2,
    visit_date: TODAY,
    visit_time: '12:00',
    phone: '',
    email: '',
    note: '',
    status: 'pending',
    source: 'web',
    created_at: '',
    updated_at: '',
    ...partial,
  };
}

describe('countActionablePending', () => {
  it('returns 0 for empty list', () => {
    expect(countActionablePending([], TODAY)).toBe(0);
  });

  it('counts pending on today and tomorrow, excludes yesterday', () => {
    const list = [
      reservation({ id: 'today', visit_date: '2026-09-14' }),
      reservation({ id: 'tomorrow', visit_date: '2026-09-15' }),
      reservation({ id: 'yesterday', visit_date: '2026-09-13' }),
    ];
    expect(countActionablePending(list, TODAY)).toBe(2);
  });

  it('ignores statuses other than pending', () => {
    const list = [
      reservation({ id: 'a', status: 'approved' }),
      reservation({ id: 'b', status: 'rejected' }),
      reservation({ id: 'c', status: 'cancelled' }),
      reservation({ id: 'd', status: 'no_show' }),
      reservation({ id: 'e', status: 'pending' }),
    ];
    expect(countActionablePending(list, TODAY)).toBe(1);
  });

  it('compares dates across month and year boundaries', () => {
    const list = [
      reservation({ id: 'next-month', visit_date: '2026-10-01' }),
      reservation({ id: 'next-year', visit_date: '2027-01-01' }),
      reservation({ id: 'last-year', visit_date: '2025-12-31' }),
    ];
    expect(countActionablePending(list, TODAY)).toBe(2);
  });
});

describe('formatBadgeCount', () => {
  it('formats 0 and 9 as-is', () => {
    expect(formatBadgeCount(0)).toBe('0');
    expect(formatBadgeCount(9)).toBe('9');
  });

  it('caps 10 or more as 9+', () => {
    expect(formatBadgeCount(10)).toBe('9+');
    expect(formatBadgeCount(123)).toBe('9+');
  });
});

describe('PENDING_LIST_HREF', () => {
  it('全期間の申請中一覧を開くクエリ付きのパスになっている', () => {
    expect(PENDING_LIST_HREF).toBe('/admin/reservations?range=all&status=pending');
  });
});

describe('pendingBadgeLabel', () => {
  it('includes the exact count', () => {
    expect(pendingBadgeLabel(12)).toBe('未対応の予約 12 件');
  });
});

describe('fetchActionablePendingCount', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-14T10:00:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('status=pending で取得し、今日以降の件数だけを返す', async () => {
    listReservationsMock.mockResolvedValue([
      reservation({ id: 'yesterday', visit_date: '2026-09-13' }),
      reservation({ id: 'today', visit_date: '2026-09-14' }),
      reservation({ id: 'next-month', visit_date: '2026-10-01' }),
    ]);

    await expect(fetchActionablePendingCount()).resolves.toBe(2);
    expect(listReservationsMock).toHaveBeenCalledTimes(1);
    expect(listReservationsMock).toHaveBeenCalledWith(undefined, 'pending');
  });

  it('同時に呼ぶと 1 リクエストにまとめ、完了後の呼び出しは取り直す', async () => {
    const deferred = createDeferred<Reservation[]>();
    listReservationsMock.mockReturnValueOnce(deferred.promise);

    const first = fetchActionablePendingCount();
    const second = fetchActionablePendingCount();
    expect(second).toBe(first);
    expect(listReservationsMock).toHaveBeenCalledTimes(1);

    deferred.resolve([reservation({ id: 'today' })]);
    await expect(first).resolves.toBe(1);
    await expect(second).resolves.toBe(1);

    listReservationsMock.mockResolvedValueOnce([]);
    await expect(fetchActionablePendingCount()).resolves.toBe(0);
    expect(listReservationsMock).toHaveBeenCalledTimes(2);
  });

  it('失敗はそのまま投げ、次の呼び出しは取り直す', async () => {
    const error = new ReservationApiError({ code: 'INTERNAL_ERROR', message: '予約一覧を取得できません' });
    listReservationsMock.mockRejectedValueOnce(error);

    await expect(fetchActionablePendingCount()).rejects.toBe(error);

    listReservationsMock.mockResolvedValueOnce([reservation({ id: 'today' })]);
    await expect(fetchActionablePendingCount()).resolves.toBe(1);
    expect(listReservationsMock).toHaveBeenCalledTimes(2);
  });

  it('変更通知の後は通知前に始まった取得を使い回さない', async () => {
    const stale = createDeferred<Reservation[]>();
    const fresh = createDeferred<Reservation[]>();
    listReservationsMock.mockReturnValueOnce(stale.promise).mockReturnValueOnce(fresh.promise);
    const before = fetchActionablePendingCount();

    notifyPendingCountChanged();

    const after = fetchActionablePendingCount();
    expect(after).not.toBe(before);
    expect(listReservationsMock).toHaveBeenCalledTimes(2);

    // 古い取得が先に終わっても、通知後の取得の共有は続く
    stale.resolve([reservation({ id: 'today' })]);
    await expect(before).resolves.toBe(1);
    expect(fetchActionablePendingCount()).toBe(after);
    expect(listReservationsMock).toHaveBeenCalledTimes(2);

    fresh.resolve([]);
    await expect(after).resolves.toBe(0);
  });
});

describe('件数変更の通知と購読', () => {
  it('notifyPendingCountChanged は pending-count-changed を window に投げる', () => {
    const listener = vi.fn();
    window.addEventListener(PENDING_COUNT_CHANGED_EVENT, listener);

    notifyPendingCountChanged();

    window.removeEventListener(PENDING_COUNT_CHANGED_EVENT, listener);
    expect(listener).toHaveBeenCalledTimes(1);
    expect(listener.mock.calls[0][0]).toBeInstanceOf(CustomEvent);
  });

  it('subscribePendingCountChanged は通知で呼ばれ、解除後は呼ばれない', () => {
    const handler = vi.fn();
    const unsubscribe = subscribePendingCountChanged(handler);

    notifyPendingCountChanged();
    expect(handler).toHaveBeenCalledTimes(1);

    unsubscribe();
    notifyPendingCountChanged();
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it('subscribePendingCountRefresh は通知と bfcache 復帰（persisted）で呼ばれる', () => {
    const handler = vi.fn();
    const unsubscribe = subscribePendingCountRefresh(handler);

    notifyPendingCountChanged();
    expect(handler).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: false }));
    expect(handler).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true }));
    expect(handler).toHaveBeenCalledTimes(2);

    unsubscribe();
    notifyPendingCountChanged();
    window.dispatchEvent(new PageTransitionEvent('pageshow', { persisted: true }));
    expect(handler).toHaveBeenCalledTimes(2);
  });
});
