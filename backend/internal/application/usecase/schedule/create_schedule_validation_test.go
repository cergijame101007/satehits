package usecase

import (
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
)

func validCreateScheduleCommand() CreateScheduleCommand {
	return CreateScheduleCommand{
		Date:         datetime.MustParseDate("2026-05-20"),
		ScheduleType: "normal",
		Capacity:     10,
	}
}

func violationFields(v []FieldViolation) []string {
	fields := make([]string, len(v))
	for i, item := range v {
		fields[i] = item.Field
	}
	return fields
}

func TestValidateCreateSchedule_valid(t *testing.T) {
	cmd := validCreateScheduleCommand()
	cmd.EventName = "和紅茶をしばく会"
	cmd.EventDescription = "詳細はInstagramをご覧ください"
	cmd.OpenTime = datetime.MustParseTime("11:30")
	cmd.LastOrderTime = datetime.MustParseTime("14:00")
	cmd.CloseTime = datetime.MustParseTime("15:00")

	if v := validateCreateSchedule(cmd); len(v) != 0 {
		t.Fatalf("expected no violations, got %#v", v)
	}
}

func TestValidateCreateSchedule_dateRequired(t *testing.T) {
	cmd := validCreateScheduleCommand()
	cmd.Date = datetime.Date{}

	v := validateCreateSchedule(cmd)
	if len(v) != 1 || v[0].Field != "date" {
		t.Fatalf("expected single date violation, got %#v", v)
	}
}

func TestValidateCreateSchedule_scheduleType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string // "" ならバリデーションエラーなし
	}{
		{"empty", "", "schedule_type"},
		{"whitespace", " \t　 ", "schedule_type"},
		{"invalid", "holiday", "schedule_type"},
		{"normal", "normal", ""},
		{"morning", "morning", ""},
		{"event", "event", ""},
		{"special_menu", "special_menu", ""},
		{"closed", "closed", ""},
		{"trimmed", " normal ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := validCreateScheduleCommand()
			cmd.ScheduleType = tt.in
			v := validateCreateSchedule(cmd)
			if tt.want == "" {
				if len(v) != 0 {
					t.Fatalf("expected no violations, got %#v", v)
				}
				return
			}
			if len(v) == 0 || v[0].Field != tt.want {
				t.Fatalf("expected %q violation, got %#v", tt.want, v)
			}
		})
	}
}

func TestValidateCreateSchedule_capacity(t *testing.T) {
	cmd := validCreateScheduleCommand()
	cmd.Capacity = -1

	v := validateCreateSchedule(cmd)
	if len(v) != 1 || v[0].Field != "capacity" {
		t.Fatalf("expected single capacity violation, got %#v", v)
	}

	cmd.Capacity = 0
	if v := validateCreateSchedule(cmd); len(v) != 0 {
		t.Fatalf("expected capacity 0 allowed, got %#v", v)
	}
}

func TestValidateCreateSchedule_eventNameMaxLength(t *testing.T) {
	cmd := validCreateScheduleCommand()
	cmd.EventName = strings.Repeat("あ", maxEventNameRunes+1)

	v := validateCreateSchedule(cmd)
	if len(v) != 1 || v[0].Field != "event_name" {
		t.Fatalf("expected single event_name violation, got %#v", v)
	}
}

func TestValidateCreateSchedule_eventDescriptionMaxLength(t *testing.T) {
	cmd := validCreateScheduleCommand()
	cmd.EventDescription = strings.Repeat("あ", maxEventDescriptionRunes+1)

	v := validateCreateSchedule(cmd)
	if len(v) != 1 || v[0].Field != "event_description" {
		t.Fatalf("expected single event_description violation, got %#v", v)
	}
}

func TestValidateScheduleTimes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		open      string
		lastOrder string
		close     string
		want      []string
	}{
		{
			name: "all_omitted",
			want: nil,
		},
		{
			name:      "valid_order",
			open:      "11:30",
			lastOrder: "14:00",
			close:     "15:00",
			want:      nil,
		},
		{
			name:      "open_after_last_order",
			open:      "15:00",
			lastOrder: "14:00",
			close:     "16:00",
			want:      []string{"last_order_time"},
		},
		{
			name:      "last_order_after_close",
			open:      "11:30",
			lastOrder: "16:00",
			close:     "15:00",
			want:      []string{"close_time"},
		},
		{
			name:      "open_after_close_without_last_order",
			open:      "16:00",
			close:     "15:00",
			want:      []string{"close_time"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			open := datetime.Time{}
			lastOrder := datetime.Time{}
			closeTime := datetime.Time{}
			if tt.open != "" {
				open = datetime.MustParseTime(tt.open)
			}
			if tt.lastOrder != "" {
				lastOrder = datetime.MustParseTime(tt.lastOrder)
			}
			if tt.close != "" {
				closeTime = datetime.MustParseTime(tt.close)
			}
			got := violationFields(validateScheduleTimes(open, lastOrder, closeTime))
			if len(got) != len(tt.want) {
				t.Fatalf("fields = %v, want %v (violations %#v)", got, tt.want, validateScheduleTimes(open, lastOrder, closeTime))
			}
			for i, field := range tt.want {
				if got[i] != field {
					t.Fatalf("fields = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
