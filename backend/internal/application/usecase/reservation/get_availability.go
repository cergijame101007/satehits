package usecase

import (
	"context"
	"errors"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

// ListAvailabilityResult は月間空き状況一覧の結果
type ListAvailabilityResult struct {
	Year           int
	Month          int
	Availabilities []service.Availability
}

// GetAvailabilityUseCase は指定日の残り食数を返す
type GetAvailabilityUseCase struct {
	availability *service.AvailabilityService
}

// NewGetAvailabilityUseCase は GetAvailabilityUseCase を生成する
func NewGetAvailabilityUseCase(availability *service.AvailabilityService) *GetAvailabilityUseCase {
	return &GetAvailabilityUseCase{availability: availability}
}

// Execute は指定日の Availability を返す
func (u *GetAvailabilityUseCase) Execute(ctx context.Context, date datetime.Date) (*service.Availability, error) {
	avail, err := u.availability.ResolveForDate(ctx, date)
	if err != nil {
		return nil, mapAvailabilityServiceError(err)
	}
	return &avail, nil
}

// ExecuteMonth は指定年月の Availability 一覧を返す
func (u *GetAvailabilityUseCase) ExecuteMonth(ctx context.Context, year, month int) (*ListAvailabilityResult, error) {
	items, err := u.availability.ResolveMonth(ctx, year, month)
	if err != nil {
		return nil, mapAvailabilityServiceError(err)
	}
	return &ListAvailabilityResult{Year: year, Month: month, Availabilities: items}, nil
}

func mapAvailabilityServiceError(err error) error {
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
