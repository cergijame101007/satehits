ALTER TABLE reservations ADD COLUMN IF NOT EXISTS visit_date DATE NOT NULL DEFAULT CURRENT_DATE;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS visit_time TIME NOT NULL DEFAULT TIME '12:00:00';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS note TEXT;
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'web';
ALTER TABLE reservations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

ALTER TABLE reservations ALTER COLUMN visit_date DROP DEFAULT;
ALTER TABLE reservations ALTER COLUMN visit_time DROP DEFAULT;
ALTER TABLE reservations ALTER COLUMN phone DROP DEFAULT;
ALTER TABLE reservations ALTER COLUMN email DROP DEFAULT;

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_people_check;
ALTER TABLE reservations
    ADD CONSTRAINT reservations_people_check CHECK (people >= 1 AND people <= 7) NOT VALID;
ALTER TABLE reservations VALIDATE CONSTRAINT reservations_people_check;

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_status_check;
ALTER TABLE reservations
    ADD CONSTRAINT reservations_status_check
        CHECK (status IN ('pending', 'approved', 'rejected', 'no_show'));

ALTER TABLE reservations DROP CONSTRAINT IF EXISTS reservations_source_check;
ALTER TABLE reservations
    ADD CONSTRAINT reservations_source_check
        CHECK (source IN ('web', 'instagram', 'phone', 'walk_in', 'other'));

CREATE INDEX IF NOT EXISTS idx_reservations_visit_date ON reservations (visit_date);
CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations (status);
