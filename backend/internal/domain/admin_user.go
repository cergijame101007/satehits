package domain

import (
	"context"
	"errors"
	"time"
)

// ErrAdminUserNotFound は管理者ユーザーが見つからないエラー
var ErrAdminUserNotFound = errors.New("admin user not found")

// ErrAdminUserUnauthorized は管理者ユーザーが認証失敗したエラー
var ErrAdminUserUnauthorized = errors.New("admin user unauthorized")

// AdminUser は管理者ユーザーのドメインエンティティ
type AdminUser struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// AdminUserRepository は管理者ユーザーの永続化するためのインターフェース
type AdminUserRepository interface {
	FindByEmail(ctx context.Context, email string) (AdminUser, error)
	FindByID(ctx context.Context, id int64) (AdminUser, error)
}
