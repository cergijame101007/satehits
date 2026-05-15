-- reservations.updated_at を daily_schedules と同じ共通関数で自動更新
DROP TRIGGER IF EXISTS update_reservations_updated_at ON reservations;
CREATE TRIGGER update_reservations_updated_at
    BEFORE UPDATE ON reservations
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();
