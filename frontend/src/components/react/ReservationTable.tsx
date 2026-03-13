import { useState, useMemo } from 'react';
import type { Reservation, ReservationStatus } from '../../types/reservation';
import { getReservations, getAvailability, statusLabels, statusColors } from '../../mocks/reservation';

function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

/** ステータス遷移ルール */
const statusTransitions: Record<ReservationStatus, ReservationStatus[]> = {
  pending: ['approved', 'rejected'],
  approved: ['no_show'],
  rejected: [],
  no_show: [],
};

const actionLabels: Record<ReservationStatus, string> = {
  approved: '承認',
  rejected: '拒否',
  no_show: 'No Show',
  pending: '',
};

const actionColors: Record<ReservationStatus, string> = {
  approved: 'bg-green-600 hover:bg-green-700',
  rejected: 'bg-red-600 hover:bg-red-700',
  no_show: 'bg-gray-600 hover:bg-gray-700',
  pending: '',
};

export default function ReservationTable() {
  const [selectedDate, setSelectedDate] = useState(formatDate(new Date()));
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [reservationList, setReservationList] = useState<Reservation[]>([]);
  const [isLoaded, setIsLoaded] = useState(false);

  // 日付変更時にデータ再取得
  useMemo(() => {
    // TODO: GET /api/v1/admin/reservations?date=YYYY-MM-DD&status=xxx に置き換え
    const data = getReservations(selectedDate, statusFilter || undefined);
    setReservationList(data);
    setIsLoaded(true);
  }, [selectedDate, statusFilter]);

  // TODO: GET /api/v1/reservations/availability?date=YYYY-MM-DD に置き換え
  const availability = getAvailability(selectedDate);

  const handleStatusChange = async (id: number, newStatus: ReservationStatus) => {
    const reservation = reservationList.find((r) => r.id === id);
    if (!reservation) return;

    const actionLabel = actionLabels[newStatus];
    if (!confirm(`${reservation.name}さんの予約を「${actionLabel}」にしますか？`)) return;

    // TODO: PATCH /api/v1/admin/reservations/{id}/status に置き換え
    setReservationList((prev) =>
      prev.map((r) =>
        r.id === id ? { ...r, status: newStatus, updated_at: new Date().toISOString() } : r,
      ),
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-medium">予約一覧</h1>
        <a
          href="/admin/reservations/new"
          className="px-4 py-2 bg-[#43676B] text-white text-sm rounded-lg hover:bg-[#365558] transition-colors"
        >
          + 予約登録
        </a>
      </div>

      {/* フィルター */}
      <div className="flex flex-wrap gap-3">
        <input
          type="date"
          value={selectedDate}
          onChange={(e) => setSelectedDate(e.target.value)}
          className="px-4 py-2 rounded-lg border border-gray-200 bg-white focus:outline-none focus:ring-2 focus:ring-[#43676B]/30 focus:border-[#43676B] transition-colors"
        />
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-4 py-2 rounded-lg border border-gray-200 bg-white focus:outline-none focus:ring-2 focus:ring-[#43676B]/30 focus:border-[#43676B] transition-colors"
        >
          <option value="">すべてのステータス</option>
          <option value="pending">申請中</option>
          <option value="approved">承認済み</option>
          <option value="rejected">拒否</option>
          <option value="no_show">No Show</option>
        </select>
      </div>

      {/* キャパシティ情報 */}
      {isLoaded && (
        <div className="flex items-center gap-4 text-sm">
          {availability.is_holiday ? (
            <span className="text-gray-400">定休日</span>
          ) : (
            <>
              <span>
                残り: <strong className="text-[#43676B]">{availability.available}食</strong> / {availability.capacity}食
              </span>
              <span className="text-gray-400">
                予約済み: {availability.reserved}食
              </span>
            </>
          )}
        </div>
      )}

      {/* 予約カード一覧 */}
      {reservationList.length === 0 ? (
        <div className="text-center py-12 text-gray-400">
          {availability.is_holiday ? 'この日は定休日です' : '予約がありません'}
        </div>
      ) : (
        <div className="space-y-3">
          {reservationList.map((r) => (
            <div
              key={r.id}
              className="rounded-xl border border-gray-200 bg-white p-5 transition-shadow hover:shadow-sm"
            >
              <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                  {/* ステータスドット */}
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
                    backgroundColor:
                      r.status === 'pending'
                        ? '#FEF3C7'
                        : r.status === 'approved'
                          ? '#DCFCE7'
                          : r.status === 'rejected'
                            ? '#FEE2E2'
                            : '#F3F4F6',
                    color:
                      r.status === 'pending'
                        ? '#92400E'
                        : r.status === 'approved'
                          ? '#166534'
                          : r.status === 'rejected'
                            ? '#991B1B'
                            : '#374151',
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

              {/* アクションボタン */}
              {statusTransitions[r.status].length > 0 && (
                <div className="flex gap-2 mt-4 ml-6">
                  {statusTransitions[r.status].map((nextStatus) => (
                    <button
                      key={nextStatus}
                      onClick={() => handleStatusChange(r.id, nextStatus)}
                      className={`px-4 py-1.5 text-sm text-white rounded-lg transition-colors ${actionColors[nextStatus]}`}
                    >
                      {actionLabels[nextStatus]}
                    </button>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* 凡例 */}
      <div className="flex flex-wrap gap-4 text-xs text-gray-500 pt-2">
        {Object.entries(statusLabels).map(([key, label]) => (
          <span key={key} className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full" style={{ backgroundColor: statusColors[key] }} />
            {label}
          </span>
        ))}
      </div>
    </div>
  );
}
