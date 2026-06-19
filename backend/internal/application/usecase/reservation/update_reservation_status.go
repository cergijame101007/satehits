package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

// UpdateReservationStatusCommand は予約ステータス更新の入力
type UpdateReservationStatusCommand struct {
	ID     uuid.UUID
	Status string
	Reason string
}

// UpdateReservationStatusUseCase は管理者向け予約ステータス更新
type UpdateReservationStatusUseCase struct {
	repo     domain.ReservationRepository
	notifier MailNotifier
}

// NewUpdateReservationStatusUseCase は UpdateReservationStatusUseCase を生成する
func NewUpdateReservationStatusUseCase(repo domain.ReservationRepository, notifier MailNotifier) *UpdateReservationStatusUseCase {
	if notifier == nil {
		notifier = NoOpMailNotifier{}
	}
	return &UpdateReservationStatusUseCase{repo: repo, notifier: notifier}
}

// Execute は遷移ルールを検証しステータスを更新する
func (u *UpdateReservationStatusUseCase) Execute(ctx context.Context, cmd UpdateReservationStatusCommand) (*domain.Reservation, error) {
	violations := validateUpdateStatusTarget(cmd.Status)
	violations = append(violations, validateUpdateStatusReason(cmd.Reason)...)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	current, err := u.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	if !domain.CanTransition(current.Status, cmd.Status) {
		return nil, &ValidationError{
			Violations: []FieldViolation{
				{Field: "status", Message: "このステータスには変更できません"},
			},
		}
	}

	updated, err := u.repo.UpdateStatus(ctx, cmd.ID, cmd.Status)
	if err != nil {
		return nil, err
	}

	switch cmd.Status {
	case "approved":
		u.notifier.ReservationApproved(updated)
	case "rejected":
		u.notifier.ReservationRejected(updated, cmd.Reason)
	}

	return &updated, nil
}
