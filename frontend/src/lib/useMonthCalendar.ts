import { useState, useEffect } from 'react';
import type { Reservation, DailySchedule } from '@/types/reservation';
import { listSchedules, ScheduleApiError } from '@/lib/schedule';
import { listReservations, ReservationApiError } from '@/lib/adminReservation';
import type { DayReservationSummary, MonthCalendarDay } from '@/components/react/MonthCalendar';

interface UseMonthCalendarDataResult {
  days: MonthCalendarDay[];
  isLoading: boolean;
  /** スケジュール取得失敗（カレンダー全体が表示不可） */
  error: string;
  /** 予約サマリー取得失敗（カレンダーは表示、件数・残食数は非表示） */
  summaryError: string;
}

interface UseMonthCalendarDataOptions {
  /** false のとき fetch しない（デフォルト true） */
  enabled?: boolean;
  includeReservationSummary?: boolean;
}

/** 月内の pending / approved 予約から日別サマリーを組み立てる */
export function buildReservationSummaryByDate(
  reservations: Reservation[],
  viewYear: number,
  viewMonth: number,
): Map<string, DayReservationSummary> {
  const monthPrefix = `${viewYear}-${String(viewMonth).padStart(2, '0')}-`;
  const activeStatuses = new Set(['pending', 'approved']);

  return reservations
    .filter((r) => r.visit_date.startsWith(monthPrefix) && activeStatuses.has(r.status))
    .reduce((map, r) => {
      const existing = map.get(r.visit_date) ?? { count: 0, reservedMeals: 0 };
      existing.count += 1;
      if (r.status === 'approved') {
        existing.reservedMeals += r.people;
      }
      map.set(r.visit_date, existing);
      return map;
    }, new Map<string, DayReservationSummary>());
}

async function loadReservationSummary(
  viewYear: number,
  viewMonth: number,
): Promise<{ byDate: Map<string, DayReservationSummary>; error: string }> {
  try {
    const reservations = await listReservations();
    return {
      byDate: buildReservationSummaryByDate(reservations, viewYear, viewMonth),
      error: '',
    };
  } catch (err) {
    return {
      byDate: new Map(),
      error:
        err instanceof ReservationApiError
          ? err.message
          : '予約件数の取得に失敗しました。カレンダーの件数表示は利用できません。',
    };
  }
}

function buildMonthDays(
  schedules: DailySchedule[],
  viewYear: number,
  viewMonth: number,
  reservationByDate: Map<string, DayReservationSummary>,
): MonthCalendarDay[] {
  const daysInMonth = new Date(viewYear, viewMonth, 0).getDate();
  const built: MonthCalendarDay[] = [];

  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${viewYear}-${String(viewMonth).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
    const schedule = schedules.find((s) => s.date === dateStr) ?? {
      date: dateStr,
      type: 'normal' as const,
      capacity: 10,
      is_default: true,
    };
    const reservation = reservationByDate.get(dateStr);

    built.push({
      date: dateStr,
      schedule,
      reservation,
    });
  }

  return built;
}

/** 月間スケジュール＋予約サマリーを取得 */
export function useMonthCalendarData(
  viewYear: number,
  viewMonth: number,
  options?: UseMonthCalendarDataOptions,
): UseMonthCalendarDataResult {
  const enabled = options?.enabled !== false;
  const includeReservationSummary = options?.includeReservationSummary === true;

  const [days, setDays] = useState<MonthCalendarDay[]>([]);
  const [isLoading, setIsLoading] = useState(enabled);
  const [error, setError] = useState('');
  const [summaryError, setSummaryError] = useState('');

  useEffect(() => {
    if (!enabled) {
      setDays([]);
      setIsLoading(false);
      setError('');
      setSummaryError('');
      return;
    }

    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError('');
      setSummaryError('');

      try {
        const schedules = await listSchedules(viewYear, viewMonth);

        let reservationByDate = new Map<string, DayReservationSummary>();
        if (includeReservationSummary) {
          const summary = await loadReservationSummary(viewYear, viewMonth);
          reservationByDate = summary.byDate;
          if (!cancelled && summary.error) {
            setSummaryError(summary.error);
          }
        }

        if (cancelled) return;

        setDays(buildMonthDays(schedules, viewYear, viewMonth, reservationByDate));
      } catch (err) {
        if (!cancelled) {
          setDays([]);
          setError(
            err instanceof ScheduleApiError
              ? err.message
              : 'カレンダーデータの取得に失敗しました。時間をおいて再度お試しください。',
          );
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
  }, [enabled, viewYear, viewMonth, includeReservationSummary]);

  return { days, isLoading, error, summaryError };
}

export type { DailySchedule, MonthCalendarDay, DayReservationSummary };
