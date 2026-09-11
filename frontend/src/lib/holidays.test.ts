import { describe, expect, it } from 'vitest';
import { isHoliday } from '@/lib/holidays';

// 同梱の src/data/holidays.json（内閣府 CSV 由来）に依存する
describe('isHoliday', () => {
  it('returns true for New Year holiday', () => {
    expect(isHoliday('2026-01-01')).toBe(true);
  });

  it('returns true for a substitute holiday', () => {
    // 2026-05-06 は振替休日
    expect(isHoliday('2026-05-06')).toBe(true);
  });

  it('returns false for an ordinary weekday', () => {
    expect(isHoliday('2026-05-18')).toBe(false);
  });

  it('returns false for a year outside the bundled data', () => {
    expect(isHoliday('2099-01-01')).toBe(false);
  });
});
