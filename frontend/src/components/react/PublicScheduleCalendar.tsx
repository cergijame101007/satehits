import { useEffect, useMemo, useState } from 'react';
import type { PublicDaySchedule } from '@/types/reservation';
import { buildMonthDatesFixedGrid } from '@/lib/calendarUtils';
import { getJapaneseMonthName } from '@/lib/japaneseMonth';
import { getStoreHoursHeaderContent } from '@/lib/storeHours';
import { getPublicScheduleCellDisplay, shouldShowClosedMark } from '@/lib/publicScheduleCell';
import { listPublicSchedules } from '@/lib/publicSchedule';

const WEEKDAYS = ['日', '月', '火', '水', '木', '金', '土'] as const;

const GRID_CELL_BORDER =
  'border-r border-b border-primary/40 [&:nth-child(7n)]:border-r-0 [&:nth-last-child(-n+7)]:border-b-0';

/** 日付・休・Event バーを縦に収めるセル高さ */
const CELL_MIN_HEIGHT = 'min-h-[4.5rem] sm:min-h-[5.25rem]';

/** Event バー行の高さ（休の位置を Event あり/なしで揃える） */
const EVENT_BAR_ROW = 'h-[1.125rem] shrink-0 sm:h-[1.25rem]';

function EventBar() {
  return (
    <span className="flex h-full w-full items-center justify-center bg-[#6B3A32] text-[9px] tracking-wide text-white sm:text-[10px]">
      Event
    </span>
  );
}

function parseDateLocal(dateStr: string): Date {
  return new Date(`${dateStr}T00:00:00`);
}

function hasMorningCircle(schedule: PublicDaySchedule | undefined): boolean {
  if (!schedule || shouldShowClosedMark(schedule)) return false;
  return schedule.schedule_type === 'morning';
}

interface EventInfoItem {
  date: string;
  dateLabel: string;
  name: string;
  description?: string;
}

function toEventInfoItem(schedule: PublicDaySchedule): EventInfoItem {
  const date = parseDateLocal(schedule.date);
  const weekday = WEEKDAYS[date.getDay()];
  const month = date.getMonth() + 1;
  const day = date.getDate();
  const name = schedule.event_name?.trim() ?? '';

  return {
    date: schedule.date,
    dateLabel: `${month}/${day}(${weekday})…`,
    name,
    description: schedule.event_description?.trim() || undefined,
  };
}

interface CalendarCellProps {
  dateStr: string;
  schedule: PublicDaySchedule | undefined;
}

function CalendarCell({ dateStr, schedule }: CalendarCellProps) {
  const date = parseDateLocal(dateStr);
  const day = date.getDate();
  const isSunday = date.getDay() === 0;
  const morning = hasMorningCircle(schedule);
  const { showClosedMark, showEventBar } = getPublicScheduleCellDisplay(schedule);

  return (
    <div className={`flex ${CELL_MIN_HEIGHT} flex-col ${GRID_CELL_BORDER}`}>
      <div className="flex min-h-0 flex-1 flex-col px-1 pt-1 sm:px-1.5 sm:pt-1.5">
        <span
          className={`shrink-0 self-center text-xs leading-none sm:text-sm ${
            isSunday ? 'text-accent' : 'text-primary'
          } ${morning ? 'flex h-6 w-6 items-center justify-center rounded-full border border-accent sm:h-7 sm:w-7' : ''}`}
        >
          {day}
        </span>

        {showClosedMark && (
          <div className="flex flex-1 items-center justify-center">
            <span className="text-base font-medium leading-none text-accent sm:text-lg">休</span>
          </div>
        )}

        {!showClosedMark && showEventBar && <div className="flex-1" aria-hidden="true" />}
      </div>

      {(showEventBar || showClosedMark) && (
        <div
          className={`${EVENT_BAR_ROW} w-full ${showEventBar ? '' : 'invisible'}`}
          aria-hidden={!showEventBar}
        >
          {showEventBar && <EventBar />}
        </div>
      )}
    </div>
  );
}

