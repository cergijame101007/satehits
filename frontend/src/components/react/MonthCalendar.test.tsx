import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import MonthCalendar, { type MonthCalendarDay } from '@/components/react/MonthCalendar';
import type { DailySchedule } from '@/types/reservation';

function day(
  date: string,
  schedule: Partial<DailySchedule> = {},
  reservation?: MonthCalendarDay['reservation'],
): MonthCalendarDay {
  return {
    date,
    schedule: { date, type: 'normal', capacity: 10, is_default: true, ...schedule },
    reservation,
  };
}

/** 日付セル（先頭の span がその日の数字）を取得する */
function getDayCell(dayNumber: number): HTMLButtonElement {
  const cell = screen
    .getAllByRole('button')
    .find((b) => b.querySelector('span')?.textContent === String(dayNumber));
  if (!cell) throw new Error(`${dayNumber} 日のセルが見つかりません`);
  return cell as HTMLButtonElement;
}

const days: MonthCalendarDay[] = [
  day('2026-09-15'),
  day('2026-09-16', { type: 'closed', capacity: 0 }),
  day('2026-09-17', { type: 'event', capacity: 8, event_name: '夜会', is_default: false }),
  day('2026-09-18', {}, { count: 2, reservedMeals: 10 }),
  day('2026-09-19', {}, { count: 1, reservedMeals: 4 }),
];

