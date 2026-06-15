package usecase

import (
	"context"
	"errors"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

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

func mapAvailabilityServiceError(err error) error {
	var dateErr *service.DateValidationError
	if errors.As(err, &dateErr) {
		return serviceViolationsToValidationError(dateErr.Violations)
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
