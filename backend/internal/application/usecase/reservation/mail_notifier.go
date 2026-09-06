package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/domain"
)

// MailEnqueuer は予約関連メールを Outbox へ記録する（同一トランザクション参加）
type MailEnqueuer interface {
	EnqueueReservationReceived(ctx context.Context, r domain.Reservation) error
	EnqueueReservationApproved(ctx context.Context, r domain.Reservation) error
	EnqueueReservationRejected(ctx context.Context, r domain.Reservation, reason string) error
}

// NoOpMailEnqueuer はメール送信意図を記録しない
type NoOpMailEnqueuer struct{}

func (NoOpMailEnqueuer) EnqueueReservationReceived(context.Context, domain.Reservation) error {
	return nil
}
func (NoOpMailEnqueuer) EnqueueReservationApproved(context.Context, domain.Reservation) error {
	return nil
}
func (NoOpMailEnqueuer) EnqueueReservationRejected(context.Context, domain.Reservation, string) error {
	return nil
}
