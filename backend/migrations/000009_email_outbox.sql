-- メール送信意図を永続化する Outbox。予約 INSERT と同一トランザクションで記録する。
-- UNIQUE (reservation_id, mail_type) は防御的な不変条件（通常フローでは衝突しない）。
CREATE TABLE IF NOT EXISTS email_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reservation_id UUID NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    mail_type TEXT NOT NULL
        CHECK (mail_type IN ('reservation_received', 'reservation_approved', 'reservation_rejected')),
    from_address TEXT NOT NULL,
    to_address TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_html TEXT NOT NULL,
    body_text TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'failed')),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_outbox_logical_event_unique UNIQUE (reservation_id, mail_type)
);

CREATE INDEX IF NOT EXISTS idx_email_outbox_dispatch
    ON email_outbox (next_attempt_at) WHERE status = 'pending';

-- update_updated_at_column() は 000002_daily_schedules.sql で定義済み
DROP TRIGGER IF EXISTS update_email_outbox_updated_at ON email_outbox;
CREATE TRIGGER update_email_outbox_updated_at
    BEFORE UPDATE ON email_outbox
    FOR EACH ROW
    EXECUTE PROCEDURE update_updated_at_column();
