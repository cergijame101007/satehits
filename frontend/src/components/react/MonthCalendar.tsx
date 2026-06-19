import type { DailySchedule, ScheduleType } from '@/types/reservation';
import { scheduleTypeShort, scheduleTypeLabels } from '@/mocks/reservation';
import { editableScheduleTypes } from '@/lib/schedule';
import { typeColors, typeTextColors, WEEKDAYS, isClosedScheduleType } from '@/lib/calendarTheme';
import { buildMonthDates, formatDate, shiftMonth } from '@/lib/calendarUtils';
import { cx } from '@/lib/cx';

export interface DayReservationSummary {
  count: number;
  reservedMeals: number;
}

export interface MonthCalendarDay {
  date: string;
  schedule: DailySchedule;
  reservation?: DayReservationSummary;
}

interface MonthCalendarProps {
  viewYear: number;
  viewMonth: number;
  onViewChange: (year: number, month: number) => void;
  days: MonthCalendarDay[];
  selectedDate?: string;
  onSelectDate?: (date: string) => void;
  variant: 'schedule' | 'reservation' | 'picker';
  /** picker のみ有効。true なら休業・満席日を選択不可（顧客向け）。管理者フォームは false */
  restrictSelection?: boolean;
  isLoading?: boolean;
  compact?: boolean;
  showLegend?: boolean;
  className?: string;
}

function ScheduleTypeBadge({ type, compact }: { type: ScheduleType; compact?: boolean }) {
  return (
    <span
      className={`inline-flex items-center justify-center font-medium rounded ${
        compact ? 'min-w-3.5 h-3.5 px-0.5 text-[9px] leading-none' : 'min-w-4 h-4 px-0.5 text-[10px] leading-none'
      }`}
      style={{
        backgroundColor: typeColors[type],
        color: typeTextColors[type],
      }}
    >
      {scheduleTypeShort[type]}
    </span>
  );
}

export default function MonthCalendar({
  viewYear,
  viewMonth,
  onViewChange,
  days,
  selectedDate,
  onSelectDate,
  variant,
  restrictSelection = true,
  isLoading = false,
  compact = false,
  showLegend = true,
  className = '',
}: MonthCalendarProps) {
  const dayMap = new Map(days.map((d) => [d.date, d]));
  const calendarCells = buildMonthDates(viewYear, viewMonth);
  const todayStr = formatDate(new Date());

  const prevMonth = () => {
    const next = shiftMonth(viewYear, viewMonth, -1);
    onViewChange(next.year, next.month);
  };

  const nextMonth = () => {
    const next = shiftMonth(viewYear, viewMonth, 1);
    onViewChange(next.year, next.month);
  };

  const legendTypes =
    variant === 'schedule' || variant === 'reservation'
      ? editableScheduleTypes
      : (['normal', 'morning', 'closed'] as ScheduleType[]);

  return (
    <div className={className}>
      <div className="flex items-center justify-between mb-3">
        <button
          type="button"
          onClick={prevMonth}
          disabled={isLoading}
          className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors disabled:opacity-40"
          aria-label="前の月"
        >
          ‹
        </button>
        <span className={`font-medium ${compact ? 'text-base' : 'text-lg'}`}>
          {viewYear}年{viewMonth}月
        </span>
        <button
          type="button"
          onClick={nextMonth}
          disabled={isLoading}
          className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors disabled:opacity-40"
          aria-label="次の月"
        >
          ›
        </button>
      </div>

      {isLoading ? (
        <div className={`text-center text-gray-400 text-sm ${compact ? 'py-8' : 'py-16'}`}>読み込み中...</div>
      ) : (
        <>
          <div className="grid grid-cols-7 mb-1">
            {WEEKDAYS.map((w, i) => (
              <div
                key={w}
                className={`text-center text-xs font-medium py-1 ${
                  i === 0 ? 'text-red-400' : i === 6 ? 'text-blue-400' : 'text-gray-500'
                }`}
              >
                {w}
              </div>
            ))}
          </div>

          <div className={`grid grid-cols-7 ${compact ? 'gap-0.5' : 'gap-1'}`}>
            {calendarCells.map((dateStr, i) => {
              if (!dateStr) return <div key={`pad-${i}`} />;

              const day = parseInt(dateStr.split('-')[2]);
              const dayData = dayMap.get(dateStr);
              const schedule = dayData?.schedule ?? { date: dateStr, type: 'normal' as ScheduleType, capacity: 10 };
              const isSelected = selectedDate === dateStr;
              const isToday = dateStr === todayStr;
              const isClosed = isClosedScheduleType(schedule.type);
              const isFull =
                !isClosed &&
                !!dayData?.reservation &&
                dayData.reservation.reservedMeals >= schedule.capacity;
              const isDisabled = variant === 'picker' && restrictSelection && (isClosed || isFull);

              const cellClass = cx(
                'flex flex-col items-center rounded-lg transition-colors',
                compact ? 'py-1 min-h-[52px]' : 'py-2',
                isSelected && 'ring-2 ring-primary ring-offset-1',
                isToday && !isDisabled && 'bg-primary/10 hover:bg-primary/15',
                isDisabled && 'text-gray-300 cursor-not-allowed opacity-60',
                !isDisabled && onSelectDate && !isToday && 'cursor-pointer hover:bg-gray-50',
                !isDisabled && onSelectDate && isToday && 'cursor-pointer',
              );

              const cellStyle =
                isClosed && variant === 'picker' ? { backgroundColor: '#F9FAFB' } : undefined;

              return (
                <button
                  key={dateStr}
                  type="button"
                  disabled={isDisabled || !onSelectDate}
                  onClick={() => !isDisabled && onSelectDate?.(dateStr)}
                  className={cellClass}
                  style={cellStyle}
                >
                  <span className={`text-gray-600 ${compact ? 'text-[10px]' : 'text-xs'}`}>{day}</span>

                  {(variant === 'schedule' || variant === 'reservation' || variant === 'picker') && (
                    <ScheduleTypeBadge type={schedule.type} compact={compact} />
                  )}

                  {variant === 'reservation' && dayData?.reservation && !isClosed && (
                    <span className="text-[9px] leading-tight text-gray-500 mt-0.5 text-center">
                      {dayData.reservation.count}件
                      <br />
                      {dayData.reservation.reservedMeals}/{schedule.capacity}食
                    </span>
                  )}

                  {variant === 'picker' && !isClosed && !isDisabled && schedule.capacity > 0 && (
                    <span
                      className={`text-[9px] leading-none mt-0.5 ${
                        isSelected ? 'text-primary font-medium' : 'text-gray-400'
                      }`}
                    >
                      残{Math.max(0, schedule.capacity - (dayData?.reservation?.reservedMeals ?? 0))}
                    </span>
                  )}

                  {variant === 'picker' && isClosed && (
                    <span className="text-[9px] leading-none mt-0.5 text-gray-400">休</span>
                  )}
                </button>
              );
            })}
          </div>

          {showLegend && (
            <div
              className={`flex flex-wrap items-center gap-x-3 gap-y-2 pt-3 mt-3 border-t border-gray-100 text-xs ${
                compact ? 'text-[10px] gap-x-2' : ''
              }`}
            >
              {legendTypes.map((key) => (
                <span key={key} className="inline-flex items-center gap-1">
                  <ScheduleTypeBadge type={key} compact />
                  {scheduleTypeLabels[key]}
                </span>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
