package usecase

import "github.com/cergijame101007/satehits/internal/domain"

// MailNotifier は予約関連メール通知（fire-and-forget）
type MailNotifier interface {
	ReservationReceived(r domain.Reservation)
	ReservationApproved(r domain.Reservation)
	ReservationRejected(r domain.Reservation, reason string)
}

// NoOpMailNotifier はメール送信を行わない
type NoOpMailNotifier struct{}

func (NoOpMailNotifier) ReservationReceived(domain.Reservation)         {}
func (NoOpMailNotifier) ReservationApproved(domain.Reservation)         {}
func (NoOpMailNotifier) ReservationRejected(domain.Reservation, string) {}
