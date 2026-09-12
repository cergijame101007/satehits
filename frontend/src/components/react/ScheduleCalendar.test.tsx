import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ScheduleCalendar from '@/components/react/ScheduleCalendar';
import { deleteSchedule, listSchedules, ScheduleApiError, setSchedule } from '@/lib/schedule';
import type { DailySchedule } from '@/types/reservation';

vi.mock('@/lib/schedule', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/schedule')>();
  return { ...actual, listSchedules: vi.fn(), setSchedule: vi.fn(), deleteSchedule: vi.fn() };
});

const listSchedulesMock = vi.mocked(listSchedules);
const setScheduleMock = vi.mocked(setSchedule);
const deleteScheduleMock = vi.mocked(deleteSchedule);

function schedule(date: string, overrides: Partial<DailySchedule> = {}): DailySchedule {
  return { date, type: 'normal', capacity: 10, is_default: true, ...overrides };
}

/** 2026 年 9 月: 15 日は通常（定例）、16 日はイベント（個別設定）、17 日は定休（定例） */
const septemberSchedules: DailySchedule[] = [
  schedule('2026-09-15'),
  schedule('2026-09-16', {
    type: 'event',
    capacity: 8,
    event_name: '羊の夜会',
    description: 'ラム尽くしのコース',
    is_default: false,
  }),
  schedule('2026-09-17', { type: 'closed', capacity: 0 }),
];

function getDayCell(dayNumber: number): HTMLButtonElement {
  const cell = screen
    .getAllByRole('button')
    .find((b) => b.querySelector('span')?.textContent === String(dayNumber));
  if (!cell) throw new Error(`${dayNumber} 日のセルが見つかりません`);
  return cell as HTMLButtonElement;
}

async function renderLoaded() {
  render(<ScheduleCalendar />);
  await waitFor(() => {
    expect(screen.queryByText('読み込み中...')).not.toBeInTheDocument();
  });
}

