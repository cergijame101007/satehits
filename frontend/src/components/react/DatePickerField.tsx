import { useState, useEffect, useRef } from 'react';
import MonthCalendar from '@/components/react/MonthCalendar';
import { useMonthCalendarData } from '@/lib/useMonthCalendar';

interface DatePickerFieldProps {
  value: string;
  onChange: (date: string) => void;
  error?: string;
  label?: string;
  required?: boolean;
}

/** スケジュール設定画面と統一したポップアップ日付ピッカー */
export default function DatePickerField({
  value,
  onChange,
  error,
  label = '来店日',
  required = false,
}: DatePickerFieldProps) {
  const now = new Date();
  const [isOpen, setIsOpen] = useState(false);
  const [viewYear, setViewYear] = useState(() =>
    value ? parseInt(value.split('-')[0]) : now.getFullYear(),
  );
  const [viewMonth, setViewMonth] = useState(() =>
    value ? parseInt(value.split('-')[1]) : now.getMonth() + 1,
  );
  const containerRef = useRef<HTMLDivElement>(null);

  const { days, isLoading, error: loadError, summaryError } = useMonthCalendarData(viewYear, viewMonth, {
    includeReservationSummary: true,
  });

  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isOpen]);

  const handleSelect = (date: string) => {
    onChange(date);
    setIsOpen(false);
  };

  const displayValue = value ? value.replace(/-/g, '/') : '日付を選択';

  return (
    <div ref={containerRef} className="relative">
      {label && (
        <label className="block text-sm font-medium mb-2">
          {label} {required && <span className="text-red-500">*</span>}
        </label>
      )}
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className={`w-full px-4 py-3 rounded-lg border text-left flex items-center justify-between transition-colors ${
          error ? 'border-red-400 bg-red-50/50' : 'border-gray-200 bg-white hover:border-primary/40'
        } focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary`}
      >
        <span className={value ? 'text-base' : 'text-gray-400'}>{displayValue}</span>
        <svg className="w-5 h-5 text-gray-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={1.5}
            d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
      </button>

      {isOpen && (
        <div className="absolute z-20 mt-1 w-full min-w-[280px] rounded-xl border border-gray-200 bg-white p-4 shadow-lg">
          {loadError && (
            <div className="rounded-lg bg-red-50 border border-red-200 p-2 text-red-700 text-xs mb-3">
              {loadError}
            </div>
          )}
          {summaryError && (
            <div className="rounded-lg bg-amber-50 border border-amber-200 p-2 text-amber-800 text-xs mb-3">
              {summaryError}
            </div>
          )}
          <MonthCalendar
            viewYear={viewYear}
            viewMonth={viewMonth}
            onViewChange={(y, m) => {
              setViewYear(y);
              setViewMonth(m);
            }}
            days={days}
            selectedDate={value || undefined}
            onSelectDate={handleSelect}
            variant="picker"
            restrictSelection={false}
            isLoading={isLoading}
            compact
            showLegend
          />
        </div>
      )}

      {error && <p className="text-red-500 text-sm mt-1">{error}</p>}
    </div>
  );
}
