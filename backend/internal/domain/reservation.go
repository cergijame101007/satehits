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
}
