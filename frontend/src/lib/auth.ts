import type { LoginRequest, LoginResponse } from '@/types/reservation';

const API_BASE = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';
const ADMIN_BASE = `${API_BASE}/api/v1/admin`;

let accessToken: string | null = null;
let refreshPromise: Promise<string> | null = null;

export interface ApiErrorBody {
  code: string;
  message: string;
  details?: Array<{ field: string; message: string }>;
}

export class AuthError extends Error {
  code: string;
  details?: Array<{ field: string; message: string }>;

  constructor(error: ApiErrorBody) {
    super(error.message);
    this.name = 'AuthError';
    this.code = error.code;
    this.details = error.details;
  }
}

interface ErrorResponse {
  error: ApiErrorBody;
}

type RefreshResponse = Pick<LoginResponse, 'token' | 'expires_at'>;

async function parseErrorResponse(res: Response): Promise<AuthError> {
  try {
    const body = (await res.json()) as ErrorResponse;
    if (body?.error?.message) {
      return new AuthError(body.error);
    }
  } catch {
    // JSON パース失敗時は汎用メッセージへフォールバック
  }
  return new AuthError({
    code: 'INTERNAL_ERROR',
    message: 'サーバー内部でエラーが発生しました',
  });
}

/** POST /admin/login — AT をメモリに保存し、RT は httpOnly Cookie で受け取る */
export async function login(email: string, password: string): Promise<LoginResponse> {
  const res = await fetch(`${ADMIN_BASE}/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ email, password } satisfies LoginRequest),
  });

  if (!res.ok) {
    throw await parseErrorResponse(res);
  }

  const data = (await res.json()) as LoginResponse;
  accessToken = data.token;
  return data;
}

/** POST /admin/refresh — in-flight Promise を共有して多重 refresh を防ぐ */
export async function refresh(): Promise<string> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      const res = await fetch(`${ADMIN_BASE}/refresh`, {
        method: 'POST',
        credentials: 'include',
      });

      if (!res.ok) {
        clearAccessToken();
        throw await parseErrorResponse(res);
      }

      const data = (await res.json()) as RefreshResponse;
      accessToken = data.token;
      return data.token;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

/** メモリに AT があれば返す。なければ refresh で再取得 */
export async function ensureAccessToken(): Promise<string> {
  if (accessToken) {
    return accessToken;
  }
  return refresh();
}

export function clearAccessToken(): void {
  accessToken = null;
  refreshPromise = null;
}

/** AT のみ無効化（進行中の refresh Promise は維持） */
export function invalidateAccessToken(): void {
  accessToken = null;
}

/** POST /admin/logout — サーバ側で RT 失効後、メモリの AT をクリア */
export async function logout(): Promise<void> {
  const token = await ensureAccessToken().catch(() => null);

  try {
    if (token) {
      await fetch(`${ADMIN_BASE}/logout`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
        credentials: 'include',
      });
    }
  } finally {
    clearAccessToken();
  }
}

export function getAccessToken(): string | null {
  return accessToken;
}
