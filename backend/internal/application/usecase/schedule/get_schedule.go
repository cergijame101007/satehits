package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

// GetScheduleResult は日別取得の結果（有効スケジュール）
type GetScheduleResult = service.EffectiveSchedule

// GetScheduleUseCase は指定日のスケジュールを返す
type GetScheduleUseCase struct {
	resolver *service.ScheduleResolver
}

// NewGetScheduleUseCase は GetScheduleUseCase を生成する
func NewGetScheduleUseCase(resolver *service.ScheduleResolver) *GetScheduleUseCase {
	return &GetScheduleUseCase{resolver: resolver}
}

// Execute は有効スケジュールを返す（保存行優先、無ければ店舗定例）
func (u *GetScheduleUseCase) Execute(ctx context.Context, date datetime.Date) (*GetScheduleResult, error) {
	eff, err := u.resolver.ResolveForDate(ctx, date)
	if err != nil {
		return nil, mapScheduleServiceError(err)
	}
	return &eff, nil
}
