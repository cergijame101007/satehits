import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import PublicScheduleCalendar from '@/components/react/PublicScheduleCalendar';
import { listPublicSchedules, PublicScheduleApiError } from '@/lib/publicSchedule';
import type { PublicDaySchedule } from '@/types/reservation';

vi.mock('@/lib/publicSchedule', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/publicSchedule')>();
  return { ...actual, listPublicSchedules: vi.fn() };
});

const listMock = vi.mocked(listPublicSchedules);

function day(date: string, overrides: Partial<PublicDaySchedule> = {}): PublicDaySchedule {
  return { date, schedule_type: 'normal', capacity: 10, available: 10, is_holiday: false, ...overrides };
}

/** 2026 年 9 月: 6 日（日）朝営業、10 日（木）定休、12 日イベント、19 日外部イベント、26 日特別メニュー */
const septemberSchedules: PublicDaySchedule[] = [
  day('2026-09-06', { schedule_type: 'morning' }),
  day('2026-09-10', { schedule_type: 'closed', capacity: 0, available: 0, is_holiday: true }),
  day('2026-09-12', { schedule_type: 'event', event_name: '羊の夜会', event_description: 'ラム尽くしのコース' }),
  day('2026-09-19', {
    schedule_type: 'external_event',
    capacity: 0,
    available: 0,
    is_holiday: true,
    event_name: 'マルシェ出店',
  }),
  day('2026-09-26', { schedule_type: 'special_menu', event_name: '秋の特別メニュー' }),
  day('2026-09-27', { schedule_type: 'special_menu' }),
];

/** グリッドの日付セル（数字を持つ最初の span の親）を取得する */
function getDayCell(dayNumber: number): HTMLElement {
  const calendar = screen.getByLabelText('2026年9月の営業スケジュール');
  const numberSpan = within(calendar)
    .getAllByText(String(dayNumber))
    .find((el) => el.tagName === 'SPAN');
  // Event バーは日付の入った内側 div の外にあるため、セル全体（min-h 指定の要素）まで遡る
  const cell = numberSpan?.closest('[class*="min-h-[4.5rem]"]');
  if (!cell) throw new Error(`${dayNumber} 日のセルが見つかりません`);
  return cell as HTMLElement;
}

async function renderLoaded() {
  render(<PublicScheduleCalendar />);
  return screen.findByLabelText('2026年9月の営業スケジュール');
}

describe('PublicScheduleCalendar', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
    listMock.mockResolvedValue(septemberSchedules);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('読み込み中の表示後、今月のスケジュールを取得して年・月・和名と営業時間を表示する', async () => {
    render(<PublicScheduleCalendar />);

    expect(screen.getByText('読み込み中…')).toBeInTheDocument();

    const calendar = await screen.findByLabelText('2026年9月の営業スケジュール');
    expect(listMock).toHaveBeenCalledWith(2026, 9);
    expect(within(calendar).getByText('2026')).toBeInTheDocument();
    // 「9」は日付セルにもあるため、月表示の大きな数字に限定する
    expect(within(calendar).getByText('9', { selector: '.text-4xl' })).toBeInTheDocument();
    expect(within(calendar).getByText('長月')).toBeInTheDocument();
    expect(within(calendar).getByText('11:30-15:00')).toBeInTheDocument();
    expect(within(calendar).getByText('日 8:30-15:00')).toBeInTheDocument();
    expect(within(calendar).getByText('木、金定休日')).toBeInTheDocument();
  });

  it('取得に失敗したら案内メッセージを表示する', async () => {
    listMock.mockRejectedValue(new PublicScheduleApiError({ code: 'INTERNAL_ERROR', message: 'error' }));
    render(<PublicScheduleCalendar />);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'スケジュールを表示できません。時間をおいて再度ご確認ください。',
    );
    expect(screen.queryByLabelText(/営業スケジュール/)).not.toBeInTheDocument();
  });

  it('定休日・外部イベント日には「休」を表示し、通常日には表示しない', async () => {
    await renderLoaded();

    expect(within(getDayCell(10)).getByText('休')).toBeInTheDocument();
    expect(within(getDayCell(19)).getByText('休')).toBeInTheDocument();
    expect(within(getDayCell(11)).queryByText('休')).not.toBeInTheDocument();
  });

  it('イベント名のある日に Event バーを表示し、イベント名のない特別メニュー日には表示しない', async () => {
    await renderLoaded();

    expect(within(getDayCell(12)).getByText('Event')).toBeInTheDocument();
    expect(within(getDayCell(19)).getByText('Event')).toBeInTheDocument();
    expect(within(getDayCell(26)).getByText('Event')).toBeInTheDocument();
    expect(within(getDayCell(27)).queryByText('Event')).not.toBeInTheDocument();
    expect(within(getDayCell(11)).queryByText('Event')).not.toBeInTheDocument();
  });

  it('朝営業の日は日付を丸で囲む', async () => {
    await renderLoaded();

    expect(within(getDayCell(6)).getByText('6')).toHaveClass('rounded-full');
    expect(within(getDayCell(11)).getByText('11')).not.toHaveClass('rounded-full');
  });

  it('Event Info にイベントを日付順に、説明付きで一覧表示する', async () => {
    await renderLoaded();

    expect(screen.getByText('◎ Event Info')).toBeInTheDocument();
    const items = screen.getAllByRole('listitem');
    expect(items.map((li) => li.textContent)).toEqual([
      '9/12(土)…羊の夜会ラム尽くしのコース',
      '9/19(土)…マルシェ出店',
      '9/26(土)…秋の特別メニュー',
    ]);
  });

  it('イベントがない月は Event Info を表示しない', async () => {
    listMock.mockResolvedValue([day('2026-09-11')]);
    await renderLoaded();

    expect(screen.queryByText('◎ Event Info')).not.toBeInTheDocument();
    expect(screen.getByText('〇…朝営業')).toBeInTheDocument();
  });
});
