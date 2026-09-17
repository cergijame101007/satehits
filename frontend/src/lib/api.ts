import { invalidateAccessToken, ensureAccessToken, refresh, clearAccessToken } from '@/lib/auth';

const API_BASE = import.meta.env.PUBLIC_API_URL ?? 'http://localhost:8080';

/** 認証切れでログイン画面へリダイレクトしたときに投げる。呼び出し側はエラー表示・ログを省略してよい */
export class UnauthorizedError extends Error {
  constructor() {
    super('Unauthorized');
    this.name = 'UnauthorizedError';
  }
}

function redirectToLogin(): never {
  clearAccessToken();
  window.location.href = '/admin/login';
  throw new UnauthorizedError();
}

/** 保護 API 用 fetch — Bearer 付与、401 時は refresh 1 回リトライ */
export async function authedFetch(path: string, init: RequestInit = {}): Promise<Response> {
  let token: string;
  try {
    token = await ensureAccessToken();
  } catch {
    redirectToLogin();
  }

  const headers = new Headers(init.headers);
  headers.set('Authorization', `Bearer ${token}`);

  let res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
    credentials: 'include',
  });

  if (res.status === 401) {
    invalidateAccessToken();
    try {
      token = await refresh();
    } catch {
      redirectToLogin();
    }

    headers.set('Authorization', `Bearer ${token}`);
    res = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
      credentials: 'include',
    });

    if (res.status === 401) {
      redirectToLogin();
    }
  }

  return res;
}
