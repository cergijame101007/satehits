import { useState, useEffect } from 'react';
import type { DailySchedule, ScheduleType } from '@/types/reservation';
import { scheduleTypeLabels, scheduleTypeShort } from '@/mocks/reservation';
import {
  editableScheduleTypes,
  listSchedules,
  ScheduleApiError,
  setSchedule,
} from '@/lib/schedule';

/** スケジュールタイプの背景色 */
const typeColors: Record<ScheduleType, string> = {
  normal: '#E0F2FE',
  morning: '#FEF3C7',
  event: '#EDE9FE',
  special: '#FCE7F3',
  closed: '#F3F4F6',
  temporary_closed: '#FEE2E2',
};

const typeTextColors: Record<ScheduleType, string> = {
  normal: '#0369A1',
  morning: '#92400E',
  event: '#6D28D9',
  special: '#BE185D',
  closed: '#6B7280',
  temporary_closed: '#DC2626',
};

export default function ScheduleCalendar() {
  const now = new Date();
  const [viewYear, setViewYear] = useState(now.getFullYear());
  const [viewMonth, setViewMonth] = useState(now.getMonth() + 1);
  const [selectedSchedule, setSelectedSchedule] = useState<DailySchedule | null>(null);

  const [editType, setEditType] = useState<ScheduleType>('normal');
  const [editCapacity, setEditCapacity] = useState(10);
  const [editEventName, setEditEventName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const [saveError, setSaveError] = useState('');

  const [schedules, setSchedules] = useState<DailySchedule[]>([]);

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      setIsLoading(true);
      setLoadError('');
      setSelectedSchedule(null);

      try {
        const data = await listSchedules(viewYear, viewMonth);
        if (!cancelled) {
          setSchedules(data);
        }
      } catch (err) {
        if (!cancelled) {
          setSchedules([]);
          setLoadError(
            err instanceof ScheduleApiError
              ? err.message
              : 'スケジュールの取得に失敗しました。時間をおいて再度お試しください。',
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
  }, [viewYear, viewMonth]);

  const prevMonth = () => {
    if (viewMonth === 1) {
      setViewYear((y) => y - 1);
      setViewMonth(12);
    } else {
      setViewMonth((m) => m - 1);
    }
  };

  const nextMonth = () => {
    if (viewMonth === 12) {
      setViewYear((y) => y + 1);
      setViewMonth(1);
    } else {
      setViewMonth((m) => m + 1);
    }
  };

  const firstDayOfMonth = new Date(viewYear, viewMonth - 1, 1).getDay();
  const daysInMonth = new Date(viewYear, viewMonth, 0).getDate();

  const calendarCells: (DailySchedule | null)[] = [];
  for (let i = 0; i < firstDayOfMonth; i++) calendarCells.push(null);
  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${viewYear}-${String(viewMonth).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
    const schedule = schedules.find((s) => s.date === dateStr);
    calendarCells.push(schedule ?? { date: dateStr, type: 'normal', capacity: 10 });
  }

  const handleSelectDay = (schedule: DailySchedule) => {
    setSelectedSchedule(schedule);
    setEditType(schedule.type);
    setEditCapacity(schedule.capacity);
    setEditEventName(schedule.event_name ?? '');
    setEditDescription(schedule.description ?? '');
    setSaveError('');
  };

  const handleSave = async () => {
    if (!selectedSchedule) return;
    setSaveError('');

    if (editType === 'event' && (!editEventName.trim() || !editDescription.trim())) {
      setSaveError('イベント名と説明は必須です');
      return;
    }

    setIsSaving(true);

    try {
      const saved = await setSchedule(
        selectedSchedule.date,
        editType,
        editCapacity,
        editEventName,
        editDescription,
      );

      setSchedules((prev) => prev.map((s) => (s.date === saved.date ? saved : s)));
      setSelectedSchedule(saved);
      setEditType(saved.type);
      setEditCapacity(saved.capacity);
      setEditEventName(saved.event_name ?? '');
      setEditDescription(saved.description ?? '');
    } catch (err) {
      if (err instanceof ScheduleApiError && err.code === 'VALIDATION_ERROR' && err.details?.length) {
        setSaveError(err.details.map((d) => d.message).join(' '));
      } else {
        setSaveError(
          err instanceof ScheduleApiError
            ? err.message
            : '保存に失敗しました。時間をおいて再度お試しください。',
        );
      }
    } finally {
      setIsSaving(false);
    }
  };

  const weekdays = ['日', '月', '火', '水', '木', '金', '土'];

  const inputClass =
    'w-full px-4 py-3 rounded-lg border border-gray-200 bg-white focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors text-base';

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-medium">スケジュール設定</h1>

      {loadError && (
        <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-red-700 text-sm">
          {loadError}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* カレンダー */}
        <div className="lg:col-span-2 rounded-xl border border-gray-200 bg-white p-5">
          <div className="flex items-center justify-between mb-4">
            <button
              type="button"
              onClick={prevMonth}
              disabled={isLoading}
              className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors disabled:opacity-40"
            >
              ‹
            </button>
            <span className="text-lg font-medium">
              {viewYear}年{viewMonth}月
            </span>
            <button
              type="button"
              onClick={nextMonth}
              disabled={isLoading}
              className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors disabled:opacity-40"
            >
              ›
            </button>
          </div>

          {isLoading ? (
            <div className="text-center py-16 text-gray-400 text-sm">読み込み中...</div>
          ) : loadError ? (
            <div className="text-center py-16 text-gray-400 text-sm">スケジュールを表示できません</div>
          ) : (
            <>
              <div className="grid grid-cols-7 mb-1">
                {weekdays.map((w, i) => (
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

              <div className="grid grid-cols-7 gap-1">
                {calendarCells.map((schedule, i) => {
                  if (!schedule) return <div key={`pad-${i}`} />;

                  const day = parseInt(schedule.date.split('-')[2]);
                  const isSelected = selectedSchedule?.date === schedule.date;

                  return (
                    <button
                      key={schedule.date}
                      type="button"
                      onClick={() => handleSelectDay(schedule)}
                      className={`flex flex-col items-center py-2 rounded-lg text-sm transition-colors cursor-pointer ${
                        isSelected ? 'ring-2 ring-primary ring-offset-1' : 'hover:bg-gray-50'
                      }`}
                      style={{
                        backgroundColor: isSelected ? typeColors[schedule.type] + '80' : undefined,
                      }}
                    >
                      <span className="text-xs text-gray-600">{day}</span>
                      <span
                        className="inline-flex items-center justify-center min-w-4 h-4 px-0.5 text-[10px] leading-none font-medium mt-0.5 rounded"
                        style={{
                          backgroundColor: typeColors[schedule.type],
                          color: typeTextColors[schedule.type],
                        }}
                      >
                        {scheduleTypeShort[schedule.type]}
                      </span>
                    </button>
                  );
                })}
              </div>

              <div className="flex flex-wrap items-center gap-x-3 gap-y-2 mt-4 pt-4 border-t border-gray-100 text-xs">
                {editableScheduleTypes.map((key) => (
                  <span key={key} className="inline-flex items-center gap-1">
                    <span
                      className="inline-flex items-center justify-center w-4 h-4 shrink-0 rounded text-[10px] leading-none font-medium"
                      style={{
                        backgroundColor: typeColors[key],
                        color: typeTextColors[key],
                      }}
                    >
                      {scheduleTypeShort[key]}
                    </span>
                    {scheduleTypeLabels[key]}
                  </span>
                ))}
              </div>
            </>
          )}
        </div>

        {/* 設定フォーム */}
        <div className="rounded-xl border border-gray-200 bg-white p-5">
          {selectedSchedule ? (
            <div className="space-y-4">
              <h3 className="font-medium">
                {selectedSchedule.date.replace(/-/g, '/')} の設定
              </h3>

              <div>
                <label className="block text-sm font-medium mb-2">タイプ</label>
                <select
                  value={editType}
                  onChange={(e) => setEditType(e.target.value as ScheduleType)}
                  className={inputClass}
                >
                  {editableScheduleTypes.map((key) => (
                    <option key={key} value={key}>
                      {scheduleTypeLabels[key]}
                    </option>
                  ))}
                </select>
              </div>

              {editType === 'event' && (
                <div>
                  <label className="block text-sm font-medium mb-2">イベント名</label>
                  <input
                    type="text"
                    value={editEventName}
                    onChange={(e) => setEditEventName(e.target.value)}
                    className={inputClass}
                    required
                  />
                </div>
              )}

              {(editType === 'event' || editType === 'special') && (
                <div>
                  <label className="block text-sm font-medium mb-2">
                    説明（顧客に表示）
                  </label>
                  <textarea
                    value={editDescription}
                    onChange={(e) => setEditDescription(e.target.value)}
                    rows={3}
                    className={inputClass}
                    required={editType === 'event'}
                  />
                </div>
              )}

              <div>
                <label className="block text-sm font-medium mb-2">提供可能数</label>
                <select
                  value={editCapacity}
                  onChange={(e) => setEditCapacity(parseInt(e.target.value))}
                  className={inputClass}
                >
                  {Array.from({ length: 16 }, (_, i) => i).map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
              </div>

              {saveError && (
                <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-red-700 text-sm">
                  {saveError}
                </div>
              )}

              <button
                type="button"
                onClick={handleSave}
                disabled={isSaving || isLoading}
                className={`w-full py-3 rounded-xl text-white font-medium transition-all ${
                  isSaving || isLoading
                    ? 'bg-gray-400 cursor-not-allowed'
                    : 'bg-primary hover:bg-primary-dark active:scale-[0.98] shadow-lg'
                }`}
              >
                {isSaving ? '保存中...' : '保存する'}
              </button>
            </div>
          ) : (
            <div className="text-center py-12 text-gray-400 text-sm">
              カレンダーから日付を選択してください
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
