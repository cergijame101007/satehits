CREATE TABLE IF NOT EXISTS daily_schedules (
    date DATE PRIMARY KEY,
    schedule_type TEXT NOT NULL,
    capacity INTEGER NOT NULL DEFAULT 10,
    event_name TEXT,
    event_description TEXT,
    open_time TIME,
    last_order_time TIME,
    close_time TIME,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE daily_schedules ADD CONSTRAINT daily_schedules_schedule_type_check CHECK (schedule_type IN ('normal', 'morning', 'event', 'special_menu', 'closed'));
ALTER TABLE daily_schedules ADD CONSTRAINT daily_schedules_capacity_check CHECK (capacity >= 0);

-- updated_at を UPDATE 時に自動更新する関数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW IS DISTINCT FROM OLD THEN
        NEW.updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_daily_schedules_updated_at ON daily_schedules;
CREATE TRIGGER update_daily_schedules_updated_at
    BEFORE UPDATE ON daily_schedules
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();
