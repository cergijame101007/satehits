import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { confirmIfCapacityExceeded } from '@/components/react/ReservationCreateForm';
import { getAvailability } from '@/lib/availability';

vi.mock('@/lib/availability', () => ({
  getAvailability: vi.fn(),
}));

const mockedGetAvailability = vi.mocked(getAvailability);

describe('confirmIfCapacityExceeded', () => {
  let confirmSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    confirmSpy = vi.spyOn(window, 'confirm');
  });

  afterEach(() => {
    vi.restoreAllMocks();
    mockedGetAvailability.mockReset();
  });

  it('pending / approved 以外は空き確認なしで登録を許可する', async () => {
    const ok = await confirmIfCapacityExceeded('2026-10-01', 2, 'rejected');

    expect(ok).toBe(true);
    expect(mockedGetAvailability).not.toHaveBeenCalled();
    expect(confirmSpy).not.toHaveBeenCalled();
  });

  it('提供可能数に収まる場合は確認なしで登録を許可する', async () => {
    mockedGetAvailability.mockResolvedValue({
      date: '2026-10-01',
      capacity: 10,
      reserved: 4,
      available: 6,
      is_holiday: false,
    });

    const ok = await confirmIfCapacityExceeded('2026-10-01', 2, 'approved');

    expect(ok).toBe(true);
    expect(confirmSpy).not.toHaveBeenCalled();
  });

  it('提供可能数を超える場合は確認ダイアログの結果に従う', async () => {
    mockedGetAvailability.mockResolvedValue({
      date: '2026-10-01',
      capacity: 10,
      reserved: 9,
      available: 1,
      is_holiday: false,
    });
    confirmSpy.mockReturnValue(false);

    const ok = await confirmIfCapacityExceeded('2026-10-01', 2, 'pending');

    expect(ok).toBe(false);
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(confirmSpy.mock.calls[0][0]).toContain('提供可能数（10食）を超えて登録されます');
  });

  it('空き取得に失敗した場合は確認ダイアログを出し、キャンセルなら登録を止める', async () => {
    mockedGetAvailability.mockRejectedValue(new Error('network error'));
    confirmSpy.mockReturnValue(false);

    const ok = await confirmIfCapacityExceeded('2026-10-01', 2, 'approved');

    expect(ok).toBe(false);
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(confirmSpy.mock.calls[0][0]).toContain('空き状況を確認できませんでした');
  });

  it('空き取得に失敗した場合でも OK なら登録を許可する', async () => {
    mockedGetAvailability.mockRejectedValue(new Error('network error'));
    confirmSpy.mockReturnValue(true);

    const ok = await confirmIfCapacityExceeded('2026-10-01', 2, 'approved');

    expect(ok).toBe(true);
    expect(confirmSpy).toHaveBeenCalledTimes(1);
  });
});
