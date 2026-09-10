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

// Outbox 行の status。pending → sent / failed のみで、sent と failed は終端
const (
	OutboxStatusPending = "pending"
	OutboxStatusSent    = "sent"
	OutboxStatusFailed  = "failed"
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

// EmailOutboxRepository は Outbox の永続化抽象（Resend 非依存）。
//
// 送信は claim → 送信 → mark の 3 段階で、claim と mark はそれぞれ独立してコミットされる（lease 方式）。
// claim は attempt_count を +1 し next_attempt_at を leaseUntil まで進めてからコミットするため、
// 送信中にプロセスが落ちても試行回数は消費済みで、lease 切れ後に再 claim される。
type EmailOutboxRepository interface {
	Enqueue(ctx context.Context, in EnqueueEmailInput) error
	// ClaimNextPending は送信可能な 1 行を claim して返す。対象が無ければ (nil, nil)。
	// 返る AttemptCount は今回の試行を含む値（初回は 1）
	ClaimNextPending(ctx context.Context, leaseUntil time.Time) (*EmailOutboxMessage, error)
	// MarkSent は送信成功として sent にする
	MarkSent(ctx context.Context, id uuid.UUID) error
	// MarkRetry は一時失敗として次回試行時刻を設定する（attempt_count は claim 時に加算済み）
	MarkRetry(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error
	// MarkFailed は恒久失敗または再送上限として failed にする
	MarkFailed(ctx context.Context, id uuid.UUID, lastError string) error
	// ReleaseClaim は claim で消費した試行を戻し、次回試行時刻を設定する（認証エラー等、メッセージ起因でない失敗用）
	ReleaseClaim(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time, lastError string) error
}
