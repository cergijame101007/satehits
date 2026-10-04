package usecase

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

// captureSlog は slog の既定ロガーを JSON でバッファへ向け、テスト終了時に戻す
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

// docs/monitoring.md §4.6 L4: RT の再利用を検知したら admin_user_id と検知箇所を WARN で残す
func TestRefreshUseCase_logsReuse(t *testing.T) {
	const plainToken = "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	now := time.Now()
	owner := domain.AdminUser{ID: 7, Email: "owner@example.com", Role: "owner"}
	activeRT := domain.RefreshToken{
		ID:          99,
		AdminUserID: owner.ID,
		TokenHash:   hashRefreshTokenPlain(plainToken),
		ExpiresAt:   now.Add(24 * time.Hour),
	}
	revokedAt := now.Add(-time.Minute)
	revokedRT := activeRT
	revokedRT.RevokedAt = &revokedAt
	notRevoked := false

	tests := []struct {
		name      string
		refresh   *refreshFakeTokenRepo
		wantStage string
	}{
		{
			name:      "logs reuse when the token was already revoked",
			refresh:   &refreshFakeTokenRepo{findToken: revokedRT},
			wantStage: `"stage":"lookup"`,
		},
		{
			name:      "logs reuse when a concurrent rotation already revoked the token",
			refresh:   &refreshFakeTokenRepo{findToken: activeRT, revokeIfActiveResult: &notRevoked},
			wantStage: `"stage":"rotate"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf := captureSlog(t)
			uc := newTestRefreshUseCase(t, &refreshFakeAdminRepo{user: owner}, tt.refresh, &stubTxManager{})

			_, err := uc.Execute(context.Background(), RefreshCommand{RefreshToken: plainToken})

			if !errors.Is(err, domain.ErrRefreshTokenInvalid) {
				t.Fatalf("err: got %v, want %v", err, domain.ErrRefreshTokenInvalid)
			}
			logText := logBuf.String()
			for _, want := range []string{`"msg":"refresh token reuse detected"`, `"level":"WARN"`, `"admin_user_id":7`, tt.wantStage} {
				if !strings.Contains(logText, want) {
					t.Errorf("log should contain %s: %s", want, logText)
				}
			}
			// RT の平文・ハッシュはログに出さない
			if strings.Contains(logText, plainToken) || strings.Contains(logText, activeRT.TokenHash) {
				t.Errorf("log should not contain the refresh token: %s", logText)
			}
		})
	}
}

func TestRefreshUseCase_doesNotLogReuseForExpiredToken(t *testing.T) {
	const plainToken = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	logBuf := captureSlog(t)
	expired := domain.RefreshToken{
		ID:          5,
		AdminUserID: 1,
		TokenHash:   hashRefreshTokenPlain(plainToken),
		ExpiresAt:   time.Now().Add(-time.Hour),
	}
	uc := newTestRefreshUseCase(t, &refreshFakeAdminRepo{}, &refreshFakeTokenRepo{findToken: expired}, &stubTxManager{})

	if _, err := uc.Execute(context.Background(), RefreshCommand{RefreshToken: plainToken}); !errors.Is(err, domain.ErrRefreshTokenInvalid) {
		t.Fatalf("err: got %v, want %v", err, domain.ErrRefreshTokenInvalid)
	}
	if logBuf.Len() != 0 {
		t.Errorf("expired token is not reuse and should not be logged: %s", logBuf.String())
	}
}
