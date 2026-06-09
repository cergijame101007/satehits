package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/cergijame101007/satehits/internal/domain"
)

// LogoutCommand は管理者ログアウト入力（Cookie の RT 平文）
type LogoutCommand struct {
	RefreshToken string
}

// LogoutUseCase はログアウトユースケース
type LogoutUseCase struct {
	refreshTokenRepo domain.RefreshTokenRepository
}

// NewLogoutUseCase は LogoutUseCase のインスタンスを作成する
func NewLogoutUseCase(refreshTokenRepo domain.RefreshTokenRepository) *LogoutUseCase {
	return &LogoutUseCase{refreshTokenRepo: refreshTokenRepo}
}

// Execute は RT を revoke する。AT 検証はミドルウェア側の責務。
// RT が空・未登録・失効済みでも成功扱い（204。Cookie 削除は handler 側）。
func (u *LogoutUseCase) Execute(ctx context.Context, cmd LogoutCommand) error {
	if cmd.RefreshToken == "" {
		return nil
	}

	tokenHash := HashRefreshTokenPlain(cmd.RefreshToken)

	rt, err := u.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil
		}
		return err
	}

	if err := u.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}
