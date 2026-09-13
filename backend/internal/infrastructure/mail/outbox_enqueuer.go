package mail

import (
	"context"
	"errors"
	"fmt"

	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/domain"
)

// ErrOwnerAddressNotConfigured はオーナー宛メールの宛先（MAIL_OWNER_ADDRESS）が未設定
var ErrOwnerAddressNotConfigured = errors.New("owner mail address is not configured")

// EnqueuerConfig は OutboxEnqueuer が本文・宛先に使う設定
type EnqueuerConfig struct {
	// FromAddress は全メール共通の送信元
	FromAddress string
	// OwnerAddress はオーナー向けメール（UC-S04）の宛先。空ならオーナー向け enqueue はエラー
	OwnerAddress string
	// AdminURL はオーナー向けメール本文に載せる管理画面 URL
	AdminURL string
}

// OutboxEnqueuer は予約関連メールを Outbox へ記録する
type OutboxEnqueuer struct {
	repo     domain.EmailOutboxRepository
	from     string
	owner    string
	adminURL string
}

// NewOutboxEnqueuer は OutboxEnqueuer を生成する
func NewOutboxEnqueuer(repo domain.EmailOutboxRepository, cfg EnqueuerConfig) *OutboxEnqueuer {
	return &OutboxEnqueuer{repo: repo, from: cfg.FromAddress, owner: cfg.OwnerAddress, adminURL: cfg.AdminURL}
}

var (
	_ reservationusecase.MailEnqueuer            = (*OutboxEnqueuer)(nil)
	_ reservationusecase.PendingReminderEnqueuer = (*OutboxEnqueuer)(nil)
)

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

// EnqueuePendingReminder は UC-S04 オーナー向け pending リマインドを Outbox へ記録する
func (e *OutboxEnqueuer) EnqueuePendingReminder(ctx context.Context, r domain.Reservation, mailType domain.MailType, totalPending int) error {
	if e.owner == "" {
		return ErrOwnerAddressNotConfigured
	}
	content := buildPendingReminder(r, totalPending, e.adminURL)
	return e.enqueueTo(ctx, r, mailType, e.owner, content)
}

func (e *OutboxEnqueuer) enqueue(ctx context.Context, r domain.Reservation, mailType domain.MailType, content reservationMailContent) error {
	return e.enqueueTo(ctx, r, mailType, r.Email, content)
}

func (e *OutboxEnqueuer) enqueueTo(ctx context.Context, r domain.Reservation, mailType domain.MailType, to string, content reservationMailContent) error {
	err := e.repo.Enqueue(ctx, domain.EnqueueEmailInput{
		ReservationID: r.ID,
		MailType:      mailType,
		FromAddress:   e.from,
		ToAddress:     to,
		Subject:       content.Subject,
		BodyHTML:      content.HTML,
		BodyText:      content.Text,
	})
	if err != nil {
		return fmt.Errorf("enqueue %s: %w", mailType, err)
	}
	return nil
}
