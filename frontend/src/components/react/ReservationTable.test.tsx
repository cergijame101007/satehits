import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ReservationTable from '@/components/react/ReservationTable';
import { listReservations, ReservationApiError, updateReservationStatus } from '@/lib/adminReservation';
import { getAvailability } from '@/lib/availability';
import { listSchedules } from '@/lib/schedule';
import type { AvailabilityResponse, DailySchedule, Reservation } from '@/types/reservation';

vi.mock('@/lib/adminReservation', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/adminReservation')>();
  return { ...actual, listReservations: vi.fn(), updateReservationStatus: vi.fn() };
});

vi.mock('@/lib/availability', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/availability')>();
  return { ...actual, getAvailability: vi.fn() };
});

vi.mock('@/lib/schedule', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/schedule')>();
  return { ...actual, listSchedules: vi.fn() };
});

const listReservationsMock = vi.mocked(listReservations);
const updateStatusMock = vi.mocked(updateReservationStatus);
const getAvailabilityMock = vi.mocked(getAvailability);
const listSchedulesMock = vi.mocked(listSchedules);

function reservation(overrides: Partial<Reservation>): Reservation {
  return {
    id: 'r-1',
    name: '山田太郎',
    people: 2,
    visit_date: '2026-09-10',
    visit_time: '12:00',
    phone: '090-1111-2222',
    email: 'yamada@example.com',
    note: '',
    status: 'pending',
    source: 'web',
    created_at: '2026-09-01T00:00:00Z',
    updated_at: '2026-09-01T00:00:00Z',
    ...overrides,
  };
}

const today = '2026-09-10';

const reservations: Reservation[] = [
  reservation({ id: 'r-2', name: '佐藤花子', visit_time: '13:00', status: 'approved', people: 3 }),
  reservation({ id: 'r-1', name: '山田太郎', visit_time: '12:00', status: 'pending', note: '窓側希望' }),
  reservation({ id: 'r-3', name: '鈴木一郎', visit_date: '2026-09-11', visit_time: '11:30' }),
];

function availability(date: string, overrides: Partial<AvailabilityResponse> = {}): AvailabilityResponse {
  return { date, capacity: 10, reserved: 3, available: 7, is_holiday: false, ...overrides };
}

function schedule(date: string, overrides: Partial<DailySchedule> = {}): DailySchedule {
  return { date, type: 'normal', capacity: 10, is_default: true, ...overrides };
}

function getDayCell(dayNumber: number): HTMLButtonElement {
  const cell = screen
    .getAllByRole('button')
    .find((b) => b.querySelector('span')?.textContent === String(dayNumber));
  if (!cell) throw new Error(`${dayNumber} 日のセルが見つかりません`);
  return cell as HTMLButtonElement;
}

function getReservationCard(name: string): HTMLElement {
  const card = screen.getByText(name).closest('.rounded-xl');
  if (!card) throw new Error(`${name} のカードが見つかりません`);
  return card as HTMLElement;
}

async function renderLoaded() {
  render(<ReservationTable />);
  await waitFor(() => {
    expect(screen.queryByText('読み込み中...')).not.toBeInTheDocument();
  });
}

