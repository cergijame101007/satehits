import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import DatePickerField from '@/components/react/DatePickerField';
import { useMonthCalendarData } from '@/lib/useMonthCalendar';
import type { MonthCalendarDay } from '@/components/react/MonthCalendar';

vi.mock('@/lib/useMonthCalendar', () => ({
  useMonthCalendarData: vi.fn(),
}));

const useMonthCalendarDataMock = vi.mocked(useMonthCalendarData);

const septemberDays: MonthCalendarDay[] = [
  {
    date: '2026-09-15',
    schedule: { date: '2026-09-15', type: 'normal', capacity: 10, is_default: true },
    reservation: { count: 1, reservedMeals: 3 },
  },
];

function getDayCell(dayNumber: number): HTMLButtonElement {
  const cell = screen
    .getAllByRole('button')
    .find((b) => b.querySelector('span')?.textContent === String(dayNumber));
  if (!cell) throw new Error(`${dayNumber} 日のセルが見つかりません`);
  return cell as HTMLButtonElement;
}

describe('DatePickerField', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
    useMonthCalendarDataMock.mockReturnValue({
      days: septemberDays,
      isLoading: false,
      error: '',
      summaryError: '',
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('値が空ならプレースホルダー、値があれば YYYY/MM/DD で表示する', () => {
    const { rerender } = render(<DatePickerField value="" onChange={() => {}} />);
    expect(screen.getByRole('button', { name: /日付を選択/ })).toBeInTheDocument();

    rerender(<DatePickerField value="2026-09-15" onChange={() => {}} />);
    expect(screen.getByRole('button', { name: /2026\/09\/15/ })).toBeInTheDocument();
  });

  it('ラベルと必須マークを表示する', () => {
    render(<DatePickerField value="" onChange={() => {}} label="予約日" required />);

    expect(screen.getByText('予約日')).toBeInTheDocument();
    expect(screen.getByText('*')).toBeInTheDocument();
  });

  it('閉じている間はデータを取得せず、開いたときに今月のデータを取得する', async () => {
    const user = userEvent.setup();
    render(<DatePickerField value="" onChange={() => {}} />);

    expect(useMonthCalendarDataMock).toHaveBeenLastCalledWith(
      2026,
      9,
      expect.objectContaining({ enabled: false, includeReservationSummary: true }),
    );
    expect(screen.queryByText('2026年9月')).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /日付を選択/ }));

    expect(screen.getByText('2026年9月')).toBeInTheDocument();
    expect(useMonthCalendarDataMock).toHaveBeenLastCalledWith(
      2026,
      9,
      expect.objectContaining({ enabled: true }),
    );
  });

  it('値がある場合はその年月から表示を始める', async () => {
    const user = userEvent.setup();
    render(<DatePickerField value="2026-11-03" onChange={() => {}} />);

    await user.click(screen.getByRole('button', { name: /2026\/11\/03/ }));

    expect(screen.getByText('2026年11月')).toBeInTheDocument();
    expect(useMonthCalendarDataMock).toHaveBeenLastCalledWith(2026, 11, expect.anything());
  });

  it('日付セルを選ぶと onChange に渡し、ピッカーを閉じる', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<DatePickerField value="" onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: /日付を選択/ }));
    await user.click(getDayCell(15));

    expect(onChange).toHaveBeenCalledWith('2026-09-15');
    expect(screen.queryByText('2026年9月')).not.toBeInTheDocument();
  });

  it('管理者向けなので休業日も選択できる', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    useMonthCalendarDataMock.mockReturnValue({
      days: [
        {
          date: '2026-09-17',
          schedule: { date: '2026-09-17', type: 'closed', capacity: 0, is_default: true },
        },
      ],
      isLoading: false,
      error: '',
      summaryError: '',
    });
    render(<DatePickerField value="" onChange={onChange} />);

    await user.click(screen.getByRole('button', { name: /日付を選択/ }));
    await user.click(getDayCell(17));

    expect(onChange).toHaveBeenCalledWith('2026-09-17');
  });

  it('月移動で表示年月が変わり、その月のデータを取得する', async () => {
    const user = userEvent.setup();
    render(<DatePickerField value="" onChange={() => {}} />);

    await user.click(screen.getByRole('button', { name: /日付を選択/ }));
    await user.click(screen.getByRole('button', { name: '次の月' }));

    expect(screen.getByText('2026年10月')).toBeInTheDocument();
    expect(useMonthCalendarDataMock).toHaveBeenLastCalledWith(2026, 10, expect.anything());
  });

  it('外側をクリックするとピッカーが閉じる', async () => {
    const user = userEvent.setup();
    render(
      <div>
        <p>外側</p>
        <DatePickerField value="" onChange={() => {}} />
      </div>,
    );

    await user.click(screen.getByRole('button', { name: /日付を選択/ }));
    expect(screen.getByText('2026年9月')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByText('外側'));

    expect(screen.queryByText('2026年9月')).not.toBeInTheDocument();
  });

  it('取得エラーとサマリーエラーをピッカー内に表示する', async () => {
    const user = userEvent.setup();
    useMonthCalendarDataMock.mockReturnValue({
      days: [],
      isLoading: false,
      error: 'カレンダーデータの取得に失敗しました。',
      summaryError: '予約件数の取得に失敗しました。',
    });
    render(<DatePickerField value="" onChange={() => {}} />);

    await user.click(screen.getByRole('button', { name: /日付を選択/ }));

    expect(screen.getByText('カレンダーデータの取得に失敗しました。')).toBeInTheDocument();
    expect(screen.getByText('予約件数の取得に失敗しました。')).toBeInTheDocument();
  });

  it('error プロパティをフィールドの下に表示する', () => {
    render(<DatePickerField value="" onChange={() => {}} error="来店日を入力してください" />);

    expect(screen.getByText('来店日を入力してください')).toBeInTheDocument();
  });
});
