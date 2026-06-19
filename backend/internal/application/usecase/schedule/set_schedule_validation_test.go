package usecase

import (
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

const (
	testEventNameSample = "和紅茶をしばく会"
	testEventDescShort  = "説明"
)

// validSetScheduleCommand は OpenAPI SetScheduleRequest 相当の正常系ベース
func validSetScheduleCommand() SetScheduleCommand {
	return SetScheduleCommand{
		Date:         datetime.MustParseDate("2026-05-20"),
		ScheduleType: domain.ScheduleTypeNormal,
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

func TestValidateSetSchedule(t *testing.T) {
	t.Run("accepts full event day with business hours", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = testEventNameSample
		cmd.EventDescription = "詳細はInstagramをご覧ください"
		cmd.OpenTime = datetime.MustParseTime("11:30")
		cmd.LastOrderTime = datetime.MustParseTime("13:30")
		cmd.CloseTime = datetime.MustParseTime("15:00")
		assertNoViolations(t, validateSetSchedule(cmd))
	})

	t.Run("accepts minimal normal schedule", func(t *testing.T) {
		assertNoViolations(t, validateSetSchedule(validSetScheduleCommand()))
	})

	t.Run("accepts event with name and description without times", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = testEventNameSample
		cmd.EventDescription = "通常のランチ営業はおやすみです"
		assertNoViolations(t, validateSetSchedule(cmd))
	})

	t.Run("rejects zero date", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.Date = datetime.Date{}
		assertSingleViolationField(t, validateSetSchedule(cmd), "date")
	})
}

func TestValidateSetSchedule_scheduleType(t *testing.T) {
	// docs/table_design.md の schedule_type CHECK 制約
	tests := []struct {
		name      string
		in        string
		wantField string // 空ならエラーなし
	}{
		{name: "rejects empty schedule_type", in: "", wantField: "schedule_type"},
		{name: "rejects whitespace-only schedule_type", in: " \t　 ", wantField: "schedule_type"},
		{name: "rejects unknown schedule_type", in: "holiday", wantField: "schedule_type"},
		{name: "accepts normal", in: domain.ScheduleTypeNormal, wantField: ""},
		{name: "accepts morning", in: domain.ScheduleTypeMorning, wantField: ""},
		{name: "accepts event", in: domain.ScheduleTypeEvent, wantField: ""},
		{name: "accepts external_event", in: domain.ScheduleTypeExternalEvent, wantField: ""},
		{name: "accepts special_menu", in: domain.ScheduleTypeSpecialMenu, wantField: ""},
		{name: "accepts closed", in: domain.ScheduleTypeClosed, wantField: ""},
		{name: "accepts trimmed schedule_type", in: " " + domain.ScheduleTypeNormal + " ", wantField: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := validSetScheduleCommand()
			cmd.ScheduleType = tt.in
			if strings.TrimSpace(tt.in) == domain.ScheduleTypeEvent ||
				strings.TrimSpace(tt.in) == domain.ScheduleTypeExternalEvent {
				cmd.EventName = "テストイベント"
				cmd.EventDescription = testEventDescShort
			}
			v := validateSetSchedule(cmd)
			if tt.wantField == "" {
				assertNoViolations(t, v)
				return
			}
			assertSingleViolationField(t, v, tt.wantField)
		})
	}
}

func TestValidateSetSchedule_capacity(t *testing.T) {
	t.Run("rejects negative capacity", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.Capacity = -1
		assertSingleViolationField(t, validateSetSchedule(cmd), "capacity")
	})

	t.Run("accepts zero capacity for closed day", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeClosed
		cmd.Capacity = 0
		assertNoViolations(t, validateSetSchedule(cmd))
	})
}

func TestValidateSetSchedule_eventRequiredFields(t *testing.T) {
	t.Run("rejects event without event_name", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventDescription = testEventDescShort
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})

	t.Run("accepts event without event_description", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = testEventNameSample
		assertNoViolations(t, validateSetSchedule(cmd))
	})

	t.Run("accepts external_event without event_description", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeExternalEvent
		cmd.EventName = testEventNameSample
		cmd.Capacity = 0
		assertNoViolations(t, validateSetSchedule(cmd))
	})

	t.Run("rejects external_event without event_name", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeExternalEvent
		cmd.Capacity = 0
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})

	t.Run("rejects external_event with whitespace-only event_name", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeExternalEvent
		cmd.Capacity = 0
		cmd.EventName = "  "
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})

	t.Run("rejects event with whitespace-only event_name", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = "  "
		cmd.EventDescription = testEventDescShort
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})
}

