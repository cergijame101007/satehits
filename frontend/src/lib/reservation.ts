import type { Reservation, ReservationRequest } from '@/types/reservation';

const API_BASE = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

interface ErrorResponse {
  error: {
    code: string;
    message: string;
    details?: Array<{ field: string; message: string }>;
  };
}

export class ReservationApiError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ErrorResponse['error']) {
    super(error.message);
    this.name = 'ReservationApiError';
    this.code = error.code;
    this.details = error.details;
  }
}

export function toReservationErrorMessage(err: unknown): string {
  if (err instanceof ReservationApiError) return err.message;
  if (err instanceof Error) return err.message;
  return '予約の申請に失敗しました';
}

async function parseError(res: Response): Promise<ReservationApiError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new ReservationApiError(body.error);
    }
  } catch {
    // fall through
  }
  return new ReservationApiError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

/** POST /api/v1/reservations */
export async function createReservation(req: ReservationRequest): Promise<Reservation> {
  const res = await fetch(`${API_BASE}/api/v1/reservations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  if (!res.ok) {
    throw await parseError(res);
  }

  return (await res.json()) as Reservation;
}
