import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ReservationCreateForm from '@/components/react/ReservationCreateForm';
import { createAdminReservation, ReservationApiError } from '@/lib/adminReservation';
import { getAvailability } from '@/lib/availability';
import { createDeferred } from '@/test/deferred';
import type { AvailabilityResponse, Reservation } from '@/types/reservation';

vi.mock('@/lib/adminReservation', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/adminReservation')>();
  return { ...actual, createAdminReservation: vi.fn() };
});

vi.mock('@/lib/availability', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/availability')>();
  return { ...actual, getAvailability: vi.fn() };
});

// 日付ピッカーは単体でテスト済みのため、日付を返すボタンに差し替える
vi.mock('@/components/react/DatePickerField', () => ({
  default: ({
    value,
    onChange,
    error,
  }: {
    value: string;
    onChange: (date: string) => void;
    error?: string;
  }) => (
    <div>
      <button type="button" onClick={() => onChange('2026-09-15')}>
        日付を選択
      </button>
      <span data-testid="visit-date">{value}</span>
      {error && <p>{error}</p>}
    </div>
  ),
}));

const createMock = vi.mocked(createAdminReservation);
const getAvailabilityMock = vi.mocked(getAvailability);

const availabilityOk: AvailabilityResponse = {
  date: '2026-09-15',
  capacity: 10,
  reserved: 5,
  available: 5,
  is_holiday: false,
};

const created: Reservation = {
  id: 'r-1',
  name: '山田太郎',
  people: 2,
  visit_date: '2026-09-15',
  visit_time: '12:00',
  phone: '090-1234-5678',
  email: 'yamada@example.com',
  note: '',
  status: 'approved',
  source: 'instagram',
  created_at: '2026-09-10T00:00:00Z',
  updated_at: '2026-09-10T00:00:00Z',
};

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole('button', { name: '日付を選択' }));
  await user.type(screen.getByPlaceholderText('山田太郎'), '山田太郎');
  await user.type(screen.getByPlaceholderText('090-1234-5678'), '090-1234-5678');
  await user.type(screen.getByPlaceholderText('yamada@example.com'), 'yamada@example.com');
}

