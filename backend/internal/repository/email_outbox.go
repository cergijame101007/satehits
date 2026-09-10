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

// ClaimNextPending は送信可能な 1 行を FOR UPDATE SKIP LOCKED で選び、
// attempt_count を +1・next_attempt_at を leaseUntil に進めた状態で返す（単一 UPDATE で完結）。
// 同一 reservation_id に先行 pending がある行はスキップする（未処理同士の追い越し防止）。
// lease 中の行も pending のままなので、後続行はその lease が解消されるまで取られない。
func (r *PostgresEmailOutboxRepository) ClaimNextPending(ctx context.Context, leaseUntil time.Time) (*domain.EmailOutboxMessage, error) {
	query := `
UPDATE email_outbox o
SET attempt_count = o.attempt_count + 1,
    next_attempt_at = $1
WHERE o.id = (
    SELECT c.id
    FROM email_outbox c
    WHERE c.status = 'pending'
      AND c.next_attempt_at <= NOW()
      AND NOT EXISTS (
          SELECT 1 FROM email_outbox p
          WHERE p.reservation_id = c.reservation_id
            AND p.status = 'pending'
            AND p.created_at < c.created_at
      )
    ORDER BY c.created_at
    FOR UPDATE OF c SKIP LOCKED
    LIMIT 1
)
RETURNING o.id, o.reservation_id, o.mail_type, o.from_address, o.to_address,
          o.subject, o.body_html, o.body_text, o.attempt_count, o.next_attempt_at, o.created_at`
	var msg domain.EmailOutboxMessage
	var mailType string
	err := r.getDB(ctx).QueryRowContext(ctx, query, leaseUntil).Scan(
		&msg.ID,
		&msg.ReservationID,
		&mailType,
		&msg.FromAddress,
		&msg.ToAddress,
		&msg.Subject,
		&msg.BodyHTML,
		&msg.BodyText,
		&msg.AttemptCount,
		&msg.NextAttemptAt,
		&msg.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim next pending outbox: %w", err)
	}
	msg.MailType = domain.MailType(mailType)
	msg.Status = domain.OutboxStatusPending
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
	return requirePendingRowAffected(result, "mark outbox sent", id)
}

// MarkRetry は一時失敗として next_attempt_at を設定する（attempt_count は claim 時に加算済み）
func (r *PostgresEmailOutboxRepository) MarkRetry(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
UPDATE email_outbox
SET next_attempt_at = $2, last_error = $3
WHERE id = $1 AND status = 'pending'`, id, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox retry: %w", err)
	}
	return requirePendingRowAffected(result, "mark outbox retry", id)
}

// MarkFailed は再送上限到達または恒久失敗として記録する
func (r *PostgresEmailOutboxRepository) MarkFailed(ctx context.Context, id uuid.UUID, lastError string) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
UPDATE email_outbox
SET status = 'failed', last_error = $2
WHERE id = $1 AND status = 'pending'`, id, lastError)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return requirePendingRowAffected(result, "mark outbox failed", id)
}

// ReleaseClaim は claim で加算した attempt_count を戻し、next_attempt_at を設定する。
// 認証エラーなどメッセージ起因でない失敗で、行の試行回数を消費させないために使う
func (r *PostgresEmailOutboxRepository) ReleaseClaim(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error {
	result, err := r.getDB(ctx).ExecContext(ctx, `
UPDATE email_outbox
SET attempt_count = GREATEST(attempt_count - 1, 0), next_attempt_at = $2, last_error = $3
WHERE id = $1 AND status = 'pending'`, id, nextAttemptAt, lastError)
	if err != nil {
		return fmt.Errorf("release outbox claim: %w", err)
	}
	return requirePendingRowAffected(result, "release outbox claim", id)
}

func requirePendingRowAffected(result sql.Result, op string, id uuid.UUID) error {
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows: %w", op, err)
	}
	if n == 0 {
		return fmt.Errorf("%s: no pending row for id=%s", op, id)
	}
	return nil
}
