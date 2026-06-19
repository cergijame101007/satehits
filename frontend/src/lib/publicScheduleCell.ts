import type { PublicDaySchedule } from '@/types/reservation';

export interface PublicScheduleCellDisplay {
  showClosedMark: boolean;
  showEventBar: boolean;
}

function hasEventName(schedule: PublicDaySchedule | undefined): boolean {
  return Boolean(schedule?.event_name?.trim());
}

/** セル中央の「休」表示（定休・臨時休・外部イベント） */
export function shouldShowClosedMark(schedule: PublicDaySchedule | undefined): boolean {
  if (!schedule) return false;
  if (schedule.is_holiday) return true;
  return schedule.schedule_type === 'closed';
}

/** セル下部のイベント名バー（店内イベント・外部イベント・特別メニュー） */
export function shouldShowEventBar(schedule: PublicDaySchedule | undefined): boolean {
  if (!schedule || !hasEventName(schedule)) return false;
  return (
    schedule.schedule_type === 'event' ||
    schedule.schedule_type === 'external_event' ||
    schedule.schedule_type === 'special_menu'
  );
}

export function getPublicScheduleCellDisplay(
  schedule: PublicDaySchedule | undefined,
): PublicScheduleCellDisplay {
  return {
    showClosedMark: shouldShowClosedMark(schedule),
    showEventBar: shouldShowEventBar(schedule),
  };
}
