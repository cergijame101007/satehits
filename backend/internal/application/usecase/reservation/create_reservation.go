package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

// FieldViolation はフィールド単位のバリデーションエラー
// handler.ErrorDetail と同形の別定義（handler 非依存のため）
type FieldViolation struct {
	Field   string
	Message string
}

// ValidationError はユースケース入力の検証失敗
type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string {
	return "入力内容に誤りがあります"
}

// CreateReservationCommand は顧客向け Web 予約申請の入力
type CreateReservationCommand struct {
	Name      string
	People    int
	VisitDate datetime.Date
	VisitTime datetime.Time
	Phone     string
	Email     string
	Note      string
}

// CreateReservationUseCase は顧客向け予約作成
type CreateReservationUseCase struct {
	repo         domain.ReservationRepository
	resolver     *service.ScheduleResolver
	availability *service.AvailabilityService
	txManager    application.TxManager
}

// NewCreateReservationUseCase は CreateReservationUseCase の生成
func NewCreateReservationUseCase(
	repo domain.ReservationRepository,
	resolver *service.ScheduleResolver,
	availability *service.AvailabilityService,
	txManager application.TxManager,
) *CreateReservationUseCase {
	return &CreateReservationUseCase{
		repo:         repo,
		resolver:     resolver,
		availability: availability,
		txManager:    txManager,
	}
}

// Execute は入力検証および Repository への永続化
func (u *CreateReservationUseCase) Execute(ctx context.Context, cmd CreateReservationCommand) (*domain.Reservation, error) {
	violations := validateCreateReservation(cmd, time.Now())
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	eff, err := u.resolver.ResolveForDate(ctx, cmd.VisitDate)
	if err != nil {
		return nil, mapAvailabilityServiceError(err)
	}
	if eff.Schedule.ScheduleType == domain.ScheduleTypeClosed {
		return nil, &ValidationError{Violations: []FieldViolation{
			{Field: "visit_date", Message: "この日は予約できません"},
		}}
	}
	if v := validateVisitTimeForSchedule(eff.Schedule, cmd.VisitTime); len(v) > 0 {
		return nil, &ValidationError{Violations: v}
	}

	avail, err := u.availability.ResolveForDate(ctx, cmd.VisitDate)
	if err != nil {
		return nil, mapAvailabilityServiceError(err)
	}
	if avail.IsHoliday {
		return nil, &ValidationError{Violations: []FieldViolation{
			{Field: "visit_date", Message: "この日は予約できません"},
		}}
	}
	if cmd.People > avail.Available {
		return nil, domain.ErrCapacityExceeded
	}

	in := domain.CreateReservationInput{
		Name:      strings.TrimSpace(cmd.Name),
		People:    cmd.People,
		VisitDate: cmd.VisitDate,
		VisitTime: cmd.VisitTime,
		Phone:     strings.TrimSpace(cmd.Phone),
		Email:     strings.TrimSpace(cmd.Email),
		Note:      cmd.Note,
		Status:    "pending",
		Source:    "web",
	}

	var created domain.Reservation
	err = u.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		fresh, err := u.availability.ResolveForDate(txCtx, cmd.VisitDate)
		if err != nil {
			return err
		}
		if fresh.IsHoliday {
			return &ValidationError{Violations: []FieldViolation{
				{Field: "visit_date", Message: "この日は予約できません"},
			}}
		}
		if cmd.People > fresh.Available {
			return domain.ErrCapacityExceeded
		}
		res, err := u.repo.Create(txCtx, in)
		if err != nil {
			return err
		}
		created = res
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}
