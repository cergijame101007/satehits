package service

import (
	"context"
	"errors"
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

func (r availabilityReservationRepo) SumReservedPeopleByDate(_ context.Context, date datetime.Date) (int, error) {
	if r.approvedByDate == nil {
		return 0, nil
	}
	return r.approvedByDate[date.String()], nil
}

func (r availabilityReservationRepo) SumReservedPeopleByDateRange(_ context.Context, from, to datetime.Date) (map[string]int, error) {
	if r.approvedByDate == nil {
		return map[string]int{}, nil
	}
	result := make(map[string]int)
	fromStr := from.String()
	toStr := to.String()
	for dateStr, count := range r.approvedByDate {
		if dateStr >= fromStr && dateStr <= toStr {
			result[dateStr] = count
		}
	}
	return result, nil
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

func TestAvailabilityService_ResolveForDate_rejectsZeroDate(t *testing.T) {
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{})
	_, err := svc.ResolveForDate(context.Background(), datetime.Date{})
	if err == nil {
		t.Fatal("err = nil, want DateValidationError")
	}
	var dateErr *DateValidationError
	if !errors.As(err, &dateErr) {
		t.Fatalf("err type = %T, want *DateValidationError", err)
	}
	if dateErr.Violations[0].Field != "date" {
		t.Fatalf("Field = %q, want date", dateErr.Violations[0].Field)
	}
}

func TestAvailabilityService_ResolveForDate_boundaries(t *testing.T) {
	date := datetime.MustParseDate("2026-05-18") // 月曜

	tests := []struct {
		name             string
		schedule         domain.Schedule
		approved         int
		wantCapacity     int
		wantReserved     int
		wantAvailable    int
		wantHoliday      bool
		wantScheduleType string
	}{
		{
			name:             "returns full capacity when no approved reservations",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeNormal, Capacity: 10},
			approved:         0,
			wantCapacity:     10,
			wantReserved:     0,
			wantAvailable:    10,
			wantScheduleType: domain.ScheduleTypeNormal,
		},
		{
			name:             "available is zero when reserved equals capacity",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeNormal, Capacity: 10},
			approved:         10,
			wantCapacity:     10,
			wantReserved:     10,
			wantAvailable:    0,
			wantScheduleType: domain.ScheduleTypeNormal,
		},
		{
			name:             "available is one when one seat remains",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeNormal, Capacity: 10},
			approved:         9,
			wantCapacity:     10,
			wantReserved:     9,
			wantAvailable:    1,
			wantScheduleType: domain.ScheduleTypeNormal,
		},
		{
			name:             "clips available when reserved exceeds capacity",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeNormal, Capacity: 10},
			approved:         11,
			wantCapacity:     10,
			wantReserved:     11,
			wantAvailable:    0,
			wantScheduleType: domain.ScheduleTypeNormal,
		},
		{
			name:             "capacity zero on bookable event day is not holiday",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeEvent, Capacity: 0, EventName: "evt"},
			approved:         0,
			wantCapacity:     0,
			wantAvailable:    0,
			wantScheduleType: domain.ScheduleTypeEvent,
		},
		{
			name:          "closed day ignores approved count in response",
			schedule:      domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeClosed, Capacity: 0},
			approved:      5,
			wantHoliday:   true,
			wantCapacity:  0,
			wantReserved:  0,
			wantAvailable: 0,
		},
		{
			name:             "special_menu preserves schedule type and capacity",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeSpecialMenu, Capacity: 7},
			approved:         3,
			wantCapacity:     7,
			wantReserved:     3,
			wantAvailable:    4,
			wantScheduleType: domain.ScheduleTypeSpecialMenu,
		},
		{
			name:             "minimum capacity one with no reservations",
			schedule:         domain.Schedule{Date: date, ScheduleType: domain.ScheduleTypeNormal, Capacity: 1},
			approved:         0,
			wantCapacity:     1,
			wantAvailable:    1,
			wantScheduleType: domain.ScheduleTypeNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newAvailabilityService(availabilityScheduleRepo{
				byDate: map[string]domain.Schedule{date.String(): tt.schedule},
			}, availabilityReservationRepo{
				approvedByDate: map[string]int{date.String(): tt.approved},
			})

			got, err := svc.ResolveForDate(context.Background(), date)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got.IsHoliday != tt.wantHoliday {
				t.Fatalf("IsHoliday = %v, want %v", got.IsHoliday, tt.wantHoliday)
			}
			if got.Capacity != tt.wantCapacity {
				t.Fatalf("Capacity = %d, want %d", got.Capacity, tt.wantCapacity)
			}
			if got.Reserved != tt.wantReserved {
				t.Fatalf("Reserved = %d, want %d", got.Reserved, tt.wantReserved)
			}
			if got.Available != tt.wantAvailable {
				t.Fatalf("Available = %d, want %d", got.Available, tt.wantAvailable)
			}
			if !tt.wantHoliday && got.ScheduleType != tt.wantScheduleType {
				t.Fatalf("ScheduleType = %q, want %q", got.ScheduleType, tt.wantScheduleType)
			}
		})
	}
}

