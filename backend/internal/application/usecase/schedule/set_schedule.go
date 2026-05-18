package usecase

import (
	"context"
	"strings"

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

// SetScheduleCommand は日別スケジュール設定（Upsert）の入力
type SetScheduleCommand struct {
	Date             datetime.Date
	ScheduleType     string
	Capacity         int
	EventName        string
	EventDescription string
	OpenTime         datetime.Time
	LastOrderTime    datetime.Time
	CloseTime        datetime.Time
}

// SetScheduleResult は Upsert の結果（Inserted が true なら新規挿入）
type SetScheduleResult struct {
	Schedule domain.Schedule
	Inserted bool
}

// SetScheduleUseCase は日別スケジュールの設定（Upsert）
type SetScheduleUseCase struct {
	repo domain.ScheduleRepository
}

// NewSetScheduleUseCase は SetScheduleUseCase の生成
func NewSetScheduleUseCase(repo domain.ScheduleRepository) *SetScheduleUseCase {
	return &SetScheduleUseCase{repo: repo}
}

// Execute は入力検証・デフォルト営業時刻の適用・Repository への Upsert
func (u *SetScheduleUseCase) Execute(ctx context.Context, cmd SetScheduleCommand) (*SetScheduleResult, error) {
	violations := validateSetSchedule(cmd)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	scheduleType := strings.TrimSpace(cmd.ScheduleType)
	open, lastOrder, close := service.ApplyDefaultBusinessHours(scheduleType, cmd.OpenTime, cmd.LastOrderTime, cmd.CloseTime)
	if scheduleType == "event" {
		open, lastOrder, close = service.ApplyEventDefaultBusinessHours(cmd.Date, open, lastOrder, close)
	}
	if scheduleType == "closed" {
		open, lastOrder, close = datetime.Time{}, datetime.Time{}, datetime.Time{}
	}
	if v := validateScheduleTimes(open, lastOrder, close); len(v) > 0 {
		return nil, &ValidationError{Violations: v}
	}

	in := domain.SetScheduleInput{
		Date:             cmd.Date,
		ScheduleType:     scheduleType,
		Capacity:         cmd.Capacity,
		EventName:        strings.TrimSpace(cmd.EventName),
		EventDescription: strings.TrimSpace(cmd.EventDescription),
		OpenTime:         open,
		LastOrderTime:    lastOrder,
		CloseTime:        close,
	}
	res, inserted, err := u.repo.Upsert(ctx, in)
	if err != nil {
		return nil, err
	}
	return &SetScheduleResult{Schedule: res, Inserted: inserted}, nil
}
