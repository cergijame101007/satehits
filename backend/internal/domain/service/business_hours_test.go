package service

import (
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
)

func TestApplyDefaultBusinessHours(t *testing.T) {
	tests := []struct {
		name         string
		scheduleType string
		open         string
		lastOrder    string
		close        string
		wantOpen     string
		wantLast     string
		wantClose    string
	}{
		{
			name:         "fills all times for normal when omitted",
			scheduleType: "normal",
			wantOpen:     "11:30",
			wantLast:     "14:00",
			wantClose:    "15:00",
		},
		{
			name:         "fills all times for special_menu when omitted",
			scheduleType: "special_menu",
			wantOpen:     "11:30",
			wantLast:     "14:00",
			wantClose:    "15:00",
		},
		{
			name:         "fills all times for morning when omitted",
			scheduleType: "morning",
			wantOpen:     "08:30",
			wantLast:     "14:00",
			wantClose:    "15:00",
		},
		{
			name:         "keeps provided times for normal",
			scheduleType: "normal",
			open:         "10:00",
			lastOrder:    "13:00",
			close:        "14:00",
			wantOpen:     "10:00",
			wantLast:     "13:00",
			wantClose:    "14:00",
		},
		{
			name:         "fills only omitted fields for normal",
			scheduleType: "normal",
			open:         "10:00",
			wantOpen:     "10:00",
			wantLast:     "14:00",
			wantClose:    "15:00",
		},
		{
			name:         "does not fill times for event",
			scheduleType: "event",
			wantOpen:     "",
			wantLast:     "",
			wantClose:    "",
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
			open, last, close := parseOptionalTime(tt.open), parseOptionalTime(tt.lastOrder), parseOptionalTime(tt.close)
			gotOpen, gotLast, gotClose := ApplyDefaultBusinessHours(tt.scheduleType, open, last, close)
			assertTimeString(t, "open", gotOpen, tt.wantOpen)
			assertTimeString(t, "last_order", gotLast, tt.wantLast)
			assertTimeString(t, "close", gotClose, tt.wantClose)
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
