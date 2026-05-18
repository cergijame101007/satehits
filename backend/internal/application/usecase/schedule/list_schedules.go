package usecase

import (
	"context"
	"errors"

	"github.com/cergijame101007/satehits/internal/domain/service"
)

// ListSchedulesResult は月間一覧の結果
type ListSchedulesResult struct {
	Year      int
	Month     int
	Schedules []service.EffectiveSchedule
}

// ListSchedulesUseCase は指定期間のスケジュール一覧を返す
type ListSchedulesUseCase struct {
	resolver *service.ScheduleResolver
}

// NewListSchedulesUseCase は ListSchedulesUseCase を生成する
func NewListSchedulesUseCase(resolver *service.ScheduleResolver) *ListSchedulesUseCase {
	return &ListSchedulesUseCase{resolver: resolver}
}

// Execute は year/month の各暦日について有効スケジュールを返す
func (u *ListSchedulesUseCase) Execute(ctx context.Context, year, month int) (*ListSchedulesResult, error) {
	schedules, err := u.resolver.ResolveMonth(ctx, year, month)
	if err != nil {
		return nil, mapScheduleServiceError(err)
	}

	return &ListSchedulesResult{Year: year, Month: month, Schedules: schedules}, nil
}

func mapScheduleServiceError(err error) error {
	var dateErr *service.DateValidationError
	if errors.As(err, &dateErr) {
		return serviceViolationsToValidationError(dateErr.Violations)
	}
	var ymErr *service.YearMonthValidationError
	if errors.As(err, &ymErr) {
		return serviceViolationsToValidationError(ymErr.Violations)
	}
	return err
}

func serviceViolationsToValidationError(violations []service.ScheduleFieldViolation) *ValidationError {
	out := make([]FieldViolation, len(violations))
	for i, v := range violations {
		out[i] = FieldViolation{Field: v.Field, Message: v.Message}
	}
	return &ValidationError{Violations: out}
}
