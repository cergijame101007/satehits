package usecase

import (
	"context"
	"strings"

	"github.com/cergijame101007/satehits/internal/domain"
)

// CaptchaVerifier は Turnstile 等の CAPTCHA トークン検証
type CaptchaVerifier interface {
	Verify(ctx context.Context, token string) error
}

// DevBypassTurnstileToken は開発環境のみ NoOpCaptchaVerifier が許可するプレースホルダ
const DevBypassTurnstileToken = "dev-bypass"

// NoOpCaptchaVerifier は開発環境用の検証スキップ実装
type NoOpCaptchaVerifier struct{}

func (NoOpCaptchaVerifier) Verify(_ context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return domain.ErrCaptchaFailed
	}
	return nil
}
