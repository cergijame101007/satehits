package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

const loginAttemptRetention = 24 * time.Hour

// PostgresLoginAttemptRepository は PostgreSQL を使ったログイン失敗試行リポジトリ
type PostgresLoginAttemptRepository struct {
	baseRepository
}

// NewPostgresLoginAttemptRepository は PostgresLoginAttemptRepository を生成する
func NewPostgresLoginAttemptRepository(db *sql.DB) *PostgresLoginAttemptRepository {
	return &PostgresLoginAttemptRepository{baseRepository{db: db}}
}

// CountRecent は窓内のメール単位・IP 単位の失敗件数と最古時刻を同時に返す
func (r *PostgresLoginAttemptRepository) CountRecent(
	ctx context.Context,
	emailKey, ip string,
	since time.Time,
) (domain.LoginAttemptCounts, error) {
	query := `
SELECT
    COUNT(*) FILTER (WHERE email_key = $1),
    MIN(attempted_at) FILTER (WHERE email_key = $1),
    COUNT(*) FILTER (WHERE ip = $2),
    MIN(attempted_at) FILTER (WHERE ip = $2)
FROM login_attempts
WHERE attempted_at >= $3 AND (email_key = $1 OR ip = $2)`

	var (
		byEmail       int
		oldestByEmail sql.NullTime
		byIP          int
		oldestByIP    sql.NullTime
	)
	err := r.getDB(ctx).QueryRowContext(ctx, query, emailKey, ip, since).Scan(
		&byEmail,
		&oldestByEmail,
		&byIP,
		&oldestByIP,
	)
	if err != nil {
		return domain.LoginAttemptCounts{}, err
	}

	out := domain.LoginAttemptCounts{
		ByEmail: byEmail,
		ByIP:    byIP,
	}
	if oldestByEmail.Valid {
		out.OldestByEmail = oldestByEmail.Time
	}
	if oldestByIP.Valid {
		out.OldestByIP = oldestByIP.Time
	}
	return out, nil
}

// RecordFailure は失敗試行を記録し、保持期間を超えた行を削除する
func (r *PostgresLoginAttemptRepository) RecordFailure(
	ctx context.Context,
	emailKey, ip string,
	at time.Time,
) error {
	db := r.getDB(ctx)
	_, err := db.ExecContext(ctx,
		`INSERT INTO login_attempts (email_key, ip, attempted_at) VALUES ($1, $2, $3)`,
		emailKey, ip, at,
	)
	if err != nil {
		return err
	}

	cutoff := at.Add(-loginAttemptRetention)
	_, err = db.ExecContext(ctx, `DELETE FROM login_attempts WHERE attempted_at < $1`, cutoff)
	return err
}

// ClearByEmail は指定メールキーの失敗試行をすべて削除する
func (r *PostgresLoginAttemptRepository) ClearByEmail(ctx context.Context, emailKey string) error {
	_, err := r.getDB(ctx).ExecContext(ctx, `DELETE FROM login_attempts WHERE email_key = $1`, emailKey)
	return err
}
