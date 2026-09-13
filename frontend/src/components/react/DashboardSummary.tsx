import { useEffect, useMemo, useState } from 'react';
import { statusLabels, statusBadgeBg, statusBadgeText } from '@/lib/reservationStatusTheme';
import { getAvailability, toAvailabilityErrorMessage } from '@/lib/availability';
import { listReservations, toReservationErrorMessage } from '@/lib/adminReservation';
import {
  fetchActionablePendingCount,
  formatBadgeCount,
  pendingBadgeLabel,
} from '@/lib/pendingReservations';
import { formatDate, formatDateJa } from '@/lib/calendarUtils';
import type { AvailabilityResponse, Reservation } from '@/types/reservation';
import Alert from '@/components/react/ui/Alert';
import Card from '@/components/react/ui/Card';

interface DaySummaryProps {
  title: string;
  date: Date;
}

function DaySummary({ title, date }: DaySummaryProps) {
  const dateStr = formatDate(date);
  const [availability, setAvailability] = useState<AvailabilityResponse | null>(null);
  const [reservations, setReservations] = useState<Reservation[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [availabilityError, setAvailabilityError] = useState<string | null>(null);
  const [reservationError, setReservationError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      setAvailabilityError(null);
      setReservationError(null);

      const [availResult, resResult] = await Promise.allSettled([
        getAvailability(dateStr),
        listReservations(dateStr),
      ]);

      if (cancelled) return;

      if (availResult.status === 'fulfilled') {
        setAvailability(availResult.value);
      } else {
        setAvailability(null);
        setAvailabilityError(toAvailabilityErrorMessage(availResult.reason));
      }

      if (resResult.status === 'fulfilled') {
        setReservations(resResult.value);
      } else {
        setReservations([]);
        setReservationError(toReservationErrorMessage(resResult.reason));
      }

      setIsLoading(false);
    }

    load();
    return () => {
      cancelled = true;
    };
  }, [dateStr]);

  const pending = reservations.filter((r) => r.status === 'pending');
  const approved = reservations.filter((r) => r.status === 'approved');
  const approvedPeople = approved.reduce((sum, r) => sum + r.people, 0);

  return (
    <Card>
      <h3 className="text-sm text-gray-500 mb-1">{title}</h3>
      <p className="text-lg font-medium mb-4">{formatDateJa(dateStr)}</p>

      {availabilityError && (
        <Alert variant="error" className="text-center">{availabilityError}</Alert>
      )}
      {reservationError && (
        <Alert variant="error" className="text-center">{reservationError}</Alert>
      )}

      {isLoading && (
        <p className="text-gray-400 text-center py-6">読み込み中...</p>
      )}

      {!isLoading && !availabilityError && availability?.is_holiday && (
        <p className="text-gray-400 text-center py-6">定休日</p>
      )}

      {!isLoading && !availabilityError && availability && !availability.is_holiday && (
        <>
          <div className="text-center mb-4">
            <p className="text-xs text-gray-500 mb-1">残り提供数</p>
            <p className="text-3xl font-medium text-primary tabular-nums">
              {availability.available}
              <span className="text-lg text-gray-400 font-normal"> / {availability.capacity}食</span>
            </p>
          </div>

          <div className="grid grid-cols-2 gap-3 text-sm text-center">
            <div>
              <p className="text-gray-500 text-xs mb-0.5">承認待ち</p>
              <p className="text-xl font-medium tabular-nums">
                {reservationError ? '—' : `${pending.length}件`}
              </p>
            </div>
            <div>
              <p className="text-gray-500 text-xs mb-0.5">承認済み</p>
              <p className="text-xl font-medium tabular-nums">
                {reservationError ? (
                  '—'
                ) : (
                  <>
                    {approved.length}件
                    <span className="text-sm text-gray-500 font-normal">（{approvedPeople}名）</span>
                  </>
                )}
              </p>
            </div>
          </div>
        </>
      )}

      {!reservationError && reservations.length > 0 && (
        <div className="mt-4 pt-4 border-t border-gray-100 space-y-2">
          {reservations.slice(0, 3).map((r) => (
            <div key={r.id} className="flex items-center justify-between text-sm">
              <span>
                {r.visit_time} {r.name}（{r.people}名）
              </span>
              <span
                className="text-xs px-2 py-0.5 rounded-full"
                style={{
                  backgroundColor: statusBadgeBg[r.status],
                  color: statusBadgeText[r.status],
                }}
              >
                {statusLabels[r.status]}
              </span>
            </div>
          ))}
          {reservations.length > 3 && (
            <p className="text-xs text-gray-400">他 {reservations.length - 3}件</p>
          )}
        </div>
      )}
    </Card>
  );
}

/** 未対応 pending 件数の赤バッジ（0 件は描画しない） */
function PendingCountBadge({ count }: { count: number }) {
  if (count <= 0) return null;
  return (
    <span
      aria-label={pendingBadgeLabel(count)}
      className="ml-1 align-middle inline-flex items-center justify-center min-w-5 h-5 px-1.5 rounded-full bg-red-600 text-white text-[11px] font-medium leading-none tabular-nums"
    >
      {formatBadgeCount(count)}
    </span>
  );
}

export default function DashboardSummary() {
  const today = useMemo(() => new Date(), []);
  const tomorrow = useMemo(() => {
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return d;
  }, []);
  // 取得失敗時は 0 扱いにして注意帯・バッジを出さない（日別サマリー側でエラーは見える）
  const [pendingCount, setPendingCount] = useState(0);

  useEffect(() => {
    let cancelled = false;

    fetchActionablePendingCount()
      .then((count) => {
        if (!cancelled) setPendingCount(count);
      })
      .catch((err: unknown) => {
        console.error('未対応予約件数の取得に失敗しました', err);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-medium">ダッシュボード</h1>

      {pendingCount > 0 && (
        <Alert variant="error" className="flex flex-wrap items-center justify-between gap-2">
          <span>
            未対応の予約が <span className="font-medium tabular-nums">{pendingCount}</span>{' '}
            件あります。承認または拒否を行ってください。
          </span>
          <a
            href="/admin/reservations"
            className="shrink-0 font-medium underline underline-offset-2 hover:opacity-80 transition-opacity"
          >
            予約一覧へ
          </a>
        </Alert>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <DaySummary title="今日の予約" date={today} />
        <DaySummary title="明日の予約" date={tomorrow} />
      </div>

      {/* クイックリンク */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
        {[
          { href: '/admin/reservations', label: '予約一覧', icon: 'M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2' },
          { href: '/admin/reservations/new', label: '予約登録', icon: 'M12 4v16m8-8H4' },
          { href: '/admin/schedules', label: 'スケジュール', icon: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z' },
          { href: '/admin/suppliers', label: '取引先', icon: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z' },
        ].map((link) => (
          <a
            key={link.href}
            href={link.href}
            className="rounded-xl border border-gray-200 bg-white p-4 flex flex-col items-center gap-2 hover:border-primary/30 hover:shadow-sm transition-all"
          >
            <svg className="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d={link.icon} />
            </svg>
            <span className="text-sm font-medium">
              {link.label}
              {link.href === '/admin/reservations' && <PendingCountBadge count={pendingCount} />}
            </span>
          </a>
        ))}
        <a
          href="/"
          target="_blank"
          rel="noopener noreferrer"
          className="rounded-xl border border-gray-200 bg-white p-4 flex flex-col items-center gap-2 hover:border-primary/30 hover:shadow-sm transition-all"
        >
          <svg className="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
          </svg>
          <span className="text-sm font-medium">サイト確認</span>
        </a>
      </div>
    </div>
  );
}
