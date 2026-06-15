package service

import (
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

func TestApplyDefaultBusinessHours(t *testing.T) {
	tests := []struct {
		name         string
		scheduleType string
		openTime     string
		lastOrder    string
		closeTime    string
		wantOpen     string
		wantLast     string
		wantClose    string
	}{
		{
			name:         "fills all times for normal when omitted",
			scheduleType: "normal",
			wantOpen:     "11:30",
			wantLast:     "13:30",
			wantClose:    "15:00",
		},
		{
			name:         "fills all times for special_menu when omitted",
			scheduleType: "special_menu",
			wantOpen:     "11:30",
			wantLast:     "13:30",
			wantClose:    "15:00",
		},
		{
			name:         "fills all times for morning when omitted",
			scheduleType: "morning",
			wantOpen:     "08:30",
			wantLast:     "13:30",
			wantClose:    "15:00",
		},
		{
			name:         "keeps provided times for normal",
			scheduleType: "normal",
			openTime:     "10:00",
			lastOrder:    "13:00",
			closeTime:    "14:00",
			wantOpen:     "10:00",
			wantLast:     "13:00",
			wantClose:    "14:00",
		},
		{
			name:         "fills only omitted fields for normal",
			scheduleType: "normal",
			openTime:     "10:00",
			wantOpen:     "10:00",
			wantLast:     "13:30",
			wantClose:    "15:00",
		},
		{
			name:         "does not fill times for closed",
			scheduleType: "closed",
			wantOpen:     "",
			wantLast:     "",
			wantClose:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openTime, last, closeTime := parseOptionalTime(tt.openTime), parseOptionalTime(tt.lastOrder), parseOptionalTime(tt.closeTime)
			gotOpen, gotLast, gotClose := ApplyDefaultBusinessHours(tt.scheduleType, openTime, last, closeTime)
			assertTimeString(t, "open", gotOpen, tt.wantOpen)
			assertTimeString(t, "last_order", gotLast, tt.wantLast)
			assertTimeString(t, "close", gotClose, tt.wantClose)
		})
	}
}

func TestApplyEventDefaultBusinessHours(t *testing.T) {
	tests := []struct {
		name      string
		date      string
		wantOpen  string
		wantLast  string
		wantClose string
	}{
		{
			name:      "fills weekday event with normal hours",
			date:      "2026-05-20",
			wantOpen:  "11:30",
			wantLast:  "13:30",
			wantClose: "15:00",
		},
		{
			name:      "fills saturday event with normal hours",
			date:      "2026-05-23",
			wantOpen:  "11:30",
			wantLast:  "13:30",
			wantClose: "15:00",
		},
		{
			name:      "fills sunday event with morning store hours",
			date:      "2026-05-24",
			wantOpen:  "08:30",
			wantLast:  "13:30",
			wantClose: "15:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := datetime.MustParseDate(tt.date)
			gotOpen, gotLast, gotClose := ApplyEventDefaultBusinessHours(d, datetime.Time{}, datetime.Time{}, datetime.Time{})
			assertTimeString(t, "open", gotOpen, tt.wantOpen)
			assertTimeString(t, "last_order", gotLast, tt.wantLast)
			assertTimeString(t, "close", gotClose, tt.wantClose)
		})
	}
}

func TestBookingWindowMinutes(t *testing.T) {
	tests := []struct {
		name      string
		schedule  domain.Schedule
		wantOpen  int
		wantLast  int
		wantOK    bool
	}{
		{
			name:     "normal weekday",
			schedule: domain.Schedule{ScheduleType: domain.ScheduleTypeNormal},
			wantOpen: 11*60 + 30,
			wantLast: 13*60 + 30,
			wantOK:   true,
		},
		{
			name:     "morning sunday books from lunch open not store open",
			schedule: domain.Schedule{ScheduleType: domain.ScheduleTypeMorning},
			wantOpen: 11*60 + 30,
			wantLast: 13*60 + 30,
			wantOK:   true,
		},
		{
			name:     "event on weekday",
			schedule: domain.Schedule{Date: datetime.MustParseDate("2026-05-20"), ScheduleType: domain.ScheduleTypeEvent},
			wantOpen: 11*60 + 30,
			wantLast: 13*60 + 30,
			wantOK:   true,
		},
		{
			name:     "event on saturday uses normal booking window",
			schedule: domain.Schedule{Date: datetime.MustParseDate("2026-05-23"), ScheduleType: domain.ScheduleTypeEvent},
			wantOpen: 11*60 + 30,
			wantLast: 13*60 + 30,
			wantOK:   true,
		},
		{
			name:     "event on sunday books from lunch open not morning store open",
			schedule: domain.Schedule{Date: datetime.MustParseDate("2026-05-24"), ScheduleType: domain.ScheduleTypeEvent},
			wantOpen: 11*60 + 30,
			wantLast: 13*60 + 30,
			wantOK:   true,
		},
		{
			name:     "closed",
			schedule: domain.Schedule{ScheduleType: domain.ScheduleTypeClosed},
			wantOK:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			open, last, ok := BookingWindowMinutes(tt.schedule)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if open != tt.wantOpen {
				t.Fatalf("open = %d, want %d", open, tt.wantOpen)
			}
			if last != tt.wantLast {
				t.Fatalf("last = %d, want %d", last, tt.wantLast)
			}
		})
	}
}

func parseOptionalTime(s string) datetime.Time {
	if s == "" {
		return datetime.Time{}
	}
	return datetime.MustParseTime(s)
}

func assertTimeString(t *testing.T, label string, got datetime.Time, want string) {
	t.Helper()
	if want == "" {
		if !got.IsZero() {
			t.Fatalf("%s = %s, want zero", label, got.String())
		}
		return
	}
	if got.String() != want {
		t.Fatalf("%s = %s, want %s", label, got.String(), want)
	}
}
