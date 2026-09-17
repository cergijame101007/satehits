import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { buildReservationSummaryByDate } from '@/lib/useMonthCalendar';
import type { Reservation } from '@/types/reservation';

function reservation(overrides: Partial<Reservation>): Reservation {
  return {
    id: 'r-1',
    name: '山田太郎',
    people: 2,
    visit_date: '2026-09-15',
    visit_time: '12:00',
    phone: '090-1111-2222',
    email: 'yamada@example.com',
    note: '',
    status: 'pending',
    source: 'web',
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z',
    ...overrides,
  };
}

describe('buildReservationSummaryByDate', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-15T10:00:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('当月の pending / approved だけを日別に集計する', () => {
    const summary = buildReservationSummaryByDate(
      [
        reservation({ id: 'a', visit_date: '2026-09-15', status: 'pending' }),
        reservation({ id: 'b', visit_date: '2026-09-15', status: 'approved', people: 3 }),
        reservation({ id: 'c', visit_date: '2026-09-15', status: 'rejected' }),
        reservation({ id: 'd', visit_date: '2026-10-01', status: 'pending' }),
      ],
      2026,
      9,
    );

    expect(summary.get('2026-09-15')).toEqual({ count: 2, reservedMeals: 3, pendingCount: 1 });
    expect(summary.has('2026-10-01')).toBe(false);
  });

  it('pendingCount は pending だけを数え、approved や他のステータスは数えない', () => {
    const summary = buildReservationSummaryByDate(
      [
        reservation({ id: 'a', visit_date: '2026-09-20', status: 'pending' }),
        reservation({ id: 'b', visit_date: '2026-09-20', status: 'pending' }),
        reservation({ id: 'c', visit_date: '2026-09-20', status: 'approved', people: 4 }),
        reservation({ id: 'd', visit_date: '2026-09-20', status: 'cancelled' }),
      ],
      2026,
      9,
    );

    expect(summary.get('2026-09-20')).toEqual({ count: 3, reservedMeals: 4, pendingCount: 2 });
  });

  it('今日より前の日は pending があっても pendingCount が 0 になる', () => {
    const summary = buildReservationSummaryByDate(
      [
        reservation({ id: 'yesterday', visit_date: '2026-09-14', status: 'pending' }),
        reservation({ id: 'today', visit_date: '2026-09-15', status: 'pending' }),
      ],
      2026,
      9,
    );

    expect(summary.get('2026-09-14')?.pendingCount).toBe(0);
    expect(summary.get('2026-09-14')?.count).toBe(1);
    expect(summary.get('2026-09-15')?.pendingCount).toBe(1);
  });

  it('予約がない日はエントリを作らない', () => {
    const summary = buildReservationSummaryByDate([], 2026, 9);

    expect(summary.size).toBe(0);
  });
});
