package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// DeleteScheduleUseCase は日別スケジュールの例外設定を削除し店舗定例に戻す
type DeleteScheduleUseCase struct {
	repo domain.ScheduleRepository
}

// NewDeleteScheduleUseCase は DeleteScheduleUseCase の生成
func NewDeleteScheduleUseCase(repo domain.ScheduleRepository) *DeleteScheduleUseCase {
	return &DeleteScheduleUseCase{repo: repo}
}

// Execute は指定日の daily_schedules 行を削除する
func (u *DeleteScheduleUseCase) Execute(ctx context.Context, date datetime.Date) error {
	if date.IsZero() {
		return &ValidationError{Violations: []FieldViolation{
			{Field: "date", Message: "日付は必須です"},
		}}
	}

	_, found, err := u.repo.FindByDate(ctx, date)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrScheduleNotStored
	}

	return u.repo.DeleteByDate(ctx, date)
}
