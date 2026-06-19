import { useState, useEffect } from 'react';
import type { Reservation, ReservationStatus, AvailabilityResponse } from '../../types/reservation';
import { statusLabels, statusColors, statusBadgeBg, statusBadgeText } from '../../mocks/reservation';
import { getAvailability, toAvailabilityErrorMessage } from '@/lib/availability';
import { listReservations, updateReservationStatus, toReservationErrorMessage } from '@/lib/adminReservation';
import {
  getStatusActionDialogContent,
  statusActionLabels,
  statusToButtonVariant,
  type StatusActionTarget,
} from '@/lib/reservationStatusTheme';
import { useMonthCalendarData } from '@/lib/useMonthCalendar';
import MonthCalendar from '@/components/react/MonthCalendar';
import Alert from '@/components/react/ui/Alert';
import Button from '@/components/react/ui/Button';
import Card from '@/components/react/ui/Card';
import ConfirmModal from '@/components/react/ui/ConfirmModal';
import Textarea from '@/components/react/ui/Textarea';
import { inputClassName } from '@/components/react/ui/inputStyles';
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
  const [availability, setAvailability] = useState<AvailabilityResponse | null>(null);
  const [isAvailabilityLoading, setIsAvailabilityLoading] = useState(false);
  const [availabilityError, setAvailabilityError] = useState<string | null>(null);
  const [statusActionTarget, setStatusActionTarget] = useState<StatusActionTarget | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  const [isStatusSubmitting, setIsStatusSubmitting] = useState(false);

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
          setLoadError(toReservationErrorMessage(err));
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

  useEffect(() => {
    let cancelled = false;

    async function loadAvailability() {
      setIsAvailabilityLoading(true);
      setAvailabilityError(null);
      try {
        const data = await getAvailability(selectedDate);
        if (!cancelled) {
          setAvailability(data);
        }
      } catch (err) {
        if (!cancelled) {
          setAvailabilityError(toAvailabilityErrorMessage(err));
          setAvailability(null);
        }
      } finally {
        if (!cancelled) {
          setIsAvailabilityLoading(false);
        }
      }
    }

    loadAvailability();
    return () => {
      cancelled = true;
    };
  }, [selectedDate]);

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

  const selectedDaySchedule = calendarDays.find((d) => d.date === selectedDate)?.schedule;

  const handleStatusChange = async (id: string, newStatus: ReservationStatus, reason?: string) => {
    setActionError(null);
    try {
      await updateReservationStatus(id, newStatus, reason);
      const [data, avail] = await Promise.all([
        listReservations(selectedDate, statusFilter || undefined),
        getAvailability(selectedDate),
      ]);
      setReservationList(data);
      setAvailability(avail);
    } catch (err) {
      setActionError(toReservationErrorMessage(err));
      throw err;
    }
  };

  const openStatusActionModal = (reservation: Reservation, newStatus: ReservationStatus) => {
    setStatusActionTarget({ reservation, newStatus });
    setRejectReason('');
    setActionError(null);
  };

  const closeStatusActionModal = () => {
    if (isStatusSubmitting) return;
    setStatusActionTarget(null);
    setRejectReason('');
  };

  const submitStatusAction = async () => {
    if (!statusActionTarget) return;
    setIsStatusSubmitting(true);
    setActionError(null);
    try {
      const reason = statusActionTarget.newStatus === 'rejected' ? rejectReason : undefined;
      await handleStatusChange(statusActionTarget.reservation.id, statusActionTarget.newStatus, reason);
      setStatusActionTarget(null);
      setRejectReason('');
    } catch {
      // actionError は handleStatusChange 内で設定済み
    } finally {
      setIsStatusSubmitting(false);
    }
  };

  const isHoliday =
    selectedDaySchedule && isClosedScheduleType(selectedDaySchedule.type);

  const statusActionDialog = statusActionTarget
    ? getStatusActionDialogContent(statusActionTarget)
    : null;
  const isRejectAction = statusActionTarget?.newStatus === 'rejected';

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
        <Card compact className="lg:col-span-1">
          {calendarError && (
            <Alert variant="error" compact className="mb-3">
              {calendarError}
            </Alert>
          )}
          {calendarSummaryError && (
            <Alert variant="warning" compact className="mb-3">
              {calendarSummaryError}
            </Alert>
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
        </Card>

        <div className="lg:col-span-2 space-y-4">
          <div className="flex flex-wrap items-center gap-3">
            <p className="text-sm font-medium text-gray-700">{formatDateJa(selectedDate)}</p>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className={inputClassName(false, 'sm')}
            >
              <option value="">すべてのステータス</option>
              <option value="pending">申請中</option>
              <option value="approved">承認済み</option>
              <option value="rejected">拒否</option>
              <option value="cancelled">キャンセル</option>
              <option value="no_show">No Show</option>
            </select>
          </div>

          {loadError && <Alert variant="error">{loadError}</Alert>}
          {actionError && <Alert variant="error">{actionError}</Alert>}

          {isLoaded && !isLoading && (
            <Card compact className="flex items-center gap-4 text-sm text-gray-700">
              {availabilityError && (
                <span className="text-red-600">{availabilityError}</span>
              )}
              {!availabilityError && isAvailabilityLoading && (
                <span className="text-gray-500">空き状況を読み込み中...</span>
              )}
              {!availabilityError && !isAvailabilityLoading && (isHoliday || availability?.is_holiday) && (
                <span className="text-gray-500">定休日</span>
              )}
              {!availabilityError && !isAvailabilityLoading && availability && !availability.is_holiday && (
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
            </Card>
          )}

          {isLoading && (
            <div className="text-center py-12 text-gray-400">読み込み中...</div>
          )}

          {!isLoading && reservationList.length === 0 ? (
            <div className="text-center py-12 text-gray-400">
              {loadError
                ? '予約を取得できませんでした'
                : isHoliday || availability?.is_holiday
                  ? 'この日は定休日です'
                  : '予約がありません'}
            </div>
          ) : (
            !isLoading && (
              <div className="space-y-3">
                {reservationList.map((r) => (
                  <Card key={r.id} className="transition-shadow hover:shadow-sm">
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
                          <Button
                            key={nextStatus}
                            type="button"
                            variant={statusToButtonVariant(nextStatus)}
                            size="sm"
                            onClick={() => openStatusActionModal(r, nextStatus)}
                          >
                            {statusActionLabels[nextStatus]}
                          </Button>
                        ))}
                      </div>
                    )}
                  </Card>
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

      {statusActionTarget && statusActionDialog && (
        <ConfirmModal
          open
          title={statusActionDialog.title}
          description={statusActionDialog.description}
          confirmLabel={statusActionDialog.confirmLabel}
          confirmVariant={statusActionDialog.confirmVariant}
          isSubmitting={isStatusSubmitting}
          onConfirm={submitStatusAction}
          onCancel={closeStatusActionModal}
        >
          {isRejectAction && (
            <div className="mt-4">
              <label htmlFor="reject-reason" className="block text-sm font-medium text-gray-700 mb-2">
                拒否理由（任意）
              </label>
              <Textarea
                id="reject-reason"
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                rows={4}
                inputSize="sm"
                placeholder="例: 定員超過のため"
              />
              <p className="mt-2 text-xs text-gray-500">
                入力した場合のみ、顧客への拒否メールに理由が記載されます。
              </p>
            </div>
          )}
        </ConfirmModal>
      )}
    </div>
  );
}
