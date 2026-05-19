CREATE TABLE reservations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    people      INTEGER NOT NULL CHECK (people >= 1 AND people <= 7),
    visit_date  DATE NOT NULL,
    visit_time  TIME NOT NULL,
    phone       TEXT NOT NULL,
    email       TEXT NOT NULL,
    note        TEXT,
    status      TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled', 'no_show')),
    source      TEXT NOT NULL DEFAULT 'web'
                CHECK (source IN ('web', 'instagram', 'phone', 'walk_in', 'other')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reservations_visit_date ON reservations (visit_date);
CREATE INDEX idx_reservations_status ON reservations (status);
CREATE INDEX idx_reservations_created_at ON reservations (created_at);
CREATE UNIQUE INDEX idx_reservations_unique_active
    ON reservations (visit_date, visit_time, phone)
    WHERE status IN ('pending', 'approved');
