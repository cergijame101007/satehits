package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

type getAvailScheduleRepo struct {
	byDate map[string]domain.Schedule
}

func (r getAvailScheduleRepo) Upsert(context.Context, domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (r getAvailScheduleRepo) FindByDate(_ context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	s, ok := r.byDate[date.String()]
	return s, ok, nil
}

func (r getAvailScheduleRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

type getAvailReservationRepo struct {
	approvedByDate map[string]int
}

func (r getAvailReservationRepo) Create(context.Context, domain.CreateReservationInput) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r getAvailReservationRepo) GetAll(context.Context) ([]domain.Reservation, error) {
	return nil, nil
}

func (r getAvailReservationRepo) List(context.Context, domain.ListReservationsFilter) ([]domain.Reservation, error) {
	return nil, nil
}

func (r getAvailReservationRepo) GetByID(context.Context, uuid.UUID) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r getAvailReservationRepo) UpdateStatus(context.Context, uuid.UUID, string) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r getAvailReservationRepo) SumApprovedPeopleByDate(_ context.Context, date datetime.Date) (int, error) {
	if r.approvedByDate == nil {
		return 0, nil
	}
	return r.approvedByDate[date.String()], nil
}

func newGetAvailabilityUseCaseForTest(sched getAvailScheduleRepo, res getAvailReservationRepo) *GetAvailabilityUseCase {
	resolver := service.NewScheduleResolver(sched)
	avail := service.NewAvailabilityService(resolver, res)
	return NewGetAvailabilityUseCase(avail)
}

func TestGetAvailabilityUseCase_Execute(t *testing.T) {
	date := datetime.MustParseDate("2026-05-18")

	t.Run("returns availability for bookable day", func(t *testing.T) {
		uc := newGetAvailabilityUseCaseForTest(getAvailScheduleRepo{}, getAvailReservationRepo{
			approvedByDate: map[string]int{date.String(): 4},
		})

		got, err := uc.Execute(context.Background(), date)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if got.Available != 6 {
			t.Fatalf("Available = %d, want 6", got.Available)
		}
		if got.IsHoliday {
			t.Fatal("IsHoliday = true, want false")
		}
	})

	t.Run("returns validation error for zero date", func(t *testing.T) {
		uc := newGetAvailabilityUseCaseForTest(getAvailScheduleRepo{}, getAvailReservationRepo{})

		_, err := uc.Execute(context.Background(), datetime.Date{})
		if err == nil {
			t.Fatal("Execute() err = nil, want ValidationError")
		}
		var vErr *ValidationError
		if !errors.As(err, &vErr) {
			t.Fatalf("Execute() err = %v, want ValidationError", err)
		}
		assertHasViolationField(t, vErr.Violations, "date")
	})

	t.Run("returns holiday availability with zero counts", func(t *testing.T) {
		closed := datetime.MustParseDate("2026-05-21")
		uc := newGetAvailabilityUseCaseForTest(getAvailScheduleRepo{}, getAvailReservationRepo{
			approvedByDate: map[string]int{closed.String(): 5},
		})

		got, err := uc.Execute(context.Background(), closed)
		if err != nil {
			t.Fatalf("Execute() err = %v, want nil", err)
		}
		if !got.IsHoliday {
			t.Fatal("IsHoliday = false, want true")
		}
		if got.Reserved != 0 {
			t.Fatalf("Reserved = %d, want 0 on holiday", got.Reserved)
		}
	})
}
