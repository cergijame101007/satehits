package usecase

import "context"

// CaptchaVerifier は Turnstile 等の CAPTCHA トークン検証
type CaptchaVerifier interface {
	Verify(ctx context.Context, token, remoteIP string) error
}

// NoOpCaptchaVerifier は開発環境用の検証スキップ実装
type NoOpCaptchaVerifier struct{}

func (NoOpCaptchaVerifier) Verify(context.Context, string, string) error {
	return nil
}
