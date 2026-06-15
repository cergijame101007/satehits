package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

type availabilityScheduleRepo struct {
	byDate map[string]domain.Schedule
}

func (r availabilityScheduleRepo) Upsert(context.Context, domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (r availabilityScheduleRepo) FindByDate(_ context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	s, ok := r.byDate[date.String()]
	return s, ok, nil
}

func (r availabilityScheduleRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

type availabilityReservationRepo struct {
	approvedByDate map[string]int
}

func (r availabilityReservationRepo) Create(context.Context, domain.CreateReservationInput) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r availabilityReservationRepo) GetAll(context.Context) ([]domain.Reservation, error) {
	return nil, nil
}

func (r availabilityReservationRepo) List(context.Context, domain.ListReservationsFilter) ([]domain.Reservation, error) {
	return nil, nil
}

func (r availabilityReservationRepo) GetByID(context.Context, uuid.UUID) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r availabilityReservationRepo) UpdateStatus(context.Context, uuid.UUID, string) (domain.Reservation, error) {
	return domain.Reservation{}, nil
}

func (r availabilityReservationRepo) SumApprovedPeopleByDate(_ context.Context, date datetime.Date) (int, error) {
	if r.approvedByDate == nil {
		return 0, nil
	}
	return r.approvedByDate[date.String()], nil
}

func newAvailabilityService(sched availabilityScheduleRepo, res availabilityReservationRepo) *AvailabilityService {
	return NewAvailabilityService(NewScheduleResolver(sched), res)
}

func TestAvailabilityService_ResolveForDate_normalDay(t *testing.T) {
	date := datetime.MustParseDate("2026-05-18") // 月曜・定例 normal
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{
		approvedByDate: map[string]int{date.String(): 6},
	})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Capacity != 10 {
		t.Fatalf("Capacity = %d, want 10", got.Capacity)
	}
	if got.Reserved != 6 {
		t.Fatalf("Reserved = %d, want 6", got.Reserved)
	}
	if got.Available != 4 {
		t.Fatalf("Available = %d, want 4", got.Available)
	}
	if got.ScheduleType != domain.ScheduleTypeNormal {
		t.Fatalf("ScheduleType = %q, want normal", got.ScheduleType)
	}
	if got.IsHoliday {
		t.Fatal("IsHoliday = true, want false")
	}
}

func TestAvailabilityService_ResolveForDate_eventDayCapacityZero(t *testing.T) {
	date := datetime.MustParseDate("2026-02-11")
	svc := newAvailabilityService(availabilityScheduleRepo{
		byDate: map[string]domain.Schedule{
			date.String(): {
				Date:             date,
				ScheduleType:     domain.ScheduleTypeEvent,
				Capacity:         0,
				EventName:        "和紅茶をしばく会",
				EventDescription: "入門編",
			},
		},
	}, availabilityReservationRepo{})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.ScheduleType != domain.ScheduleTypeEvent {
		t.Fatalf("ScheduleType = %q, want event", got.ScheduleType)
	}
	if got.EventName != "和紅茶をしばく会" {
		t.Fatalf("EventName = %q", got.EventName)
	}
	if got.Available != 0 {
		t.Fatalf("Available = %d, want 0", got.Available)
	}
	if got.IsHoliday {
		t.Fatal("IsHoliday = true, want false")
	}
}

func TestAvailabilityService_ResolveForDate_holiday(t *testing.T) {
	date := datetime.MustParseDate("2026-05-21") // 木曜・定例 closed
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{
		approvedByDate: map[string]int{date.String(): 3},
	})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if !got.IsHoliday {
		t.Fatal("IsHoliday = false, want true")
	}
	if got.ScheduleType != "" {
		t.Fatalf("ScheduleType = %q, want empty", got.ScheduleType)
	}
	if got.Capacity != 0 || got.Reserved != 0 || got.Available != 0 {
		t.Fatalf("capacity/reserved/available = %d/%d/%d, want 0/0/0", got.Capacity, got.Reserved, got.Available)
	}
}

func TestAvailabilityService_ResolveForDate_clipsAvailableAtZero(t *testing.T) {
	date := datetime.MustParseDate("2026-05-18")
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{
		approvedByDate: map[string]int{date.String(): 12},
	})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.Available != 0 {
		t.Fatalf("Available = %d, want 0 (clipped)", got.Available)
	}
}

func TestAvailabilityService_ResolveForDate_storedRowOverridesDefault(t *testing.T) {
	date := datetime.MustParseDate("2026-05-21") // 定例は木曜 closed
	svc := newAvailabilityService(availabilityScheduleRepo{
		byDate: map[string]domain.Schedule{
			date.String(): {
				Date:         date,
				ScheduleType: domain.ScheduleTypeNormal,
				Capacity:     8,
			},
		},
	}, availabilityReservationRepo{
		approvedByDate: map[string]int{date.String(): 2},
	})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.IsHoliday {
		t.Fatal("IsHoliday = true, want false (DB row overrides default closed)")
	}
	if got.Capacity != 8 {
		t.Fatalf("Capacity = %d, want 8", got.Capacity)
	}
	if got.Available != 6 {
		t.Fatalf("Available = %d, want 6", got.Available)
	}
}
