import type {
  Reservation,
  LoginResponse,
  DailySchedule,
  ScheduleType,
} from '../types/reservation';

/** モック予約データ */
export const mockReservations: Reservation[] = [
  {
    id: 'a1000001-0000-4000-8000-000000000001',
    name: '山田太郎',
    people: 2,
    visit_date: '2026-03-14',
    visit_time: '12:00',
    phone: '090-1234-5678',
    email: 'yamada@example.com',
    note: '',
    status: 'pending',
    source: 'web',
    created_at: '2026-03-10T10:00:00+09:00',
    updated_at: '2026-03-10T10:00:00+09:00',
  },
  {
    id: 'a1000001-0000-4000-8000-000000000002',
    name: '佐藤花子',
    people: 4,
    visit_date: '2026-03-14',
    visit_time: '12:30',
    phone: '080-9876-5432',
    email: 'sato@example.com',
    note: '魚の火入れ希望',
    status: 'approved',
    source: 'web',
    created_at: '2026-03-09T15:30:00+09:00',
    updated_at: '2026-03-10T09:00:00+09:00',
  },
  {
    id: 'a1000001-0000-4000-8000-000000000003',
    name: '鈴木一郎',
    people: 1,
    visit_date: '2026-03-14',
    visit_time: '11:30',
    phone: '070-5555-1234',
    email: 'suzuki@example.com',
    note: 'テーブル席希望',
    status: 'approved',
    source: 'phone',
    created_at: '2026-03-08T12:00:00+09:00',
    updated_at: '2026-03-09T10:00:00+09:00',
  },
  {
    id: 'a1000001-0000-4000-8000-000000000004',
    name: '田中美咲',
    people: 3,
    visit_date: '2026-03-15',
    visit_time: '9:00',
    phone: '090-3333-4444',
    email: 'tanaka@example.com',
    note: '',
    status: 'pending',
    source: 'instagram',
    created_at: '2026-03-11T08:00:00+09:00',
    updated_at: '2026-03-11T08:00:00+09:00',
  },
  {
    id: 'a1000001-0000-4000-8000-000000000005',
    name: '高橋健太',
    people: 2,
    visit_date: '2026-03-15',
    visit_time: '10:00',
    phone: '080-7777-8888',
    email: 'takahashi@example.com',
    note: '',
    status: 'cancelled',
    source: 'web',
    created_at: '2026-03-10T20:00:00+09:00',
    updated_at: '2026-03-11T09:00:00+09:00',
  },
];

/**
 * 予約一覧を取得（モック）
 * TODO: GET /api/v1/admin/reservations?date=YYYY-MM-DD&status=xxx に置き換え
 */
export function getReservations(date?: string, status?: string): Reservation[] {
  let filtered = [...mockReservations];
  if (date) filtered = filtered.filter((r) => r.visit_date === date);
  if (status) filtered = filtered.filter((r) => r.status === status);
  return filtered.sort((a, b) => a.visit_time.localeCompare(b.visit_time));
}

/**
 * ログイン（モック）
 * TODO: POST /api/v1/admin/login に置き換え
 */
export function login(email: string, password: string): LoginResponse | null {
  if (email === 'owner@example.com' && password === 'password123') {
    return {
      token: 'mock-jwt-token-xxxxx',
      expires_at: new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString(),
      user: { id: 1, email: 'owner@example.com', role: 'owner' },
    };
  }
  return null;
}

/**
 * 月間スケジュールを生成（モック）
 * TODO: GET /api/v1/admin/schedules?month=YYYY-MM に置き換え
 */
export function getMonthlySchedules(year: number, month: number): DailySchedule[] {
  const schedules: DailySchedule[] = [];
  const daysInMonth = new Date(year, month, 0).getDate();

  for (let day = 1; day <= daysInMonth; day++) {
    const date = new Date(year, month - 1, day);
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
    const dayOfWeek = date.getDay();

    let type: ScheduleType;
    let capacity = 10;

    if (dayOfWeek === 4 || dayOfWeek === 5) {
      type = 'closed';
      capacity = 0;
    } else if (dayOfWeek === 0) {
      type = 'morning';
    } else {
      type = 'normal';
    }

    schedules.push({ date: dateStr, type, capacity });
  }

  // モック: 特別な日を追加
  const eventDay = schedules.find((s) => s.date === `${year}-${String(month).padStart(2, '0')}-11`);
  if (eventDay && eventDay.type !== 'closed') {
    eventDay.type = 'event';
    eventDay.event_name = '和紅茶をしばく会';
    eventDay.description = '和紅茶をしばく会 入門編\n@WINE LAB.\n通常のランチ営業はおやすみ';
    eventDay.capacity = 0;
  }

  return schedules;
}

/** ステータスの日本語表示 */
export const statusLabels: Record<string, string> = {
  pending: '申請中',
  approved: '承認済み',
  rejected: '拒否',
  cancelled: 'キャンセル',
  no_show: 'No Show',
};

/** ステータスの色クラス */
export const statusColors: Record<string, string> = {
  pending: '#EAB308',
  approved: '#22C55E',
  rejected: '#EF4444',
  cancelled: '#9CA3AF',
  no_show: '#EA580C',
};

/** ステータスバッジの背景色 */
export const statusBadgeBg: Record<string, string> = {
  pending: '#FEF3C7',
  approved: '#DCFCE7',
  rejected: '#FEE2E2',
  cancelled: '#F3F4F6',
  no_show: '#FED7AA',
};

/** ステータスバッジの文字色 */
export const statusBadgeText: Record<string, string> = {
  pending: '#92400E',
  approved: '#166534',
  rejected: '#991B1B',
  cancelled: '#4B5563',
  no_show: '#C2410C',
};

/** スケジュールタイプの表示ラベル */
export const scheduleTypeLabels: Record<ScheduleType, string> = {
  normal: '通常',
  morning: '朝営業',
  event: 'イベント',
  special: '特別メニュー',
  closed: '定休日',
  temporary_closed: '臨時休',
};

/** スケジュールタイプの短縮表示 */
export const scheduleTypeShort: Record<ScheduleType, string> = {
  normal: '通',
  morning: '朝',
  event: 'イ',
  special: '特',
  closed: '休',
  temporary_closed: '臨',
};
