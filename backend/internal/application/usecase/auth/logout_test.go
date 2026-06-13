package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

type logoutFakeRefreshTokenRepo struct {
	findToken domain.RefreshToken
	findErr   error
	lastHash  string

	revokeID    int64
	revokeErr   error
	revokeCalls int
}

func (f *logoutFakeRefreshTokenRepo) Issue(context.Context, domain.CreateRefreshTokenInput) (domain.RefreshToken, error) {
	return domain.RefreshToken{}, errors.New("not implemented")
}

func (f *logoutFakeRefreshTokenRepo) FindByTokenHash(_ context.Context, tokenHash string) (domain.RefreshToken, error) {
	f.lastHash = tokenHash
	if f.findErr != nil {
		return domain.RefreshToken{}, f.findErr
	}
	if f.findToken.ID == 0 {
		return domain.RefreshToken{}, domain.ErrRefreshTokenNotFound
	}
	return f.findToken, nil
}

func (f *logoutFakeRefreshTokenRepo) Revoke(_ context.Context, id int64) error {
	f.revokeCalls++
	f.revokeID = id
	return f.revokeErr
}

func (f *logoutFakeRefreshTokenRepo) RevokeIfActive(context.Context, int64) (bool, error) {
	return false, errors.New("not implemented")
}

func (f *logoutFakeRefreshTokenRepo) RevokeAllByUser(context.Context, int64) error {
	return errors.New("not implemented")
}

func newTestLogoutUseCase(refresh *logoutFakeRefreshTokenRepo) *LogoutUseCase {
	return NewLogoutUseCase(refresh)
}

func TestLogoutUseCase_Execute(t *testing.T) {
	plainToken := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tokenHash := hashRefreshTokenPlain(plainToken)
	revokedAt := time.Now()

	tests := []struct {
		name            string
		cmd             LogoutCommand
		refresh         *logoutFakeRefreshTokenRepo
		wantErrContains string
		check           func(t *testing.T, refresh *logoutFakeRefreshTokenRepo)
	}{
		{
			name:    "succeeds and revokes valid refresh token",
			cmd:     LogoutCommand{RefreshToken: plainToken},
			refresh: &logoutFakeRefreshTokenRepo{findToken: domain.RefreshToken{ID: 42, TokenHash: tokenHash}},
			check: func(t *testing.T, refresh *logoutFakeRefreshTokenRepo) {
				t.Helper()
				assertRefreshTokenHash(t, plainToken, refresh.lastHash)
				if refresh.revokeCalls != 1 {
					t.Fatalf("Revoke calls = %d, want 1", refresh.revokeCalls)
				}
				if refresh.revokeID != 42 {
					t.Fatalf("Revoke id = %d, want 42", refresh.revokeID)
				}
			},
		},
		{
			name: "succeeds when refresh token is empty",
			cmd:  LogoutCommand{RefreshToken: ""},
			refresh: &logoutFakeRefreshTokenRepo{
				findToken: domain.RefreshToken{ID: 99},
			},
			check: func(t *testing.T, refresh *logoutFakeRefreshTokenRepo) {
				t.Helper()
				if refresh.lastHash != "" {
					t.Fatalf("FindByTokenHash hash = %q, want empty (no lookup)", refresh.lastHash)
				}
				if refresh.revokeCalls != 0 {
					t.Fatalf("Revoke calls = %d, want 0", refresh.revokeCalls)
				}
			},
		},
		{
			name:    "succeeds when refresh token is not found",
			cmd:     LogoutCommand{RefreshToken: plainToken},
			refresh: &logoutFakeRefreshTokenRepo{},
			check: func(t *testing.T, refresh *logoutFakeRefreshTokenRepo) {
				t.Helper()
				if refresh.revokeCalls != 0 {
					t.Fatalf("Revoke calls = %d, want 0", refresh.revokeCalls)
				}
			},
		},
		{
			name: "succeeds when refresh token is already revoked",
			cmd:  LogoutCommand{RefreshToken: plainToken},
			refresh: &logoutFakeRefreshTokenRepo{
				findToken: domain.RefreshToken{ID: 7, TokenHash: tokenHash, RevokedAt: &revokedAt},
			},
			check: func(t *testing.T, refresh *logoutFakeRefreshTokenRepo) {
				t.Helper()
				if refresh.revokeCalls != 1 {
					t.Fatalf("Revoke calls = %d, want 1", refresh.revokeCalls)
				}
				if refresh.revokeID != 7 {
					t.Fatalf("Revoke id = %d, want 7", refresh.revokeID)
				}
			},
		},
		{
			name: "returns error when FindByTokenHash fails",
			cmd:  LogoutCommand{RefreshToken: plainToken},
			refresh: &logoutFakeRefreshTokenRepo{
				findErr: errors.New("connection refused"),
			},
			wantErrContains: "connection refused",
			check: func(t *testing.T, refresh *logoutFakeRefreshTokenRepo) {
				t.Helper()
				if refresh.revokeCalls != 0 {
					t.Fatalf("Revoke calls = %d, want 0", refresh.revokeCalls)
				}
			},
		},
		{
			name: "returns error when Revoke fails",
			cmd:  LogoutCommand{RefreshToken: plainToken},
			refresh: &logoutFakeRefreshTokenRepo{
				findToken: domain.RefreshToken{ID: 11, TokenHash: tokenHash},
				revokeErr: errors.New("update failed"),
			},
			wantErrContains: "failed to revoke refresh token",
			check: func(t *testing.T, refresh *logoutFakeRefreshTokenRepo) {
				t.Helper()
				if refresh.revokeCalls != 1 {
					t.Fatalf("Revoke calls = %d, want 1", refresh.revokeCalls)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newTestLogoutUseCase(tt.refresh)
			err := uc.Execute(context.Background(), tt.cmd)

			if tt.wantErrContains != "" {
				assertExecuteError(t, err, nil, tt.wantErrContains)
			} else if err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if tt.check != nil {
				tt.check(t, tt.refresh)
			}
		})
	}
}