describe('ReservationCreateForm', () => {
  let confirmSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true);
    getAvailabilityMock.mockResolvedValue(availabilityOk);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.clearAllMocks();
  });

  it('初期値は Instagram 経由・12:00・2名・承認済み', () => {
    render(<ReservationCreateForm />);

    expect(screen.getByDisplayValue('Instagram')).toBeInTheDocument();
    expect(screen.getByDisplayValue('12:00')).toBeInTheDocument();
    expect(screen.getByDisplayValue('2名')).toBeInTheDocument();
    expect(screen.getByDisplayValue('承認済み')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '登録する' })).toBeEnabled();
  });

  it('必須項目が空なら送信せずエラーを表示する', async () => {
    const user = userEvent.setup();
    render(<ReservationCreateForm />);

    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(screen.getByText('お名前を入力してください')).toBeInTheDocument();
    expect(screen.getByText('来店日を入力してください')).toBeInTheDocument();
    expect(screen.getByText('電話番号を入力してください')).toBeInTheDocument();
    expect(screen.getByText('メールアドレスを入力してください')).toBeInTheDocument();
    expect(getAvailabilityMock).not.toHaveBeenCalled();
    expect(createMock).not.toHaveBeenCalled();
  });

  it('入力するとその項目のエラーが消える', async () => {
    const user = userEvent.setup();
    render(<ReservationCreateForm />);

    await user.click(screen.getByRole('button', { name: '登録する' }));
    await user.type(screen.getByPlaceholderText('山田太郎'), '山');
    await user.click(screen.getByRole('button', { name: '日付を選択' }));

    expect(screen.queryByText('お名前を入力してください')).not.toBeInTheDocument();
    expect(screen.queryByText('来店日を入力してください')).not.toBeInTheDocument();
    expect(screen.getByText('電話番号を入力してください')).toBeInTheDocument();
  });

  it('提供数に収まる場合は確認なしで登録し、完了画面を表示する', async () => {
    const user = userEvent.setup();
    createMock.mockResolvedValue(created);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(await screen.findByText('予約を登録しました')).toBeInTheDocument();
    expect(getAvailabilityMock).toHaveBeenCalledWith('2026-09-15');
    expect(confirmSpy).not.toHaveBeenCalled();
    expect(createMock).toHaveBeenCalledWith({
      source: 'instagram',
      name: '山田太郎',
      people: 2,
      visit_date: '2026-09-15',
      visit_time: '12:00',
      phone: '090-1234-5678',
      email: 'yamada@example.com',
      note: '',
      status: 'approved',
    });
    expect(screen.getByRole('link', { name: '予約一覧へ' })).toHaveAttribute(
      'href',
      '/admin/reservations',
    );
  });

  it('経路・時間・人数・ステータス・備考の変更が送信内容に反映される', async () => {
    const user = userEvent.setup();
    createMock.mockResolvedValue(created);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.selectOptions(screen.getByDisplayValue('Instagram'), 'phone');
    await user.selectOptions(screen.getByDisplayValue('12:00'), '11:30');
    await user.selectOptions(screen.getByDisplayValue('2名'), '3');
    await user.selectOptions(screen.getByDisplayValue('承認済み'), 'pending');
    await user.type(document.querySelector<HTMLTextAreaElement>('textarea[name="note"]')!, '窓側希望');
    await user.click(screen.getByRole('button', { name: '登録する' }));

    await waitFor(() => {
      expect(createMock).toHaveBeenCalledWith(
        expect.objectContaining({
          source: 'phone',
          visit_time: '11:30',
          people: 3,
          status: 'pending',
          note: '窓側希望',
        }),
      );
    });
  });

  it('提供数を超える場合は確認ダイアログを出し、キャンセルなら登録しない', async () => {
    const user = userEvent.setup();
    getAvailabilityMock.mockResolvedValue({ ...availabilityOk, reserved: 9, available: 1 });
    confirmSpy.mockReturnValue(false);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));

    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalledTimes(1);
    });
    expect(confirmSpy).toHaveBeenCalledWith(
      '提供可能数（10食）を超えて登録されます（予約済み 9食 + 今回 2食 = 11食）。このまま登録しますか？',
    );
    expect(createMock).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '登録する' })).toBeEnabled();
  });

  it('提供数を超えても確認で OK なら登録する', async () => {
    const user = userEvent.setup();
    getAvailabilityMock.mockResolvedValue({ ...availabilityOk, reserved: 9, available: 1 });
    confirmSpy.mockReturnValue(true);
    createMock.mockResolvedValue(created);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(await screen.findByText('予約を登録しました')).toBeInTheDocument();
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    expect(createMock).toHaveBeenCalledTimes(1);
  });

  it('定休日は超過確認をせずそのまま登録する', async () => {
    const user = userEvent.setup();
    getAvailabilityMock.mockResolvedValue({
      ...availabilityOk,
      is_holiday: true,
      capacity: 0,
      reserved: 0,
      available: 0,
    });
    createMock.mockResolvedValue(created);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(await screen.findByText('予約を登録しました')).toBeInTheDocument();
    expect(confirmSpy).not.toHaveBeenCalled();
  });

  it('API の details をフィールドごとのエラーとして表示する', async () => {
    const user = userEvent.setup();
    createMock.mockRejectedValue(
      new ReservationApiError({
        code: 'VALIDATION_ERROR',
        message: '入力内容に誤りがあります',
        details: [{ field: 'email', message: 'メールアドレスの形式が正しくありません' }],
      }),
    );
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));

    expect(await screen.findByText('メールアドレスの形式が正しくありません')).toBeInTheDocument();
    expect(screen.queryByText('予約を登録しました')).not.toBeInTheDocument();
  });

  it('details のない API エラーはメッセージを、想定外の例外は汎用メッセージを表示する', async () => {
    const user = userEvent.setup();
    createMock.mockRejectedValueOnce(
      new ReservationApiError({ code: 'CAPACITY_EXCEEDED', message: '残り食数を超えています' }),
    );
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('残り食数を超えています');

    createMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
    await user.click(screen.getByRole('button', { name: '登録する' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('登録に失敗しました');
  });

  it('送信中はボタンが無効になり二重送信されない', async () => {
    const user = userEvent.setup();
    const deferred = createDeferred<Reservation>();
    createMock.mockReturnValue(deferred.promise);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));

    const submitting = await screen.findByRole('button', { name: '登録中...' });
    expect(submitting).toBeDisabled();
    await user.click(submitting);
    expect(createMock).toHaveBeenCalledTimes(1);

    deferred.resolve(created);
    expect(await screen.findByText('予約を登録しました')).toBeInTheDocument();
  });

  it('「続けて登録」で初期状態のフォームに戻る', async () => {
    const user = userEvent.setup();
    createMock.mockResolvedValue(created);
    render(<ReservationCreateForm />);

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '登録する' }));
    await screen.findByText('予約を登録しました');

    await user.click(screen.getByRole('button', { name: '続けて登録' }));

    expect(screen.getByRole('heading', { name: '予約登録' })).toBeInTheDocument();
    expect(screen.getByPlaceholderText('山田太郎')).toHaveValue('');
    expect(screen.getByTestId('visit-date')).toHaveTextContent('');
  });
});
