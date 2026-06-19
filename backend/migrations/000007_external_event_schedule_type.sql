ALTER TABLE daily_schedules DROP CONSTRAINT daily_schedules_schedule_type_check;
ALTER TABLE daily_schedules ADD CONSTRAINT daily_schedules_schedule_type_check
  CHECK (schedule_type IN ('normal', 'morning', 'event', 'external_event', 'special_menu', 'closed'));

UPDATE daily_schedules
SET schedule_type = 'external_event', capacity = 0
WHERE schedule_type = 'event' AND capacity = 0;
