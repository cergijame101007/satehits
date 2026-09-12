import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import DashboardSummary from '@/components/react/DashboardSummary';
import { listReservations, ReservationApiError } from '@/lib/adminReservation';
import { AvailabilityApiError, getAvailability } from '@/lib/availability';
import type { AvailabilityResponse, Reservation } from '@/types/reservation';

vi.mock('@/lib/adminReservation', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/adminReservation')>();
  return { ...actual, listReservations: vi.fn() };
});

vi.mock('@/lib/availability', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/availability')>();
  return { ...actual, getAvailability: vi.fn() };
});

const listReservationsMock = vi.mocked(listReservations);
const getAvailabilityMock = vi.mocked(getAvailability);

const today = '2026-09-10';
const tomorrow = '2026-09-11';

function reservation(overrides: Partial<Reservation>): Reservation {
  return {
    id: 'r-1',
    name: '山田太郎',
    people: 2,
    visit_date: today,
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

function availability(date: string, overrides: Partial<AvailabilityResponse> = {}): AvailabilityResponse {
  return { date, capacity: 10, reserved: 5, available: 5, is_holiday: false, ...overrides };
}

const todayReservations: Reservation[] = [
  reservation({ id: 'r-1', name: '山田太郎', visit_time: '12:00', status: 'pending' }),
  reservation({ id: 'r-2', name: '佐藤花子', visit_time: '12:30', status: 'approved', people: 3 }),
  reservation({ id: 'r-3', name: '鈴木一郎', visit_time: '13:00', status: 'approved', people: 2 }),
  reservation({ id: 'r-4', name: '高橋次郎', visit_time: '13:30', status: 'rejected' }),
];

function getCard(title: string): HTMLElement {
  const card = screen.getByText(title).closest('.rounded-xl');
  if (!card) throw new Error(`${title} のカードが見つかりません`);
  return card as HTMLElement;
}

async function renderLoaded() {
  render(<DashboardSummary />);
  await waitFor(() => {
    expect(screen.queryAllByText('読み込み中...')).toHaveLength(0);
  });
}

describe('DashboardSummary', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
    getAvailabilityMock.mockImplementation(async (date) => availability(date));
    listReservationsMock.mockImplementation(async (date) => (date === today ? todayReservations : []));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('今日と明日の日付でそれぞれ空き状況と予約を取得する', async () => {
    render(<DashboardSummary />);

    expect(screen.getAllByText('読み込み中...')).toHaveLength(2);
    expect(screen.getByText('2026年9月10日（木）')).toBeInTheDocument();
    expect(screen.getByText('2026年9月11日（金）')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.queryAllByText('読み込み中...')).toHaveLength(0);
    });
    expect(getAvailabilityMock).toHaveBeenCalledWith(today);
    expect(getAvailabilityMock).toHaveBeenCalledWith(tomorrow);
    expect(listReservationsMock).toHaveBeenCalledWith(today);
    expect(listReservationsMock).toHaveBeenCalledWith(tomorrow);
  });

  it('残り提供数・承認待ち件数・承認済み件数（人数）と直近 3 件を表示する', async () => {
    await renderLoaded();

    const card = getCard('今日の予約');
    expect(within(card).getByText('5')).toBeInTheDocument();
    expect(within(card).getByText('/ 10食')).toBeInTheDocument();
    expect(within(card).getByText('1件')).toBeInTheDocument();
    expect(within(card).getByText('2件')).toBeInTheDocument();
    expect(within(card).getByText('（5名）')).toBeInTheDocument();

    expect(within(card).getByText('12:00 山田太郎（2名）')).toBeInTheDocument();
    expect(within(card).getByText('12:30 佐藤花子（3名）')).toBeInTheDocument();
    expect(within(card).getByText('13:00 鈴木一郎（2名）')).toBeInTheDocument();
    expect(within(card).queryByText(/高橋次郎/)).not.toBeInTheDocument();
    expect(within(card).getByText('他 1件')).toBeInTheDocument();
    expect(within(card).getByText('申請中')).toBeInTheDocument();
  });

  it('予約がない日は件数 0 で一覧を出さない', async () => {
    await renderLoaded();

    const card = getCard('明日の予約');
    // 承認待ち・承認済みの両方が 0 件
    expect(within(card).getAllByText('0件')).toHaveLength(2);
    expect(within(card).queryByText(/他 /)).not.toBeInTheDocument();
  });

  it('定休日は残り提供数の代わりに定休日と表示する', async () => {
    getAvailabilityMock.mockImplementation(async (date) =>
      availability(date, { is_holiday: date === tomorrow, capacity: 0, available: 0 }),
    );
    await renderLoaded();

    const card = getCard('明日の予約');
    expect(within(card).getByText('定休日')).toBeInTheDocument();
    expect(within(card).queryByText('残り提供数')).not.toBeInTheDocument();
  });

  it('空き状況の取得に失敗したらエラーを表示し、残数を出さない', async () => {
    getAvailabilityMock.mockImplementation(async (date) => {
      if (date === today) {
        throw new AvailabilityApiError({ code: 'INTERNAL_ERROR', message: '空き状況を取得できません' });
      }
      return availability(date);
    });
    await renderLoaded();

    const card = getCard('今日の予約');
    expect(within(card).getByRole('alert')).toHaveTextContent('空き状況を取得できません');
    expect(within(card).queryByText('残り提供数')).not.toBeInTheDocument();
    // 予約一覧は取得できているので表示される
    expect(within(card).getByText('12:00 山田太郎（2名）')).toBeInTheDocument();
  });

  it('予約の取得に失敗したらエラーを表示し、件数を「—」にする', async () => {
    listReservationsMock.mockImplementation(async (date) => {
      if (date === today) {
        throw new ReservationApiError({ code: 'INTERNAL_ERROR', message: '予約一覧を取得できません' });
      }
      return [];
    });
    await renderLoaded();

    const card = getCard('今日の予約');
    expect(within(card).getByRole('alert')).toHaveTextContent('予約一覧を取得できません');
    expect(within(card).getAllByText('—')).toHaveLength(2);
    expect(within(card).getByText('5')).toBeInTheDocument();
  });

  it('管理画面へのクイックリンクとサイト確認リンクを表示する', async () => {
    await renderLoaded();

    expect(screen.getByRole('link', { name: '予約一覧' })).toHaveAttribute('href', '/admin/reservations');
    expect(screen.getByRole('link', { name: '予約登録' })).toHaveAttribute('href', '/admin/reservations/new');
    expect(screen.getByRole('link', { name: 'スケジュール' })).toHaveAttribute('href', '/admin/schedules');
    expect(screen.getByRole('link', { name: '取引先' })).toHaveAttribute('href', '/admin/suppliers');
    const site = screen.getByRole('link', { name: 'サイト確認' });
    expect(site).toHaveAttribute('href', '/');
    expect(site).toHaveAttribute('target', '_blank');
  });
});
