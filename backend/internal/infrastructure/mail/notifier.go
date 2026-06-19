package mail

import (
	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/domain"
)

// ReservationNotifier は予約関連メールを非同期キューへ投入する
type ReservationNotifier struct {
	queue *Queue
	from  string
}

// NewReservationNotifier は ReservationNotifier を生成する
func NewReservationNotifier(queue *Queue, fromAddress string) *ReservationNotifier {
	return &ReservationNotifier{queue: queue, from: fromAddress}
}

var _ reservationusecase.MailNotifier = (*ReservationNotifier)(nil)

// ReservationReceived は UC-S01 受付メールをキューへ投入する
func (n *ReservationNotifier) ReservationReceived(r domain.Reservation) {
	content := buildReservationReceived(r)
	n.enqueue(r.Email, content)
}

// ReservationApproved は UC-S02 承認メールをキューへ投入する
func (n *ReservationNotifier) ReservationApproved(r domain.Reservation) {
	content := buildReservationApproved(r)
	n.enqueue(r.Email, content)
}

// ReservationRejected は UC-S03 拒否メールをキューへ投入する
func (n *ReservationNotifier) ReservationRejected(r domain.Reservation, reason string) {
	content := buildReservationRejected(r, reason)
	n.enqueue(r.Email, content)
}

func (n *ReservationNotifier) enqueue(to string, content reservationMailContent) {
	n.queue.Enqueue(domain.MailMessage{
		From:    n.from,
		To:      to,
		Subject: content.Subject,
		HTML:    content.HTML,
		Text:    content.Text,
	})
}
