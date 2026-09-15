import { afterEach, describe, expect, it, vi } from 'vitest';
import { authedFetch } from '@/lib/api';
import { listSchedules, setSchedule } from '@/lib/schedule';

vi.mock('@/lib/api', () => ({ authedFetch: vi.fn() }));

const authedFetchMock = vi.mocked(authedFetch);

function okResponse(body: unknown): Response {
  return { ok: true, json: async () => body } as Response;
}

function sentBody(): unknown {
  const init = authedFetchMock.mock.calls[0][1];
  return JSON.parse(String(init?.body));
}

describe('schedule lib', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('listSchedules は API の schedule_type とイベント欄をそのまま DailySchedule に渡す', async () => {
    authedFetchMock.mockResolvedValue(
      okResponse({
        year: 2026,
        month: 9,
        schedules: [
          { date: '2026-09-17', schedule_type: 'closed', capacity: 0, is_default: true },
          {
            date: '2026-09-19',
            schedule_type: 'special_menu',
            capacity: 10,
            event_name: '秋のラム',
            event_description: '数量限定',
            is_default: false,
          },
        ],
      }),
    );

    const result = await listSchedules(2026, 9);

    expect(authedFetchMock).toHaveBeenCalledWith('/api/v1/admin/schedules?year=2026&month=9');
    expect(result).toEqual([
      { date: '2026-09-17', schedule_type: 'closed', capacity: 0, is_default: true },
      {
        date: '2026-09-19',
        schedule_type: 'special_menu',
        capacity: 10,
        event_name: '秋のラム',
        event_description: '数量限定',
        is_default: false,
      },
    ]);
  });

  it('setSchedule は special_menu をそのまま送り、名称・説明は入力があるときだけ付ける', async () => {
    authedFetchMock.mockResolvedValue(
      okResponse({ date: '2026-09-19', schedule_type: 'special_menu', capacity: 8, event_name: '秋のラム', is_default: false }),
    );

    const saved = await setSchedule('2026-09-19', 'special_menu', 8, ' 秋のラム ', '');

    expect(authedFetchMock).toHaveBeenCalledWith(
      '/api/v1/admin/schedules/2026-09-19',
      expect.objectContaining({ method: 'PUT' }),
    );
    expect(sentBody()).toEqual({ schedule_type: 'special_menu', capacity: 8, event_name: '秋のラム' });
    expect(saved.schedule_type).toBe('special_menu');
  });

  it('setSchedule の closed（臨時休業）はイベント欄を送らない', async () => {
    authedFetchMock.mockResolvedValue(
      okResponse({ date: '2026-09-15', schedule_type: 'closed', capacity: 0, is_default: false }),
    );

    const saved = await setSchedule('2026-09-15', 'closed', 0, '名前', '説明');

    expect(sentBody()).toEqual({ schedule_type: 'closed', capacity: 0 });
    expect(saved).toEqual({ date: '2026-09-15', schedule_type: 'closed', capacity: 0, is_default: false });
  });
});