func TestValidateSetSchedule_eventTextLength(t *testing.T) {
	t.Run("accepts event_name at max rune length for event type", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = strings.Repeat("あ", maxEventNameRunes)
		cmd.EventDescription = testEventDescShort
		assertNoViolations(t, validateSetSchedule(cmd))
	})
	t.Run("rejects event_name over max rune length for event type", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = strings.Repeat("あ", maxEventNameRunes+1)
		cmd.EventDescription = testEventDescShort
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})

	t.Run("rejects event_name on normal schedule", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.EventName = "不要なイベント名"
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})

	t.Run("accepts event_description at max rune length for event type", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = "イベント"
		cmd.EventDescription = strings.Repeat("あ", maxEventDescriptionRunes)
		assertNoViolations(t, validateSetSchedule(cmd))
	})

	t.Run("rejects event_description over max rune length for event type", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeEvent
		cmd.EventName = "イベント"
		cmd.EventDescription = strings.Repeat("あ", maxEventDescriptionRunes+1)
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_description")
	})

	t.Run("rejects event_description on normal schedule", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.EventDescription = "不要な説明"
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_description")
	})
}

func TestValidateSetSchedule_forbiddenEventText(t *testing.T) {
	t.Run("rejects event_name on morning schedule", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeMorning
		cmd.EventName = "朝イベント"
		assertSingleViolationField(t, validateSetSchedule(cmd), "event_name")
	})

	t.Run("rejects event fields on closed schedule", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeClosed
		cmd.Capacity = 0
		cmd.EventName = "残したくない"
		cmd.EventDescription = testEventDescShort
		got := violationFields(validateSetSchedule(cmd))
		want := []string{"event_name", "event_description"}
		if len(got) != len(want) {
			t.Fatalf("violation fields = %v, want %v", got, want)
		}
		for i, field := range want {
			if got[i] != field {
				t.Fatalf("violation fields = %v, want %v", got, want)
			}
		}
	})

	t.Run("accepts special_menu with optional event_name", func(t *testing.T) {
		cmd := validSetScheduleCommand()
		cmd.ScheduleType = domain.ScheduleTypeSpecialMenu
		cmd.EventName = "リゾットランチ"
		cmd.EventDescription = "本日はリゾットランチの日です"
		assertNoViolations(t, validateSetSchedule(cmd))
	})
}

func TestValidateScheduleTimes(t *testing.T) {
	// 任意指定の営業時刻の前後関係（未指定はドメインサービスで補完後に再検証）
	tests := []struct {
		name      string
		openTime  string
		lastOrder string
		close     string
		want      []string
	}{
		{name: "accepts all times omitted"},
		{name: "accepts open before last order before close", openTime: "11:30", lastOrder: "13:30", close: "15:00"},
		{name: "rejects open after last order", openTime: "15:00", lastOrder: "13:30", close: "16:00", want: []string{"last_order_time"}},
		{name: "rejects last order after close", openTime: "11:30", lastOrder: "16:00", close: "15:00", want: []string{"close_time"}},
		{name: "rejects open after close without last order", openTime: "16:00", close: "15:00", want: []string{"close_time"}},
		{name: "accepts only open and close when open before close", openTime: "11:30", close: "15:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openTime, lastOrder, closeTime := datetime.Time{}, datetime.Time{}, datetime.Time{}
			if tt.openTime != "" {
				openTime = datetime.MustParseTime(tt.openTime)
			}
			if tt.lastOrder != "" {
				lastOrder = datetime.MustParseTime(tt.lastOrder)
			}
			if tt.close != "" {
				closeTime = datetime.MustParseTime(tt.close)
			}
			got := violationFields(validateScheduleTimes(openTime, lastOrder, closeTime))
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