describe('MonthCalendar', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('表示中の年月と曜日ヘッダーを表示し、当月の日数分のセルを描画する', () => {
    render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={days} variant="schedule" />,
    );

    expect(screen.getByText('2026年9月')).toBeInTheDocument();
    expect(screen.getByText('日')).toBeInTheDocument();
    expect(screen.getByText('土')).toBeInTheDocument();
    // 前月・次月ボタン + 30 日分
    expect(screen.getAllByRole('button')).toHaveLength(32);
  });

  it('前の月・次の月ボタンで onViewChange に移動先の年月を渡す（年またぎ含む）', async () => {
    const user = userEvent.setup();
    const onViewChange = vi.fn();
    render(
      <MonthCalendar viewYear={2026} viewMonth={1} onViewChange={onViewChange} days={[]} variant="schedule" />,
    );

    await user.click(screen.getByRole('button', { name: '前の月' }));
    expect(onViewChange).toHaveBeenLastCalledWith(2025, 12);

    await user.click(screen.getByRole('button', { name: '次の月' }));
    expect(onViewChange).toHaveBeenLastCalledWith(2026, 2);
  });

  it('isLoading 中は読み込み中表示になり、月移動ボタンが無効になる', () => {
    render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={days} variant="schedule" isLoading />,
    );

    expect(screen.getByText('読み込み中...')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '前の月' })).toBeDisabled();
    expect(screen.getByRole('button', { name: '次の月' })).toBeDisabled();
    expect(screen.getAllByRole('button')).toHaveLength(2);
  });

  it('日付セルのクリックで onSelectDate に YYYY-MM-DD を渡す', async () => {
    const user = userEvent.setup();
    const onSelectDate = vi.fn();
    render(
      <MonthCalendar
        viewYear={2026}
        viewMonth={9}
        onViewChange={() => {}}
        days={days}
        variant="schedule"
        onSelectDate={onSelectDate}
      />,
    );

    await user.click(getDayCell(15));

    expect(onSelectDate).toHaveBeenCalledWith('2026-09-15');
  });

  it('onSelectDate がなければ日付セルは押せない', () => {
    render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={days} variant="schedule" />,
    );

    expect(getDayCell(15)).toBeDisabled();
  });

  it('各日のスケジュール種別を短縮ラベルで表示する', () => {
    render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={days} variant="schedule" />,
    );

    expect(within(getDayCell(15)).getByText('通')).toBeInTheDocument();
    expect(within(getDayCell(16)).getByText('休')).toBeInTheDocument();
    expect(within(getDayCell(17)).getByText('イ')).toBeInTheDocument();
  });

  it('days にない日は通常営業として扱う', () => {
    render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={days} variant="schedule" />,
    );

    expect(within(getDayCell(1)).getByText('通')).toBeInTheDocument();
  });

  it('selectedDate のセルに選択スタイルが付く', () => {
    render(
      <MonthCalendar
        viewYear={2026}
        viewMonth={9}
        onViewChange={() => {}}
        days={days}
        variant="schedule"
        selectedDate="2026-09-15"
      />,
    );

    expect(getDayCell(15)).toHaveClass('ring-2');
    expect(getDayCell(14)).not.toHaveClass('ring-2');
  });

  it('reservation variant では件数と予約済み/提供数を表示し、休業日には表示しない', () => {
    render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={days} variant="reservation" />,
    );

    expect(getDayCell(19)).toHaveTextContent('1件');
    expect(getDayCell(19)).toHaveTextContent('4/10食');
    expect(getDayCell(16)).not.toHaveTextContent('件');
  });

  it('picker（restrictSelection）では休業日・満席日を選択不可にし、残数を表示する', async () => {
    const user = userEvent.setup();
    const onSelectDate = vi.fn();
    render(
      <MonthCalendar
        viewYear={2026}
        viewMonth={9}
        onViewChange={() => {}}
        days={days}
        variant="picker"
        onSelectDate={onSelectDate}
      />,
    );

    const closed = getDayCell(16);
    expect(closed).toBeDisabled();
    // 種別バッジの「休」に加え、picker 用の「休」ラベルが付く
    expect(within(closed).getAllByText('休')).toHaveLength(2);

    const full = getDayCell(18);
    expect(full).toBeDisabled();

    const open = getDayCell(19);
    expect(open).toBeEnabled();
    expect(within(open).getByText('残6')).toBeInTheDocument();

    await user.click(closed);
    await user.click(open);
    expect(onSelectDate).toHaveBeenCalledTimes(1);
    expect(onSelectDate).toHaveBeenCalledWith('2026-09-19');
  });

  it('picker でも restrictSelection=false なら休業日・満席日を選択できる', async () => {
    const user = userEvent.setup();
    const onSelectDate = vi.fn();
    render(
      <MonthCalendar
        viewYear={2026}
        viewMonth={9}
        onViewChange={() => {}}
        days={days}
        variant="picker"
        restrictSelection={false}
        onSelectDate={onSelectDate}
      />,
    );

    expect(getDayCell(16)).toBeEnabled();
    expect(getDayCell(18)).toBeEnabled();
    await user.click(getDayCell(16));

    expect(onSelectDate).toHaveBeenCalledWith('2026-09-16');
  });

  it('凡例は schedule では編集可能な全種別、picker では通常・朝営業・定休日のみ', () => {
    const { rerender } = render(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={[]} variant="schedule" />,
    );

    expect(screen.getByText('イベント')).toBeInTheDocument();
    expect(screen.getByText('外部イベント（店休）')).toBeInTheDocument();
    expect(screen.getByText('特別メニュー')).toBeInTheDocument();

    rerender(
      <MonthCalendar viewYear={2026} viewMonth={9} onViewChange={() => {}} days={[]} variant="picker" />,
    );

    expect(screen.getByText('通常')).toBeInTheDocument();
    expect(screen.getByText('朝営業')).toBeInTheDocument();
    expect(screen.getByText('定休日')).toBeInTheDocument();
    expect(screen.queryByText('イベント')).not.toBeInTheDocument();
  });

  it('showLegend=false なら凡例を表示しない', () => {
    render(
      <MonthCalendar
        viewYear={2026}
        viewMonth={9}
        onViewChange={() => {}}
        days={[]}
        variant="schedule"
        showLegend={false}
      />,
    );

    expect(screen.queryByText('特別メニュー')).not.toBeInTheDocument();
  });
});
