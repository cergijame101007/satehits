package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/holiday"
)

const (
	defaultScheduleCapacity = 10 // 店舗定例の提供数（docs/domain_knowledge.md §3）
	minScheduleYear         = 2000
	maxScheduleYear         = 2100
	minScheduleMonth        = 1
	maxScheduleMonth        = 12
)

// EffectiveSchedule — その日の営業設定（有効スケジュール）
// IsDefault: DB 行なし・店舗定例から合成
type EffectiveSchedule struct {
	Schedule  domain.Schedule
	IsDefault bool
}

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

// DateValidationError は日付検証失敗
type DateValidationError struct {
	Violations []ScheduleFieldViolation
}

func (e *DateValidationError) Error() string {
	return "入力内容に誤りがあります"
}

// ScheduleResolver — DB 保存行と店舗定例の合成（docs/data_flow.md §3）
type ScheduleResolver struct {
	repo domain.ScheduleRepository
}

// NewScheduleResolver は ScheduleResolver を生成する
func NewScheduleResolver(repo domain.ScheduleRepository) *ScheduleResolver {
	return &ScheduleResolver{repo: repo}
}

// ResolveForDate — 指定日の有効スケジュール（行優先、無ければ定例合成）
func (r *ScheduleResolver) ResolveForDate(ctx context.Context, date datetime.Date) (EffectiveSchedule, error) {
	if v := validateDate(date); len(v) > 0 {
		return EffectiveSchedule{}, &DateValidationError{Violations: v}
	}
	stored, found, err := r.repo.FindByDate(ctx, date)
	if err != nil {
		return EffectiveSchedule{}, err
	}
	if found {
		return EffectiveSchedule{Schedule: stored, IsDefault: false}, nil
	}
	return EffectiveSchedule{Schedule: synthesizeFromStoreCalendar(date), IsDefault: true}, nil
}

// ResolveMonth — 指定年月の各暦日の有効スケジュール（日付昇順）
func (r *ScheduleResolver) ResolveMonth(ctx context.Context, year, month int) ([]EffectiveSchedule, error) {
	if v := validateYearMonth(year, month); len(v) > 0 {
		return nil, &YearMonthValidationError{Violations: v}
	}

	stored, err := r.repo.ListStoredByYearMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}
	byDate := make(map[string]domain.Schedule, len(stored))
	for _, s := range stored {
		byDate[s.Date.String()] = s
	}

	lastDay := daysInMonth(year, time.Month(month))
	items := make([]EffectiveSchedule, 0, lastDay)
	for day := 1; day <= lastDay; day++ {
		d := datetime.NewDate(year, time.Month(month), day)
		if s, ok := byDate[d.String()]; ok {
			items = append(items, EffectiveSchedule{Schedule: s, IsDefault: false})
			continue
		}
		items = append(items, EffectiveSchedule{Schedule: synthesizeFromStoreCalendar(d), IsDefault: true})
	}
	return items, nil
}

// MonthDateRange は指定年月の月初・月末（validateYearMonth 済みであること）
func MonthDateRange(year, month int) (from, to datetime.Date) {
	lastDay := daysInMonth(year, time.Month(month))
	from = datetime.NewDate(year, time.Month(month), 1)
	to = datetime.NewDate(year, time.Month(month), lastDay)
	return from, to
}

// validateDate は日別解決用の日付検証（ゼロ値不可）
func validateDate(date datetime.Date) []ScheduleFieldViolation {
	if date.IsZero() {
		return []ScheduleFieldViolation{{Field: "date", Message: "日付は必須です"}}
	}
	return nil
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

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// synthesizeFromStoreCalendar — daily_schedules 行なし日の店舗定例合成
// 祝日は曜日にかかわらず normal（日曜と重なっても朝営業なし）、それ以外は木金 closed・日曜 morning・月〜水土 normal
// （docs/domain_knowledge.md §3・§4・§6）。祝日判定は同梱の内閣府 CSV（internal/domain/holiday）に基づく
func synthesizeFromStoreCalendar(d datetime.Date) domain.Schedule {
	scheduleType, capacity := defaultScheduleTypeAndCapacity(d)
	openTime, lastOrder, closeTime := ApplyDefaultBusinessHours(scheduleType, datetime.Time{}, datetime.Time{}, datetime.Time{})
	return domain.Schedule{
		Date:          d,
		ScheduleType:  scheduleType,
		Capacity:      capacity,
		OpenTime:      openTime,
		LastOrderTime: lastOrder,
		CloseTime:     closeTime,
	}
}

// defaultScheduleTypeAndCapacity は店舗定例のタイプ・提供数。祝日が最優先、次に曜日
func defaultScheduleTypeAndCapacity(d datetime.Date) (scheduleType string, capacity int) {
	if holiday.IsNationalHoliday(d) {
		return domain.ScheduleTypeNormal, defaultScheduleCapacity
	}
	switch d.Weekday() {
	case time.Thursday, time.Friday:
		return domain.ScheduleTypeClosed, 0
	case time.Sunday:
		return domain.ScheduleTypeMorning, defaultScheduleCapacity
	default:
		return domain.ScheduleTypeNormal, defaultScheduleCapacity
	}
}
