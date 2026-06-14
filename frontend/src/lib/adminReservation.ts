import { authedFetch } from '@/lib/api';
import type {
  AdminReservationRequest,
  Reservation,
  ReservationListResponse,
  ReservationStatus,
  ReservationSource,
  UpdateStatusResponse,
} from '@/types/reservation';

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

/** GET /api/v1/admin/reservations */
export async function listReservations(
  date?: string,
  status?: string,
  source?: string,
): Promise<Reservation[]> {
  const params = new URLSearchParams();
  if (date) params.set('date', date);
  if (status) params.set('status', status);
  if (source) params.set('source', source);

  const query = params.toString();
  const path = query ? `/api/v1/admin/reservations?${query}` : '/api/v1/admin/reservations';
  const res = await authedFetch(path);

  if (!res.ok) {
    throw await parseError(res);
  }

  const data = (await res.json()) as ReservationListResponse;
  return data.reservations;
}

/** POST /api/v1/admin/reservations */
export async function createAdminReservation(req: AdminReservationRequest): Promise<Reservation> {
  const res = await authedFetch('/api/v1/admin/reservations', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });

  if (!res.ok) {
    throw await parseError(res);
  }

  return (await res.json()) as Reservation;
}

/** PATCH /api/v1/admin/reservations/{id}/status */
export async function updateReservationStatus(
  id: string,
  status: ReservationStatus,
): Promise<UpdateStatusResponse> {
  const res = await authedFetch(`/api/v1/admin/reservations/${id}/status`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ status }),
  });

  if (!res.ok) {
    throw await parseError(res);
  }

  return (await res.json()) as UpdateStatusResponse;
}

export type { ReservationSource };
