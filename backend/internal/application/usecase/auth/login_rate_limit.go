package usecase

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// LoginRateLimitPolicy はログイン失敗のレートリミットしきい値
type LoginRateLimitPolicy struct {
	EmailMax int
	IPMax    int
	Window   time.Duration
}

// RateLimitedError はログイン試行がしきい値を超えたときのエラー
type RateLimitedError struct {
	RetryAfter time.Duration
}

func (e *RateLimitedError) Error() string {
	minutes := int(math.Ceil(e.RetryAfter.Minutes()))
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("ログイン試行回数が上限に達しました。%d分後に再度お試しください", minutes)
}

func normalizeLoginEmailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func retryAfterFromOldest(oldest time.Time, window time.Duration, now time.Time) time.Duration {
	if oldest.IsZero() {
		return window
	}
	until := oldest.Add(window).Sub(now)
	if until < time.Second {
		return time.Second
	}
	return until.Truncate(time.Second)
}
