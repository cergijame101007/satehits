package service

import (
	"fmt"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

const (
	minScheduleYear  = 2000
	maxScheduleYear  = 2100
	minScheduleMonth = 1
	maxScheduleMonth = 12
)

// ScheduleFieldViolation はスケジュール系のフィールド単位検証エラー
type ScheduleFieldViolation struct {
	Field   string
	Message string
}

// YearMonthValidationError は year/month 検証失敗
type YearMonthValidationError struct {
	Violations []ScheduleFieldViolation
}

func (e *YearMonthValidationError) Error() string {
	return "入力内容に誤りがあります"
}

// validateYearMonth は一覧・月次解決用の year/month 範囲検証
func validateYearMonth(year, month int) []ScheduleFieldViolation {
	var violations []ScheduleFieldViolation
	if year < minScheduleYear || year > maxScheduleYear {
		violations = append(violations, ScheduleFieldViolation{
			Field:   "year",
			Message: fmt.Sprintf("年は%d〜%dの範囲で指定してください", minScheduleYear, maxScheduleYear),
		})
	}
	if month < minScheduleMonth || month > maxScheduleMonth {
		violations = append(violations, ScheduleFieldViolation{
			Field:   "month",
			Message: fmt.Sprintf("月は%d〜%dの範囲で指定してください", minScheduleMonth, maxScheduleMonth),
		})
	}
	return violations
}

// MonthDateRange は指定年月の月初・月末（validateYearMonth 済みであること）
func MonthDateRange(year, month int) (from, to datetime.Date) {
	lastDay := daysInMonth(year, time.Month(month))
	from = datetime.NewDate(year, time.Month(month), 1)
	to = datetime.NewDate(year, time.Month(month), lastDay)
	return from, to
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
