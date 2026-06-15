package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

type fakeReservationRepo struct {
	reservation domain.Reservation
	getErr      error
	updateErr   error
	lastStatus  string
}

func (f *fakeReservationRepo) Create(context.Context, domain.CreateReservationInput) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (f *fakeReservationRepo) GetAll(context.Context) ([]domain.Reservation, error) {
	return nil, nil
}

func (f *fakeReservationRepo) List(context.Context, domain.ListReservationsFilter) ([]domain.Reservation, error) {
	return nil, nil
}

func (f *fakeReservationRepo) GetByID(_ context.Context, id uuid.UUID) (domain.Reservation, error) {
	if f.getErr != nil {
		return domain.Reservation{}, f.getErr
	}
	if f.reservation.ID != id {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	return f.reservation, nil
}

func (f *fakeReservationRepo) UpdateStatus(_ context.Context, id uuid.UUID, status string) (domain.Reservation, error) {
	if f.updateErr != nil {
		return domain.Reservation{}, f.updateErr
	}
	f.lastStatus = status
	updated := f.reservation
	updated.Status = status
	return updated, nil
}

func TestUpdateReservationStatusUseCase(t *testing.T) {
	reservationID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name       string
		current    string
		target     string
		wantErr    bool
		wantField  string
		wantStatus string
	}{
		{name: "allows pending to approved", current: "pending", target: "approved", wantStatus: "approved"},
		{name: "allows pending to rejected", current: "pending", target: "rejected", wantStatus: "rejected"},
		{name: "allows approved to no_show", current: "approved", target: "no_show", wantStatus: "no_show"},
		{name: "rejects rejected to approved", current: "rejected", target: "approved", wantErr: true, wantField: "status"},
		{name: "rejects pending to no_show", current: "pending", target: "no_show", wantErr: true, wantField: "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeReservationRepo{
				reservation: domain.Reservation{ID: reservationID, Status: tt.current},
			}
			uc := NewUpdateReservationStatusUseCase(repo)

			result, err := uc.Execute(context.Background(), UpdateReservationStatusCommand{
				ID:     reservationID,
				Status: tt.target,
			})

			if tt.wantErr {
				if err == nil {
					t.Fatal("Execute() err = nil, want error")
				}
				var vErr *ValidationError
				if !errors.As(err, &vErr) {
					t.Fatalf("Execute() err = %v, want ValidationError", err)
				}
				assertHasViolationField(t, vErr.Violations, tt.wantField)
				return
			}

			if err != nil {
				t.Fatalf("Execute() err = %v, want nil", err)
			}
			if result.Status != tt.wantStatus {
				t.Fatalf("result.Status = %q, want %q", result.Status, tt.wantStatus)
			}
			if repo.lastStatus != tt.wantStatus {
				t.Fatalf("repo.lastStatus = %q, want %q", repo.lastStatus, tt.wantStatus)
			}
		})
	}
}
