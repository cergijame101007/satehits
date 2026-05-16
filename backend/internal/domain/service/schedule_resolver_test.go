package service

import (
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

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
