package usecase

import (
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// validCreateScheduleCommand は OpenAPI SetScheduleRequest 相当の正常系ベース
func validCreateScheduleCommand() CreateScheduleCommand {
	return CreateScheduleCommand{
		Date:         datetime.MustParseDate("2026-05-20"),
		ScheduleType: "normal",
		Capacity:     10,
	}
}

func assertNoViolations(t *testing.T, violations []FieldViolation) {
	t.Helper()
	if len(violations) != 0 {
		t.Fatalf("violations = %#v, want none", violations)
	}
}

func assertSingleViolationField(t *testing.T, violations []FieldViolation, field string) {
	t.Helper()
	if len(violations) != 1 {
		t.Fatalf("violations count = %d, want 1; violations = %#v", len(violations), violations)
	}
	if violations[0].Field != field {
		t.Fatalf("violations[0].Field = %q, want %q; violations = %#v", violations[0].Field, field, violations)
	}
}

func violationFields(v []FieldViolation) []string {
	fields := make([]string, len(v))
	for i, item := range v {
		fields[i] = item.Field
	}
	return fields
}

func TestValidateCreateSchedule(t *testing.T) {
	t.Run("accepts full event day with business hours", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.ScheduleType = "event"
		cmd.EventName = "和紅茶をしばく会"
		cmd.EventDescription = "詳細はInstagramをご覧ください"
		cmd.OpenTime = datetime.MustParseTime("11:30")
		cmd.LastOrderTime = datetime.MustParseTime("14:00")
		cmd.CloseTime = datetime.MustParseTime("15:00")
		assertNoViolations(t, validateCreateSchedule(cmd))
	})

	t.Run("accepts minimal normal schedule", func(t *testing.T) {
		assertNoViolations(t, validateCreateSchedule(validCreateScheduleCommand()))
	})

	t.Run("rejects zero date", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.Date = datetime.Date{}
		assertSingleViolationField(t, validateCreateSchedule(cmd), "date")
	})
}

func TestValidateCreateSchedule_scheduleType(t *testing.T) {
	// docs/table_design.md の schedule_type CHECK 制約
	tests := []struct {
		name      string
		in        string
		wantField string // 空ならエラーなし
	}{
		{name: "rejects empty schedule_type", in: "", wantField: "schedule_type"},
		{name: "rejects whitespace-only schedule_type", in: " \t　 ", wantField: "schedule_type"},
		{name: "rejects unknown schedule_type", in: "holiday", wantField: "schedule_type"},
		{name: "accepts normal", in: "normal", wantField: ""},
		{name: "accepts morning", in: "morning", wantField: ""},
		{name: "accepts event", in: "event", wantField: ""},
		{name: "accepts special_menu", in: "special_menu", wantField: ""},
		{name: "accepts closed", in: "closed", wantField: ""},
		{name: "accepts trimmed schedule_type", in: " normal ", wantField: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validCreateScheduleCommand()
			cmd.ScheduleType = tt.in
			v := validateCreateSchedule(cmd)
			if tt.wantField == "" {
				assertNoViolations(t, v)
				return
			}
			assertSingleViolationField(t, v, tt.wantField)
		})
	}
}

func TestValidateCreateSchedule_capacity(t *testing.T) {
	t.Run("rejects negative capacity", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.Capacity = -1
		assertSingleViolationField(t, validateCreateSchedule(cmd), "capacity")
	})

	t.Run("accepts zero capacity for closed day", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.ScheduleType = "closed"
		cmd.Capacity = 0
		assertNoViolations(t, validateCreateSchedule(cmd))
	})
}

func TestValidateCreateSchedule_eventTextLength(t *testing.T) {
	t.Run("accepts event_name at max rune length", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.EventName = strings.Repeat("あ", maxEventNameRunes)
		assertNoViolations(t, validateCreateSchedule(cmd))
	})

	t.Run("rejects event_name over max rune length", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.EventName = strings.Repeat("あ", maxEventNameRunes+1)
		assertSingleViolationField(t, validateCreateSchedule(cmd), "event_name")
	})

	t.Run("accepts event_description at max rune length", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.EventDescription = strings.Repeat("あ", maxEventDescriptionRunes)
		assertNoViolations(t, validateCreateSchedule(cmd))
	})

	t.Run("rejects event_description over max rune length", func(t *testing.T) {
		cmd := validCreateScheduleCommand()
		cmd.EventDescription = strings.Repeat("あ", maxEventDescriptionRunes+1)
		assertSingleViolationField(t, validateCreateSchedule(cmd), "event_description")
	})
}

func TestValidateScheduleTimes(t *testing.T) {
	// 任意指定の営業時刻の前後関係（未指定は DB デフォルト相当）
	tests := []struct {
		name      string
		open      string
		lastOrder string
		close     string
		want      []string
	}{
		{name: "accepts all times omitted"},
		{name: "accepts open before last order before close", open: "11:30", lastOrder: "14:00", close: "15:00"},
		{name: "rejects open after last order", open: "15:00", lastOrder: "14:00", close: "16:00", want: []string{"last_order_time"}},
		{name: "rejects last order after close", open: "11:30", lastOrder: "16:00", close: "15:00", want: []string{"close_time"}},
		{name: "rejects open after close without last order", open: "16:00", close: "15:00", want: []string{"close_time"}},
		{name: "accepts only open and close when open before close", open: "11:30", close: "15:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			open, lastOrder, closeTime := datetime.Time{}, datetime.Time{}, datetime.Time{}
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
				t.Fatalf("violation fields = %v, want %v", got, tt.want)
			}
			for i, field := range tt.want {
				if got[i] != field {
					t.Fatalf("violation fields = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
