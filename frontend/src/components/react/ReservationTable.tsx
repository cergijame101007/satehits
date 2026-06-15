import { useState, useEffect } from 'react';
import type { Reservation, ReservationStatus } from '../../types/reservation';
import { getAvailability, statusLabels, statusColors, statusBadgeBg, statusBadgeText } from '../../mocks/reservation';
import { listReservations, updateReservationStatus } from '@/lib/adminReservation';
import { useMonthCalendarData } from '@/lib/useMonthCalendar';
import MonthCalendar from '@/components/react/MonthCalendar';
import { formatDate, formatDateJa } from '@/lib/calendarUtils';
import { isClosedScheduleType } from '@/lib/calendarTheme';

/** ステータス遷移ルール */
const statusTransitions: Record<ReservationStatus, ReservationStatus[]> = {
  pending: ['approved', 'rejected', 'cancelled'],
  approved: ['no_show', 'cancelled'],
  rejected: [],
  cancelled: [],
  no_show: [],
};

const actionLabels: Record<ReservationStatus, string> = {
  approved: '承認',
  rejected: '拒否',
  cancelled: 'キャンセル',
  no_show: 'No Show',
  pending: '',
};

/** アクションボタンのスタイル（現在のステータスバッジと区別） */
function getActionButtonClass(status: ReservationStatus): string {
  const base = 'px-4 py-1.5 text-sm rounded-lg transition-colors font-medium';
  switch (status) {
    case 'approved':
      return `${base} bg-green-600 text-white hover:bg-green-700`;
    case 'rejected':
      return `${base} bg-red-600 text-white hover:bg-red-700`;
    case 'cancelled':
      return `${base} border border-gray-300 text-gray-600 bg-white hover:bg-gray-50`;
    case 'no_show':
      return `${base} bg-orange-600 text-white hover:bg-orange-700`;
    default:
      return base;
  }
}

