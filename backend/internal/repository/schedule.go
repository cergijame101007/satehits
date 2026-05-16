package repository

import (
	"context"
	"database/sql"

	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresScheduleRepository はPostgreSQLを使ったスケジュールリポジトリの実装
type PostgresScheduleRepository struct {
	db *sql.DB
}

// NewPostgresScheduleRepository はPostgresScheduleRepositoryのインスタンスを作成する
func NewPostgresScheduleRepository(db *sql.DB) *PostgresScheduleRepository {
	return &PostgresScheduleRepository{db: db}
}

// Create はスケジュールデータを永続化し挿入結果を返す
func (r *PostgresScheduleRepository) Create(ctx context.Context, in domain.CreateScheduleInput) (domain.Schedule, error) {
	query := `
INSERT INTO daily_schedules (
	date, schedule_type, capacity, event_name, event_description, open_time, last_order_time, close_time
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING date, schedule_type, capacity, event_name, event_description, open_time, last_order_time, close_time, created_at, updated_at`
	var out domain.Schedule
	err := r.db.QueryRowContext(ctx, query,
		in.Date, in.ScheduleType, in.Capacity, in.EventName, in.EventDescription, in.OpenTime, in.LastOrderTime, in.CloseTime,
	).Scan(
		&out.Date,
		&out.ScheduleType,
		&out.Capacity,
		&out.EventName,
		&out.EventDescription,
		&out.OpenTime,
		&out.LastOrderTime,
		&out.CloseTime,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	return out, err
}