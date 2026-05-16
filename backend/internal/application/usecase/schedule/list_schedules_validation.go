package usecase

import "fmt"

const (
	minScheduleYear  = 2000
	maxScheduleYear  = 2100
	minScheduleMonth = 1
	maxScheduleMonth = 12
)

func validateListSchedulesYearMonth(year, month int) []FieldViolation {
	var violations []FieldViolation
	if year < minScheduleYear || year > maxScheduleYear {
		violations = append(violations, FieldViolation{
			Field:   "year",
			Message: fmt.Sprintf("年は%d〜%dの範囲で指定してください", minScheduleYear, maxScheduleYear),
		})
	}
	if month < minScheduleMonth || month > maxScheduleMonth {
		violations = append(violations, FieldViolation{
			Field:   "month",
			Message: fmt.Sprintf("月は%d〜%dの範囲で指定してください", minScheduleMonth, maxScheduleMonth),
		})
	}
	return violations
}
