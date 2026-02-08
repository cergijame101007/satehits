package domain

import (
	"context"
	"time"
)

// Reservation はドメインエンティティ
type Reservation struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	People    int       `json:"people"`
	CreatedAt time.Time `json:"created_at"`
}

// ReservationRepository は予約データを永続化するためのインターフェース
type ReservationRepository interface {
	Create(ctx context.Context, name string, people int) error
	GetAll(ctx context.Context) ([]Reservation, error)
}
