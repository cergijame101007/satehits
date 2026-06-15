import type { AvailabilityListResponse, AvailabilityResponse } from '@/types/reservation';

const API_BASE = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}

export class AvailabilityApiError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ErrorResponse['error']) {
    super(error.message);
    this.name = 'AvailabilityApiError';
    this.code = error.code;
    this.details = error.details;
  }
}

export function toAvailabilityErrorMessage(err: unknown): string {
  if (err instanceof AvailabilityApiError) return err.message;
  if (err instanceof Error) return err.message;
  return '空き状況の取得に失敗しました';
}

async function parseError(res: Response): Promise<AvailabilityApiError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new AvailabilityApiError(body.error);
    }
  } catch {
    // fall through
  }
  return new AvailabilityApiError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

/** GET /api/v1/reservations/availability?date=YYYY-MM-DD */
export async function getAvailability(date: string): Promise<AvailabilityResponse> {
  const params = new URLSearchParams({ date });
  const res = await fetch(`${API_BASE}/api/v1/reservations/availability?${params}`);

  if (!res.ok) {
    throw await parseError(res);
  }

  return (await res.json()) as AvailabilityResponse;
}

/** GET /api/v1/reservations/availability?year=&month= */
export async function listMonthlyAvailability(
  year: number,
  month: number,
): Promise<AvailabilityListResponse> {
  const params = new URLSearchParams({
    year: String(year),
    month: String(month),
  });
  const res = await fetch(`${API_BASE}/api/v1/reservations/availability?${params}`);

  if (!res.ok) {
    throw await parseError(res);
  }

  return (await res.json()) as AvailabilityListResponse;
}

/** 予約可能日範囲をカバーする年月の一覧を返す */
export function getMonthsInRange(min: Date, max: Date): Array<{ year: number; month: number }> {
  const months: Array<{ year: number; month: number }> = [];
  const current = new Date(min.getFullYear(), min.getMonth(), 1);
  const end = new Date(max.getFullYear(), max.getMonth(), 1);

  while (current <= end) {
    months.push({ year: current.getFullYear(), month: current.getMonth() + 1 });
    current.setMonth(current.getMonth() + 1);
  }

  return months;
}

/** 複数月の空き状況を取得して date → AvailabilityResponse の Map にまとめる */
export async function fetchAvailabilityMapForRange(
  min: Date,
  max: Date,
): Promise<Map<string, AvailabilityResponse>> {
  const months = getMonthsInRange(min, max);
  const results = await Promise.all(months.map(({ year, month }) => listMonthlyAvailability(year, month)));

  const map = new Map<string, AvailabilityResponse>();
  for (const result of results) {
    for (const item of result.availabilities) {
      map.set(item.date, item);
    }
  }
  return map;
}
