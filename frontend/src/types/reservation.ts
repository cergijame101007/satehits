/** 予約ステータス */
export type ReservationStatus = 'pending' | 'approved' | 'rejected' | 'cancelled' | 'no_show';

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
  recaptcha_token: string;
}

/** 空き状況レスポンス */
export interface AvailabilityResponse {
  date: string;
  capacity: number;
  reserved: number;
  available: number;
  is_holiday: boolean;
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
export type ScheduleType = 'normal' | 'morning' | 'event' | 'special' | 'closed' | 'temporary_closed';

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
  source: 'instagram' | 'phone' | 'walk_in' | 'other';
  name: string;
  people: number;
  visit_date: string;
  visit_time: string;
  phone: string;
  email: string;
  note: string;
  status: ReservationStatus;
}

