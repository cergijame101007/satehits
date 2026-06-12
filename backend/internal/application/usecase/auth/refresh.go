package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

type RefreshCommand struct {
	RefreshToken string
}

type RefreshUseCase struct {
	adminUserRepo    domain.AdminUserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       *jwt.JWTService
	txManager        application.TxManager
}

func NewRefreshUseCase(
	adminUserRepo domain.AdminUserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService *jwt.JWTService,
	txManager application.TxManager,
) *RefreshUseCase {
	return &RefreshUseCase{
		adminUserRepo:    adminUserRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtService:       jwtService,
		txManager:        txManager,
	}
}

type RefreshResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
}

func (u *RefreshUseCase) Execute(ctx context.Context, cmd RefreshCommand) (*RefreshResult, error) {
	if cmd.RefreshToken == "" {
		return nil, domain.ErrRefreshTokenNotFound
	}

	now := time.Now()

	tokenHash := hashRefreshTokenPlain(cmd.RefreshToken)

	rt, err := u.refreshTokenRepo.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenNotFound) {
			return nil, domain.ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	// revoke 済み RT の再利用検知: 当該ユーザーの全 RT を失効させる
	if rt.RevokedAt != nil {
		if err := u.refreshTokenRepo.RevokeAllByUser(ctx, rt.AdminUserID); err != nil {
			return nil, fmt.Errorf("failed to revoke refresh token: %w", err)
		}
		return nil, domain.ErrRefreshTokenInvalid
	}

	if !rt.ExpiresAt.After(now) {
		return nil, domain.ErrRefreshTokenInvalid
	}

	user, err := u.adminUserRepo.FindByID(ctx, rt.AdminUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to find admin user: %w", err)
	}

	accessToken, expiresAt, err := u.jwtService.GenerateAccessToken(rt.AdminUserID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshTokenHash, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	if err := u.txManager.DoInTx(ctx, func(ctx context.Context) error {
		if err := u.refreshTokenRepo.Revoke(ctx, rt.ID); err != nil {
			return err
		}
		_, err := u.refreshTokenRepo.Issue(ctx, domain.CreateRefreshTokenInput{
			AdminUserID: rt.AdminUserID,
			TokenHash:   refreshTokenHash,
			ExpiresAt:   now.Add(refreshTokenTTL),
		})
		return err
	}); err != nil {
		return nil, fmt.Errorf("failed to rotate refresh token: %w", err)
	}

	return &RefreshResult{AccessToken: accessToken, ExpiresAt: expiresAt, RefreshToken: refreshToken}, nil
}
