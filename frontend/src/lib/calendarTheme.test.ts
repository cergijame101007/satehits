import { describe, expect, it } from 'vitest';
import {
  getScheduleLegendLabel,
  getScheduleTypeLabel,
  isClosedScheduleType,
  scheduleTypeLabels,
} from '@/lib/calendarTheme';

describe('calendarTheme', () => {
  it('closed の種別ラベルは臨時休', () => {
    expect(scheduleTypeLabels.closed).toBe('臨時休');
  });

  it('店舗定例の closed は定休日、個別設定の closed は臨時休と表示する', () => {
    expect(getScheduleTypeLabel('closed', true)).toBe('定休日');
    expect(getScheduleTypeLabel('closed', false)).toBe('臨時休');
  });

  it('closed 以外は is_default にかかわらず種別ラベルを返す', () => {
    expect(getScheduleTypeLabel('normal', true)).toBe('通常');
    expect(getScheduleTypeLabel('special_menu', false)).toBe('特別メニュー');
  });

  it('凡例の closed は定休日と臨時休を併記する', () => {
    expect(getScheduleLegendLabel('closed')).toBe('定休日・臨時休');
    expect(getScheduleLegendLabel('morning')).toBe('朝営業');
  });

  it('closed と external_event を休業として扱う', () => {
    expect(isClosedScheduleType('closed')).toBe(true);
    expect(isClosedScheduleType('external_event')).toBe(true);
    expect(isClosedScheduleType('special_menu')).toBe(false);
  });
});
