package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/cergijame101007/satehits/internal/domain"
)

// PostgresAdminUserRepository は PostgreSQL を使った管理者ユーザーリポジトリの実装
// 管理者ユーザーは seed で DB に手動登録のみで、アプリからの作成・更新・削除は不要
type PostgresAdminUserRepository struct {
	db *sql.DB
}

// NewPostgresAdminUserRepository はPostgresAdminUserRepositoryのインスタンスを作成する
func NewPostgresAdminUserRepository(db *sql.DB) *PostgresAdminUserRepository {
	return &PostgresAdminUserRepository{db: db}
}

// FindByEmail はメールアドレスで管理者ユーザーを検索する
func (r *PostgresAdminUserRepository) FindByEmail(ctx context.Context, email string) (domain.AdminUser, error) {
	query := `SELECT id, email, password_hash, role, created_at, updated_at FROM admin_users WHERE email = $1`
	var out domain.AdminUser
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&out.ID,
		&out.Email,
		&out.PasswordHash,
		&out.Role,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AdminUser{}, domain.ErrAdminUserNotFound
		}
		return domain.AdminUser{}, err
	}
	return out, nil
}

// FindByID はユーザーIDで管理者ユーザーを検索する
func (r *PostgresAdminUserRepository) FindByID(ctx context.Context, id int64) (domain.AdminUser, error) {
	query := `SELECT id, email, password_hash, role, created_at, updated_at FROM admin_users WHERE id = $1`
	var out domain.AdminUser
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&out.ID,
		&out.Email,
		&out.PasswordHash,
		&out.Role,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AdminUser{}, domain.ErrAdminUserNotFound
		}
		return domain.AdminUser{}, err
	}
	return out, nil
}
