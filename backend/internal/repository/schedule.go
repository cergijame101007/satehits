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

// Upsert は daily_schedules を日付キーで挿入または更新する
func (r *PostgresScheduleRepository) Upsert(ctx context.Context, in domain.SetScheduleInput) (domain.Schedule, bool, error) {
	query := `
INSERT INTO daily_schedules (
	date, schedule_type, capacity, event_name, event_description, open_time, last_order_time, close_time
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (date) DO UPDATE SET
	schedule_type = EXCLUDED.schedule_type,
	capacity = EXCLUDED.capacity,
	event_name = EXCLUDED.event_name,
	event_description = EXCLUDED.event_description,
	open_time = EXCLUDED.open_time,
	last_order_time = EXCLUDED.last_order_time,
	close_time = EXCLUDED.close_time
RETURNING
	date, schedule_type, capacity, event_name, event_description,
	open_time, last_order_time, close_time, created_at, updated_at,
	(xmax = 0) AS inserted`
	var out domain.Schedule
	var inserted bool
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
		&inserted,
	)
	return out, inserted, err
}
