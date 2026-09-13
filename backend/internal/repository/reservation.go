package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresReservationRepository はPostgreSQLを使った予約リポジトリの実装
type PostgresReservationRepository struct {
	baseRepository
}

// NewPostgresReservationRepository はPostgresReservationRepositoryのインスタンスを作成する
func NewPostgresReservationRepository(db *sql.DB) *PostgresReservationRepository {
	return &PostgresReservationRepository{baseRepository{db: db}}
}

// Create は予約データを永続化し挿入結果を返す
func (r *PostgresReservationRepository) Create(ctx context.Context, in domain.CreateReservationInput) (domain.Reservation, error) {
	query := `
INSERT INTO reservations (
    name, people, visit_date, visit_time, phone, email, note, status, source
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, name, people, visit_date, visit_time, phone, email, COALESCE(note, ''), status, source, created_at, updated_at`
	var out domain.Reservation
	err := r.getDB(ctx).QueryRowContext(ctx, query,
		in.Name, in.People, in.VisitDate, in.VisitTime, in.Phone, in.Email, in.Note, in.Status, in.Source,
	).Scan(
		&out.ID,
		&out.Name,
		&out.People,
		&out.VisitDate,
		&out.VisitTime,
		&out.Phone,
		&out.Email,
		&out.Note,
		&out.Status,
		&out.Source,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Reservation{}, domain.ErrReservationConflict
		}
		return domain.Reservation{}, err
	}
	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

const reservationSelectColumns = `
SELECT id, name, people, visit_date, visit_time, phone, email,
       COALESCE(note, ''), status, source, created_at, updated_at
FROM reservations`

func scanReservation(row interface {
	Scan(dest ...any) error
}) (domain.Reservation, error) {
	var reservation domain.Reservation
	err := row.Scan(
		&reservation.ID,
		&reservation.Name,
		&reservation.People,
		&reservation.VisitDate,
		&reservation.VisitTime,
		&reservation.Phone,
		&reservation.Email,
		&reservation.Note,
		&reservation.Status,
		&reservation.Source,
		&reservation.CreatedAt,
		&reservation.UpdatedAt,
	)
	return reservation, err
}

// List は管理者向け予約一覧を取得する（絞り込み条件は usecase で検証済み）
func (r *PostgresReservationRepository) List(ctx context.Context, f domain.ListReservationsFilter) ([]domain.Reservation, error) {
	var (
		conditions []string
		args       []any
		argNum     = 1
	)

	if f.Date != nil {
		conditions = append(conditions, fmt.Sprintf("visit_date = $%d", argNum))
		args = append(args, *f.Date)
		argNum++
	}
	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, f.Status)
		argNum++
	}
	if f.Source != "" {
		conditions = append(conditions, fmt.Sprintf("source = $%d", argNum))
		args = append(args, f.Source)
	}

	query := reservationSelectColumns
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY visit_date, visit_time"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []domain.Reservation
	for rows.Next() {
		reservation, err := scanReservation(rows)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reservations, nil
}

// GetByID は ID で予約を1件取得する
func (r *PostgresReservationRepository) GetByID(ctx context.Context, id uuid.UUID) (domain.Reservation, error) {
	query := reservationSelectColumns + " WHERE id = $1"
	reservation, err := scanReservation(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Reservation{}, domain.ErrReservationNotFound
		}
		return domain.Reservation{}, err
	}
	return reservation, nil
}

// UpdateStatus は予約ステータスを更新する
func (r *PostgresReservationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (domain.Reservation, error) {
	query := `
UPDATE reservations SET status = $1, updated_at = NOW() WHERE id = $2
RETURNING id, name, people, visit_date, visit_time, phone, email,
          COALESCE(note, ''), status, source, created_at, updated_at`
	reservation, err := scanReservation(r.db.QueryRowContext(ctx, query, status, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Reservation{}, domain.ErrReservationNotFound
		}
		if isUniqueViolation(err) {
			return domain.Reservation{}, domain.ErrReservationConflict
		}
		return domain.Reservation{}, err
	}
	return reservation, nil
}

// SumReservedPeopleByDate は指定日の予約済み人数合計（pending + approved）を返す
// NOTE: ローンチ前に要確認 pending を reserved に含める仮方針
func (r *PostgresReservationRepository) SumReservedPeopleByDate(ctx context.Context, date datetime.Date) (int, error) {
	query := `
SELECT COALESCE(SUM(people), 0)
FROM reservations
WHERE visit_date = $1 AND status IN ('pending', 'approved')`
	var total int
	if err := r.getDB(ctx).QueryRowContext(ctx, query, date).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

// SumReservedPeopleByDateRange は期間内の日付別・予約済み人数合計（pending + approved）を返す
// NOTE: ローンチ前に要確認 pending を reserved に含める仮方針
func (r *PostgresReservationRepository) SumReservedPeopleByDateRange(ctx context.Context, from, to datetime.Date) (map[string]int, error) {
	query := `
SELECT visit_date, COALESCE(SUM(people), 0)
FROM reservations
WHERE visit_date >= $1 AND visit_date <= $2 AND status IN ('pending', 'approved')
GROUP BY visit_date`
	rows, err := r.getDB(ctx).QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var date datetime.Date
		var total int
		if err := rows.Scan(&date, &total); err != nil {
			return nil, err
		}
		result[date.String()] = total
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

var _ domain.PendingReminderReservationReader = (*PostgresReservationRepository)(nil)

// ListPendingWebByVisitDateRange は status=pending かつ source=web で来店日が [from, to] の予約を返す（UC-S04）
func (r *PostgresReservationRepository) ListPendingWebByVisitDateRange(ctx context.Context, from, to datetime.Date) ([]domain.Reservation, error) {
	query := reservationSelectColumns + `
WHERE status = 'pending' AND source = 'web'
  AND visit_date >= $1 AND visit_date <= $2
ORDER BY visit_date, visit_time, created_at`
	rows, err := r.getDB(ctx).QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("list pending web reservations: %w", err)
	}
	defer rows.Close()

	var reservations []domain.Reservation
	for rows.Next() {
		reservation, err := scanReservation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pending web reservation: %w", err)
		}
		reservations = append(reservations, reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending web reservations: %w", err)
	}
	return reservations, nil
}

// CountPendingFrom は来店日が from 以降の pending 予約数（source を問わない）を返す
func (r *PostgresReservationRepository) CountPendingFrom(ctx context.Context, from datetime.Date) (int, error) {
	var total int
	err := r.getDB(ctx).QueryRowContext(ctx, `
SELECT COUNT(*) FROM reservations
WHERE status = 'pending' AND visit_date >= $1`, from).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count pending reservations: %w", err)
	}
	return total, nil
}
