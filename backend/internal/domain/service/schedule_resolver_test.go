package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// stubNationalHolidays はテスト用の祝日集合（同梱の内閣府 CSV に依存しない）
type stubNationalHolidays map[string]bool

func (s stubNationalHolidays) IsNationalHoliday(d datetime.Date) bool {
	return s[d.String()]
}

// testStoreCalendar は 2026 年の代表的な祝日（元日・春分の日・憲法記念日・振替休日）だけを持つ StoreCalendar
func testStoreCalendar() *StoreCalendar {
	return NewStoreCalendar(stubNationalHolidays{
		"2026-01-01": true, // 木曜
		"2026-03-20": true, // 金曜
		"2026-05-03": true, // 日曜
		"2026-05-06": true, // 水曜（振替休日）
	})
}

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

func (resolveMonthStubRepo) DeleteByDate(context.Context, datetime.Date) error {
	return nil
}

func TestScheduleResolver_ResolveForDate_rejectsZeroDate(t *testing.T) {
	r := NewScheduleResolver(resolveMonthStubRepo{}, testStoreCalendar())
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
	r := NewScheduleResolver(resolveMonthStubRepo{}, testStoreCalendar())
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

func TestStoreCalendar_DefaultSchedule_holidayAndWeekdayRules(t *testing.T) {
	tests := []struct {
		name         string
		date         string
		wantType     string
		wantCapacity int
	}{
		{name: "monday is normal", date: "2026-05-18", wantType: "normal", wantCapacity: 10},
		{name: "thursday is closed", date: "2026-05-21", wantType: "closed", wantCapacity: 0},
		{name: "friday is closed", date: "2026-05-22", wantType: "closed", wantCapacity: 0},
		{name: "saturday is normal", date: "2026-05-16", wantType: "normal", wantCapacity: 10},
		{name: "sunday is morning", date: "2026-05-17", wantType: "morning", wantCapacity: 10},
		// 祝日は testStoreCalendar の stub で与える（同梱 CSV には依存しない）
		{name: "thursday holiday is normal instead of closed", date: "2026-01-01", wantType: "normal", wantCapacity: 10},
		{name: "friday holiday is normal instead of closed", date: "2026-03-20", wantType: "normal", wantCapacity: 10},
		{name: "sunday holiday is normal without morning hours", date: "2026-05-03", wantType: "normal", wantCapacity: 10},
		{name: "wednesday substitute holiday stays normal", date: "2026-05-06", wantType: "normal", wantCapacity: 10},
		{name: "thursday not in holiday set falls back to closed", date: "2026-01-08", wantType: "closed", wantCapacity: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := datetime.MustParseDate(tt.date)
			got := testStoreCalendar().DefaultSchedule(d)
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

func TestStoreCalendar_DefaultSchedule_appliesBusinessHours(t *testing.T) {
	tests := []struct {
		name        string
		date        string
		wantWeekday time.Weekday
		wantOpen    string
	}{
		{name: "monday opens at lunch", date: "2026-05-18", wantWeekday: time.Monday, wantOpen: "11:30"},
		{name: "sunday opens with morning hours", date: "2026-05-17", wantWeekday: time.Sunday, wantOpen: "08:30"},
		{name: "sunday holiday opens at lunch without morning hours", date: "2026-05-03", wantWeekday: time.Sunday, wantOpen: "11:30"},
		{name: "thursday holiday opens at lunch", date: "2026-01-01", wantWeekday: time.Thursday, wantOpen: "11:30"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := datetime.MustParseDate(tt.date)
			if d.Weekday() != tt.wantWeekday {
				t.Fatalf("test date weekday = %v, want %v", d.Weekday(), tt.wantWeekday)
			}
			got := testStoreCalendar().DefaultSchedule(d)
			if got.OpenTime.String() != tt.wantOpen {
				t.Fatalf("OpenTime = %s, want %s", got.OpenTime, tt.wantOpen)
			}
		})
	}
}

type resolveHolidayOverrideRepo struct {
	resolveMonthStubRepo
	stored domain.Schedule
}

func (r resolveHolidayOverrideRepo) FindByDate(_ context.Context, d datetime.Date) (domain.Schedule, bool, error) {
	if d.String() == r.stored.Date.String() {
		return r.stored, true, nil
	}
	return domain.Schedule{}, false, nil
}

func TestScheduleResolver_ResolveForDate_storedRowWinsOverHolidayDefault(t *testing.T) {
	// 祝日でも daily_schedules 行があればそちらが優先（行優先は祝日対応後も変えない）
	holidayDate := datetime.MustParseDate("2026-01-01")
	stored := domain.Schedule{Date: holidayDate, ScheduleType: "closed", Capacity: 0}
	r := NewScheduleResolver(resolveHolidayOverrideRepo{stored: stored}, testStoreCalendar())

	got, err := r.ResolveForDate(context.Background(), holidayDate)
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if got.IsDefault {
		t.Fatal("IsDefault = true, want false")
	}
	if got.Schedule.ScheduleType != "closed" {
		t.Fatalf("ScheduleType = %q, want closed", got.Schedule.ScheduleType)
	}
}
