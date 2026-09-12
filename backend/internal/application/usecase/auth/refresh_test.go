package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

type refreshFakeAdminRepo struct {
	user        domain.AdminUser
	findErr     error
	findIDCalls int
	lastID      int64
}

func (f *refreshFakeAdminRepo) FindByEmail(context.Context, string) (domain.AdminUser, error) {
	return domain.AdminUser{}, errors.New("not implemented")
}

func (f *refreshFakeAdminRepo) FindByID(_ context.Context, id int64) (domain.AdminUser, error) {
	f.findIDCalls++
	f.lastID = id
	if f.findErr != nil {
		return domain.AdminUser{}, f.findErr
	}
	return f.user, nil
}

type refreshFakeTokenRepo struct {
	findToken domain.RefreshToken
	findErr   error
	lastHash  string

	revokeID    int64
	revokeErr   error
	revokeCalls int

	revokeIfActiveResult *bool
	revokeIfActiveErr    error
	revokeIfActiveCalls  int

	revokeAllUserID int64
	revokeAllErr    error
	revokeAllCalls  int

	issueInput *domain.CreateRefreshTokenInput
	issueErr   error
}

func (f *refreshFakeTokenRepo) Issue(_ context.Context, input domain.CreateRefreshTokenInput) (domain.RefreshToken, error) {
	if f.issueErr != nil {
		return domain.RefreshToken{}, f.issueErr
	}
	in := input
	f.issueInput = &in
	return domain.RefreshToken{
		ID:          999,
		AdminUserID: input.AdminUserID,
		TokenHash:   input.TokenHash,
		ExpiresAt:   input.ExpiresAt,
	}, nil
}

func (f *refreshFakeTokenRepo) FindByTokenHash(_ context.Context, tokenHash string) (domain.RefreshToken, error) {
	f.lastHash = tokenHash
	if f.findErr != nil {
		return domain.RefreshToken{}, f.findErr
	}
	return f.findToken, nil
}

func (f *refreshFakeTokenRepo) Revoke(_ context.Context, id int64) error {
	f.revokeCalls++
	f.revokeID = id
	return f.revokeErr
}

func (f *refreshFakeTokenRepo) RevokeIfActive(_ context.Context, id int64) (bool, error) {
	f.revokeIfActiveCalls++
	f.revokeID = id
	if f.revokeIfActiveErr != nil {
		return false, f.revokeIfActiveErr
	}
	if f.revokeIfActiveResult != nil {
		return *f.revokeIfActiveResult, nil
	}
	return true, nil
}

func (f *refreshFakeTokenRepo) RevokeAllByUser(_ context.Context, adminUserID int64) error {
	f.revokeAllCalls++
	f.revokeAllUserID = adminUserID
	return f.revokeAllErr
}

// stubTxManager は DoInTx の中身を素通しで fn(ctx) に流すスタブ。
// usecase 側の「Revoke + Issue を DoInTx に渡しているか」「呼ばれた回数」を検証する。
type stubTxManager struct {
	calls int
	fail  error
}

func (s *stubTxManager) DoInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	s.calls++
	if s.fail != nil {
		return s.fail
	}
	return fn(ctx)
}

func newTestRefreshUseCase(t *testing.T, admin *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm application.TxManager) *RefreshUseCase {
	t.Helper()
	jwtSvc := jwt.NewJWTService([]byte(testJWTSecret), "satehits-api", "satehits-admin", time.Hour)
	return NewRefreshUseCase(admin, refresh, jwtSvc, txm)
}

