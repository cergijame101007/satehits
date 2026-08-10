package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

const (
	testLoginPassword = "password123"
	testLoginEmail    = "owner@example.com"
	testJWTSecret     = "test-jwt-secret-minimum-32-bytes-long"
	testClientIP      = "203.0.113.10"
)

var testLoginRateLimit = LoginRateLimitPolicy{
	EmailMax: 5,
	IPMax:    20,
	Window:   15 * time.Minute,
}

type fakeAdminUserRepo struct {
	user      domain.AdminUser
	findErr   error
	lastEmail string
	findCalls int
}

func (f *fakeAdminUserRepo) FindByEmail(_ context.Context, email string) (domain.AdminUser, error) {
	f.findCalls++
	f.lastEmail = email
	if f.findErr != nil {
		return domain.AdminUser{}, f.findErr
	}
	return f.user, nil
}

func (f *fakeAdminUserRepo) FindByID(context.Context, int64) (domain.AdminUser, error) {
	return domain.AdminUser{}, errors.New("not implemented")
}

type fakeRefreshTokenRepo struct {
	issueInput *domain.CreateRefreshTokenInput
	issueErr   error
}

func (f *fakeRefreshTokenRepo) Issue(_ context.Context, input domain.CreateRefreshTokenInput) (domain.RefreshToken, error) {
	if f.issueErr != nil {
		return domain.RefreshToken{}, f.issueErr
	}
	in := input
	f.issueInput = &in
	return domain.RefreshToken{
		ID:          1,
		AdminUserID: input.AdminUserID,
		TokenHash:   input.TokenHash,
		ExpiresAt:   input.ExpiresAt,
	}, nil
}

func (f *fakeRefreshTokenRepo) FindByTokenHash(context.Context, string) (domain.RefreshToken, error) {
	return domain.RefreshToken{}, errors.New("not implemented")
}

func (f *fakeRefreshTokenRepo) Revoke(context.Context, int64) error {
	return errors.New("not implemented")
}

func (f *fakeRefreshTokenRepo) RevokeIfActive(context.Context, int64) (bool, error) {
	return true, nil
}

func (f *fakeRefreshTokenRepo) RevokeAllByUser(context.Context, int64) error {
	return errors.New("not implemented")
}

type fakeLoginAttemptRepo struct {
	counts        domain.LoginAttemptCounts
	countErr      error
	recordErr     error
	clearErr      error
	recordCalls   int
	clearCalls    int
	lastRecordKey string
	lastRecordIP  string
	lastClearKey  string
	countCalled   bool
	lastCountKey  string
	lastCountIP   string
}

func (f *fakeLoginAttemptRepo) CountRecent(_ context.Context, emailKey, ip string, _ time.Time) (domain.LoginAttemptCounts, error) {
	f.countCalled = true
	f.lastCountKey = emailKey
	f.lastCountIP = ip
	if f.countErr != nil {
		return domain.LoginAttemptCounts{}, f.countErr
	}
	return f.counts, nil
}

func (f *fakeLoginAttemptRepo) RecordFailure(_ context.Context, emailKey, ip string, _ time.Time) error {
	f.recordCalls++
	f.lastRecordKey = emailKey
	f.lastRecordIP = ip
	return f.recordErr
}

func (f *fakeLoginAttemptRepo) ClearByEmail(_ context.Context, emailKey string) error {
	f.clearCalls++
	f.lastClearKey = emailKey
	return f.clearErr
}

func mustPasswordHash(t *testing.T, plain string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	return string(hash)
}

func newTestLoginUseCase(
	t *testing.T,
	admin *fakeAdminUserRepo,
	refresh *fakeRefreshTokenRepo,
	attempts *fakeLoginAttemptRepo,
) *LoginUseCase {
	t.Helper()
	if attempts == nil {
		attempts = &fakeLoginAttemptRepo{}
	}
	jwtSvc := jwt.NewJWTService([]byte(testJWTSecret), "satehits-api", "satehits-admin", time.Hour)
	return NewLoginUseCase(admin, refresh, attempts, jwtSvc, testLoginRateLimit)
}

