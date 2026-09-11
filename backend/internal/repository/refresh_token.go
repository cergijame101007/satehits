package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresRefreshTokenRepository はPostgreSQLを使ったリフレッシュトークンリポジトリの実装
type PostgresRefreshTokenRepository struct {
	baseRepository
}

// NewPostgresRefreshTokenRepository はPostgresRefreshTokenRepositoryのインスタンスを作成する
func NewPostgresRefreshTokenRepository(db *sql.DB) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{baseRepository{db: db}}
}

// Issue はリフレッシュトークンを永続化し挿入結果を返す
func (r *PostgresRefreshTokenRepository) Issue(ctx context.Context, input domain.CreateRefreshTokenInput) (domain.RefreshToken, error) {
	query := `INSERT INTO refresh_tokens (admin_user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id, admin_user_id, token_hash, expires_at, revoked_at, created_at`
	var out domain.RefreshToken
	err := r.getDB(ctx).QueryRowContext(ctx, query,
		input.AdminUserID, input.TokenHash, input.ExpiresAt,
	).Scan(
		&out.ID,
		&out.AdminUserID,
		&out.TokenHash,
		&out.ExpiresAt,
		&out.RevokedAt,
		&out.CreatedAt,
	)
	if err != nil {
		return domain.RefreshToken{}, err
	}
	return out, nil
}

// FindByTokenHash はトークンハッシュでリフレッシュトークンを検索する
func (r *PostgresRefreshTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (domain.RefreshToken, error) {
	query := `SELECT id, admin_user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE token_hash = $1`
	var out domain.RefreshToken
	err := r.getDB(ctx).QueryRowContext(ctx, query, tokenHash).Scan(
		&out.ID,
		&out.AdminUserID,
		&out.TokenHash,
		&out.ExpiresAt,
		&out.RevokedAt,
		&out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RefreshToken{}, domain.ErrRefreshTokenNotFound
		}
		return domain.RefreshToken{}, err
	}
	return out, nil
}

// Revoke はリフレッシュトークンを失効させる（ログアウト用）
// 存在しない id・既に revoke 済みは 0 行更新だがエラーにしない
func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	_, err := r.getDB(ctx).ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	return nil
}

// RevokeIfActive は未失効の RT を revoke する（refresh ローテーション用）
// 1 行更新できたら true。既に revoke 済み・存在しない id は false（エラーなし）
func (r *PostgresRefreshTokenRepository) RevokeIfActive(ctx context.Context, id int64) (bool, error) {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND revoked_at IS NULL`
	res, err := r.getDB(ctx).ExecContext(ctx, query, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// RevokeAllByUser はユーザーIDでリフレッシュトークンを失効させる
func (r *PostgresRefreshTokenRepository) RevokeAllByUser(ctx context.Context, adminUserID int64) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE admin_user_id = $1 AND revoked_at IS NULL`
	_, err := r.getDB(ctx).ExecContext(ctx, query, adminUserID)
	if err != nil {
		return err
	}
	return nil
}

// DeleteExpired は有効期限を過ぎたリフレッシュトークン行を削除し、削除件数を返す
// revoke 済みでも未期限の行は再利用検知（docs/table_design.md §2.4）のため残す
func (r *PostgresRefreshTokenRepository) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`
	res, err := r.getDB(ctx).ExecContext(ctx, query, now)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}
