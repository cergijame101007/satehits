package domain

import (
	"context"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// Schedule はドメインエンティティ
type Schedule struct {
	Date             datetime.Date  `json:"date"`
	ScheduleType     string         `json:"schedule_type"`
	Capacity         int            `json:"capacity"`
	EventName        string         `json:"event_name"`
	EventDescription string         `json:"event_description"`
	OpenTime         datetime.Time  `json:"open_time"`
	LastOrderTime    datetime.Time  `json:"last_order_time"`
	CloseTime        datetime.Time  `json:"close_time"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
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