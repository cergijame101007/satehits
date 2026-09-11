package domain

import (
	"context"
	"errors"
	"time"
)

// ErrRefreshTokenNotFound はリフレッシュトークンが見つからないエラー
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

// ErrRefreshTokenInvalid はリフレッシュトークンが無効なエラー
var ErrRefreshTokenInvalid = errors.New("refresh token invalid")

// RefreshToken はリフレッシュトークンのドメインエンティティ
type RefreshToken struct {
	ID          int64      `json:"id"`
	AdminUserID int64      `json:"admin_user_id"`
	TokenHash   string     `json:"token_hash"`
	ExpiresAt   time.Time  `json:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateRefreshTokenInput struct {
	AdminUserID int64
	TokenHash   string
	ExpiresAt   time.Time
}

type RefreshTokenRepository interface {
	Issue(ctx context.Context, input CreateRefreshTokenInput) (RefreshToken, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	Revoke(ctx context.Context, id int64) error
	RevokeIfActive(ctx context.Context, id int64) (bool, error)
	RevokeAllByUser(ctx context.Context, adminUserID int64) error
	// DeleteExpired は expires_at < now の行（revoke 済みかどうかを問わない）を削除し、削除件数を返す
	// 期限切れ RT は再利用検知にも使えないため安全に消せる。未期限の revoke 済み行は再利用検知のため残す
	DeleteExpired(ctx context.Context, now time.Time) (int64, error)
}
