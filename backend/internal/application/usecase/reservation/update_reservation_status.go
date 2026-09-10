package usecase

import (
	"context"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/application"
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
	repo      domain.ReservationRepository
	txManager application.TxManager
	enqueuer  MailEnqueuer
}

// NewUpdateReservationStatusUseCase は UpdateReservationStatusUseCase を生成する
func NewUpdateReservationStatusUseCase(
	repo domain.ReservationRepository,
	txManager application.TxManager,
	enqueuer MailEnqueuer,
) *UpdateReservationStatusUseCase {
	if enqueuer == nil {
		enqueuer = NoOpMailEnqueuer{}
	}
	return &UpdateReservationStatusUseCase{repo: repo, txManager: txManager, enqueuer: enqueuer}
}

// Execute は遷移ルールを検証しステータスを更新する（メール Outbox は同一トランザクション）
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

	var updated domain.Reservation
	err = u.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		res, err := u.repo.UpdateStatus(txCtx, cmd.ID, cmd.Status)
		if err != nil {
			return err
		}
		updated = res
		switch cmd.Status {
		case defaultAdminReservationStatus:
			if err := u.enqueuer.EnqueueReservationApproved(txCtx, updated); err != nil {
				return ignoreAlreadyEnqueued(err, updated.ID, "reservation_approved")
			}
		case "rejected":
			if err := u.enqueuer.EnqueueReservationRejected(txCtx, updated, cmd.Reason); err != nil {
				return ignoreAlreadyEnqueued(err, updated.ID, "reservation_rejected")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
