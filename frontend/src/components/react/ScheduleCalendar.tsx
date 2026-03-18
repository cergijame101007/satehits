import { useState, useEffect } from 'react';
import type { DailySchedule, ScheduleType } from '../../types/reservation';
import { getMonthlySchedules, scheduleTypeLabels, scheduleTypeShort } from '../../mocks/reservation';

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

  // 編集フォーム用ステート
  const [editType, setEditType] = useState<ScheduleType>('normal');
  const [editCapacity, setEditCapacity] = useState(10);
  const [editEventName, setEditEventName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  // TODO: GET /api/v1/admin/schedules?month=YYYY-MM に置き換え
  const [schedules, setSchedules] = useState<DailySchedule[]>(() =>
    getMonthlySchedules(viewYear, viewMonth),
  );

  useEffect(() => {
    setSchedules(getMonthlySchedules(viewYear, viewMonth));
    setSelectedSchedule(null);
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
  };

  const handleSave = async () => {
    if (!selectedSchedule) return;
    setIsSaving(true);

    // TODO: PUT /api/v1/admin/capacity/{date} または POST /api/v1/admin/schedules に置き換え
    await new Promise((resolve) => setTimeout(resolve, 300));

    setSchedules((prev) =>
      prev.map((s) =>
        s.date === selectedSchedule.date
          ? {
              ...s,
              type: editType,
              capacity: editCapacity,
              event_name: editType === 'event' ? editEventName : undefined,
              description: editType === 'event' || editType === 'special' ? editDescription : undefined,
            }
          : s,
      ),
    );

    setSelectedSchedule((prev) =>
      prev
        ? {
            ...prev,
            type: editType,
            capacity: editCapacity,
            event_name: editType === 'event' ? editEventName : undefined,
            description: editType === 'event' || editType === 'special' ? editDescription : undefined,
          }
        : null,
    );

    setIsSaving(false);
  };

  const weekdays = ['日', '月', '火', '水', '木', '金', '土'];

  const inputClass =
    'w-full px-4 py-3 rounded-lg border border-gray-200 bg-white focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors text-base';

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-medium">スケジュール設定</h1>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* カレンダー */}
        <div className="lg:col-span-2 rounded-xl border border-gray-200 bg-white p-5">
          {/* ヘッダー */}
          <div className="flex items-center justify-between mb-4">
            <button
              type="button"
              onClick={prevMonth}
              className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors"
            >
              ‹
            </button>
            <span className="text-lg font-medium">
              {viewYear}年{viewMonth}月
            </span>
            <button
              type="button"
              onClick={nextMonth}
              className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-gray-100 transition-colors"
            >
              ›
            </button>
          </div>

          {/* 曜日ヘッダー */}
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

          {/* カレンダーグリッド */}
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
                  style={{ backgroundColor: isSelected ? typeColors[schedule.type] + '80' : undefined }}
                >
                  <span className="text-xs text-gray-600">{day}</span>
                  <span
                    className="text-[11px] font-medium mt-0.5 px-1 rounded"
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

          {/* 凡例 */}
          <div className="flex flex-wrap gap-3 mt-4 pt-4 border-t border-gray-100 text-xs">
            {Object.entries(scheduleTypeLabels).map(([key, label]) => (
              <span key={key} className="flex items-center gap-1">
                <span
                  className="w-4 h-4 rounded text-center text-[10px] leading-4 font-medium"
                  style={{
                    backgroundColor: typeColors[key as ScheduleType],
                    color: typeTextColors[key as ScheduleType],
                  }}
                >
                  {scheduleTypeShort[key as ScheduleType]}
                </span>
                {label}
              </span>
            ))}
          </div>
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
                  {Object.entries(scheduleTypeLabels).map(([key, label]) => (
                    <option key={key} value={key}>
                      {label}
                    </option>
                  ))}
                </select>
              </div>

              {(editType === 'event') && (
                <div>
                  <label className="block text-sm font-medium mb-2">イベント名</label>
                  <input
                    type="text"
                    value={editEventName}
                    onChange={(e) => setEditEventName(e.target.value)}
                    placeholder="和紅茶をしばく会"
                    className={inputClass}
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

              <button
                onClick={handleSave}
                disabled={isSaving}
                className={`w-full py-3 rounded-xl text-white font-medium transition-all ${
                  isSaving
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
