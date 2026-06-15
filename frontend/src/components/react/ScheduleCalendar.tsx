import { useState, useEffect } from 'react';
import type { DailySchedule, ScheduleType } from '@/types/reservation';
import { scheduleTypeLabels } from '@/mocks/reservation';
import {
  editableScheduleTypes,
  listSchedules,
  ScheduleApiError,
  setSchedule,
} from '@/lib/schedule';
import MonthCalendar from '@/components/react/MonthCalendar';
import type { MonthCalendarDay } from '@/components/react/MonthCalendar';

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

  const calendarDays: MonthCalendarDay[] = schedules.length
    ? schedules.map((schedule) => ({ date: schedule.date, schedule }))
    : Array.from({ length: new Date(viewYear, viewMonth, 0).getDate() }, (_, i) => {
        const d = i + 1;
        const dateStr = `${viewYear}-${String(viewMonth).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
        return { date: dateStr, schedule: { date: dateStr, type: 'normal' as ScheduleType, capacity: 10 } };
      });

  const handleSelectDay = (dateStr: string) => {
    const schedule =
      schedules.find((s) => s.date === dateStr) ??
      ({ date: dateStr, type: 'normal' as ScheduleType, capacity: 10 } satisfies DailySchedule);
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
        <div className="lg:col-span-2 rounded-xl border border-gray-200 bg-white p-5">
          {loadError && !isLoading ? (
            <div className="text-center py-16 text-gray-400 text-sm">スケジュールを表示できません</div>
          ) : (
            <MonthCalendar
              viewYear={viewYear}
              viewMonth={viewMonth}
              onViewChange={(y, m) => {
                setViewYear(y);
                setViewMonth(m);
              }}
              days={calendarDays}
              selectedDate={selectedSchedule?.date}
              onSelectDate={handleSelectDay}
              variant="schedule"
              isLoading={isLoading}
            />
          )}
        </div>

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
