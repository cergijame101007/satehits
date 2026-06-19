package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

type fakeMailNotifier struct {
	received   int
	approved   int
	rejected   int
	lastReason string
}

func (f *fakeMailNotifier) ReservationReceived(domain.Reservation) {
	f.received++
}

func (f *fakeMailNotifier) ReservationApproved(domain.Reservation) {
	f.approved++
}

func (f *fakeMailNotifier) ReservationRejected(_ domain.Reservation, reason string) {
	f.rejected++
	f.lastReason = reason
}

func TestUpdateReservationStatusUseCaseSendsApprovedMail(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	notifier := &fakeMailNotifier{}
	uc := NewUpdateReservationStatusUseCase(repo, notifier)

	_, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "approved",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if notifier.approved != 1 {
		t.Fatalf("approved notifications = %d, want 1", notifier.approved)
	}
}

func TestUpdateReservationStatusUseCaseSendsRejectedMailWithReason(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	notifier := &fakeMailNotifier{}
	uc := NewUpdateReservationStatusUseCase(repo, notifier)

	_, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "rejected",
		Reason: "定員超過",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if notifier.rejected != 1 {
		t.Fatalf("rejected notifications = %d, want 1", notifier.rejected)
	}
	if notifier.lastReason != "定員超過" {
		t.Fatalf("lastReason = %q, want 定員超過", notifier.lastReason)
	}
}

func TestUpdateReservationStatusUseCaseDoesNotSendMailForCancelled(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	notifier := &fakeMailNotifier{}
	uc := NewUpdateReservationStatusUseCase(repo, notifier)

	_, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "cancelled",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if notifier.received+notifier.approved+notifier.rejected != 0 {
		t.Fatalf("unexpected mail notifications: %+v", notifier)
	}
}