export default function PublicScheduleCalendar() {
  const now = new Date();
  const viewYear = now.getFullYear();
  const viewMonth = now.getMonth() + 1;

  const [schedules, setSchedules] = useState<PublicDaySchedule[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError('');

      try {
        const data = await listPublicSchedules(viewYear, viewMonth);
        if (!cancelled) {
          setSchedules(data);
        }
      } catch {
        if (!cancelled) {
          setSchedules([]);
          setError('スケジュールを表示できません。時間をおいて再度ご確認ください。');
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, []);

  const scheduleMap = useMemo(() => new Map(schedules.map((s) => [s.date, s])), [schedules]);

  const calendarCells = useMemo(
    () => buildMonthDatesFixedGrid(viewYear, viewMonth, 6),
    [viewYear, viewMonth],
  );

  const eventItems = useMemo(
    () =>
      schedules
        .filter((s) => s.event_name?.trim())
        .sort((a, b) => a.date.localeCompare(b.date))
        .map(toEventInfoItem),
    [schedules],
  );

  const storeHours = getStoreHoursHeaderContent();
  const monthName = getJapaneseMonthName(viewMonth);

  if (isLoading) {
    return (
      <p className="py-8 text-center text-sm text-primary/70" aria-live="polite">
        読み込み中…
      </p>
    );
  }

  if (error) {
    return (
      <p className="py-6 text-center text-sm text-gray-600" role="alert">
        {error}
      </p>
    );
  }

  return (
    <div
      className="public-schedule-calendar rounded-lg bg-[linear-gradient(145deg,#faf8f4_0%,#f3f0ea_50%,#faf6f2_100%)] px-8 py-5 sm:p-6"
      aria-label={`${viewYear}年${viewMonth}月の営業スケジュール`}
    >
      {/* Header */}
      <div className="mb-0.5 text-primary">
        <p className="text-center text-2xl font-bold tracking-widest sm:text-3xl">{viewYear}</p>

        <div className="flex items-end justify-between gap-2">
          <div className="flex items-baseline gap-3 sm:gap-4">
            <span className="text-4xl font-medium leading-none sm:text-5xl">{viewMonth}</span>
            <span className="text-base sm:text-lg">{monthName}</span>
          </div>

          <div className="flex items-start gap-2 text-[11px] leading-snug sm:text-xs">
            <div className="text-primary">{storeHours.openLabel}</div>
            <div className="text-right">
              {storeHours.hoursLines.map((line) => (
                <p key={line} className="text-primary">
                  {line}
                </p>
              ))}
              <p className="text-accent">{storeHours.closedWeekdaysLabel}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Calendar grid（曜日ヘッダー + 日付を同一グリッドで罫線を統一） */}
      <div className="grid grid-cols-7 border border-primary/40">
        {WEEKDAYS.map((label, index) => (
          <div
            key={label}
            className={`py-1.5 text-center text-xs sm:text-sm ${GRID_CELL_BORDER} ${
              index === 0 ? 'text-accent' : 'text-primary'
            }`}
          >
            {label}
          </div>
        ))}

        {calendarCells.map((dateStr, index) => {
          if (!dateStr) {
            return (
              <div
                key={`pad-${index}`}
                className={`${CELL_MIN_HEIGHT} ${GRID_CELL_BORDER}`}
                aria-hidden="true"
              />
            );
          }

          return (
            <CalendarCell
              key={dateStr}
              dateStr={dateStr}
              schedule={scheduleMap.get(dateStr)}
            />
          );
        })}
      </div>

      {/* Footer legend & events */}
      <div className="mt-2 space-y-2 text-xs sm:text-sm">
        <p className="flex items-center gap-1 text-accent">
          <span>〇…朝営業</span>
        </p>

        {eventItems.length > 0 && (
          <div className="space-y-2 font-bold text-xs sm:text-sm">
            <p className="text-accent">◎ Event Info</p>
            <ul className="text-xs sm:text-sm">
              {eventItems.map((item) => (
                <li key={item.date} className="flex items-start text-accent">
                  <span className="shrink-0">{item.dateLabel}</span>
                  <div className="min-w-0">
                    <p>{item.name}</p>
                    {item.description && (
                      <p className="text-[11px] sm:text-xs">{item.description}</p>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
