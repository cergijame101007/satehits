package domain

import (
	"context"
	"time"
)

// LoginAttemptCounts はスライディングウィンドウ内の失敗試行集計
type LoginAttemptCounts struct {
	ByEmail       int
	ByIP          int
	OldestByEmail time.Time // 該当なしはゼロ値
	OldestByIP    time.Time
}

// LoginAttemptRepository はログイン失敗試行の永続化インターフェース
type LoginAttemptRepository interface {
	CountRecent(ctx context.Context, emailKey, ip string, since time.Time) (LoginAttemptCounts, error)
	RecordFailure(ctx context.Context, emailKey, ip string, at time.Time) error
	ClearByEmail(ctx context.Context, emailKey string) error
}
