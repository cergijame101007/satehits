import { useState, useEffect, useMemo, useRef } from 'react';
import type { Reservation, ReservationStatus, AvailabilityResponse } from '../../types/reservation';
import { statusLabels, statusColors, statusBadgeBg, statusBadgeText } from '@/lib/reservationStatusTheme';
import { getAvailability, toAvailabilityErrorMessage } from '@/lib/availability';
import { listReservations, updateReservationStatus, toReservationErrorMessage } from '@/lib/adminReservation';
import { notifyPendingCountChanged } from '@/lib/pendingReservations';
import {
  getStatusActionDialogContent,
  statusActionLabels,
  statusToButtonVariant,
  type StatusActionTarget,
} from '@/lib/reservationStatusTheme';
import { buildReservationSummaryByDate, useMonthCalendarData } from '@/lib/useMonthCalendar';
import MonthCalendar from '@/components/react/MonthCalendar';
import Alert from '@/components/react/ui/Alert';
import Button from '@/components/react/ui/Button';
import ButtonLink from '@/components/react/ui/ButtonLink';
import Card from '@/components/react/ui/Card';
import ConfirmModal from '@/components/react/ui/ConfirmModal';
import Textarea from '@/components/react/ui/Textarea';
import { inputClassName } from '@/lib/ui/inputStyles';
import { cx } from '@/lib/cx';
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

const statusFilterOptions = Object.entries(statusLabels) as [ReservationStatus, string][];

function isValidStatusFilter(value: string | null): value is ReservationStatus {
  return statusFilterOptions.some(([key]) => key === value);
}

/** URL クエリから初期表示状態を読む。`range` は all のみ、`status` は既存の選択肢のみ有効 */
function readListViewParams(search: string): { isAllRange: boolean; statusFilter: string } {
  const params = new URLSearchParams(search);
  const status = params.get('status');
  return {
    isAllRange: params.get('range') === 'all',
    statusFilter: isValidStatusFilter(status) ? status : '',
  };
}

interface DateGroup {
  date: string;
  reservations: Reservation[];
}

/** 来店日順に並んだ予約を、日付ごとのグループに分ける */
function groupByDate(reservations: Reservation[]): DateGroup[] {
  return reservations.reduce<DateGroup[]>((groups, r) => {
    const last = groups[groups.length - 1];
    if (last && last.date === r.visit_date) {
      last.reservations.push(r);
    } else {
      groups.push({ date: r.visit_date, reservations: [r] });
    }
    return groups;
  }, []);
}

interface ReservationCardProps {
  reservation: Reservation;
  onStatusAction: (reservation: Reservation, newStatus: ReservationStatus) => void;
}

function ReservationCard({ reservation: r, onStatusAction }: ReservationCardProps) {
  return (
    <Card className="transition-shadow hover:shadow-sm">
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
              onClick={() => onStatusAction(r, nextStatus)}
            >
              {statusActionLabels[nextStatus]}
            </Button>
          ))}
        </div>
      )}
    </Card>
  );
}

interface DateGroupSectionProps {
  group: DateGroup;
  onSelectDate: (date: string) => void;
  onStatusAction: (reservation: Reservation, newStatus: ReservationStatus) => void;
}

/** 全期間モードの日付見出し + その日の予約カード。見出しを押すと選択日モードに戻る */
function DateGroupSection({ group, onSelectDate, onStatusAction }: DateGroupSectionProps) {
  return (
    <section className="space-y-3">
      <button
        type="button"
        onClick={() => onSelectDate(group.date)}
        className="flex items-baseline gap-2 text-sm font-medium text-gray-700 hover:text-primary transition-colors"
      >
        <span>{formatDateJa(group.date)}</span>
        <span className="text-xs text-gray-400">{group.reservations.length}件</span>
      </button>
      {group.reservations.map((r) => (
        <ReservationCard key={r.id} reservation={r} onStatusAction={onStatusAction} />
      ))}
    </section>
  );
}

