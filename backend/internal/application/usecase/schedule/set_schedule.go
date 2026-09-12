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
	repo     domain.ScheduleRepository
	calendar *service.StoreCalendar
}

// NewSetScheduleUseCase は SetScheduleUseCase の生成（calendar は event の省略時刻の補完に使う）
func NewSetScheduleUseCase(repo domain.ScheduleRepository, calendar *service.StoreCalendar) *SetScheduleUseCase {
	return &SetScheduleUseCase{repo: repo, calendar: calendar}
}

// Execute は入力検証・デフォルト営業時刻の適用・Repository への Upsert
func (u *SetScheduleUseCase) Execute(ctx context.Context, cmd SetScheduleCommand) (*SetScheduleResult, error) {
	violations := validateSetSchedule(cmd)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	scheduleType := strings.TrimSpace(cmd.ScheduleType)
	openTime, lastOrder, closeTime := service.ApplyDefaultBusinessHours(scheduleType, cmd.OpenTime, cmd.LastOrderTime, cmd.CloseTime)
	if scheduleType == domain.ScheduleTypeEvent {
		openTime, lastOrder, closeTime = u.calendar.ApplyEventDefaultBusinessHours(cmd.Date, openTime, lastOrder, closeTime)
	}
	if scheduleType == domain.ScheduleTypeClosed || scheduleType == domain.ScheduleTypeExternalEvent {
		openTime, lastOrder, closeTime = datetime.Time{}, datetime.Time{}, datetime.Time{}
	}
	if scheduleType == domain.ScheduleTypeExternalEvent {
		cmd.Capacity = 0
	}
	if v := validateScheduleTimes(openTime, lastOrder, closeTime); len(v) > 0 {
		return nil, &ValidationError{Violations: v}
	}

	eventName := strings.TrimSpace(cmd.EventName)
	eventDescription := strings.TrimSpace(cmd.EventDescription)
	if scheduleTypeForbidsEventText(scheduleType) {
		eventName = ""
		eventDescription = ""
	}

	in := domain.SetScheduleInput{
		Date:             cmd.Date,
		ScheduleType:     scheduleType,
		Capacity:         cmd.Capacity,
		EventName:        eventName,
		EventDescription: eventDescription,
		OpenTime:         openTime,
		LastOrderTime:    lastOrder,
		CloseTime:        closeTime,
	}
	res, inserted, err := u.repo.Upsert(ctx, in)
	if err != nil {
		return nil, err
	}
	return &SetScheduleResult{Schedule: res, Inserted: inserted}, nil
}