func assertValidationError(t *testing.T, err error, field string) {
	t.Helper()
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("err = %T(%v), want *ValidationError", err, err)
	}
	for _, v := range vErr.Violations {
		if v.Field == field {
			return
		}
	}
	t.Fatalf("violations = %#v, want field %q", vErr.Violations, field)
}

func assertRefreshTokenHash(t *testing.T, plainToken, wantHash string) {
	t.Helper()
	sum := sha256.Sum256([]byte(plainToken))
	gotHash := hex.EncodeToString(sum[:])
	if gotHash != wantHash {
		t.Fatalf("token hash = %q, want %q", gotHash, wantHash)
	}
}

func assertExecuteError(t *testing.T, err error, wantIs error, wantContains string) {
	t.Helper()
	if err == nil {
		t.Fatalf("err = nil, want error")
	}
	if wantIs != nil && errors.Is(err, wantIs) {
		return
	}
	if wantContains != "" && strings.Contains(err.Error(), wantContains) {
		return
	}
	t.Fatalf("err = %v, want Is(%v) or contains %q", err, wantIs, wantContains)
}

func TestLoginUseCase_Execute(t *testing.T) {
	ownerHash := mustPasswordHash(t, testLoginPassword)
	owner := domain.AdminUser{
		ID:           1,
		Email:        testLoginEmail,
		PasswordHash: ownerHash,
		Role:         "owner",
	}
	oldest := time.Now().Add(-5 * time.Minute)

	tests := []struct {
		name            string
		cmd             LoginCommand
		adminRepo       *fakeAdminUserRepo
		refresh         *fakeRefreshTokenRepo
		attempts        *fakeLoginAttemptRepo
		wantValErr      string
		wantErrIs       error
		wantErrContains string
		wantRateLimited bool
		checkResult     func(t *testing.T, admin *fakeAdminUserRepo, refresh *fakeRefreshTokenRepo, attempts *fakeLoginAttemptRepo, result *LoginResult)
	}{
		{
			name:       "returns validation error when email is empty",
			cmd:        LoginCommand{Email: "", Password: testLoginPassword, ClientIP: testClientIP},
			adminRepo:  &fakeAdminUserRepo{user: owner},
			refresh:    &fakeRefreshTokenRepo{},
			wantValErr: "email",
		},
		{
			name:       "returns validation error when password is empty",
			cmd:        LoginCommand{Email: owner.Email, Password: "", ClientIP: testClientIP},
			adminRepo:  &fakeAdminUserRepo{user: owner},
			refresh:    &fakeRefreshTokenRepo{},
			wantValErr: "password",
		},
		{
			name:      "returns unauthorized when user is not found",
			cmd:       validLoginCommand(),
			adminRepo: &fakeAdminUserRepo{findErr: domain.ErrAdminUserNotFound},
			refresh:   &fakeRefreshTokenRepo{},
			attempts:  &fakeLoginAttemptRepo{},
			wantErrIs: domain.ErrAdminUserUnauthorized,
			checkResult: func(t *testing.T, _ *fakeAdminUserRepo, _ *fakeRefreshTokenRepo, attempts *fakeLoginAttemptRepo, _ *LoginResult) {
				t.Helper()
				if attempts.recordCalls != 1 {
					t.Fatalf("RecordFailure calls = %d, want 1", attempts.recordCalls)
				}
				if attempts.lastRecordKey != testLoginEmail {
					t.Fatalf("RecordFailure emailKey = %q, want %q", attempts.lastRecordKey, testLoginEmail)
				}
				if attempts.lastRecordIP != testClientIP {
					t.Fatalf("RecordFailure ip = %q, want %q", attempts.lastRecordIP, testClientIP)
				}
			},
		},
		{
			name:      "returns unauthorized when password does not match",
			cmd:       LoginCommand{Email: owner.Email, Password: "wrong-password", ClientIP: testClientIP},
			adminRepo: &fakeAdminUserRepo{user: owner},
			refresh:   &fakeRefreshTokenRepo{},
			attempts:  &fakeLoginAttemptRepo{},
			wantErrIs: domain.ErrAdminUserUnauthorized,
			checkResult: func(t *testing.T, _ *fakeAdminUserRepo, _ *fakeRefreshTokenRepo, attempts *fakeLoginAttemptRepo, _ *LoginResult) {
				t.Helper()
				if attempts.recordCalls != 1 {
					t.Fatalf("RecordFailure calls = %d, want 1", attempts.recordCalls)
				}
			},
		},
		{
			name: "returns unauthorized when password hash is malformed",
			cmd:  validLoginCommand(),
			adminRepo: &fakeAdminUserRepo{user: domain.AdminUser{
				ID:           1,
				Email:        testLoginEmail,
				PasswordHash: "not-a-bcrypt-hash",
				Role:         "owner",
			}},
			refresh:   &fakeRefreshTokenRepo{},
			wantErrIs: domain.ErrAdminUserUnauthorized,
		},
		{
			name:            "returns repository error when FindByEmail fails",
			cmd:             validLoginCommand(),
			adminRepo:       &fakeAdminUserRepo{findErr: errors.New("connection refused")},
			refresh:         &fakeRefreshTokenRepo{},
			wantErrContains: "connection refused",
		},
		{
			name:            "returns error when refresh token issue fails",
			cmd:             validLoginCommand(),
			adminRepo:       &fakeAdminUserRepo{user: owner},
			refresh:         &fakeRefreshTokenRepo{issueErr: errors.New("insert failed")},
			wantErrContains: "insert failed",
		},
		{
			name:      "returns rate limited when email attempts exceed threshold without FindByEmail",
			cmd:       validLoginCommand(),
			adminRepo: &fakeAdminUserRepo{user: owner},
			refresh:   &fakeRefreshTokenRepo{},
			attempts: &fakeLoginAttemptRepo{
				counts: domain.LoginAttemptCounts{
					ByEmail:       5,
					OldestByEmail: oldest,
				},
			},
			wantRateLimited: true,
			checkResult: func(t *testing.T, admin *fakeAdminUserRepo, _ *fakeRefreshTokenRepo, attempts *fakeLoginAttemptRepo, _ *LoginResult) {
				t.Helper()
				if admin.findCalls != 0 {
					t.Fatalf("FindByEmail calls = %d, want 0", admin.findCalls)
				}
				if attempts.recordCalls != 0 {
					t.Fatalf("RecordFailure calls = %d, want 0", attempts.recordCalls)
				}
			},
		},
		{
			name:      "returns rate limited when IP attempts exceed threshold without FindByEmail",
			cmd:       validLoginCommand(),
			adminRepo: &fakeAdminUserRepo{user: owner},
			refresh:   &fakeRefreshTokenRepo{},
			attempts: &fakeLoginAttemptRepo{
				counts: domain.LoginAttemptCounts{
					ByIP:       20,
					OldestByIP: oldest,
				},
			},
			wantRateLimited: true,
			checkResult: func(t *testing.T, admin *fakeAdminUserRepo, _ *fakeRefreshTokenRepo, _ *fakeLoginAttemptRepo, _ *LoginResult) {
				t.Helper()
				if admin.findCalls != 0 {
					t.Fatalf("FindByEmail calls = %d, want 0", admin.findCalls)
				}
			},
		},
		{
			name:            "returns error when CountRecent fails",
			cmd:             validLoginCommand(),
			adminRepo:       &fakeAdminUserRepo{user: owner},
			refresh:         &fakeRefreshTokenRepo{},
			attempts:        &fakeLoginAttemptRepo{countErr: errors.New("count failed")},
			wantErrContains: "count failed",
		},
		{
			name:            "returns error when RecordFailure fails on auth failure",
			cmd:             LoginCommand{Email: owner.Email, Password: "wrong-password", ClientIP: testClientIP},
			adminRepo:       &fakeAdminUserRepo{user: owner},
			refresh:         &fakeRefreshTokenRepo{},
			attempts:        &fakeLoginAttemptRepo{recordErr: errors.New("record failed")},
			wantErrContains: "record failed",
		},
		{
			name:            "returns error when ClearByEmail fails before token issue",
			cmd:             validLoginCommand(),
			adminRepo:       &fakeAdminUserRepo{user: owner},
			refresh:         &fakeRefreshTokenRepo{},
			attempts:        &fakeLoginAttemptRepo{clearErr: errors.New("clear failed")},
			wantErrContains: "clear failed",
			checkResult: func(t *testing.T, _ *fakeAdminUserRepo, refresh *fakeRefreshTokenRepo, _ *fakeLoginAttemptRepo, _ *LoginResult) {
				t.Helper()
				if refresh.issueInput != nil {
					t.Fatal("Issue was called despite ClearByEmail failure")
				}
			},
		},
		{
			name:      "succeeds with valid credentials",
			cmd:       validLoginCommand(),
			adminRepo: &fakeAdminUserRepo{user: owner},
			refresh:   &fakeRefreshTokenRepo{},
			attempts:  &fakeLoginAttemptRepo{},
			checkResult: func(t *testing.T, admin *fakeAdminUserRepo, refresh *fakeRefreshTokenRepo, attempts *fakeLoginAttemptRepo, result *LoginResult) {
				t.Helper()
				if result == nil {
					t.Fatal("result is nil, want LoginResult")
				}
				if result.AccessToken == "" {
					t.Fatal("AccessToken is empty")
				}
				if result.RefreshToken == "" {
					t.Fatal("RefreshToken is empty")
				}
				if result.ExpiresAt.IsZero() {
					t.Fatal("ExpiresAt is zero")
				}
				wantUser := LoginUser{ID: owner.ID, Email: owner.Email, Role: owner.Role}
				if result.User != wantUser {
					t.Fatalf("User = %#v, want %#v", result.User, wantUser)
				}
				if refresh.issueInput == nil {
					t.Fatal("Issue was not called")
				}
				if refresh.issueInput.AdminUserID != owner.ID {
					t.Fatalf("Issue AdminUserID = %d, want %d", refresh.issueInput.AdminUserID, owner.ID)
				}
				assertRefreshTokenHash(t, result.RefreshToken, refresh.issueInput.TokenHash)
				expiresIn := time.Until(refresh.issueInput.ExpiresAt)
				if expiresIn < refreshTokenTTL-time.Second || expiresIn > refreshTokenTTL+time.Second {
					t.Fatalf("Issue ExpiresAt is %v from now, want about %v", expiresIn, refreshTokenTTL)
				}
				if admin.lastEmail != owner.Email {
					t.Fatalf("FindByEmail email = %q, want %q", admin.lastEmail, owner.Email)
				}
				if attempts.clearCalls != 1 {
					t.Fatalf("ClearByEmail calls = %d, want 1", attempts.clearCalls)
				}
				if attempts.lastClearKey != testLoginEmail {
					t.Fatalf("ClearByEmail key = %q, want %q", attempts.lastClearKey, testLoginEmail)
				}
				if attempts.recordCalls != 0 {
					t.Fatalf("RecordFailure calls = %d, want 0", attempts.recordCalls)
				}
				if len(result.RefreshToken) != 64 {
					t.Fatalf("RefreshToken length = %d, want 64 hex chars", len(result.RefreshToken))
				}
				for _, c := range result.RefreshToken {
					if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
						t.Fatalf("RefreshToken %q contains non-hex char %q", result.RefreshToken, c)
					}
				}
			},
		},
		{
			name: "trims email before FindByEmail and lowercases rate-limit key",
			cmd: LoginCommand{
				Email:    "  Owner@Example.com  ",
				Password: testLoginPassword,
				ClientIP: testClientIP,
			},
			adminRepo: &fakeAdminUserRepo{user: owner},
			refresh:   &fakeRefreshTokenRepo{},
			attempts:  &fakeLoginAttemptRepo{},
			checkResult: func(t *testing.T, admin *fakeAdminUserRepo, _ *fakeRefreshTokenRepo, attempts *fakeLoginAttemptRepo, result *LoginResult) {
				t.Helper()
				if result == nil {
					t.Fatal("result is nil, want success")
				}
				if admin.lastEmail != "Owner@Example.com" {
					t.Fatalf("FindByEmail email = %q, want trimmed case-preserved", admin.lastEmail)
				}
				if attempts.lastCountKey != testLoginEmail {
					t.Fatalf("CountRecent emailKey = %q, want lowercased %q", attempts.lastCountKey, testLoginEmail)
				}
				if attempts.lastClearKey != testLoginEmail {
					t.Fatalf("ClearByEmail key = %q, want lowercased %q", attempts.lastClearKey, testLoginEmail)
				}
			},
		},
		{
			name: "passes email case as-is to FindByEmail (no normalization)",
			cmd: LoginCommand{
				Email:    "Owner@Example.COM",
				Password: testLoginPassword,
				ClientIP: testClientIP,
			},
			adminRepo: &fakeAdminUserRepo{user: owner},
			refresh:   &fakeRefreshTokenRepo{},
			attempts:  &fakeLoginAttemptRepo{},
			checkResult: func(t *testing.T, admin *fakeAdminUserRepo, _ *fakeRefreshTokenRepo, _ *fakeLoginAttemptRepo, result *LoginResult) {
				t.Helper()
				if result == nil {
					t.Fatal("result is nil, want success")
				}
				if admin.lastEmail != "Owner@Example.COM" {
					t.Fatalf("FindByEmail email = %q, want case-preserved %q", admin.lastEmail, "Owner@Example.COM")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := tt.attempts
			if attempts == nil {
				attempts = &fakeLoginAttemptRepo{}
			}
			uc := newTestLoginUseCase(t, tt.adminRepo, tt.refresh, attempts)
			result, err := uc.Execute(context.Background(), tt.cmd)

			if tt.wantValErr != "" {
				assertValidationError(t, err, tt.wantValErr)
				if result != nil {
					t.Fatalf("result = %#v, want nil", result)
				}
				return
			}

			if tt.wantRateLimited {
				var rateErr *RateLimitedError
				if !errors.As(err, &rateErr) {
					t.Fatalf("err = %T(%v), want *RateLimitedError", err, err)
				}
				if rateErr.RetryAfter <= 0 {
					t.Fatalf("RetryAfter = %v, want > 0", rateErr.RetryAfter)
				}
				if result != nil {
					t.Fatalf("result = %#v, want nil on rate limit", result)
				}
				if tt.checkResult != nil {
					tt.checkResult(t, tt.adminRepo, tt.refresh, attempts, result)
				}
				return
			}

			if tt.wantErrIs != nil || tt.wantErrContains != "" {
				assertExecuteError(t, err, tt.wantErrIs, tt.wantErrContains)
				if result != nil {
					t.Fatalf("result = %#v, want nil on error", result)
				}
				if tt.checkResult != nil {
					tt.checkResult(t, tt.adminRepo, tt.refresh, attempts, result)
				}
				return
			}

			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if tt.checkResult != nil {
				tt.checkResult(t, tt.adminRepo, tt.refresh, attempts, result)
			}
		})
	}
}

func TestGenerateRefreshToken_produces_matching_hash(t *testing.T) {
	token, hash, err := generateRefreshToken()
	if err != nil {
		t.Fatalf("generateRefreshToken: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("token length = %d, want 64 hex chars", len(token))
	}
	assertRefreshTokenHash(t, token, hash)
}

func TestLoginUseCase_Execute_both_fields_empty_returns_two_violations(t *testing.T) {
	uc := newTestLoginUseCase(t, &fakeAdminUserRepo{}, &fakeRefreshTokenRepo{}, &fakeLoginAttemptRepo{})
	_, err := uc.Execute(context.Background(), LoginCommand{Email: "", Password: ""})

	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("err = %T(%v), want *ValidationError", err, err)
	}
	if len(vErr.Violations) != 2 {
		t.Fatalf("violations count = %d, want 2; violations = %#v", len(vErr.Violations), vErr.Violations)
	}
	seen := make(map[string]bool)
	for _, v := range vErr.Violations {
		seen[v.Field] = true
	}
	for _, field := range []string{"email", "password"} {
		if !seen[field] {
			t.Errorf("missing violation for field %q", field)
		}
	}
}
