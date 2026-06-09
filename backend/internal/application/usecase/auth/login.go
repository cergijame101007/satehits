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
}

// LoginUseCase はログインユースケース
type LoginUseCase struct {
	adminUserRepo    domain.AdminUserRepository
	refreshTokenRepo domain.RefreshTokenRepository
	jwtService       *jwt.JWTService
}

// NewLoginUseCase はLoginUseCaseのインスタンスを作成する
func NewLoginUseCase(
	adminUserRepo domain.AdminUserRepository,
	refreshTokenRepo domain.RefreshTokenRepository,
	jwtService *jwt.JWTService,
) *LoginUseCase {
	return &LoginUseCase{adminUserRepo: adminUserRepo, refreshTokenRepo: refreshTokenRepo, jwtService: jwtService}
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

	user, err := u.adminUserRepo.FindByEmail(ctx, strings.TrimSpace(cmd.Email))
	if err != nil {
		if errors.Is(err, domain.ErrAdminUserNotFound) {
			return nil, domain.ErrAdminUserUnauthorized
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.Password)); err != nil {
		return nil, domain.ErrAdminUserUnauthorized
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
