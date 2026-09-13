-- オーナー向け pending リマインド（UC-S04）の mail_type を追加する。
-- 000009 では CHECK が無名のため PostgreSQL の自動命名（email_outbox_mail_type_check）を前提に置き換える。
-- リマインドはタイミングごとに別 mail_type なので UNIQUE (reservation_id, mail_type) がそのまま
-- 「予約 1 件・タイミング 1 種につき最大 1 通」を保証する。
ALTER TABLE email_outbox DROP CONSTRAINT IF EXISTS email_outbox_mail_type_check;
ALTER TABLE email_outbox ADD CONSTRAINT email_outbox_mail_type_check
  CHECK (mail_type IN (
    'reservation_received',
    'reservation_approved',
    'reservation_rejected',
    'pending_reminder_3d',
    'pending_reminder_1d'
  ));
