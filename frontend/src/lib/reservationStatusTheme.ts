import type { Reservation, ReservationStatus } from '@/types/reservation';
import type { ButtonVariant } from '@/lib/ui/types';

/** ステータスの日本語表示 */
export const statusLabels: Record<ReservationStatus, string> = {
  pending: '申請中',
  approved: '承認済み',
  rejected: '拒否',
  cancelled: 'キャンセル',
  no_show: 'No Show',
};

/** ステータスの色クラス */
export const statusColors: Record<ReservationStatus, string> = {
  pending: '#EAB308',
  approved: '#22C55E',
  rejected: '#EF4444',
  cancelled: '#9CA3AF',
  no_show: '#EA580C',
};

/** ステータスバッジの背景色 */
export const statusBadgeBg: Record<ReservationStatus, string> = {
  pending: '#FEF3C7',
  approved: '#DCFCE7',
  rejected: '#FEE2E2',
  cancelled: '#F3F4F6',
  no_show: '#FED7AA',
};

/** ステータスバッジの文字色 */
export const statusBadgeText: Record<ReservationStatus, string> = {
  pending: '#92400E',
  approved: '#166534',
  rejected: '#991B1B',
  cancelled: '#4B5563',
  no_show: '#C2410C',
};

export const statusActionLabels: Record<ReservationStatus, string> = {
  approved: '承認',
  rejected: '拒否',
  cancelled: 'キャンセル',
  no_show: 'No Show',
  pending: '',
};

export type StatusActionTarget = {
  reservation: Reservation;
  newStatus: ReservationStatus;
};

export function statusToButtonVariant(status: ReservationStatus): ButtonVariant {
  switch (status) {
    case 'approved':
      return 'success';
    case 'rejected':
      return 'danger';
    case 'cancelled':
      return 'ghost';
    case 'no_show':
      return 'warning';
    default:
      return 'primary';
  }
}

export function getStatusActionDialogContent(target: StatusActionTarget): {
  title: string;
  description: string;
  confirmLabel: string;
  confirmVariant: ButtonVariant;
} {
  const { reservation, newStatus } = target;
  const summary = `${reservation.name}さん（${reservation.visit_time}・${reservation.people}名）`;

  switch (newStatus) {
    case 'approved':
      return {
        title: '予約を承認',
        description: `${summary}の予約を承認します。`,
        confirmLabel: '承認する',
        confirmVariant: 'success',
      };
    case 'rejected':
      return {
        title: '予約を拒否',
        description: `${summary}の予約を拒否します。`,
        confirmLabel: '拒否する',
        confirmVariant: 'danger',
      };
    case 'cancelled':
      return {
        title: '予約をキャンセル',
        description: `${summary}の予約をキャンセルに変更します。`,
        confirmLabel: 'キャンセルする',
        confirmVariant: 'neutral',
      };
    case 'no_show':
      return {
        title: 'No Show に変更',
        description: `${summary}の予約を「No Show（無断キャンセル）」にします。この操作は取り消せません。`,
        confirmLabel: 'No Show にする',
        confirmVariant: 'warning',
      };
    default:
      return {
        title: 'ステータスを変更',
        description: `${summary}の予約ステータスを変更します。`,
        confirmLabel: '変更する',
        confirmVariant: 'primary',
      };
  }
}
