package repository

import (
	"context"
	"database/sql"

	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresReservationRepository はPostgreSQLを使った予約リポジトリの実装
type PostgresReservationRepository struct {
	db *sql.DB
}

// NewPostgresReservationRepository はPostgresReservationRepositoryのインスタンスを作成する
func NewPostgresReservationRepository(db *sql.DB) *PostgresReservationRepository {
	return &PostgresReservationRepository{db: db}
}

// Create は予約データを永続化する
func (r *PostgresReservationRepository) Create(ctx context.Context, name string, people int) error {
	query := `INSERT INTO reservations (name, people) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, name, people)
	return err
}

// GetAll は全ての予約データを取得する
func (r *PostgresReservationRepository) GetAll(ctx context.Context) ([]domain.Reservation, error) {
	query := `SELECT id, name, people, created_at FROM reservations`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []domain.Reservation
	for rows.Next() {
		var reservation domain.Reservation
		if err := rows.Scan(&reservation.ID, &reservation.Name, &reservation.People, &reservation.CreatedAt); err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reservations, nil
}
