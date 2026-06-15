package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// ErrReservationConflict は同一来店日時・電話番号のアクティブ予約が既にある
var ErrReservationConflict = errors.New("reservation conflict")

// ErrReservationNotFound は指定 ID の予約が存在しない
var ErrReservationNotFound = errors.New("reservation not found")

// ErrCapacityExceeded はその日の予約可能数を超過した
var ErrCapacityExceeded = errors.New("capacity exceeded")

// ValidReservationStatuses は reservations.status の許容値
var ValidReservationStatuses = []string{"pending", "approved", "rejected", "cancelled", "no_show"}

// ValidReservationSources は reservations.source の許容値
var ValidReservationSources = []string{"web", "instagram", "phone", "walk_in", "other"}

// UpdateStatusTargets は PATCH /admin/reservations/{id}/status で指定可能な遷移先
var UpdateStatusTargets = []string{"approved", "rejected", "cancelled", "no_show"}

var statusTransitions = map[string][]string{
	"pending":  {"approved", "rejected", "cancelled"},
	"approved": {"no_show", "cancelled"},
}

// CanTransition は docs/api_design.md のステータス遷移表に従い from→to が許可されるか判定する
func CanTransition(from, to string) bool {
	targets, ok := statusTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// ListReservationsFilter は管理者予約一覧の絞り込み条件
type ListReservationsFilter struct {
	Date   *datetime.Date
	Status string
	Source string
}

// Reservation はドメインエンティティ
type Reservation struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	People    int           `json:"people"`
	VisitDate datetime.Date `json:"visit_date"`
	VisitTime datetime.Time `json:"visit_time"`
	Phone     string        `json:"phone"`
	Email     string        `json:"email"`
	Note      string        `json:"note"`
	Status    string        `json:"status"`
	Source    string        `json:"source"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

type CreateReservationInput struct {
	Name      string
	People    int
	VisitDate datetime.Date
	VisitTime datetime.Time
	Phone     string
	Email     string
	Note      string
	Status    string
	Source    string
}

// ReservationRepository は予約データを永続化するためのインターフェース
type ReservationRepository interface {
	Create(ctx context.Context, r CreateReservationInput) (Reservation, error)
	GetAll(ctx context.Context) ([]Reservation, error)
	List(ctx context.Context, f ListReservationsFilter) ([]Reservation, error)
	GetByID(ctx context.Context, id uuid.UUID) (Reservation, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (Reservation, error)
	// SumApprovedPeopleByDate は指定日の承認済み予約人数合計を返す
	SumApprovedPeopleByDate(ctx context.Context, date datetime.Date) (int, error)
	// SumApprovedPeopleByDateRange は期間内の日付別・承認済み予約人数合計を返す（キー: YYYY-MM-DD）
	SumApprovedPeopleByDateRange(ctx context.Context, from, to datetime.Date) (map[string]int, error)
}
