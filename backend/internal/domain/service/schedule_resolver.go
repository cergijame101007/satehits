package service

import (
	"context"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// 店舗定例の提供数（docs/domain_knowledge.md §3）
const defaultScheduleCapacity = 10

// EffectiveSchedule — その日の営業設定（有効スケジュール）
// IsDefault: DB 行なし・店舗定例から合成
type EffectiveSchedule struct {
	Schedule  domain.Schedule
	IsDefault bool
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

// synthesizeFromStoreCalendar — daily_schedules 行なし日の店舗定例合成
// 木金 closed、土日 morning、月〜水 normal（祝日未考慮・TODO）
func synthesizeFromStoreCalendar(d datetime.Date) domain.Schedule {
	scheduleType, capacity := defaultScheduleTypeAndCapacity(d.Weekday())
	open, lastOrder, close := ApplyDefaultBusinessHours(scheduleType, datetime.Time{}, datetime.Time{}, datetime.Time{})
	return domain.Schedule{
		Date:          d,
		ScheduleType:  scheduleType,
		Capacity:      capacity,
		OpenTime:      open,
		LastOrderTime: lastOrder,
		CloseTime:     close,
	}
}

func defaultScheduleTypeAndCapacity(wd time.Weekday) (scheduleType string, capacity int) {
	switch wd {
	case time.Thursday, time.Friday:
		return "closed", 0
	case time.Saturday, time.Sunday:
		return "morning", defaultScheduleCapacity
	default:
		return "normal", defaultScheduleCapacity
	}
}
