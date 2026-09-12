import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ReservationForm from '@/components/react/ReservationForm';
import { fetchAvailabilityMapForRange } from '@/lib/availability';
import { createReservation, ReservationApiError } from '@/lib/reservation';
import { createDeferred } from '@/test/deferred';
import type { AvailabilityResponse, Reservation } from '@/types/reservation';

vi.mock('@/lib/availability', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/availability')>();
  return { ...actual, fetchAvailabilityMapForRange: vi.fn() };
});

vi.mock('@/lib/reservation', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/reservation')>();
  return { ...actual, createReservation: vi.fn() };
});

// Turnstile 本体は外部スクリプトに依存するため、トークンを発行するボタンに差し替える
vi.mock('@/components/react/TurnstileWidget', () => ({
  default: ({ onToken }: { onToken: (token: string) => void }) => (
    <button type="button" onClick={() => onToken('turnstile-token')}>
      認証する
    </button>
  ),
}));

const fetchAvailabilityMock = vi.mocked(fetchAvailabilityMapForRange);
const createReservationMock = vi.mocked(createReservation);

function availability(date: string, overrides: Partial<AvailabilityResponse> = {}): AvailabilityResponse {
  return { date, capacity: 10, reserved: 5, available: 5, is_holiday: false, ...overrides };
}

/** 今日 = 2026-09-10（木）。予約可能範囲は 09-11〜09-24 */
function buildAvailabilityMap(): Map<string, AvailabilityResponse> {
  const map = new Map<string, AvailabilityResponse>();
  for (let d = 11; d <= 24; d++) {
    const date = `2026-09-${String(d).padStart(2, '0')}`;
    map.set(date, availability(date));
  }
  map.set('2026-09-12', availability('2026-09-12', { is_holiday: true, capacity: 0, available: 0 }));
  map.set('2026-09-13', availability('2026-09-13', { reserved: 10, available: 0 }));
  map.set('2026-09-14', availability('2026-09-14', { reserved: 8, available: 2 }));
  map.delete('2026-09-20');
  return map;
}

function getDayCell(dayNumber: number): HTMLButtonElement {
  const cell = screen
    .getAllByRole('button')
    .find((b) => b.querySelector('span')?.textContent === String(dayNumber));
  if (!cell) throw new Error(`${dayNumber} 日のセルが見つかりません`);
  return cell as HTMLButtonElement;
}

const createdReservation: Reservation = {
  id: 'r-1',
  name: '山田太郎',
  people: 2,
  visit_date: '2026-09-15',
  visit_time: '12:00',
  phone: '090-1234-5678',
  email: 'yamada@example.com',
  note: '',
  status: 'pending',
  source: 'web',
  created_at: '2026-09-10T00:00:00Z',
  updated_at: '2026-09-10T00:00:00Z',
};

async function renderLoaded() {
  render(<ReservationForm />);
  await screen.findByText('2026年9月');
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.click(getDayCell(15));
  await user.selectOptions(screen.getByDisplayValue('選択してください'), '12:00');
  await user.type(screen.getByPlaceholderText('山田太郎'), '山田太郎');
  await user.type(screen.getByPlaceholderText('090-1234-5678'), '090-1234-5678');
  await user.type(screen.getByPlaceholderText('yamada@example.com'), 'yamada@example.com');
}

