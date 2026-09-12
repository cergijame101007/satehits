import { describe, expect, it } from 'vitest';
import { formatStoreDefaultPreview, getStoreDefaultSchedule } from '@/lib/storeDefaultSchedule';

describe('getStoreDefaultSchedule', () => {
  it('returns closed on Thursday', () => {
    // 2026-05-21 is Thursday
    expect(getStoreDefaultSchedule('2026-05-21')).toEqual({ type: 'closed', capacity: 0 });
  });

  it('returns closed on Friday', () => {
    // 2026-05-22 is Friday
    expect(getStoreDefaultSchedule('2026-05-22')).toEqual({ type: 'closed', capacity: 0 });
  });

  it('returns morning on Sunday', () => {
    // 2026-05-17 is Sunday
    expect(getStoreDefaultSchedule('2026-05-17')).toEqual({ type: 'morning', capacity: 10 });
  });

  it('returns normal on Monday', () => {
    // 2026-05-18 is Monday
    expect(getStoreDefaultSchedule('2026-05-18')).toEqual({ type: 'normal', capacity: 10 });
  });

  // 祝日ケースは同梱の src/data/holidays.json（内閣府 CSV 由来）に依存する
  it('returns normal on a Thursday holiday instead of closed', () => {
    // 2026-01-01 is Thursday (元日)
    expect(getStoreDefaultSchedule('2026-01-01')).toEqual({ type: 'normal', capacity: 10 });
  });

  it('returns normal on a Friday holiday instead of closed', () => {
    // 2026-03-20 is Friday (春分の日)
    expect(getStoreDefaultSchedule('2026-03-20')).toEqual({ type: 'normal', capacity: 10 });
  });

  it('returns normal on a Sunday holiday without morning hours', () => {
    // 2026-05-03 is Sunday (憲法記念日)
    expect(getStoreDefaultSchedule('2026-05-03')).toEqual({ type: 'normal', capacity: 10 });
  });

  it('falls back to weekday rule for a year outside the bundled data', () => {
    // 2099-01-01 is Thursday and not in holidays.json
    expect(getStoreDefaultSchedule('2099-01-01')).toEqual({ type: 'closed', capacity: 0 });
  });
});

describe('formatStoreDefaultPreview', () => {
  it('marks Thursday as closed with reservation unavailable', () => {
    const preview = formatStoreDefaultPreview('2026-05-21');
    expect(preview.isClosed).toBe(true);
    expect(preview.typeLabel).toBe('定休日');
    expect(preview.capacityLabel).toBe('予約不可');
  });

  it('shows capacity for normal weekday', () => {
    const preview = formatStoreDefaultPreview('2026-05-18');
    expect(preview.isClosed).toBe(false);
    expect(preview.typeLabel).toBe('通常');
    expect(preview.capacityLabel).toBe('10食');
  });

  it('does not mark a Thursday holiday as closed', () => {
    const preview = formatStoreDefaultPreview('2026-01-01');
    expect(preview.isClosed).toBe(false);
    expect(preview.typeLabel).toBe('通常');
    expect(preview.capacityLabel).toBe('10食');
    expect(preview.summaryLines).not.toContain('この日は定休日として扱われます');
  });
});
