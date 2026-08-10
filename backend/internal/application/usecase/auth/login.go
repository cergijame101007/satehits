package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

const refreshTokenTTL = 30 * 24 * time.Hour

// FieldViolation はフィールド単位のバリデーションエラー
// handler.ErrorDetail と同形の別定義（handler 非依存のため）
type FieldViolation struct {
	Field   string
	Message string
}

// ValidationError はユースケース入力の検証失敗
type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string {
	return "入力内容に誤りがあります"
}

// LoginCommand は管理者ユーザーの Web ログイン入力
type LoginCommand struct {
	Email    string
	Password string
	ClientIP string
}

// LoginUseCase はログインユースケース
type LoginUseCase struct {
	adminUserRepo    domain.AdminUserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	loginAttemptRepo domain.LoginAttemptRepository
	jwtService       *jwt.JWTService
	rateLimit        LoginRateLimitPolicy
}

// NewLoginUseCase はLoginUseCaseのインスタンスを作成する
func NewLoginUseCase(
	adminUserRepo domain.AdminUserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	loginAttemptRepo domain.LoginAttemptRepository,
	jwtService *jwt.JWTService,
	rateLimit LoginRateLimitPolicy,
) *LoginUseCase {
	return &LoginUseCase{
		adminUserRepo:    adminUserRepo,
		refreshTokenRepo: refreshTokenRepo,
		loginAttemptRepo: loginAttemptRepo,
		jwtService:       jwtService,
		rateLimit:        rateLimit,
	}
}

type LoginUser struct {
	ID    int64
	Email string
	Role  string
}

// LoginResult はログイン結果
type LoginResult struct {
	AccessToken  string
	ExpiresAt    time.Time
	RefreshToken string
	User         LoginUser
}

func (u *LoginUseCase) Execute(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	violations := validateLogin(cmd)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	emailKey := normalizeLoginEmailKey(cmd.Email)
	now := time.Now()
	since := now.Add(-u.rateLimit.Window)

	counts, err := u.loginAttemptRepo.CountRecent(ctx, emailKey, cmd.ClientIP, since)
	if err != nil {
		return nil, fmt.Errorf("failed to count login attempts: %w", err)
	}
	if err := u.checkRateLimit(counts, now); err != nil {
		return nil, err
	}

	user, err := u.adminUserRepo.FindByEmail(ctx, strings.TrimSpace(cmd.Email))
	if err != nil {
		if errors.Is(err, domain.ErrAdminUserNotFound) {
			if recErr := u.loginAttemptRepo.RecordFailure(ctx, emailKey, cmd.ClientIP, now); recErr != nil {
				return nil, fmt.Errorf("failed to record login failure: %w", recErr)
			}
			return nil, domain.ErrAdminUserUnauthorized
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.Password)); err != nil {
		if recErr := u.loginAttemptRepo.RecordFailure(ctx, emailKey, cmd.ClientIP, now); recErr != nil {
			return nil, fmt.Errorf("failed to record login failure: %w", recErr)
		}
		return nil, domain.ErrAdminUserUnauthorized
	}

	if err := u.loginAttemptRepo.ClearByEmail(ctx, emailKey); err != nil {
		return nil, fmt.Errorf("failed to clear login attempts: %w", err)
	}

	accessToken, expiresAt, err := u.jwtService.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// リフレッシュトークンを作成
	refreshToken, refreshTokenHash, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	_, err = u.refreshTokenRepo.Issue(ctx, domain.CreateRefreshTokenInput{
		AdminUserID: user.ID,
		TokenHash:   refreshTokenHash,
		ExpiresAt:   time.Now().Add(refreshTokenTTL),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to issue refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		ExpiresAt:    expiresAt,
		RefreshToken: refreshToken,
		User: LoginUser{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		},
	}, nil
}

func (u *LoginUseCase) checkRateLimit(counts domain.LoginAttemptCounts, now time.Time) error {
	emailLimited := counts.ByEmail >= u.rateLimit.EmailMax
	ipLimited := counts.ByIP >= u.rateLimit.IPMax
	if !emailLimited && !ipLimited {
		return nil
	}

	var retryAfter time.Duration
	if emailLimited {
		retryAfter = retryAfterFromOldest(counts.OldestByEmail, u.rateLimit.Window, now)
	}
	if ipLimited {
		ipRetry := retryAfterFromOldest(counts.OldestByIP, u.rateLimit.Window, now)
		if ipRetry > retryAfter {
			retryAfter = ipRetry
		}
	}
	return &RateLimitedError{RetryAfter: retryAfter}
}

// generateRefreshToken はリフレッシュトークンを生成する
func generateRefreshToken() (token string, tokenHash string, err error) {
	b := make([]byte, 32) // 256bit のエントロピー
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b)

	tokenHash = hashRefreshTokenPlain(token)

	return token, tokenHash, nil
}