describe('ReservationForm', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
    vi.stubEnv('PUBLIC_TURNSTILE_SITE_KEY', '');
    vi.stubGlobal('location', { href: '/reservation' });
    sessionStorage.clear();
    fetchAvailabilityMock.mockResolvedValue(buildAvailabilityMap());
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  it('翌日〜14日先の空き状況を取得し、読み込み後にカレンダーを表示する', async () => {
    render(<ReservationForm />);

    expect(screen.getByText('空き状況を読み込み中...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '予約を申請する' })).toBeDisabled();

    await screen.findByText('2026年9月');

    const [min, max] = fetchAvailabilityMock.mock.calls[0];
    expect(min).toEqual(new Date(2026, 8, 11, 0, 0, 0, 0));
    expect(max).toEqual(new Date(2026, 8, 24, 0, 0, 0, 0));
    expect(screen.getByRole('button', { name: '予約を申請する' })).toBeEnabled();
  });

  it('空き状況の取得に失敗したらメッセージを表示し、申請できない', async () => {
    fetchAvailabilityMock.mockRejectedValue(new Error('取得に失敗'));
    render(<ReservationForm />);

    expect(await screen.findByText('取得に失敗')).toBeInTheDocument();
    expect(
      screen.getByText('空き状況を取得できないため、来店日を選択できません'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '予約を申請する' })).toBeDisabled();
  });

  it('範囲外・休業日・満席・空き情報なしの日は選択できない', async () => {
    await renderLoaded();

    expect(getDayCell(10)).toBeDisabled(); // 今日（範囲外）
    expect(getDayCell(25)).toBeDisabled(); // 15 日先（範囲外）
    expect(getDayCell(20)).toBeDisabled(); // 空き情報なし
    expect(getDayCell(13)).toBeDisabled(); // 満席

    const holiday = getDayCell(12);
    expect(holiday).toBeDisabled();
    expect(within(holiday).getByText('休')).toBeInTheDocument();

    const open = getDayCell(15);
    expect(open).toBeEnabled();
    expect(within(open).getByText('残5')).toBeInTheDocument();
  });

  it('残り 3 食以下の日は残数を強調表示する', async () => {
    await renderLoaded();

    expect(within(getDayCell(14)).getByText('残2')).toHaveClass('text-red-500');
    expect(within(getDayCell(15)).getByText('残5')).toHaveClass('text-primary');
  });

  it('日付を選ぶと選択中の表示と残り食数が出て、来店時間を選べるようになる', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    const timeSelect = screen.getByDisplayValue('選択してください');
    expect(timeSelect).toBeDisabled();

    await user.click(getDayCell(15));

    expect(screen.getByText(/選択中: 2026年9月15日/)).toHaveTextContent('（残り 5 食）');
    expect(timeSelect).toBeEnabled();
    const slots = within(timeSelect).getAllByRole('option').map((o) => o.textContent);
    expect(slots).toEqual(['選択してください', '11:30', '12:00', '12:30', '13:00', '13:30']);
  });

  it('月移動で翌月のカレンダーを表示する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '›' }));
    expect(screen.getByText('2026年10月')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: '‹' }));
    expect(screen.getByText('2026年9月')).toBeInTheDocument();
  });

  it('必須項目が空なら送信せずエラーを表示する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '予約を申請する' }));

    expect(screen.getByText('来店日を選択してください')).toBeInTheDocument();
    expect(screen.getByText('来店時間を選択してください')).toBeInTheDocument();
    expect(screen.getByText('お名前を入力してください')).toBeInTheDocument();
    // 電話番号・メールアドレスは未入力でも形式チェックが後勝ちするため、文言は固定しない
    expect(screen.getByText(/^電話番号(を入力してください|の形式が正しくありません)$/)).toBeInTheDocument();
    expect(screen.getByText(/^メールアドレス(を入力してください|の形式が正しくありません)$/)).toBeInTheDocument();
    expect(createReservationMock).not.toHaveBeenCalled();
  });

  it('電話番号・メールアドレスの形式不正をエラー表示する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.type(screen.getByPlaceholderText('090-1234-5678'), 'abc');
    // ブラウザ（jsdom）の email 検証は通るが、コンポーネントの形式チェックには落ちる値
    await user.type(screen.getByPlaceholderText('yamada@example.com'), 'user@localhost');
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));

    expect(screen.getByText('電話番号の形式が正しくありません')).toBeInTheDocument();
    expect(screen.getByText('メールアドレスの形式が正しくありません')).toBeInTheDocument();
    expect(createReservationMock).not.toHaveBeenCalled();
  });

  it('入力を修正するとその項目のエラーが消える', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '予約を申請する' }));
    expect(screen.getByText('お名前を入力してください')).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText('山田太郎'), '山');

    expect(screen.queryByText('お名前を入力してください')).not.toBeInTheDocument();
    expect(screen.getByText(/^電話番号(を入力してください|の形式が正しくありません)$/)).toBeInTheDocument();
  });

  it('正常入力で API を呼び、完了情報を保存して完了ページへ遷移する', async () => {
    const user = userEvent.setup();
    createReservationMock.mockResolvedValue(createdReservation);
    await renderLoaded();

    await fillValidForm(user);
    await user.type(screen.getByPlaceholderText('魚の火入れ希望・テーブル席希望等'), ' テーブル席希望 ');
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));

    await waitFor(() => {
      expect(window.location.href).toBe('/reservation/complete');
    });
    expect(createReservationMock).toHaveBeenCalledWith({
      name: '山田太郎',
      people: 2,
      visit_date: '2026-09-15',
      visit_time: '12:00',
      phone: '090-1234-5678',
      email: 'yamada@example.com',
      note: 'テーブル席希望',
      turnstile_token: 'dev-bypass',
    });
    const stored = JSON.parse(sessionStorage.getItem('reservation_complete') ?? '{}');
    expect(stored).toMatchObject({ visit_time: '12:00', people: '2', name: '山田太郎' });
    expect(stored.visit_date).toMatch(/^2026年9月15日/);
  });

  it('人数の変更が送信内容に反映される', async () => {
    const user = userEvent.setup();
    createReservationMock.mockResolvedValue(createdReservation);
    await renderLoaded();

    await fillValidForm(user);
    await user.selectOptions(screen.getByDisplayValue('2名'), '4');
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));

    await waitFor(() => {
      expect(createReservationMock).toHaveBeenCalledWith(expect.objectContaining({ people: 4 }));
    });
  });

  it('VALIDATION_ERROR の details を項目ごとのエラーとして表示する', async () => {
    const user = userEvent.setup();
    createReservationMock.mockRejectedValue(
      new ReservationApiError({
        code: 'VALIDATION_ERROR',
        message: '入力内容に誤りがあります',
        details: [{ field: 'visit_date', message: '予約可能期間外です' }],
      }),
    );
    await renderLoaded();

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));

    expect(await screen.findByText('予約可能期間外です')).toBeInTheDocument();
    expect(window.location.href).toBe('/reservation');
  });

  it('その他の API エラーはメッセージをそのまま、想定外の例外は汎用メッセージを表示する', async () => {
    const user = userEvent.setup();
    createReservationMock.mockRejectedValueOnce(
      new ReservationApiError({ code: 'CAPACITY_EXCEEDED', message: '残り食数を超えています' }),
    );
    await renderLoaded();

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));
    expect(await screen.findByText('残り食数を超えています')).toBeInTheDocument();

    createReservationMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));
    expect(
      await screen.findByText('予約の申請に失敗しました。時間をおいて再度お試しください。'),
    ).toBeInTheDocument();
  });

  it('送信中はボタンが無効になり二重送信されない', async () => {
    const user = userEvent.setup();
    const deferred = createDeferred<Reservation>();
    createReservationMock.mockReturnValue(deferred.promise);
    await renderLoaded();

    await fillValidForm(user);
    await user.click(screen.getByRole('button', { name: '予約を申請する' }));

    const submitting = screen.getByRole('button', { name: /送信中/ });
    expect(submitting).toBeDisabled();
    await user.click(submitting);
    expect(createReservationMock).toHaveBeenCalledTimes(1);

    deferred.resolve(createdReservation);
    await waitFor(() => {
      expect(window.location.href).toBe('/reservation/complete');
    });
  });

  describe('Turnstile が有効な場合', () => {
    beforeEach(() => {
      vi.stubEnv('PUBLIC_TURNSTILE_SITE_KEY', 'site-key');
    });

    it('トークン取得前は申請できず、取得後はトークン付きで送信する', async () => {
      const user = userEvent.setup();
      createReservationMock.mockResolvedValue(createdReservation);
      await renderLoaded();

      expect(screen.getByRole('button', { name: '予約を申請する' })).toBeDisabled();
      expect(screen.queryByText(/開発環境: Turnstile/)).not.toBeInTheDocument();

      await user.click(screen.getByRole('button', { name: '認証する' }));
      expect(screen.getByRole('button', { name: '予約を申請する' })).toBeEnabled();

      await fillValidForm(user);
      await user.click(screen.getByRole('button', { name: '予約を申請する' }));

      await waitFor(() => {
        expect(createReservationMock).toHaveBeenCalledWith(
          expect.objectContaining({ turnstile_token: 'turnstile-token' }),
        );
      });
    });
  });
});