describe('ReservationTable', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
    listReservationsMock.mockResolvedValue(reservations);
    getAvailabilityMock.mockImplementation(async (date) => availability(date));
    listSchedulesMock.mockResolvedValue([schedule(today), schedule('2026-09-11')]);
    updateStatusMock.mockResolvedValue({ id: 'r-1', status: 'approved', updated_at: '2026-09-10T00:00:00Z' });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('今日の予約を来店時間順に表示し、残り食数と予約登録リンクを表示する', async () => {
    await renderLoaded();

    expect(screen.getByRole('heading', { name: '予約一覧' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '+ 予約登録' })).toHaveAttribute('href', '/admin/reservations/new');
    expect(screen.getByText('2026年9月10日（木）')).toBeInTheDocument();

    const yamada = getReservationCard('山田太郎（2名）');
    const sato = getReservationCard('佐藤花子（3名）');
    expect(yamada.compareDocumentPosition(sato) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(within(yamada).getByText('申請中')).toBeInTheDocument();
    expect(within(yamada).getByText('備考: 窓側希望')).toBeInTheDocument();
    expect(within(yamada).getByText('090-1111-2222')).toBeInTheDocument();
    expect(within(yamada).getByText('yamada@example.com')).toBeInTheDocument();
    expect(within(sato).getByText('承認済み')).toBeInTheDocument();
    expect(screen.queryByText('鈴木一郎（2名）')).not.toBeInTheDocument();

    expect(await screen.findByText('7食')).toBeInTheDocument();
    expect(screen.getByText('予約済み: 3食')).toBeInTheDocument();
    expect(getAvailabilityMock).toHaveBeenCalledWith(today);
    expect(listSchedulesMock).toHaveBeenCalledWith(2026, 9);
  });

  it('ステータスに応じた操作ボタンだけを表示する', async () => {
    await renderLoaded();

    const pending = getReservationCard('山田太郎（2名）');
    expect(within(pending).getByRole('button', { name: '承認' })).toBeInTheDocument();
    expect(within(pending).getByRole('button', { name: '拒否' })).toBeInTheDocument();
    expect(within(pending).getByRole('button', { name: 'キャンセル' })).toBeInTheDocument();

    const approved = getReservationCard('佐藤花子（3名）');
    expect(within(approved).queryByRole('button', { name: '承認' })).not.toBeInTheDocument();
    expect(within(approved).getByRole('button', { name: 'No Show' })).toBeInTheDocument();
    expect(within(approved).getByRole('button', { name: 'キャンセル' })).toBeInTheDocument();
  });

  it('予約がない日は「予約がありません」を表示する', async () => {
    listReservationsMock.mockResolvedValue([]);
    await renderLoaded();

    expect(screen.getByText('予約がありません')).toBeInTheDocument();
  });

  it('一覧取得に失敗したらエラーを表示する', async () => {
    listReservationsMock.mockRejectedValue(
      new ReservationApiError({ code: 'INTERNAL_ERROR', message: 'サーバー内部でエラーが発生しました' }),
    );
    await renderLoaded();

    expect(screen.getByRole('alert')).toHaveTextContent('サーバー内部でエラーが発生しました');
    expect(screen.getByText('予約を取得できませんでした')).toBeInTheDocument();
  });

  it('空き状況の取得に失敗したらメッセージを表示する', async () => {
    getAvailabilityMock.mockRejectedValue(new Error('空き状況の取得に失敗しました'));
    await renderLoaded();

    expect(await screen.findByText('空き状況の取得に失敗しました')).toBeInTheDocument();
  });

  it('定休日は残り食数の代わりに定休日と表示する', async () => {
    listReservationsMock.mockResolvedValue([]);
    getAvailabilityMock.mockResolvedValue(availability(today, { is_holiday: true, capacity: 0, available: 0 }));
    listSchedulesMock.mockResolvedValue([schedule(today, { type: 'closed', capacity: 0 })]);
    await renderLoaded();

    expect(await screen.findByText('この日は定休日です')).toBeInTheDocument();
    // カレンダー凡例の「定休日」に加え、空き状況カードにも「定休日」が出る
    expect(screen.getAllByText('定休日')).toHaveLength(2);
    expect(screen.queryByText('予約済み: 3食')).not.toBeInTheDocument();
  });

  it('ステータスで絞り込める', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.selectOptions(screen.getByDisplayValue('すべてのステータス'), 'approved');

    expect(screen.getByText('佐藤花子（3名）')).toBeInTheDocument();
    expect(screen.queryByText('山田太郎（2名）')).not.toBeInTheDocument();
  });

  it('カレンダーで別の日を選ぶとその日の予約と空き状況を表示する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(getDayCell(11));

    expect(screen.getByText('2026年9月11日（金）')).toBeInTheDocument();
    expect(screen.getByText('鈴木一郎（2名）')).toBeInTheDocument();
    expect(screen.queryByText('山田太郎（2名）')).not.toBeInTheDocument();
    await waitFor(() => {
      expect(getAvailabilityMock).toHaveBeenCalledWith('2026-09-11');
    });
  });

  it('カレンダーの月移動で翌月のスケジュールと予約を取得する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(screen.getByRole('button', { name: '次の月' }));

    await waitFor(() => {
      expect(listSchedulesMock).toHaveBeenCalledWith(2026, 10);
    });
    expect(listReservationsMock).toHaveBeenCalledTimes(2);
  });

  it('承認は確認モーダルを経て API を呼び、一覧と空き状況を再取得する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(within(getReservationCard('山田太郎（2名）')).getByRole('button', { name: '承認' }));

    const dialog = screen.getByRole('dialog', { name: '予約を承認' });
    expect(within(dialog).getByText('山田太郎さん（12:00・2名）の予約を承認します。')).toBeInTheDocument();
    expect(within(dialog).queryByLabelText('拒否理由（任意）')).not.toBeInTheDocument();

    listReservationsMock.mockResolvedValue([
      reservation({ id: 'r-1', name: '山田太郎', visit_time: '12:00', status: 'approved' }),
    ]);
    await user.click(within(dialog).getByRole('button', { name: '承認する' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(updateStatusMock).toHaveBeenCalledWith('r-1', 'approved', undefined);
    expect(listReservationsMock).toHaveBeenCalledTimes(2);
    expect(getAvailabilityMock).toHaveBeenLastCalledWith(today);
    expect(within(getReservationCard('山田太郎（2名）')).getByText('承認済み')).toBeInTheDocument();
  });

  it('拒否は理由を入力でき、入力した理由付きで API を呼ぶ', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(within(getReservationCard('山田太郎（2名）')).getByRole('button', { name: '拒否' }));

    const dialog = screen.getByRole('dialog', { name: '予約を拒否' });
    await user.type(within(dialog).getByLabelText('拒否理由（任意）'), '定員超過のため');
    await user.click(within(dialog).getByRole('button', { name: '拒否する' }));

    await waitFor(() => {
      expect(updateStatusMock).toHaveBeenCalledWith('r-1', 'rejected', '定員超過のため');
    });
  });

  it('確認モーダルをキャンセルすると API を呼ばない', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(within(getReservationCard('山田太郎（2名）')).getByRole('button', { name: '承認' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'キャンセル' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(updateStatusMock).not.toHaveBeenCalled();
  });

  it('ステータス更新に失敗したらエラーを表示し、モーダルは開いたままにする', async () => {
    const user = userEvent.setup();
    updateStatusMock.mockRejectedValue(
      new ReservationApiError({ code: 'INVALID_TRANSITION', message: 'このステータスには変更できません' }),
    );
    await renderLoaded();

    await user.click(within(getReservationCard('山田太郎（2名）')).getByRole('button', { name: '承認' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '承認する' }));

    expect(await screen.findByText('このステータスには変更できません')).toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: '予約を承認' })).toBeInTheDocument();
    expect(within(screen.getByRole('dialog')).getByRole('button', { name: '承認する' })).toBeEnabled();
  });
});
