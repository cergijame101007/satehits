package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

type resolveMonthStubRepo struct{}

func (resolveMonthStubRepo) Upsert(context.Context, domain.SetScheduleInput) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (resolveMonthStubRepo) FindByDate(context.Context, datetime.Date) (domain.Schedule, bool, error) {
	return domain.Schedule{}, false, nil
}

func (resolveMonthStubRepo) ListStoredByYearMonth(context.Context, int, int) ([]domain.Schedule, error) {
	return nil, nil
}

func TestScheduleResolver_ResolveForDate_rejectsZeroDate(t *testing.T) {
	r := NewScheduleResolver(resolveMonthStubRepo{})
	_, err := r.ResolveForDate(context.Background(), datetime.Date{})
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

func TestValidateYearMonth(t *testing.T) {
	tests := []struct {
		name      string
		year      int
		month     int
		wantField string
	}{
		{name: "accepts valid year and month", year: 2026, month: 2, wantField: ""},
		{name: "rejects year below range", year: 1999, month: 1, wantField: "year"},
		{name: "rejects month above range", year: 2026, month: 13, wantField: "month"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateYearMonth(tt.year, tt.month)
			if tt.wantField == "" {
				if len(got) != 0 {
					t.Fatalf("violations = %+v, want none", got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatal("violations empty, want at least one")
			}
			if got[0].Field != tt.wantField {
				t.Fatalf("Field = %q, want %q", got[0].Field, tt.wantField)
			}
		})
	}
}

func TestScheduleResolver_ResolveMonth_rejectsInvalidYearMonth(t *testing.T) {
	r := NewScheduleResolver(resolveMonthStubRepo{})
	_, err := r.ResolveMonth(context.Background(), 1999, 1)
	if err == nil {
		t.Fatal("err = nil, want YearMonthValidationError")
	}
	var ymErr *YearMonthValidationError
	if !errors.As(err, &ymErr) {
		t.Fatalf("err type = %T, want *YearMonthValidationError", err)
	}
	if ymErr.Violations[0].Field != "year" {
		t.Fatalf("Field = %q, want year", ymErr.Violations[0].Field)
	}
}

func TestSynthesizeFromStoreCalendar_weekdayDefaults(t *testing.T) {
	tests := []struct {
		name         string
		date         string
		wantType     string
		wantCapacity int
	}{
		{name: "monday is normal", date: "2026-05-18", wantType: "normal", wantCapacity: 10},
		{name: "thursday is closed", date: "2026-05-21", wantType: "closed", wantCapacity: 0},
		{name: "saturday is morning", date: "2026-05-16", wantType: "morning", wantCapacity: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := datetime.MustParseDate(tt.date)
			got := synthesizeFromStoreCalendar(d)
			if got.ScheduleType != tt.wantType {
				t.Fatalf("ScheduleType = %q, want %q", got.ScheduleType, tt.wantType)
			}
			if got.Capacity != tt.wantCapacity {
				t.Fatalf("Capacity = %d, want %d", got.Capacity, tt.wantCapacity)
			}
			if got.Date.String() != tt.date {
				t.Fatalf("Date = %s, want %s", got.Date, tt.date)
			}
		})
	}
}

func TestSynthesizeFromStoreCalendar_appliesBusinessHoursForNormal(t *testing.T) {
	d := datetime.MustParseDate("2026-05-18")
	if d.Weekday() != time.Monday {
		t.Fatalf("test date weekday = %v, want Monday", d.Weekday())
	}
	got := synthesizeFromStoreCalendar(d)
	if got.OpenTime.String() != "11:30" {
		t.Fatalf("OpenTime = %s, want 11:30", got.OpenTime)
	}
}
