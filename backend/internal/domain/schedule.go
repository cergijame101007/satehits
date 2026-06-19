package domain

import (
	"context"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// schedule_type の列挙値（docs/table_design.md daily_schedules CHECK 制約）
const (
	ScheduleTypeNormal        = "normal"
	ScheduleTypeMorning       = "morning"
	ScheduleTypeEvent         = "event"
	ScheduleTypeExternalEvent = "external_event"
	ScheduleTypeSpecialMenu   = "special_menu"
	ScheduleTypeClosed        = "closed"
)

// Schedule はドメインエンティティ
type Schedule struct {
	Date             datetime.Date `json:"date"`
	ScheduleType     string        `json:"schedule_type"`
	Capacity         int           `json:"capacity"`
	EventName        string        `json:"event_name"`
	EventDescription string        `json:"event_description"`
	OpenTime         datetime.Time `json:"open_time"`
	LastOrderTime    datetime.Time `json:"last_order_time"`
	CloseTime        datetime.Time `json:"close_time"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// SetScheduleInput は日別スケジュール設定（Upsert）の入力
type SetScheduleInput struct {
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
	// Upsert は日付をキーに挿入または更新する。第2戻り値は新規挿入なら true
	Upsert(ctx context.Context, s SetScheduleInput) (Schedule, bool, error)
	// FindByDate は日付で1件取得する。第2戻り値は DB に行があれば true
	FindByDate(ctx context.Context, date datetime.Date) (Schedule, bool, error)
	// ListStoredByYearMonth は指定年月に保存されている行のみを日付昇順で返す（定例合成は含まない）
	ListStoredByYearMonth(ctx context.Context, year, month int) ([]Schedule, error)
}
