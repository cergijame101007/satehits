package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

type fakeMailEnqueuer struct {
	received   int
	approved   int
	rejected   int
	lastReason string
	err        error
}

func (f *fakeMailEnqueuer) EnqueueReservationReceived(context.Context, domain.Reservation) error {
	f.received++
	return f.err
}

func (f *fakeMailEnqueuer) EnqueueReservationApproved(context.Context, domain.Reservation) error {
	f.approved++
	return f.err
}

func (f *fakeMailEnqueuer) EnqueueReservationRejected(_ context.Context, _ domain.Reservation, reason string) error {
	f.rejected++
	f.lastReason = reason
	return f.err
}

func TestUpdateReservationStatusUseCaseSendsApprovedMail(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	enqueuer := &fakeMailEnqueuer{}
	uc := NewUpdateReservationStatusUseCase(repo, passThroughTxManager{}, enqueuer)

	_, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "approved",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if enqueuer.approved != 1 {
		t.Fatalf("approved enqueues = %d, want 1", enqueuer.approved)
	}
}

func TestUpdateReservationStatusUseCaseSendsRejectedMailWithReason(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	enqueuer := &fakeMailEnqueuer{}
	uc := NewUpdateReservationStatusUseCase(repo, passThroughTxManager{}, enqueuer)

	_, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "rejected",
		Reason: "定員超過",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if enqueuer.rejected != 1 {
		t.Fatalf("rejected enqueues = %d, want 1", enqueuer.rejected)
	}
	if enqueuer.lastReason != "定員超過" {
		t.Fatalf("lastReason = %q, want 定員超過", enqueuer.lastReason)
	}
}

func TestUpdateReservationStatusUseCaseDoesNotSendMailForCancelled(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	enqueuer := &fakeMailEnqueuer{}
	uc := NewUpdateReservationStatusUseCase(repo, passThroughTxManager{}, enqueuer)

	_, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "cancelled",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v", err)
	}
	if enqueuer.received+enqueuer.approved+enqueuer.rejected != 0 {
		t.Fatalf("unexpected mail enqueues: %+v", enqueuer)
	}
}

func TestUpdateReservationStatusUseCaseContinuesWhenAlreadyEnqueued(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	repo := &fakeReservationRepo{
		reservation: domain.Reservation{ID: reservationID, Status: "pending"},
	}
	enqueuer := &fakeMailEnqueuer{err: domain.ErrMailAlreadyEnqueued}
	uc := NewUpdateReservationStatusUseCase(repo, passThroughTxManager{}, enqueuer)

	result, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
		ID:     reservationID,
		Status: "approved",
	})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil (already enqueued is non-fatal)", err)
	}
	if result.Status != "approved" {
		t.Fatalf("result.Status = %q, want approved", result.Status)
	}
	if repo.lastStatus != "approved" {
		t.Fatalf("repo.lastStatus = %q, want approved", repo.lastStatus)
	}
}
