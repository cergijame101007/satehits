package domain

import "errors"

// ErrCaptchaFailed は Turnstile 検証に失敗した
var ErrCaptchaFailed = errors.New("captcha failed")
