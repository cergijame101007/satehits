package domain

import (
	"context"
	"time"
)

// Reservation はドメインエンティティ
type Reservation struct {
	ID        int
	Name      string
	People    int
	VisitDate time.Time
	VisitTime time.Time
	Phone     string
	Email     string
	Note      string
	Status    string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateReservationInput struct {
	Name      string
	People    int
	VisitDate time.Time
	VisitTime time.Time
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
