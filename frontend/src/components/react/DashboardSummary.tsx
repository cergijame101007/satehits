import { useMemo } from 'react';
import { getReservations, getAvailability, statusLabels, statusBadgeBg, statusBadgeText } from '../../mocks/reservation';
import { formatDate, formatDateJa } from '@/lib/calendarUtils';

interface DaySummaryProps {
  title: string;
  date: Date;
}

function DaySummary({ title, date }: DaySummaryProps) {
  const dateStr = formatDate(date);
  // TODO: GET /api/v1/admin/reservations?date=YYYY-MM-DD に置き換え
  const reservations = getReservations(dateStr);
  // TODO: GET /api/v1/reservations/availability?date=YYYY-MM-DD に置き換え
  const availability = getAvailability(dateStr);

  const pending = reservations.filter((r) => r.status === 'pending');
  const approved = reservations.filter((r) => r.status === 'approved');
  const approvedPeople = approved.reduce((sum, r) => sum + r.people, 0);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5">
      <h3 className="text-sm text-gray-500 mb-1">{title}</h3>
      <p className="text-lg font-medium mb-4">{formatDateJa(formatDate(date))}</p>

      {availability.is_holiday ? (
        <p className="text-gray-400 text-center py-6">定休日</p>
      ) : (
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
              <p className="text-xl font-medium tabular-nums">{pending.length}件</p>
            </div>
            <div>
              <p className="text-gray-500 text-xs mb-0.5">承認済み</p>
              <p className="text-xl font-medium tabular-nums">
                {approved.length}件
                <span className="text-sm text-gray-500 font-normal">（{approvedPeople}名）</span>
              </p>
            </div>
          </div>
        </>
      )}

      {reservations.length > 0 && (
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
    </div>
  );
}

export default function DashboardSummary() {
  const today = useMemo(() => new Date(), []);
  const tomorrow = useMemo(() => {
    const d = new Date();
    d.setDate(d.getDate() + 1);
    return d;
  }, []);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-medium">ダッシュボード</h1>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <DaySummary title="今日の予約" date={today} />
        <DaySummary title="明日の予約" date={tomorrow} />
      </div>

      {/* クイックリンク */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
        <a
          href="/admin/reservations"
          className="flex flex-col items-center gap-2 p-4 rounded-xl border border-gray-200 bg-white hover:border-primary/30 hover:shadow-sm transition-all"
        >
          <svg className="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
          </svg>
          <span className="text-sm font-medium">予約一覧</span>
        </a>
        <a
          href="/admin/reservations/new"
          className="flex flex-col items-center gap-2 p-4 rounded-xl border border-gray-200 bg-white hover:border-primary/30 hover:shadow-sm transition-all"
        >
          <svg className="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 4v16m8-8H4" />
          </svg>
          <span className="text-sm font-medium">予約登録</span>
        </a>
        <a
          href="/admin/schedules"
          className="flex flex-col items-center gap-2 p-4 rounded-xl border border-gray-200 bg-white hover:border-primary/30 hover:shadow-sm transition-all"
        >
          <svg className="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <span className="text-sm font-medium">スケジュール</span>
        </a>
        <a
          href="/admin/suppliers"
          className="flex flex-col items-center gap-2 p-4 rounded-xl border border-gray-200 bg-white hover:border-primary/30 hover:shadow-sm transition-all"
        >
          <svg className="w-6 h-6 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <span className="text-sm font-medium">取引先</span>
        </a>
        <a
          href="/"
          target="_blank"
          rel="noopener noreferrer"
          className="flex flex-col items-center gap-2 p-4 rounded-xl border border-gray-200 bg-white hover:border-primary/30 hover:shadow-sm transition-all"
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
