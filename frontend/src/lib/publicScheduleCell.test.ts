import { describe, expect, it } from 'vitest';
import type { PublicDaySchedule } from '@/types/reservation';
import {
  getPublicScheduleCellDisplay,
  shouldShowClosedMark,
  shouldShowEventBar,
  truncateEventName,
} from '@/lib/publicScheduleCell';

function schedule(partial: Partial<PublicDaySchedule> & Pick<PublicDaySchedule, 'date'>): PublicDaySchedule {
  return {
    capacity: 0,
    available: 0,
    is_holiday: false,
    schedule_type: null,
    ...partial,
  };
}

describe('publicScheduleCell', () => {
  it('shows event bar only for in-store event', () => {
    const s = schedule({
      date: '2026-02-09',
      schedule_type: 'event',
      event_name: '和紅茶をしばく会',
      capacity: 10,
      available: 8,
    });
    const display = getPublicScheduleCellDisplay(s);

    expect(shouldShowClosedMark(s)).toBe(false);
    expect(shouldShowEventBar(s)).toBe(true);
    expect(display.showClosedMark).toBe(false);
    expect(display.showEventBar).toBe(true);
    expect(display.eventBarLabel).toContain('和紅茶');
  });

  it('shows closed mark and event bar for external event', () => {
    const s = schedule({
      date: '2026-02-11',
      schedule_type: 'external_event',
      event_name: '和紅茶をしばく会',
      is_holiday: true,
    });
    const display = getPublicScheduleCellDisplay(s);

    expect(display.showClosedMark).toBe(true);
    expect(display.showEventBar).toBe(true);
    expect(display.eventBarTitle).toBe('和紅茶をしばく会');
  });

  it('shows closed mark only for regular holiday', () => {
    const s = schedule({
      date: '2026-02-13',
      is_holiday: true,
    });
    const display = getPublicScheduleCellDisplay(s);

    expect(display.showClosedMark).toBe(true);
    expect(display.showEventBar).toBe(false);
  });

  it('truncates long event names', () => {
    expect(truncateEventName('和紅茶をしばく会入門編', 4)).toBe('和紅茶を…');
  });
});
