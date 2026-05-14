package domain

import (
	"context"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// Reservation はドメインエンティティ
type Reservation struct {
	ID        int           `json:"id"`
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
	Create(ctx context.Context, r CreateReservationInput) error
	GetAll(ctx context.Context) ([]Reservation, error)
}
