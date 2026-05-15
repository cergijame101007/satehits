package domain

import (
	"context"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// Schedule はドメインエンティティ
type Schedule struct {
	Date             datetime.Date
	ScheduleType     string
	Capacity         int
	EventName        string
	EventDescription string
	OpenTime         datetime.Time
	LastOrderTime    datetime.Time
	CloseTime        datetime.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CreateScheduleInput はスケジュールを作成するための入力
type CreateScheduleInput struct {
	Date             datetime.Date
	ScheduleType     string
	Capacity         int
	EventName        string
	EventDescription string
	OpenTime         datetime.Time
	LastOrderTime    datetime.Time
	CloseTime        datetime.Time
}

// ScheduleRepository はスケジュールデータを永続化するためのインターフェース
type ScheduleRepository interface {
	Create(ctx context.Context, s CreateScheduleInput) (Schedule, error)
}