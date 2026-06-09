package domain

import (
	"context"
	"errors"
	"time"
)

// ErrRefreshTokenNotFound はリフレッシュトークンが見つからないエラー
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

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
	RevokeAllByUser(ctx context.Context, adminUserID int64) error
}