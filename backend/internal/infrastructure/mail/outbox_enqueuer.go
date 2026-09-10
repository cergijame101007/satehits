package mail

import (
	"context"
	"fmt"

	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/domain"
)

// OutboxEnqueuer は予約関連メールを Outbox へ記録する
type OutboxEnqueuer struct {
	repo domain.EmailOutboxRepository
	from string
}

// NewOutboxEnqueuer は OutboxEnqueuer を生成する
func NewOutboxEnqueuer(repo domain.EmailOutboxRepository, fromAddress string) *OutboxEnqueuer {
	return &OutboxEnqueuer{repo: repo, from: fromAddress}
}

var _ reservationusecase.MailEnqueuer = (*OutboxEnqueuer)(nil)

// EnqueueReservationReceived は UC-S01 受付メールを Outbox へ記録する
func (e *OutboxEnqueuer) EnqueueReservationReceived(ctx context.Context, r domain.Reservation) error {
	content := buildReservationReceived(r)
	return e.enqueue(ctx, r, domain.MailTypeReservationReceived, content)
}

// EnqueueReservationApproved は UC-S02 承認メールを Outbox へ記録する
func (e *OutboxEnqueuer) EnqueueReservationApproved(ctx context.Context, r domain.Reservation) error {
	content := buildReservationApproved(r)
	return e.enqueue(ctx, r, domain.MailTypeReservationApproved, content)
}

// EnqueueReservationRejected は UC-S03 拒否メールを Outbox へ記録する
func (e *OutboxEnqueuer) EnqueueReservationRejected(ctx context.Context, r domain.Reservation, reason string) error {
	content := buildReservationRejected(r, reason)
	return e.enqueue(ctx, r, domain.MailTypeReservationRejected, content)
}

func (e *OutboxEnqueuer) enqueue(ctx context.Context, r domain.Reservation, mailType domain.MailType, content reservationMailContent) error {
	err := e.repo.Enqueue(ctx, domain.EnqueueEmailInput{
		ReservationID: r.ID,
		MailType:      mailType,
		FromAddress:   e.from,
		ToAddress:     r.Email,
		Subject:       content.Subject,
		BodyHTML:      content.HTML,
		BodyText:      content.Text,
	})
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", mailType, err)
	}
	return nil
}
