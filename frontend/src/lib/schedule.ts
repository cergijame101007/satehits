import { authedFetch } from '@/lib/api';
import type { DailySchedule, ScheduleType } from '@/types/reservation';

/** フォームで選択可能なタイプ */
export const editableScheduleTypes: ScheduleType[] = [
  'normal',
  'morning',
  'event',
  'external_event',
  'special_menu',
  'closed',
];

interface ScheduleResponse {
  date: string;
  schedule_type: ScheduleType;
  capacity: number;
  event_name?: string;
  event_description?: string;
  is_default: boolean;
}

interface ScheduleListResponse {
  year: number;
  month: number;
  schedules: ScheduleResponse[];
}

interface SetScheduleRequest {
  schedule_type: ScheduleType;
  capacity: number;
  event_name?: string;
  event_description?: string;
}

interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}

export class ScheduleApiError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ErrorResponse['error']) {
    super(error.message);
    this.name = 'ScheduleApiError';
    this.code = error.code;
    this.details = error.details;
  }
}

async function parseError(res: Response): Promise<ScheduleApiError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new ScheduleApiError(body.error);
    }
  } catch {
    // fall through
  }
  return new ScheduleApiError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

function toDailySchedule(item: ScheduleResponse): DailySchedule {
  return {
    date: item.date,
    type: item.schedule_type,
    capacity: item.capacity,
    event_name: item.event_name || undefined,
    description: item.event_description || undefined,
    is_default: item.is_default,
  };
}

function buildSetScheduleRequest(
  type: ScheduleType,
  capacity: number,
  eventName: string,
  description: string,
): SetScheduleRequest {
  const body: SetScheduleRequest = {
    schedule_type: type,
    capacity: type === 'external_event' ? 0 : capacity,
  };

  if (type === 'event' || type === 'external_event') {
    body.event_name = eventName.trim();
    if (description.trim()) {
      body.event_description = description.trim();
    }
  } else if (type === 'special_menu') {
    if (eventName.trim()) body.event_name = eventName.trim();
    if (description.trim()) body.event_description = description.trim();
  }

  return body;
}

/** GET /api/v1/admin/schedules?year=&month= */
export async function listSchedules(year: number, month: number): Promise<DailySchedule[]> {
  const params = new URLSearchParams({
    year: String(year),
    month: String(month),
  });
  const res = await authedFetch(`/api/v1/admin/schedules?${params}`);

  if (!res.ok) {
    throw await parseError(res);
  }

  const data = (await res.json()) as ScheduleListResponse;
  return data.schedules.map(toDailySchedule);
}

/** PUT /api/v1/admin/schedules/{date} */
export async function setSchedule(
  date: string,
  type: ScheduleType,
  capacity: number,
  eventName: string,
  description: string,
): Promise<DailySchedule> {
  const res = await authedFetch(`/api/v1/admin/schedules/${date}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(buildSetScheduleRequest(type, capacity, eventName, description)),
  });

  if (!res.ok) {
    throw await parseError(res);
  }

  const data = (await res.json()) as ScheduleResponse;
  return toDailySchedule(data);
}

/** DELETE /api/v1/admin/schedules/{date} — 例外設定を削除し店舗定例に戻す */
export async function deleteSchedule(date: string): Promise<void> {
  const res = await authedFetch(`/api/v1/admin/schedules/${date}`, {
    method: 'DELETE',
  });

  if (!res.ok) {
    throw await parseError(res);
  }
}
