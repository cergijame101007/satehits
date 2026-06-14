package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

const defaultAdminReservationStatus = "approved"

// CreateAdminReservationCommand は管理者手動予約登録の入力
type CreateAdminReservationCommand struct {
	Name      string
	People    int
	VisitDate datetime.Date
	VisitTime datetime.Time
	Phone     string
	Email     string
	Note      string
	Source    string
	Status    string
}

// CreateAdminReservationUseCase は管理者向け予約手動登録
type CreateAdminReservationUseCase struct {
	repo domain.ReservationRepository
}

// NewCreateAdminReservationUseCase は CreateAdminReservationUseCase を生成する
func NewCreateAdminReservationUseCase(repo domain.ReservationRepository) *CreateAdminReservationUseCase {
	return &CreateAdminReservationUseCase{repo: repo}
}

// Execute は入力検証および Repository への永続化
func (u *CreateAdminReservationUseCase) Execute(ctx context.Context, cmd CreateAdminReservationCommand) (*domain.Reservation, error) {
	violations := validateCreateAdminReservation(cmd, time.Now())
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	status := strings.TrimSpace(cmd.Status)
	if status == "" {
		status = defaultAdminReservationStatus
	}

	in := domain.CreateReservationInput{
		Name:      strings.TrimSpace(cmd.Name),
		People:    cmd.People,
		VisitDate: cmd.VisitDate,
		VisitTime: cmd.VisitTime,
		Phone:     strings.TrimSpace(cmd.Phone),
		Email:     strings.TrimSpace(cmd.Email),
		Note:      cmd.Note,
		Status:    status,
		Source:    strings.TrimSpace(cmd.Source),
	}
	res, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func validateCreateAdminReservation(cmd CreateAdminReservationCommand, now time.Time) []FieldViolation {
	base := validateCreateReservation(CreateReservationCommand{
		Name:      cmd.Name,
		People:    cmd.People,
		VisitDate: cmd.VisitDate,
		VisitTime: cmd.VisitTime,
		Phone:     cmd.Phone,
		Email:     cmd.Email,
		Note:      cmd.Note,
	}, now)

	var violations []FieldViolation
	violations = append(violations, base...)
	violations = append(violations, validateReservationSource(cmd.Source)...)

	status := strings.TrimSpace(cmd.Status)
	if status != "" && !isValidEnum(status, domain.ValidReservationStatuses) {
		violations = append(violations, FieldViolation{Field: "status", Message: "ステータスの値が不正です"})
	}

	return violations
}
