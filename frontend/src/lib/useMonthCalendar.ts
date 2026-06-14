import { useState, useEffect } from 'react';
import type { DailySchedule } from '@/types/reservation';
import { listSchedules, ScheduleApiError } from '@/lib/schedule';
import { listReservations } from '@/lib/adminReservation';
import type { DayReservationSummary, MonthCalendarDay } from '@/components/react/MonthCalendar';
import { isClosedScheduleType } from '@/lib/calendarTheme';

interface UseMonthCalendarDataResult {
  days: MonthCalendarDay[];
  isLoading: boolean;
  error: string;
}

/** 月間スケジュール＋予約サマリーを取得 */
export function useMonthCalendarData(
  viewYear: number,
  viewMonth: number,
  options?: { includeReservationSummary?: boolean },
): UseMonthCalendarDataResult {
  const [days, setDays] = useState<MonthCalendarDay[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setError('');

      try {
        const schedules = await listSchedules(viewYear, viewMonth);

        let reservationByDate = new Map<string, DayReservationSummary>();
        if (options?.includeReservationSummary) {
          const reservations = await listReservations();
          const monthPrefix = `${viewYear}-${String(viewMonth).padStart(2, '0')}-`;
          const activeStatuses = new Set(['pending', 'approved']);

          reservationByDate = reservations
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

        if (cancelled) return;

        const daysInMonth = new Date(viewYear, viewMonth, 0).getDate();
        const built: MonthCalendarDay[] = [];

        for (let d = 1; d <= daysInMonth; d++) {
          const dateStr = `${viewYear}-${String(viewMonth).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
          const schedule = schedules.find((s) => s.date === dateStr) ?? {
            date: dateStr,
            type: 'normal' as const,
            capacity: 10,
          };
          const reservation = reservationByDate.get(dateStr);
          const isClosed = isClosedScheduleType(schedule.type);
          const isFull = !isClosed && reservation && reservation.reservedMeals >= schedule.capacity;

          built.push({
            date: dateStr,
            schedule,
            reservation,
            disabled: isClosed || !!isFull,
          });
        }

        setDays(built);
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
  }, [viewYear, viewMonth, options?.includeReservationSummary]);

  return { days, isLoading, error };
}

export type { DailySchedule, MonthCalendarDay, DayReservationSummary };