describe('ScheduleCalendar', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-09-10T10:00:00'));
    listSchedulesMock.mockResolvedValue(septemberSchedules);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it('今月のスケジュールを取得してカレンダーに種別を表示する', async () => {
    await renderLoaded();

    expect(listSchedulesMock).toHaveBeenCalledWith(2026, 9);
    expect(screen.getByText('2026年9月')).toBeInTheDocument();
    expect(within(getDayCell(16)).getByText('イ')).toBeInTheDocument();
    expect(within(getDayCell(17)).getByText('休')).toBeInTheDocument();
    expect(screen.getByText('カレンダーから日付を選択してください')).toBeInTheDocument();
  });

  it('取得に失敗したらエラーを表示し、カレンダーを出さない', async () => {
    listSchedulesMock.mockRejectedValue(
      new ScheduleApiError({ code: 'INTERNAL_ERROR', message: 'サーバー内部でエラーが発生しました' }),
    );
    await renderLoaded();

    expect(screen.getByRole('alert')).toHaveTextContent('サーバー内部でエラーが発生しました');
    expect(screen.getByText('スケジュールを表示できません')).toBeInTheDocument();
    expect(screen.queryByText('2026年9月')).not.toBeInTheDocument();
  });

  it('月移動で翌月を取得し、選択中の日付をクリアする', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(getDayCell(15));
    expect(screen.getByText('2026/09/15 の設定')).toBeInTheDocument();

    listSchedulesMock.mockResolvedValue([]);
    await user.click(screen.getByRole('button', { name: '次の月' }));

    await waitFor(() => {
      expect(listSchedulesMock).toHaveBeenCalledWith(2026, 10);
    });
    expect(await screen.findByText('2026年10月')).toBeInTheDocument();
    expect(screen.getByText('カレンダーから日付を選択してください')).toBeInTheDocument();
  });

  it('日付を選ぶと設定パネルに現在の値を表示し、定例日には「定例に戻す」を出さない', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(getDayCell(15));

    expect(screen.getByText('2026/09/15 の設定')).toBeInTheDocument();
    expect(screen.getByDisplayValue('通常')).toBeInTheDocument();
    expect(screen.getByDisplayValue('10')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存する' })).toBeEnabled();
    expect(screen.queryByRole('button', { name: '定例に戻す' })).not.toBeInTheDocument();
  });

  it('個別設定のある日はイベント名・説明を表示し、「定例に戻す」を出す', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(getDayCell(16));

    expect(screen.getByDisplayValue('イベント')).toBeInTheDocument();
    expect(screen.getByDisplayValue('羊の夜会')).toBeInTheDocument();
    expect(screen.getByDisplayValue('ラム尽くしのコース')).toBeInTheDocument();
    expect(screen.getByDisplayValue('8')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '定例に戻す' })).toBeInTheDocument();
  });

  it('イベントでイベント名が空なら保存せずエラーを表示する', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(getDayCell(15));
    await user.selectOptions(screen.getByDisplayValue('通常'), 'event');
    await user.click(screen.getByRole('button', { name: '保存する' }));

    expect(screen.getByRole('alert')).toHaveTextContent('イベント名は必須です');
    expect(setScheduleMock).not.toHaveBeenCalled();
  });

  it('保存すると API に日付・種別・提供数・イベント情報を渡し、結果をパネルとカレンダーに反映する', async () => {
    const user = userEvent.setup();
    setScheduleMock.mockResolvedValue(
      schedule('2026-09-15', {
        type: 'event',
        capacity: 6,
        event_name: '限定ランチ',
        description: '説明文',
        is_default: false,
      }),
    );
    await renderLoaded();

    await user.click(getDayCell(15));
    await user.selectOptions(screen.getByDisplayValue('通常'), 'event');
    await user.type(document.querySelector<HTMLInputElement>('input[type="text"]')!, '限定ランチ');
    await user.type(document.querySelector('textarea')!, '説明文');
    await user.selectOptions(screen.getByDisplayValue('10'), '6');
    await user.click(screen.getByRole('button', { name: '保存する' }));

    await waitFor(() => {
      expect(setScheduleMock).toHaveBeenCalledWith('2026-09-15', 'event', 6, '限定ランチ', '説明文');
    });
    expect(await screen.findByRole('button', { name: '定例に戻す' })).toBeInTheDocument();
    expect(within(getDayCell(15)).getByText('イ')).toBeInTheDocument();
  });

  it('外部イベントは提供数の入力を隠し、0 で保存する', async () => {
    const user = userEvent.setup();
    setScheduleMock.mockResolvedValue(
      schedule('2026-09-15', { type: 'external_event', capacity: 0, event_name: 'マルシェ出店', is_default: false }),
    );
    await renderLoaded();

    await user.click(getDayCell(15));
    await user.selectOptions(screen.getByDisplayValue('通常'), 'external_event');

    expect(screen.queryByDisplayValue('10')).not.toBeInTheDocument();
    await user.type(document.querySelector<HTMLInputElement>('input[type="text"]')!, 'マルシェ出店');
    await user.click(screen.getByRole('button', { name: '保存する' }));

    await waitFor(() => {
      expect(setScheduleMock).toHaveBeenCalledWith('2026-09-15', 'external_event', 0, 'マルシェ出店', '');
    });
  });

  it('保存の VALIDATION_ERROR は details のメッセージを連結して表示する', async () => {
    const user = userEvent.setup();
    setScheduleMock.mockRejectedValue(
      new ScheduleApiError({
        code: 'VALIDATION_ERROR',
        message: '入力内容に誤りがあります',
        details: [
          { field: 'capacity', message: '提供数は0以上です' },
          { field: 'event_name', message: 'イベント名が長すぎます' },
        ],
      }),
    );
    await renderLoaded();

    await user.click(getDayCell(15));
    await user.click(screen.getByRole('button', { name: '保存する' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('提供数は0以上です イベント名が長すぎます');
  });

  it('保存のその他のエラーはメッセージを表示する', async () => {
    const user = userEvent.setup();
    setScheduleMock.mockRejectedValueOnce(new TypeError('Failed to fetch'));
    await renderLoaded();

    await user.click(getDayCell(15));
    await user.click(screen.getByRole('button', { name: '保存する' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      '保存に失敗しました。時間をおいて再度お試しください。',
    );
  });

  it('「定例に戻す」は戻り先の定例をプレビューし、確認後に削除して再取得する', async () => {
    const user = userEvent.setup();
    deleteScheduleMock.mockResolvedValue(undefined);
    await renderLoaded();

    await user.click(getDayCell(16));
    await user.click(screen.getByRole('button', { name: '定例に戻す' }));

    const dialog = screen.getByRole('dialog', { name: '店舗定例に戻す' });
    // 2026-09-16 は水曜 → 通常営業 10 食
    expect(within(dialog).getByText('タイプ: 通常')).toBeInTheDocument();
    expect(within(dialog).getByText('提供可能数: 10食')).toBeInTheDocument();

    listSchedulesMock.mockResolvedValue([schedule('2026-09-15'), schedule('2026-09-16'), schedule('2026-09-17', { type: 'closed', capacity: 0 })]);
    await user.click(within(dialog).getByRole('button', { name: '定例に戻す' }));

    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
    expect(deleteScheduleMock).toHaveBeenCalledWith('2026-09-16');
    expect(listSchedulesMock).toHaveBeenCalledTimes(2);
    expect(screen.getByDisplayValue('通常')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '定例に戻す' })).not.toBeInTheDocument();
  });

  it('定休日に戻る場合はその旨をプレビューに表示する', async () => {
    const user = userEvent.setup();
    listSchedulesMock.mockResolvedValue([
      schedule('2026-09-18', { type: 'event', capacity: 5, event_name: '金曜特別営業', is_default: false }),
    ]);
    await renderLoaded();

    await user.click(getDayCell(18));
    await user.click(screen.getByRole('button', { name: '定例に戻す' }));

    const dialog = screen.getByRole('dialog', { name: '店舗定例に戻す' });
    expect(within(dialog).getByText('タイプ: 定休日')).toBeInTheDocument();
    expect(within(dialog).getByText('提供可能数: 予約不可')).toBeInTheDocument();
    expect(within(dialog).getByText('この日は定休日として扱われます')).toBeInTheDocument();
  });

  it('削除の NOT_FOUND は「すでに店舗定例です」と表示する', async () => {
    const user = userEvent.setup();
    deleteScheduleMock.mockRejectedValue(new ScheduleApiError({ code: 'NOT_FOUND', message: 'not found' }));
    await renderLoaded();

    await user.click(getDayCell(16));
    await user.click(screen.getByRole('button', { name: '定例に戻す' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: '定例に戻す' }));

    expect(await screen.findByText('すでに店舗定例です')).toBeInTheDocument();
    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('確認モーダルをキャンセルすると削除しない', async () => {
    const user = userEvent.setup();
    await renderLoaded();

    await user.click(getDayCell(16));
    await user.click(screen.getByRole('button', { name: '定例に戻す' }));
    await user.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'キャンセル' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(deleteScheduleMock).not.toHaveBeenCalled();
  });
});
