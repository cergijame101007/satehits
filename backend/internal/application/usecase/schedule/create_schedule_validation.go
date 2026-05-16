package usecase

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// OpenAPI SetScheduleRequest / docs/table_design.md の daily_schedules 制約に準拠
const (
	maxEventNameRunes        = 100
	maxEventDescriptionRunes = 500
)

var allowedScheduleTypes = map[string]struct{}{
	"normal":       {},
	"morning":      {},
	"event":        {},
	"special_menu": {},
	"closed":       {},
}

func validateCreateSchedule(cmd CreateScheduleCommand) []FieldViolation {
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
	if utf8.RuneCountInString(eventName) > maxEventNameRunes {
		violations = append(violations, FieldViolation{Field: "event_name", Message: fmt.Sprintf("イベント名は%d文字以内で入力してください", maxEventNameRunes)})
	}
	if utf8.RuneCountInString(eventDesc) > maxEventDescriptionRunes {
		violations = append(violations, FieldViolation{Field: "event_description", Message: fmt.Sprintf("イベント説明は%d文字以内で入力してください", maxEventDescriptionRunes)})
	}

	violations = append(violations, validateScheduleTimes(cmd.OpenTime, cmd.LastOrderTime, cmd.CloseTime)...)

	return violations
}

// validateScheduleTimes は任意指定の営業時刻の整合性を検証する（未指定は DB デフォルト相当）
func validateScheduleTimes(open, lastOrder, close datetime.Time) []FieldViolation {
	var violations []FieldViolation
	o := clockMinutes(open)
	lo := clockMinutes(lastOrder)
	c := clockMinutes(close)

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
