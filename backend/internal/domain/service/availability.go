package service

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// Availability は指定日の残り食数・予約可否の算出結果
type Availability struct {
	Date             datetime.Date
	Capacity         int
	Reserved         int
	Available        int
	ScheduleType     string // 休業時は空文字（API では null）
	EventName        string
	EventDescription string
	IsHoliday        bool
}

// AvailabilityService は空き状況を計算する Domain Service
type AvailabilityService struct {
	resolver *ScheduleResolver
	repo     domain.ReservationRepository
}

// NewAvailabilityService は AvailabilityService を生成する
func NewAvailabilityService(resolver *ScheduleResolver, repo domain.ReservationRepository) *AvailabilityService {
	return &AvailabilityService{resolver: resolver, repo: repo}
}

// ResolveForDate は指定日の残り食数・予約可否を返す
func (s *AvailabilityService) ResolveForDate(ctx context.Context, date datetime.Date) (Availability, error) {
	eff, err := s.resolver.ResolveForDate(ctx, date)
	if err != nil {
		return Availability{}, err
	}

	reserved, err := s.repo.SumApprovedPeopleByDate(ctx, date)
	if err != nil {
		return Availability{}, err
	}

	return buildAvailability(date, eff.Schedule, reserved), nil
}

// ResolveMonth は指定年月の各暦日の残り食数・予約可否を返す（日付昇順）
func (s *AvailabilityService) ResolveMonth(ctx context.Context, year, month int) ([]Availability, error) {
	effective, err := s.resolver.ResolveMonth(ctx, year, month)
	if err != nil {
		return nil, err
	}

	from, to := MonthDateRange(year, month)
	reservedByDate, err := s.repo.SumApprovedPeopleByDateRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	items := make([]Availability, 0, len(effective))
	for _, eff := range effective {
		date := eff.Schedule.Date
		reserved := reservedByDate[date.String()]
		items = append(items, buildAvailability(date, eff.Schedule, reserved))
	}
	return items, nil
}

func buildAvailability(date datetime.Date, sch domain.Schedule, reserved int) Availability {
	if sch.ScheduleType == domain.ScheduleTypeClosed {
		return Availability{
			Date:      date,
			Capacity:  0,
			Reserved:  0,
			Available: 0,
			IsHoliday: true,
		}
	}

	capacity := sch.Capacity
	available := capacity - reserved
	if available < 0 {
		available = 0
	}

	return Availability{
		Date:             date,
		Capacity:         capacity,
		Reserved:         reserved,
		Available:        available,
		ScheduleType:     sch.ScheduleType,
		EventName:        sch.EventName,
		EventDescription: sch.EventDescription,
		IsHoliday:        false,
	}
}
