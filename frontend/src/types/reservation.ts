/** 予約ステータス */
export type ReservationStatus = 'pending' | 'approved' | 'rejected' | 'cancelled' | 'no_show';

/** 予約経路 */
export type ReservationSource = 'web' | 'instagram' | 'phone' | 'walk_in' | 'other';

/** 予約データ */
export interface Reservation {
  id: string;
  name: string;
  people: number;
  visit_date: string;
  visit_time: string;
  phone: string;
  email: string;
  note: string;
  status: ReservationStatus;
  source: ReservationSource;
  created_at: string;
  updated_at: string;
}

/** 予約申請リクエスト */
export interface ReservationRequest {
  name: string;
  people: number;
  visit_date: string;
  visit_time: string;
  phone: string;
  email: string;
  note: string;
  turnstile_token: string;
}

/** 空き状況レスポンス */
export interface AvailabilityResponse {
  date: string;
  capacity: number;
  reserved: number;
  available: number;
  schedule_type?: string | null;
  event_name?: string;
  event_description?: string;
  is_holiday: boolean;
}

/** 月間空き状況レスポンス */
export interface AvailabilityListResponse {
  year: number;
  month: number;
  availabilities: AvailabilityResponse[];
}

/** 顧客向け公開スケジュールの schedule_type（OpenAPI DaySchedule） */
export type PublicScheduleType =
  | 'normal'
  | 'morning'
  | 'event'
  | 'external_event'
  | 'special_menu'
  | 'closed';

/** 顧客向け日別スケジュール（GET /schedules） */
export interface PublicDaySchedule {
  date: string;
  schedule_type: PublicScheduleType | null;
  capacity: number;
  available: number;
  event_name?: string;
  event_description?: string;
  is_holiday: boolean;
}

/** 顧客向け月間スケジュールレスポンス */
export interface MonthlyPublicScheduleResponse {
  year: number;
  month: number;
  schedules: PublicDaySchedule[];
}

/** ログインリクエスト */
export interface LoginRequest {
  email: string;
  password: string;
}

/** ログインレスポンス */
export interface LoginResponse {
  token: string;
  expires_at: string;
  user: {
    id: number;
    email: string;
    role: string;
  };
}

/** 日別スケジュールタイプ */
export type ScheduleType =
  | 'normal'
  | 'morning'
  | 'event'
  | 'external_event'
  | 'special'
  | 'closed'
  | 'temporary_closed';

/** 日別スケジュール */
export interface DailySchedule {
  date: string;
  type: ScheduleType;
  capacity: number;
  event_name?: string;
  description?: string;
}

/** 管理者用予約登録リクエスト */
export interface AdminReservationRequest {
  source: Exclude<ReservationSource, 'web'>;
  name: string;
  people: number;
  visit_date: string;
  visit_time: string;
  phone: string;
  email: string;
  note: string;
  status?: ReservationStatus;
}

/** 予約一覧レスポンス */
export interface ReservationListResponse {
  reservations: Reservation[];
  total: number;
}

/** ステータス更新レスポンス */
export interface UpdateStatusResponse {
  id: string;
  status: ReservationStatus;
  updated_at: string;
}

