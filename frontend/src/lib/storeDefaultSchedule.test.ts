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
});
