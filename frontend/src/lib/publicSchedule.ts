import type { MonthlyPublicScheduleResponse, PublicDaySchedule } from '@/types/reservation';

const API_BASE = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}

export class PublicScheduleApiError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ErrorResponse['error']) {
    super(error.message);
    this.name = 'PublicScheduleApiError';
    this.code = error.code;
    this.details = error.details;
  }
}

async function parseError(res: Response): Promise<PublicScheduleApiError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new PublicScheduleApiError(body.error);
    }
  } catch {
    // fall through
  }
  return new PublicScheduleApiError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

/** GET /api/v1/schedules?year=&month= */
export async function listPublicSchedules(
  year: number,
  month: number,
): Promise<PublicDaySchedule[]> {
  const params = new URLSearchParams({
    year: String(year),
    month: String(month),
  });
  const res = await fetch(`${API_BASE}/api/v1/schedules?${params}`);

  if (!res.ok) {
    throw await parseError(res);
  }

  const data = (await res.json()) as MonthlyPublicScheduleResponse;
  return data.schedules;
}

export type { PublicDaySchedule, MonthlyPublicScheduleResponse };
