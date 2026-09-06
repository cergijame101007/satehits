package mail

import (
	"testing"
	"time"
)

func TestNextRetryAt(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name         string
		attemptCount int
		wantOK       bool
		wantAfter    time.Duration
	}{
		{name: "attempt 1 schedules +1m", attemptCount: 1, wantOK: true, wantAfter: 1 * time.Minute},
		{name: "attempt 2 schedules +5m", attemptCount: 2, wantOK: true, wantAfter: 5 * time.Minute},
		{name: "attempt 3 schedules +15m", attemptCount: 3, wantOK: true, wantAfter: 15 * time.Minute},
		{name: "attempt 4 schedules +1h", attemptCount: 4, wantOK: true, wantAfter: 1 * time.Hour},
		{name: "attempt 5 schedules +4h", attemptCount: 5, wantOK: true, wantAfter: 4 * time.Hour},
		{name: "attempt 0 is invalid", attemptCount: 0, wantOK: false},
		{name: "attempt 6 is exhausted", attemptCount: 6, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := NextRetryAt(tt.attemptCount, now)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			want := now.Add(tt.wantAfter)
			if !got.Equal(want) {
				t.Fatalf("next = %v, want %v", got, want)
			}
		})
	}
}

func TestShouldMarkFailed(t *testing.T) {
	tests := []struct {
		name         string
		attemptCount int
		want         bool
	}{
		{name: "attempt 5 still retries", attemptCount: 5, want: false},
		{name: "attempt 6 marks failed", attemptCount: 6, want: true},
		{name: "attempt 7 marks failed", attemptCount: 7, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldMarkFailed(tt.attemptCount); got != tt.want {
				t.Fatalf("ShouldMarkFailed(%d) = %v, want %v", tt.attemptCount, got, tt.want)
			}
		})
	}
}
