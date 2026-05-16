package usecase

import (
	"context"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

func TestGetScheduleUseCase_Execute_returnsStoredRow(t *testing.T) {
	d := datetime.MustParseDate("2026-03-01")
	repo := &listSchedulesStubRepo{
		byDate: map[string]domain.Schedule{
			d.String(): {Date: d, ScheduleType: "special_menu", Capacity: 8},
		},
	}
	uc := NewGetScheduleUseCase(service.NewScheduleResolver(repo))

	res, err := uc.Execute(context.Background(), d)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if res.IsDefault {
		t.Fatal("IsDefault = true, want false")
	}
	if res.Schedule.ScheduleType != "special_menu" {
		t.Fatalf("ScheduleType = %q, want special_menu", res.Schedule.ScheduleType)
	}
}

func TestGetScheduleUseCase_Execute_returnsDomainDefaultWhenMissing(t *testing.T) {
	d := datetime.MustParseDate("2026-03-05") // Thursday
	repo := &listSchedulesStubRepo{byDate: map[string]domain.Schedule{}}
	uc := NewGetScheduleUseCase(service.NewScheduleResolver(repo))

	res, err := uc.Execute(context.Background(), d)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if !res.IsDefault {
		t.Fatal("IsDefault = false, want true")
	}
	if res.Schedule.ScheduleType != "closed" {
		t.Fatalf("ScheduleType = %q, want closed", res.Schedule.ScheduleType)
	}
}
