package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
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
	var eventName, eventDesc sql.NullString
	err := r.db.QueryRowContext(ctx, query,
		in.Date, in.ScheduleType, in.Capacity,
		eventTextParam(in.EventName), eventTextParam(in.EventDescription),
		in.OpenTime, in.LastOrderTime, in.CloseTime,
	).Scan(
		&out.Date,
		&out.ScheduleType,
		&out.Capacity,
		&eventName,
		&eventDesc,
		&out.OpenTime,
		&out.LastOrderTime,
		&out.CloseTime,
		&out.CreatedAt,
		&out.UpdatedAt,
		&inserted,
	)
	if err != nil {
		return out, false, err
	}
	out.EventName = scanEventText(eventName)
	out.EventDescription = scanEventText(eventDesc)
	return out, inserted, nil
}

const scheduleSelectColumns = `
	date, schedule_type, capacity, event_name, event_description,
	open_time, last_order_time, close_time, created_at, updated_at`

func scanSchedule(row interface {
	Scan(dest ...any) error
}) (domain.Schedule, error) {
	var out domain.Schedule
	var eventName, eventDesc sql.NullString
	err := row.Scan(
		&out.Date,
		&out.ScheduleType,
		&out.Capacity,
		&eventName,
		&eventDesc,
		&out.OpenTime,
		&out.LastOrderTime,
		&out.CloseTime,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return domain.Schedule{}, err
	}
	out.EventName = scanEventText(eventName)
	out.EventDescription = scanEventText(eventDesc)
	return out, nil
}

// scanEventText は DB の NULL / 空をドメインの空文字に正規化する
func scanEventText(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

// eventTextParam は空・空白のみを NULL、それ以外を TEXT として DB に渡す
func eventTextParam(s string) sql.NullString {
	s = strings.TrimSpace(s)
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// FindByDate は指定日の保存済みスケジュールを返す
func (r *PostgresScheduleRepository) FindByDate(ctx context.Context, date datetime.Date) (domain.Schedule, bool, error) {
	query := `SELECT` + scheduleSelectColumns + ` FROM daily_schedules WHERE date = $1`
	out, err := scanSchedule(r.db.QueryRowContext(ctx, query, date))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Schedule{}, false, nil
	}
	if err != nil {
		return domain.Schedule{}, false, err
	}
	return out, true, nil
}

// ListStoredByYearMonth は指定年月に保存されている行のみを日付昇順で返す
// year/month は呼び出し前に service 側で検証済みであること
func (r *PostgresScheduleRepository) ListStoredByYearMonth(ctx context.Context, year, month int) ([]domain.Schedule, error) {
	from, to := service.MonthDateRange(year, month)

	query := `SELECT` + scheduleSelectColumns + `
FROM daily_schedules
WHERE date >= $1 AND date <= $2
ORDER BY date`
	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.Schedule
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}
