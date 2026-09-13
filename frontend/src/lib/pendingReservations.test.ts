import { describe, expect, it } from 'vitest';
import type { Reservation } from '@/types/reservation';
import {
  countActionablePending,
  formatBadgeCount,
  pendingBadgeLabel,
} from '@/lib/pendingReservations';

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

describe('pendingBadgeLabel', () => {
  it('includes the exact count', () => {
    expect(pendingBadgeLabel(12)).toBe('未対応の予約 12 件');
  });
});