export default function ReservationTable() {
  const now = new Date();
  const todayStr = formatDate(now);
  const initialView = useMemo(
    () => readListViewParams(typeof window === 'undefined' ? '' : window.location.search),
    [],
  );
  const [selectedDate, setSelectedDate] = useState(todayStr);
  const [viewYear, setViewYear] = useState(now.getFullYear());
  const [viewMonth, setViewMonth] = useState(now.getMonth() + 1);
  const [statusFilter, setStatusFilter] = useState<string>(initialView.statusFilter);
  const [isAllRange, setIsAllRange] = useState(initialView.isAllRange);
  const [monthReservations, setMonthReservations] = useState<Reservation[]>([]);
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
  const emptyHeadingRef = useRef<HTMLParagraphElement>(null);
  const hadAllRangeItemsRef = useRef(false);

  const {
    days: scheduleDays,
    isLoading: calendarLoading,
    error: calendarError,
  } = useMonthCalendarData(viewYear, viewMonth);

  const calendarDays = useMemo(() => {
    const summaryByDate = buildReservationSummaryByDate(monthReservations, viewYear, viewMonth);
    return scheduleDays.map((day) => ({
      ...day,
      reservation: summaryByDate.get(day.date),
    }));
  }, [scheduleDays, monthReservations, viewYear, viewMonth]);

  const statusFiltered = useMemo(
    () => (statusFilter ? monthReservations.filter((r) => r.status === statusFilter) : monthReservations),
    [monthReservations, statusFilter],
  );

  const reservationList = useMemo(
    () =>
      statusFiltered
        .filter((r) => r.visit_date === selectedDate)
        .sort((a, b) => a.visit_time.localeCompare(b.visit_time)),
    [statusFiltered, selectedDate],
  );

  /** 全期間モードの一覧。来店日昇順で、今日以降と過去に分ける */
  const allRangeList = useMemo(() => {
    const sorted = [...statusFiltered].sort(
      (a, b) => a.visit_date.localeCompare(b.visit_date) || a.visit_time.localeCompare(b.visit_time),
    );
    return {
      upcoming: groupByDate(sorted.filter((r) => r.visit_date >= todayStr)),
      past: groupByDate(sorted.filter((r) => r.visit_date < todayStr)),
      total: sorted.length,
      pastCount: sorted.filter((r) => r.visit_date < todayStr).length,
    };
  }, [statusFiltered, todayStr]);

  useEffect(() => {
    let cancelled = false;

    async function fetchMonthReservations() {
      setIsLoading(true);
      setLoadError(null);
      try {
        const data = await listReservations();
        if (!cancelled) {
          setMonthReservations(data);
          setIsLoaded(true);
        }
      } catch (err) {
        if (!cancelled) {
          setLoadError(toReservationErrorMessage(err));
          setMonthReservations([]);
          setIsLoaded(true);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    fetchMonthReservations();
    return () => {
      cancelled = true;
    };
  }, [viewYear, viewMonth]);

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

  /* 表示状態を URL に反映する（履歴は増やさない。既定状態ならクエリを消す） */
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (isAllRange) {
      params.set('range', 'all');
    } else {
      params.delete('range');
    }
    if (statusFilter) {
      params.set('status', statusFilter);
    } else {
      params.delete('status');
    }
    const query = params.toString();
    window.history.replaceState(
      null,
      '',
      query ? `${window.location.pathname}?${query}` : window.location.pathname,
    );
  }, [isAllRange, statusFilter]);

  /* 全期間モードで最後の 1 件を処理して空になったら、空表示にフォーカスを移す */
  useEffect(() => {
    if (!isAllRange) {
      hadAllRangeItemsRef.current = false;
      return;
    }
    if (allRangeList.total > 0) {
      hadAllRangeItemsRef.current = true;
      return;
    }
    if (hadAllRangeItemsRef.current) {
      hadAllRangeItemsRef.current = false;
      emptyHeadingRef.current?.focus();
    }
  }, [isAllRange, allRangeList.total]);

  const handleDateSelect = (dateStr: string) => {
    // 日付を選んだら全期間モードから抜ける
    setIsAllRange(false);
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
      // 一覧の再取得に失敗してもナビのバッジは更新させる
      notifyPendingCountChanged();
      const [data, avail] = await Promise.all([listReservations(), getAvailability(selectedDate)]);
      setMonthReservations(data);
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
    selectedDaySchedule && isClosedScheduleType(selectedDaySchedule.schedule_type);

  const statusActionDialog = statusActionTarget
    ? getStatusActionDialogContent(statusActionTarget)
    : null;
  const isRejectAction = statusActionTarget?.newStatus === 'rejected';
  // 空表示は取得に成功したときだけ出す（読み込み中・失敗時は出さない）
  const showAllRangeEmpty = isAllRange && isLoaded && !isLoading && !loadError && allRangeList.total === 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-medium">予約一覧</h1>
        <ButtonLink href="/admin/reservations/new" variant="primary" size="md">
          + 予約登録
        </ButtonLink>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        <Card compact className="lg:col-span-1">
          {calendarError && (
            <Alert variant="error" compact className="mb-3">
              {calendarError}
            </Alert>
          )}
          <MonthCalendar
            viewYear={viewYear}
            viewMonth={viewMonth}
            onViewChange={handleViewChange}
            days={calendarDays}
            selectedDate={isAllRange ? undefined : selectedDate}
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
              {statusFilterOptions.map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>

            <div className="flex items-center gap-2">
              <label htmlFor="range-all-toggle" className="text-sm text-gray-700">
                全期間
              </label>
              <button
                id="range-all-toggle"
                type="button"
                role="switch"
                aria-checked={isAllRange}
                onClick={() => setIsAllRange((prev) => !prev)}
                className={cx(
                  'relative inline-flex items-center w-11 h-6 rounded-full transition-colors',
                  isAllRange ? 'bg-primary' : 'bg-gray-300',
                )}
              >
                <span
                  className={cx(
                    'inline-block w-5 h-5 rounded-full bg-white shadow transition-transform',
                    isAllRange ? 'translate-x-[22px]' : 'translate-x-0.5',
                  )}
                />
              </button>
            </div>
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

          {!isLoading && isAllRange && (
            <>
              {loadError && (
                <div className="text-center py-12 text-gray-400">予約を取得できませんでした</div>
              )}

              {showAllRangeEmpty && (
                <div className="text-center py-12 space-y-4">
                  <p ref={emptyHeadingRef} tabIndex={-1} className="text-gray-400">
                    {statusFilter === 'pending' ? '未対応の予約はありません' : '該当する予約はありません'}
                  </p>
                  <Button type="button" variant="ghost" size="sm" onClick={() => setIsAllRange(false)}>
                    選択日の表示に戻る
                  </Button>
                </div>
              )}

              {!loadError && allRangeList.total > 0 && (
                <div className="space-y-6">
                  {allRangeList.upcoming.map((group) => (
                    <DateGroupSection
                      key={group.date}
                      group={group}
                      onSelectDate={handleDateSelect}
                      onStatusAction={openStatusActionModal}
                    />
                  ))}

                  {allRangeList.past.length > 0 && (
                    <details className="rounded-xl border border-gray-200 bg-white">
                      <summary className="cursor-pointer px-4 py-3 text-sm font-medium text-gray-700">
                        過去の予約（{allRangeList.pastCount}件）
                      </summary>
                      <div className="px-4 pb-4 space-y-6">
                        {allRangeList.past.map((group) => (
                          <DateGroupSection
                            key={group.date}
                            group={group}
                            onSelectDate={handleDateSelect}
                            onStatusAction={openStatusActionModal}
                          />
                        ))}
                      </div>
                    </details>
                  )}
                </div>
              )}
            </>
          )}

          {!isLoading && !isAllRange && (
            reservationList.length === 0 ? (
              <div className="text-center py-12 text-gray-400">
                {loadError
                  ? '予約を取得できませんでした'
                  : isHoliday || availability?.is_holiday
                    ? 'この日は定休日です'
                    : '予約がありません'}
              </div>
            ) : (
              <div className="space-y-3">
                {reservationList.map((r) => (
                  <ReservationCard key={r.id} reservation={r} onStatusAction={openStatusActionModal} />
                ))}
              </div>
            )
          )}

          <div className="flex flex-wrap gap-4 text-xs text-gray-500 pt-2">
            {(Object.entries(statusLabels) as [ReservationStatus, string][]).map(([key, label]) => (
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
