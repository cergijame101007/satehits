import { describe, expect, it } from 'vitest';
import type { PublicDaySchedule } from '@/types/reservation';
import {
  getPublicScheduleCellDisplay,
  shouldShowClosedMark,
  shouldShowEventBar,
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
  });

  it('shows event bar for special menu only when it has a name', () => {
    const named = schedule({
      date: '2026-02-14',
      schedule_type: 'special_menu',
      event_name: '春のラム',
      capacity: 10,
      available: 10,
    });
    const unnamed = schedule({ date: '2026-02-15', schedule_type: 'special_menu', capacity: 10, available: 10 });

    expect(getPublicScheduleCellDisplay(named)).toEqual({ showClosedMark: false, showEventBar: true });
    expect(getPublicScheduleCellDisplay(unnamed)).toEqual({ showClosedMark: false, showEventBar: false });
  });

  it('shows closed mark for temporary closure (closed)', () => {
    const s = schedule({ date: '2026-02-16', schedule_type: 'closed' });

    expect(getPublicScheduleCellDisplay(s)).toEqual({ showClosedMark: true, showEventBar: false });
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
});
