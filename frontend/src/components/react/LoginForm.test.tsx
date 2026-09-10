import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import LoginForm from '@/components/react/LoginForm';
import { AuthError, login } from '@/lib/auth';
import { createDeferred } from '@/test/deferred';
import type { LoginResponse } from '@/types/reservation';

vi.mock('@/lib/auth', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/auth')>();
  return { ...actual, login: vi.fn() };
});

const loginMock = vi.mocked(login);

const loginResponse: LoginResponse = {
  token: 'token',
  expires_at: '2026-09-11T00:00:00Z',
  user: { id: 1, email: 'owner@example.com', role: 'owner' },
};

// label と input が htmlFor で結び付いていないため、type 属性で取得する
function emailInput(): HTMLInputElement {
  return document.querySelector<HTMLInputElement>('input[type="email"]')!;
}

function passwordInput(): HTMLInputElement {
  return document.querySelector<HTMLInputElement>('input[type="password"]')!;
}

async function fillCredentials(user: ReturnType<typeof userEvent.setup>) {
  await user.type(emailInput(), 'owner@example.com');
  await user.type(passwordInput(), 'secret');
}

describe('LoginForm', () => {
  beforeEach(() => {
    vi.stubGlobal('location', { href: '/admin/login' });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('メールアドレス・パスワードの入力欄とログインボタンを表示する', () => {
    render(<LoginForm />);

    expect(screen.getByRole('heading', { name: '管理者ログイン' })).toBeInTheDocument();
    expect(screen.getByText('メールアドレス')).toBeInTheDocument();
    expect(screen.getByText('パスワード')).toBeInTheDocument();
    expect(emailInput()).toBeRequired();
    expect(passwordInput()).toBeRequired();
    expect(screen.getByRole('button', { name: 'ログイン' })).toBeEnabled();
  });

  it('未入力で送信すると API を呼ばずエラーを表示する', async () => {
    render(<LoginForm />);

    // required 属性によるブラウザ検証を通さず、コンポーネント側のバリデーションを検証する
    fireEvent.submit(screen.getByRole('button', { name: 'ログイン' }).closest('form')!);

    expect(
      await screen.findByText('メールアドレスとパスワードを入力してください'),
    ).toBeInTheDocument();
    expect(loginMock).not.toHaveBeenCalled();
  });

  it('入力内容で login を呼び、成功したら /admin に遷移する', async () => {
    const user = userEvent.setup();
    loginMock.mockResolvedValue(loginResponse);
    render(<LoginForm />);

    await fillCredentials(user);
    await user.click(screen.getByRole('button', { name: 'ログイン' }));

    await waitFor(() => {
      expect(window.location.href).toBe('/admin');
    });
    expect(loginMock).toHaveBeenCalledWith('owner@example.com', 'secret');
  });

  it('UNAUTHORIZED なら認証情報の誤りを案内する', async () => {
    const user = userEvent.setup();
    loginMock.mockRejectedValue(new AuthError({ code: 'UNAUTHORIZED', message: 'unauthorized' }));
    render(<LoginForm />);

    await fillCredentials(user);
    await user.click(screen.getByRole('button', { name: 'ログイン' }));

    expect(
      await screen.findByText('メールアドレスまたはパスワードが正しくありません'),
    ).toBeInTheDocument();
    expect(window.location.href).toBe('/admin/login');
  });

  it('TOO_MANY_REQUESTS / VALIDATION_ERROR はサーバーのメッセージをそのまま表示する', async () => {
    const user = userEvent.setup();
    loginMock.mockRejectedValue(
      new AuthError({
        code: 'TOO_MANY_REQUESTS',
        message: 'ログイン試行が多すぎます。60秒後に再試行してください',
      }),
    );
    render(<LoginForm />);

    await fillCredentials(user);
    await user.click(screen.getByRole('button', { name: 'ログイン' }));

    expect(
      await screen.findByText('ログイン試行が多すぎます。60秒後に再試行してください'),
    ).toBeInTheDocument();
  });

  it('その他の AuthError や想定外の例外は汎用メッセージを表示する', async () => {
    const user = userEvent.setup();
    loginMock.mockRejectedValueOnce(
      new AuthError({ code: 'INTERNAL_ERROR', message: 'サーバー内部でエラーが発生しました' }),
    );
    render(<LoginForm />);

    await fillCredentials(user);
    await user.click(screen.getByRole('button', { name: 'ログイン' }));
    expect(
      await screen.findByText('ログインに失敗しました。時間をおいて再度お試しください。'),
    ).toBeInTheDocument();

    loginMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
    await user.click(screen.getByRole('button', { name: 'ログイン' }));
    expect(
      await screen.findByText('ログインに失敗しました。時間をおいて再度お試しください。'),
    ).toBeInTheDocument();
    expect(loginMock).toHaveBeenCalledTimes(2);
  });

  it('送信中はボタンが無効になり二重送信されない', async () => {
    const user = userEvent.setup();
    const deferred = createDeferred<LoginResponse>();
    loginMock.mockReturnValue(deferred.promise);
    render(<LoginForm />);

    await fillCredentials(user);
    const button = screen.getByRole('button', { name: 'ログイン' });
    await user.click(button);

    expect(screen.getByRole('button', { name: 'ログイン中...' })).toBeDisabled();
    await user.click(screen.getByRole('button', { name: 'ログイン中...' }));
    expect(loginMock).toHaveBeenCalledTimes(1);

    deferred.resolve(loginResponse);
    await waitFor(() => {
      expect(window.location.href).toBe('/admin');
    });
  });
});