func TestRefreshUseCase_Execute(t *testing.T) {
	const plainToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tokenHash := hashRefreshTokenPlain(plainToken)
	now := time.Now()
	owner := domain.AdminUser{ID: 1, Email: "owner@example.com", Role: "owner"}

	// 正常にローテーションできる有効な RT
	validRT := func() domain.RefreshToken {
		return domain.RefreshToken{
			ID:          42,
			AdminUserID: owner.ID,
			TokenHash:   tokenHash,
			ExpiresAt:   now.Add(24 * time.Hour),
		}
	}

	tests := []struct {
		name            string
		cmd             RefreshCommand
		admin           *refreshFakeAdminRepo
		refresh         *refreshFakeTokenRepo
		txm             *stubTxManager
		wantErrIs       error
		wantErrContains string
		check           func(t *testing.T, admin *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, result *RefreshResult)
	}{
		{
			name:      "returns not-found error when refresh token is empty",
			cmd:       RefreshCommand{RefreshToken: ""},
			admin:     &refreshFakeAdminRepo{user: owner},
			refresh:   &refreshFakeTokenRepo{},
			txm:       &stubTxManager{},
			wantErrIs: domain.ErrRefreshTokenNotFound,
			check: func(t *testing.T, _ *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				if refresh.lastHash != "" {
					t.Fatalf("FindByTokenHash hash = %q, want empty (no lookup)", refresh.lastHash)
				}
				if txm.calls != 0 {
					t.Fatalf("DoInTx calls = %d, want 0", txm.calls)
				}
			},
		},
		{
			name:      "returns invalid when refresh token is not found",
			cmd:       RefreshCommand{RefreshToken: plainToken},
			admin:     &refreshFakeAdminRepo{user: owner},
			refresh:   &refreshFakeTokenRepo{findErr: domain.ErrRefreshTokenNotFound},
			txm:       &stubTxManager{},
			wantErrIs: domain.ErrRefreshTokenInvalid,
		},
		{
			name:            "returns wrapped error when FindByTokenHash fails",
			cmd:             RefreshCommand{RefreshToken: plainToken},
			admin:           &refreshFakeAdminRepo{user: owner},
			refresh:         &refreshFakeTokenRepo{findErr: errors.New("connection refused")},
			txm:             &stubTxManager{},
			wantErrContains: "connection refused",
		},
		{
			name:  "detects reuse and revokes all when token is already revoked",
			cmd:   RefreshCommand{RefreshToken: plainToken},
			admin: &refreshFakeAdminRepo{user: owner},
			refresh: func() *refreshFakeTokenRepo {
				rt := validRT()
				revokedAt := now.Add(-time.Hour)
				rt.RevokedAt = &revokedAt
				return &refreshFakeTokenRepo{findToken: rt}
			}(),
			txm:       &stubTxManager{},
			wantErrIs: domain.ErrRefreshTokenInvalid,
			check: func(t *testing.T, admin *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				if refresh.revokeAllCalls != 1 {
					t.Fatalf("RevokeAllByUser calls = %d, want 1", refresh.revokeAllCalls)
				}
				if refresh.revokeAllUserID != owner.ID {
					t.Fatalf("RevokeAllByUser userID = %d, want %d", refresh.revokeAllUserID, owner.ID)
				}
				// 検証で弾くので、読み取り・ローテーションには進まない
				if admin.findIDCalls != 0 {
					t.Fatalf("FindByID calls = %d, want 0", admin.findIDCalls)
				}
				if refresh.revokeIfActiveCalls != 0 || txm.calls != 0 {
					t.Fatalf("RevokeIfActive=%d DoInTx=%d, want 0/0", refresh.revokeIfActiveCalls, txm.calls)
				}
			},
		},
		{
			name:  "returns wrapped error when RevokeAllByUser fails during reuse detection",
			cmd:   RefreshCommand{RefreshToken: plainToken},
			admin: &refreshFakeAdminRepo{user: owner},
			refresh: func() *refreshFakeTokenRepo {
				rt := validRT()
				revokedAt := now.Add(-time.Hour)
				rt.RevokedAt = &revokedAt
				return &refreshFakeTokenRepo{findToken: rt, revokeAllErr: errors.New("update failed")}
			}(),
			txm:             &stubTxManager{},
			wantErrContains: "failed to revoke refresh token",
		},
		{
			name:  "returns invalid when refresh token is expired",
			cmd:   RefreshCommand{RefreshToken: plainToken},
			admin: &refreshFakeAdminRepo{user: owner},
			refresh: func() *refreshFakeTokenRepo {
				rt := validRT()
				rt.ExpiresAt = now.Add(-time.Hour)
				return &refreshFakeTokenRepo{findToken: rt}
			}(),
			txm:       &stubTxManager{},
			wantErrIs: domain.ErrRefreshTokenInvalid,
			check: func(t *testing.T, admin *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				if admin.findIDCalls != 0 {
					t.Fatalf("FindByID calls = %d, want 0", admin.findIDCalls)
				}
				if refresh.revokeIfActiveCalls != 0 || refresh.issueInput != nil || txm.calls != 0 {
					t.Fatalf("RevokeIfActive=%d Issue=%v DoInTx=%d, want no mutation", refresh.revokeIfActiveCalls, refresh.issueInput, txm.calls)
				}
			},
		},
		{
			name:            "returns wrapped error when FindByID fails",
			cmd:             RefreshCommand{RefreshToken: plainToken},
			admin:           &refreshFakeAdminRepo{findErr: errors.New("connection refused")},
			refresh:         &refreshFakeTokenRepo{findToken: validRT()},
			txm:             &stubTxManager{},
			wantErrContains: "failed to find admin user",
			check: func(t *testing.T, _ *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				// JWT 生成・ローテーション前なので mutation は一切起きない
				if refresh.revokeIfActiveCalls != 0 || txm.calls != 0 {
					t.Fatalf("RevokeIfActive=%d DoInTx=%d, want 0/0", refresh.revokeIfActiveCalls, txm.calls)
				}
			},
		},
		{
			name:            "returns rotate error when RevokeIfActive fails inside tx",
			cmd:             RefreshCommand{RefreshToken: plainToken},
			admin:           &refreshFakeAdminRepo{user: owner},
			refresh:         &refreshFakeTokenRepo{findToken: validRT(), revokeIfActiveErr: errors.New("revoke failed")},
			txm:             &stubTxManager{},
			wantErrContains: "failed to rotate refresh token",
			check: func(t *testing.T, _ *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				if txm.calls != 1 {
					t.Fatalf("DoInTx calls = %d, want 1", txm.calls)
				}
				if refresh.revokeIfActiveCalls != 1 {
					t.Fatalf("RevokeIfActive calls = %d, want 1", refresh.revokeIfActiveCalls)
				}
				if refresh.issueInput != nil {
					t.Fatal("Issue was called, want not called when RevokeIfActive fails")
				}
			},
		},
		{
			name:  "returns invalid when RevokeIfActive finds token already revoked in tx",
			cmd:   RefreshCommand{RefreshToken: plainToken},
			admin: &refreshFakeAdminRepo{user: owner},
			refresh: func() *refreshFakeTokenRepo {
				notActive := false
				return &refreshFakeTokenRepo{findToken: validRT(), revokeIfActiveResult: &notActive}
			}(),
			txm:       &stubTxManager{},
			wantErrIs: domain.ErrRefreshTokenInvalid,
			check: func(t *testing.T, _ *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				if txm.calls != 1 {
					t.Fatalf("DoInTx calls = %d, want 1", txm.calls)
				}
				if refresh.revokeIfActiveCalls != 1 {
					t.Fatalf("RevokeIfActive calls = %d, want 1", refresh.revokeIfActiveCalls)
				}
				if refresh.revokeAllCalls != 1 {
					t.Fatalf("RevokeAllByUser calls = %d, want 1", refresh.revokeAllCalls)
				}
				if refresh.revokeAllUserID != owner.ID {
					t.Fatalf("RevokeAllByUser userID = %d, want %d", refresh.revokeAllUserID, owner.ID)
				}
				if refresh.issueInput != nil {
					t.Fatal("Issue was called, want not called when token not active")
				}
			},
		},
		{
			name:            "returns rotate error when Issue fails inside tx",
			cmd:             RefreshCommand{RefreshToken: plainToken},
			admin:           &refreshFakeAdminRepo{user: owner},
			refresh:         &refreshFakeTokenRepo{findToken: validRT(), issueErr: errors.New("insert failed")},
			txm:             &stubTxManager{},
			wantErrContains: "failed to rotate refresh token",
			check: func(t *testing.T, _ *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, _ *RefreshResult) {
				t.Helper()
				if txm.calls != 1 {
					t.Fatalf("DoInTx calls = %d, want 1", txm.calls)
				}
				if refresh.revokeIfActiveCalls != 1 {
					t.Fatalf("RevokeIfActive calls = %d, want 1", refresh.revokeIfActiveCalls)
				}
			},
		},
		{
			name:    "succeeds and rotates refresh token",
			cmd:     RefreshCommand{RefreshToken: plainToken},
			admin:   &refreshFakeAdminRepo{user: owner},
			refresh: &refreshFakeTokenRepo{findToken: validRT()},
			txm:     &stubTxManager{},
			check: func(t *testing.T, admin *refreshFakeAdminRepo, refresh *refreshFakeTokenRepo, txm *stubTxManager, result *RefreshResult) {
				t.Helper()
				if result.AccessToken == "" {
					t.Fatal("AccessToken is empty")
				}
				if result.RefreshToken == "" {
					t.Fatal("RefreshToken is empty")
				}
				if result.ExpiresAt.IsZero() {
					t.Fatal("ExpiresAt is zero")
				}
				// 検証 → 読み取り → mutation の順で、各ステップが正しい引数で呼ばれている
				if admin.findIDCalls != 1 || admin.lastID != owner.ID {
					t.Fatalf("FindByID calls=%d lastID=%d, want 1/%d", admin.findIDCalls, admin.lastID, owner.ID)
				}
				if txm.calls != 1 {
					t.Fatalf("DoInTx calls = %d, want 1", txm.calls)
				}
				if refresh.revokeIfActiveCalls != 1 || refresh.revokeID != 42 {
					t.Fatalf("RevokeIfActive calls=%d id=%d, want 1/42", refresh.revokeIfActiveCalls, refresh.revokeID)
				}
				if refresh.issueInput == nil {
					t.Fatal("Issue was not called")
				}
				if refresh.issueInput.AdminUserID != owner.ID {
					t.Fatalf("Issue AdminUserID = %d, want %d", refresh.issueInput.AdminUserID, owner.ID)
				}
				// 返した平文 RT のハッシュが、保存した TokenHash と一致する
				assertRefreshTokenHash(t, result.RefreshToken, refresh.issueInput.TokenHash)
				expiresIn := time.Until(refresh.issueInput.ExpiresAt)
				if expiresIn < refreshTokenTTL-time.Minute || expiresIn > refreshTokenTTL+time.Minute {
					t.Fatalf("Issue ExpiresAt is %v from now, want about %v", expiresIn, refreshTokenTTL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newTestRefreshUseCase(t, tt.admin, tt.refresh, tt.txm)
			result, err := uc.Execute(context.Background(), tt.cmd)

			if tt.wantErrIs != nil || tt.wantErrContains != "" {
				assertExecuteError(t, err, tt.wantErrIs, tt.wantErrContains)
				if result != nil {
					t.Fatalf("result = %#v, want nil on error", result)
				}
			} else {
				if err != nil {
					t.Fatalf("Execute: %v", err)
				}
				if result == nil {
					t.Fatal("result is nil, want RefreshResult")
				}
			}

			if tt.check != nil {
				tt.check(t, tt.admin, tt.refresh, tt.txm, result)
			}
		})
	}
}

func (f *refreshFakeTokenRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, errors.New("not implemented")
}