func TestAvailabilityService_ResolveForDate_defaultSaturdayNormal(t *testing.T) {
	date := datetime.MustParseDate("2026-05-16") // 土曜
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{
		approvedByDate: map[string]int{date.String(): 4},
	})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.ScheduleType != domain.ScheduleTypeNormal {
		t.Fatalf("ScheduleType = %q, want normal", got.ScheduleType)
	}
	if got.Available != 6 {
		t.Fatalf("Available = %d, want 6", got.Available)
	}
}

func TestAvailabilityService_ResolveForDate_defaultSundayMorning(t *testing.T) {
	date := datetime.MustParseDate("2026-05-17") // 日曜
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{
		approvedByDate: map[string]int{date.String(): 4},
	})

	got, err := svc.ResolveForDate(context.Background(), date)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if got.ScheduleType != domain.ScheduleTypeMorning {
		t.Fatalf("ScheduleType = %q, want morning", got.ScheduleType)
	}
	if got.Available != 6 {
		t.Fatalf("Available = %d, want 6", got.Available)
	}
}

func TestAvailabilityService_ResolveMonth(t *testing.T) {
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{
		approvedByDate: map[string]int{
			"2026-05-18": 6,
			"2026-05-21": 3,
		},
	})

	got, err := svc.ResolveMonth(context.Background(), 2026, 5)
	if err != nil {
		t.Fatalf("ResolveMonth() err = %v", err)
	}
	if len(got) != 31 {
		t.Fatalf("len = %d, want 31", len(got))
	}

	byDate := make(map[string]Availability, len(got))
	for _, a := range got {
		byDate[a.Date.String()] = a
	}

	monday := byDate["2026-05-18"]
	if monday.Available != 4 {
		t.Fatalf("2026-05-18 Available = %d, want 4", monday.Available)
	}
	if monday.IsHoliday {
		t.Fatal("2026-05-18 IsHoliday = true, want false")
	}

	thursday := byDate["2026-05-21"]
	if !thursday.IsHoliday {
		t.Fatal("2026-05-21 IsHoliday = false, want true")
	}
	if thursday.Reserved != 0 {
		t.Fatalf("2026-05-21 Reserved = %d, want 0 on holiday", thursday.Reserved)
	}
}

func TestAvailabilityService_ResolveMonth_invalidYearMonth(t *testing.T) {
	svc := newAvailabilityService(availabilityScheduleRepo{}, availabilityReservationRepo{})

	_, err := svc.ResolveMonth(context.Background(), 1999, 5)
	if err == nil {
		t.Fatal("ResolveMonth() err = nil, want YearMonthValidationError")
	}
	var ymErr *YearMonthValidationError
	if !errors.As(err, &ymErr) {
		t.Fatalf("ResolveMonth() err = %v, want YearMonthValidationError", err)
	}
}
