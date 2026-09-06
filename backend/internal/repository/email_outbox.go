package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresEmailOutboxRepository は PostgreSQL の email_outbox 実装
type PostgresEmailOutboxRepository struct {
	baseRepository
}

// NewPostgresEmailOutboxRepository は PostgresEmailOutboxRepository を生成する
func NewPostgresEmailOutboxRepository(db *sql.DB) *PostgresEmailOutboxRepository {
	return &PostgresEmailOutboxRepository{baseRepository{db: db}}
}

// Enqueue は Outbox 行を挿入する。UNIQUE 衝突時は Tx を aborted にせず ErrMailAlreadyEnqueued を返す
func (r *PostgresEmailOutboxRepository) Enqueue(ctx context.Context, in domain.EnqueueEmailInput) error {
	query := `
INSERT INTO email_outbox (
    reservation_id, mail_type, from_address, to_address, subject, body_html, body_text
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (reservation_id, mail_type) DO NOTHING
RETURNING id`
	var id uuid.UUID
	err := r.getDB(ctx).QueryRowContext(ctx, query,
		in.ReservationID, string(in.MailType), in.FromAddress, in.ToAddress,
		in.Subject, in.BodyHTML, in.BodyText,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrMailAlreadyEnqueued
	}
	if err != nil {
		return fmt.Errorf("enqueue email outbox: %w", err)
	}
	return nil
}

// ClaimNextPending は送信可能な 1 行を FOR UPDATE SKIP LOCKED で取得する。
// 同一 reservation_id に先行 pending がある行はスキップする（未処理同士の追い越し防止）。
func (r *PostgresEmailOutboxRepository) ClaimNextPending(ctx context.Context) (*domain.EmailOutboxMessage, error) {
	query := `
SELECT o.id, o.reservation_id, o.mail_type, o.from_address, o.to_address,
       o.subject, o.body_html, o.body_text, o.attempt_count
FROM email_outbox o
WHERE o.status = 'pending'
  AND o.next_attempt_at <= NOW()
  AND NOT EXISTS (
      SELECT 1 FROM email_outbox p
      WHERE p.reservation_id = o.reservation_id
        AND p.status = 'pending'
        AND p.created_at < o.created_at
  )
ORDER BY o.created_at
FOR UPDATE OF o SKIP LOCKED
LIMIT 1`
	var msg domain.EmailOutboxMessage
	var mailType string
	err := r.getDB(ctx).QueryRowContext(ctx, query).Scan(
		&msg.ID,
		&msg.ReservationID,
		&mailType,
		&msg.FromAddress,
		&msg.ToAddress,
		&msg.Subject,
		&msg.BodyHTML,
		&msg.BodyText,
		&msg.AttemptCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim next pending outbox: %w", err)
	}
	msg.MailType = domain.MailType(mailType)
	msg.Status = "pending"
	return &msg, nil
}

// MarkSent は送信成功として記録する
func (r *PostgresEmailOutboxRepository) MarkSent(ctx context.Context, id uuid.UUID) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
UPDATE email_outbox
SET status = 'sent', sent_at = NOW(), last_error = NULL
WHERE id = $1 AND status = 'pending'`, id)
	if err != nil {
		return fmt.Errorf("mark outbox sent: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark outbox sent rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mark outbox sent: no pending row for id=%s", id)
	}
	return nil
}

// MarkRetry は一時失敗として attempt_count と next_attempt_at を更新する
func (r *PostgresEmailOutboxRepository) MarkRetry(ctx context.Context, id uuid.UUID, attemptCount int, nextAttemptAt time.Time, lastError string) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
UPDATE email_outbox
SET attempt_count = $2, next_attempt_at = $3, last_error = $4
WHERE id = $1 AND status = 'pending'`, id, attemptCount, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox retry: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark outbox retry rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mark outbox retry: no pending row for id=%s", id)
	}
	return nil
}

// MarkFailed は再送上限到達または恒久失敗として記録する
func (r *PostgresEmailOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, attemptCount int, lastError string) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
UPDATE email_outbox
SET status = 'failed', attempt_count = $2, last_error = $3
WHERE id = $1 AND status = 'pending'`, id, attemptCount, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark outbox failed rows: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mark outbox failed: no pending row for id=%s", id)
	}
	return nil
}