export default function ReservationTable() {
  const now = new Date();
  const [selectedDate, setSelectedDate] = useState(formatDate(now));
  const [viewYear, setViewYear] = useState(now.getFullYear());
  const [viewMonth, setViewMonth] = useState(now.getMonth() + 1);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [reservationList, setReservationList] = useState<Reservation[]>([]);
  const [isLoaded, setIsLoaded] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const {
    days: calendarDays,
    isLoading: calendarLoading,
    error: calendarError,
    summaryError: calendarSummaryError,
  } = useMonthCalendarData(viewYear, viewMonth, { includeReservationSummary: true });

  useEffect(() => {
    let cancelled = false;

    async function fetchReservations() {
      setIsLoading(true);
      setLoadError(null);
      try {
        const data = await listReservations(selectedDate, statusFilter || undefined);
        if (!cancelled) {
          setReservationList(data);
          setIsLoaded(true);
        }
      } catch (err) {
        if (!cancelled) {
          setLoadError(err instanceof Error ? err.message : '予約一覧の取得に失敗しました');
          setReservationList([]);
          setIsLoaded(true);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    fetchReservations();
    return () => {
      cancelled = true;
    };
  }, [selectedDate, statusFilter]);

  const handleDateSelect = (dateStr: string) => {
    setSelectedDate(dateStr);
    const [y, m] = dateStr.split('-').map(Number);
    setViewYear(y);
    setViewMonth(m);
  };

  const handleViewChange = (year: number, month: number) => {
    setViewYear(year);
    setViewMonth(month);
  };

  // TODO: GET /api/v1/reservations/availability?date=YYYY-MM-DD に置き換え
  const availability = getAvailability(selectedDate);
  const selectedDaySchedule = calendarDays.find((d) => d.date === selectedDate)?.schedule;

  const handleStatusChange = async (id: string, newStatus: ReservationStatus) => {
    const reservation = reservationList.find((r) => r.id === id);
    if (!reservation) return;

    const actionLabel = actionLabels[newStatus];

    if (newStatus === 'no_show') {
      if (
        !confirm(
          `${reservation.name}さんの予約を「No Show（無断キャンセル）」にしますか？\n\nこの操作は取り消せません。`,
        )
      ) {
        return;
      }
    } else if (!confirm(`${reservation.name}さんの予約を「${actionLabel}」にしますか？`)) {
      return;
    }

    setActionError(null);
    try {
      await updateReservationStatus(id, newStatus);
      const data = await listReservations(selectedDate, statusFilter || undefined);
      setReservationList(data);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'ステータスの更新に失敗しました');
    }
  };

  const isHoliday =
    selectedDaySchedule && isClosedScheduleType(selectedDaySchedule.type);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-medium">予約一覧</h1>
        <a
          href="/admin/reservations/new"
          className="px-4 py-2 bg-primary text-white text-sm rounded-lg hover:bg-primary-dark transition-colors"
        >
          + 予約登録
        </a>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* ミニ月間カレンダー */}
        <div className="lg:col-span-1 rounded-xl border border-gray-200 bg-white p-4">
          {calendarError && (
            <div className="rounded-lg bg-red-50 border border-red-200 p-2 text-red-700 text-xs mb-3">
              {calendarError}
            </div>
          )}
          {calendarSummaryError && (
            <div className="rounded-lg bg-amber-50 border border-amber-200 p-2 text-amber-800 text-xs mb-3">
              {calendarSummaryError}
            </div>
          )}
          <MonthCalendar
            viewYear={viewYear}
            viewMonth={viewMonth}
            onViewChange={handleViewChange}
            days={calendarDays}
            selectedDate={selectedDate}
            onSelectDate={handleDateSelect}
            variant="reservation"
            isLoading={calendarLoading}
            compact
          />
        </div>

        {/* 予約リスト */}
        <div className="lg:col-span-2 space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <p className="text-sm font-medium text-gray-700">{formatDateJa(selectedDate)}</p>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="px-4 py-2 rounded-lg border border-gray-200 bg-white focus:outline-none focus:ring-2 focus:ring-primary/30 focus:border-primary transition-colors text-sm"
            >
              <option value="">すべてのステータス</option>
              <option value="pending">申請中</option>
              <option value="approved">承認済み</option>
              <option value="rejected">拒否</option>
              <option value="cancelled">キャンセル</option>
              <option value="no_show">No Show</option>
            </select>
          </div>

          {loadError && (
            <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-red-700 text-sm">
              {loadError}
            </div>
          )}

          {actionError && (
            <div className="rounded-lg bg-red-50 border border-red-200 p-3 text-red-700 text-sm">
              {actionError}
            </div>
          )}

          {isLoaded && !isLoading && (
            <div className="flex items-center gap-4 text-sm rounded-lg bg-primary/5 border border-primary/10 px-4 py-3">
              {isHoliday || availability.is_holiday ? (
                <span className="text-gray-500">定休日</span>
              ) : (
                <>
                  <span>
                    残り:{' '}
                    <strong className="text-primary text-base">{availability.available}食</strong> /{' '}
                    {availability.capacity}食
                  </span>
                  <span className="text-gray-400">
                    予約済み: {availability.reserved}食
                  </span>
                </>
              )}
            </div>
          )}

          {isLoading && (
            <div className="text-center py-12 text-gray-400">読み込み中...</div>
          )}

          {!isLoading && reservationList.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              {loadError
                ? '予約を取得できませんでした'
                : isHoliday || availability.is_holiday
                  ? 'この日は定休日です'
                  : '予約がありません'}
            </div>
          ) : (
            !isLoading && (
              <div className="space-y-3">
                {reservationList.map((r) => (
                  <div
                    key={r.id}
                    className="rounded-xl border border-gray-200 bg-white p-5 transition-shadow hover:shadow-sm"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className="flex items-center gap-3">
                        <span
                          className="w-3 h-3 rounded-full shrink-0"
                          style={{ backgroundColor: statusColors[r.status] }}
                        />
                        <div>
                          <span className="font-medium">{r.visit_time}</span>
                          <span className="ml-2">
                            {r.name}（{r.people}名）
                          </span>
                        </div>
                      </div>
                      <span
                        className="text-xs px-2.5 py-1 rounded-full font-medium"
                        style={{
                          backgroundColor: statusBadgeBg[r.status],
                          color: statusBadgeText[r.status],
                        }}
                      >
                        {statusLabels[r.status]}
                      </span>
                    </div>

                    <div className="text-sm text-gray-500 space-y-1 ml-6">
                      <p>
                        <span className="inline-block w-4 text-center mr-1">&#x1F4DE;</span>
                        {r.phone}
                      </p>
                      <p>
                        <span className="inline-block w-4 text-center mr-1">&#x2709;</span>
                        {r.email}
                      </p>
                      {r.note && (
                        <p className="text-gray-600 bg-gray-50 rounded-lg p-2 mt-2">
                          備考: {r.note}
                        </p>
                      )}
                    </div>

                    {statusTransitions[r.status].length > 0 && (
                      <div className="flex flex-wrap gap-2 mt-4 ml-6 pt-3 border-t border-gray-100">
                        <span className="text-xs text-gray-400 self-center mr-1">操作:</span>
                        {statusTransitions[r.status].map((nextStatus) => (
                          <button
                            key={nextStatus}
                            type="button"
                            onClick={() => handleStatusChange(r.id, nextStatus)}
                            className={getActionButtonClass(nextStatus)}
                          >
                            {actionLabels[nextStatus]}
                          </button>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )
          )}

          <div className="flex flex-wrap gap-4 text-xs text-gray-500 pt-2">
            {Object.entries(statusLabels).map(([key, label]) => (
              <span key={key} className="flex items-center gap-1.5">
                <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: statusColors[key] }} />
                {label}
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
