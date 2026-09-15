package usecase

import (
	"strings"
	"testing"
	"time"
)

func TestRetryAfterFromOldest(t *testing.T) {
	window := 15 * time.Minute
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		oldest time.Time
		want   time.Duration
	}{
		{
			name:   "returns full window when oldest is zero",
			oldest: time.Time{},
			want:   window,
		},
		{
			name:   "returns exact remaining seconds",
			oldest: now.Add(-5 * time.Minute),
			want:   10 * time.Minute,
		},
		{
			name:   "rounds fractional remaining seconds up",
			oldest: now.Add(-5*time.Minute - 300*time.Millisecond),
			want:   10 * time.Minute,
		},
		{
			name:   "returns at least one second when window is almost over",
			oldest: now.Add(-window + 200*time.Millisecond),
			want:   time.Second,
		},
		{
			name:   "returns one second when window has already elapsed",
			oldest: now.Add(-window - time.Minute),
			want:   time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := retryAfterFromOldest(tt.oldest, window, now)
			if got != tt.want {
				t.Fatalf("retryAfterFromOldest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRateLimitedError_Error(t *testing.T) {
	tests := []struct {
		name       string
		retryAfter time.Duration
		want       string
	}{
		{name: "shows one minute for one second", retryAfter: time.Second, want: "1分後"},
		{name: "rounds 61 seconds up to two minutes", retryAfter: 61 * time.Second, want: "2分後"},
		{name: "shows exact minutes", retryAfter: 10 * time.Minute, want: "10分後"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RateLimitedError{RetryAfter: tt.retryAfter}
			if got := err.Error(); !strings.Contains(got, tt.want) {
				t.Fatalf("Error() = %q, want to contain %q", got, tt.want)
			}
		})
	}
}
