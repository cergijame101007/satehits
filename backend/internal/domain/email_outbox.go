package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrMailAlreadyEnqueued は同一 reservation_id + mail_type の Outbox 行が既にある
var ErrMailAlreadyEnqueued = errors.New("mail already enqueued")

// MailType は Outbox に記録する論理メール種別
type MailType string

const (
	MailTypeReservationReceived MailType = "reservation_received"
	MailTypeReservationApproved MailType = "reservation_approved"
	MailTypeReservationRejected MailType = "reservation_rejected"
)

// EmailOutboxMessage は送信待ち・送信済みの Outbox 1 行
type EmailOutboxMessage struct {
	ID            uuid.UUID
	ReservationID uuid.UUID
	MailType      MailType
	FromAddress   string
	ToAddress     string
	Subject       string
	BodyHTML      string
	BodyText      string
	Status        string
	AttemptCount  int
	NextAttemptAt time.Time
	LastError     string
	SentAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// EnqueueEmailInput は Outbox への挿入入力（本文は enqueue 時にレンダリング済み）
type EnqueueEmailInput struct {
	ReservationID uuid.UUID
	MailType      MailType
	FromAddress   string
	ToAddress     string
	Subject       string
	BodyHTML      string
	BodyText      string
}

// IdempotencyKey は Resend 等へ渡す論理イベントキー（mail_type/reservation_id）
func IdempotencyKey(mailType MailType, reservationID uuid.UUID) string {
	return string(mailType) + "/" + reservationID.String()
}

// EmailOutboxRepository は Outbox の永続化抽象（Resend 非依存）
type EmailOutboxRepository interface {
	Enqueue(ctx context.Context, in EnqueueEmailInput) error
	// ClaimNextPending は送信可能な 1 行をロックして返す。対象が無ければ (nil, nil)
	ClaimNextPending(ctx context.Context) (*EmailOutboxMessage, error)
	MarkSent(ctx context.Context, id uuid.UUID) error
	MarkRetry(ctx context.Context, id uuid.UUID, attemptCount int, nextAttemptAt time.Time, lastError string) error
	MarkFailed(ctx context.Context, id uuid.UUID, attemptCount int, lastError string) error
}
