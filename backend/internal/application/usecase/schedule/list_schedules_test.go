package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

type listSchedulesStubRepo struct {
	byDate map[string]domain.Schedule
}

func (s *listSchedulesStubRepo) Upsert(ctx context.Context, in domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (s *listSchedulesStubRepo) FindByDate(ctx context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	sch, ok := s.byDate[date.String()]
	return sch, ok, nil
}

func (s *listSchedulesStubRepo) ListStoredByYearMonth(ctx context.Context, year, month int) ([]domain.Schedule, error) {
	prefix := fmt.Sprintf("%04d-%02d", year, month)
	var list []domain.Schedule
	for key, sch := range s.byDate {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			list = append(list, sch)
		}
	}
	return list, nil
}

func (s *listSchedulesStubRepo) DeleteByDate(ctx context.Context, date datetime.Date) error {
	return nil
}

func TestListSchedulesUseCase_Execute_mergesStoredAndDefaults(t *testing.T) {
	storedDate := datetime.MustParseDate("2026-02-11")
	repo := &listSchedulesStubRepo{
		byDate: map[string]domain.Schedule{
			storedDate.String(): {
				Date:         storedDate,
				ScheduleType: "event",
				Capacity:     5,
			},
		},
	}
	uc := NewListSchedulesUseCase(service.NewScheduleResolver(repo))

	res, err := uc.Execute(context.Background(), 2026, 2)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if len(res.Schedules) != 28 {
		t.Fatalf("len(Schedules) = %d, want 28", len(res.Schedules))
	}

	var eventDay *service.EffectiveSchedule
	for i := range res.Schedules {
		if res.Schedules[i].Schedule.Date.String() == "2026-02-11" {
			eventDay = &res.Schedules[i]
			break
		}
	}
	if eventDay == nil {
		t.Fatal("2026-02-11 not found in schedules")
	}
	if eventDay.IsDefault {
		t.Fatal("IsDefault = true, want false for stored row")
	}
	if eventDay.Schedule.ScheduleType != "event" {
		t.Fatalf("ScheduleType = %q, want event", eventDay.Schedule.ScheduleType)
	}

	if !res.Schedules[0].IsDefault {
		t.Fatal("first day IsDefault = false, want true when not stored")
	}
	if res.Schedules[0].Schedule.ScheduleType == "" {
		t.Fatal("first day schedule type empty")
	}
}
