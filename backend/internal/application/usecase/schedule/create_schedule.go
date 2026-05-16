package usecase

import (
	"context"
	"strings"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
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

// CreateScheduleCommand はスケジュール作成の入力
type CreateScheduleCommand struct {
	Date             datetime.Date
	ScheduleType     string
	Capacity         int
	EventName        string
	EventDescription string
	OpenTime         datetime.Time
	LastOrderTime    datetime.Time
	CloseTime        datetime.Time
}

// CreateScheduleUseCase はスケジュール作成
type CreateScheduleUseCase struct {
	repo domain.ScheduleRepository
}

// NewCreateScheduleUseCase は CreateScheduleUseCase の生成
func NewCreateScheduleUseCase(repo domain.ScheduleRepository) *CreateScheduleUseCase {
	return &CreateScheduleUseCase{repo: repo}
}

// Execute は入力検証および Repository への永続化
func (u *CreateScheduleUseCase) Execute(ctx context.Context, cmd CreateScheduleCommand) (*domain.Schedule, error) {
	violations := validateCreateSchedule(cmd)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	in := domain.CreateScheduleInput{
		Date:             cmd.Date,
		ScheduleType:     strings.TrimSpace(cmd.ScheduleType),
		Capacity:         cmd.Capacity,
		EventName:        strings.TrimSpace(cmd.EventName),
		EventDescription: strings.TrimSpace(cmd.EventDescription),
		OpenTime:         cmd.OpenTime,
		LastOrderTime:    cmd.LastOrderTime,
		CloseTime:        cmd.CloseTime,
	}
	res, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
