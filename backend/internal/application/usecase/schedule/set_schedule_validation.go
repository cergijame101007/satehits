package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// OpenAPI SetScheduleRequest / docs/table_design.md の daily_schedules 制約に準拠
const (
	maxEventNameRunes        = 100
	maxEventDescriptionRunes = 500
)

var allowedScheduleTypes = map[string]struct{}{
	domain.ScheduleTypeNormal:        {},
	domain.ScheduleTypeMorning:       {},
	domain.ScheduleTypeEvent:         {},
	domain.ScheduleTypeExternalEvent: {},
	domain.ScheduleTypeSpecialMenu:   {},
	domain.ScheduleTypeClosed:        {},
}

func validateSetSchedule(cmd SetScheduleCommand) []FieldViolation {
	var violations []FieldViolation

	if cmd.Date.IsZero() {
		violations = append(violations, FieldViolation{Field: "date", Message: "日付は必須です"})
	}

	st := strings.TrimSpace(cmd.ScheduleType)
	if st == "" {
		violations = append(violations, FieldViolation{Field: "schedule_type", Message: "スケジュールタイプは必須です"})
	} else if _, ok := allowedScheduleTypes[st]; !ok {
		violations = append(violations, FieldViolation{Field: "schedule_type", Message: "スケジュールタイプが正しくありません"})
	}

	if cmd.Capacity < 0 {
		violations = append(violations, FieldViolation{Field: "capacity", Message: "提供可能数は0以上で指定してください"})
	}

	eventName := strings.TrimSpace(cmd.EventName)
	eventDesc := strings.TrimSpace(cmd.EventDescription)
	if scheduleTypeForbidsEventText(st) {
		violations = append(violations, validateForbiddenEventText(eventName, eventDesc)...)
	} else {
		if utf8.RuneCountInString(eventName) > maxEventNameRunes {
			violations = append(violations, FieldViolation{Field: "event_name", Message: fmt.Sprintf("イベント名は%d文字以内で入力してください", maxEventNameRunes)})
		}
		if utf8.RuneCountInString(eventDesc) > maxEventDescriptionRunes {
			violations = append(violations, FieldViolation{Field: "event_description", Message: fmt.Sprintf("イベント説明は%d文字以内で入力してください", maxEventDescriptionRunes)})
		}
		if st == domain.ScheduleTypeEvent || st == domain.ScheduleTypeExternalEvent {
			violations = append(violations, validateEventNameRequired(eventName)...)
		}
	}

	violations = append(violations, validateScheduleTimes(cmd.OpenTime, cmd.LastOrderTime, cmd.CloseTime)...)

	return violations
}

// scheduleTypeForbidsEventText は event_name / event_description を保存しない schedule_type
func scheduleTypeForbidsEventText(scheduleType string) bool {
	return scheduleType == domain.ScheduleTypeNormal ||
		scheduleType == domain.ScheduleTypeMorning ||
		scheduleType == domain.ScheduleTypeClosed
}

// validateForbiddenEventText はイベント欄を受け付けない schedule_type 向け
func validateForbiddenEventText(eventName, eventDesc string) []FieldViolation {
	var violations []FieldViolation
	if eventName != "" {
		violations = append(violations, FieldViolation{Field: "event_name", Message: "イベント名は指定できません"})
	}
	if eventDesc != "" {
		violations = append(violations, FieldViolation{Field: "event_description", Message: "イベント説明は指定できません"})
	}
	return violations
}

// validateEventNameRequired は event / external_event 時の名称必須（説明は任意）
func validateEventNameRequired(eventName string) []FieldViolation {
	var violations []FieldViolation
	if eventName == "" {
		violations = append(violations, FieldViolation{Field: "event_name", Message: "イベント名は必須です"})
	}
	return violations
}

// validateScheduleTimes は任意指定の営業時刻の整合性を検証する
// 未指定はドメインサービスで補完後に再検証（closed は時刻を保存しない）
func validateScheduleTimes(openTime, lastOrder, closeTime datetime.Time) []FieldViolation {
	var violations []FieldViolation
	o := clockMinutes(openTime)
	lo := clockMinutes(lastOrder)
	c := clockMinutes(closeTime)

	if o >= 0 && lo >= 0 && o > lo {
		violations = append(violations, FieldViolation{Field: "last_order_time", Message: "ラストオーダーは開店時刻以降にしてください"})
	}
	if lo >= 0 && c >= 0 && lo > c {
		violations = append(violations, FieldViolation{Field: "close_time", Message: "閉店時刻はラストオーダー以降にしてください"})
	}
	if o >= 0 && c >= 0 && o > c {
		violations = append(violations, FieldViolation{Field: "close_time", Message: "閉店時刻は開店時刻以降にしてください"})
	}
	return violations
}

func clockMinutes(tm datetime.Time) int {
	if tm.IsZero() {
		return -1
	}
	u := tm.UTC()
	return u.Hour()*60 + u.Minute()
}
