package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestAuthHandler_HandleLogin_doesNotRevealUserExistence(t *testing.T) {
	owner := authTestOwner(t)

	t.Run("unknown email and wrong password produce identical 401 responses", func(t *testing.T) {
		unknownAttempts := &authTestLoginAttemptRepo{}
		wrongAttempts := &authTestLoginAttemptRepo{}
		unknown := newAuthHandlerWithDeps(t, authTestDeps{
			admin:    &authTestAdminRepo{findErr: domain.ErrAdminUserNotFound},
			attempts: unknownAttempts,
		})
		wrong := newAuthHandlerWithDeps(t, authTestDeps{
			admin:    &authTestAdminRepo{user: owner},
			attempts: wrongAttempts,
		})

		recUnknown := httptest.NewRecorder()
		unknown.HandleLogin(recUnknown, newLoginRequest("nobody@example.com", "wrong-password"))
		recWrong := httptest.NewRecorder()
		wrong.HandleLogin(recWrong, newLoginRequest(owner.Email, "wrong-password"))

		assertStatusAndCode(t, recUnknown, http.StatusUnauthorized, UnauthorizedCode)
		assertStatusAndCode(t, recWrong, http.StatusUnauthorized, UnauthorizedCode)
		if recUnknown.Body.String() != recWrong.Body.String() {
			t.Fatalf("bodies differ:\nunknown = %s\nwrong   = %s", recUnknown.Body.String(), recWrong.Body.String())
		}
		if strings.Contains(recUnknown.Body.String(), "details") {
			t.Fatalf("body = %s, want no details", recUnknown.Body.String())
		}
		if unknownAttempts.recordCalls != 1 || wrongAttempts.recordCalls != 1 {
			t.Fatalf("RecordFailure calls unknown=%d wrong=%d, want 1/1", unknownAttempts.recordCalls, wrongAttempts.recordCalls)
		}
	})

	t.Run("rate limit returns identical 429 whether or not the email exists", func(t *testing.T) {
		counts := domain.LoginAttemptCounts{ByEmail: 5, OldestByEmail: time.Now().Add(-5 * time.Minute)}
		unknownAdmin := &authTestAdminRepo{findErr: domain.ErrAdminUserNotFound}
		unknown := newAuthHandlerWithDeps(t, authTestDeps{
			admin:    unknownAdmin,
			attempts: &authTestLoginAttemptRepo{counts: counts},
		})
		wrong := newAuthHandlerWithDeps(t, authTestDeps{
			admin:    &authTestAdminRepo{user: owner},
			attempts: &authTestLoginAttemptRepo{counts: counts},
		})

		recUnknown := httptest.NewRecorder()
		unknown.HandleLogin(recUnknown, newLoginRequest("nobody@example.com", testAuthPassword))
		recWrong := httptest.NewRecorder()
		wrong.HandleLogin(recWrong, newLoginRequest(owner.Email, testAuthPassword))

		assertStatusAndCode(t, recUnknown, http.StatusTooManyRequests, TooManyRequestsCode)
		assertStatusAndCode(t, recWrong, http.StatusTooManyRequests, TooManyRequestsCode)
		if recUnknown.Body.String() != recWrong.Body.String() {
			t.Fatalf("bodies differ:\nunknown = %s\nwrong   = %s", recUnknown.Body.String(), recWrong.Body.String())
		}
		if recUnknown.Header().Get("Retry-After") != recWrong.Header().Get("Retry-After") {
			t.Fatalf("Retry-After differs: unknown=%q wrong=%q", recUnknown.Header().Get("Retry-After"), recWrong.Header().Get("Retry-After"))
		}
	})
}

func TestAuthHandler_HandleLogin_rateLimitThresholds(t *testing.T) {
	owner := authTestOwner(t)
	const window = 15 * time.Minute

	tests := []struct {
		name           string
		counts         func(now time.Time) domain.LoginAttemptCounts
		wantStatus     int
		wantRetryAfter int // 0 なら値は確認しない
	}{
		{
			name: "passes when email failures are one below the limit",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{ByEmail: 4, OldestByEmail: now}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "rejects when email failures reach the limit",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{ByEmail: 5, OldestByEmail: now}
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "passes when IP failures are one below the limit",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{ByIP: 19, OldestByIP: now}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "rejects when IP failures reach the limit",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{ByIP: 20, OldestByIP: now}
			},
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name: "uses the larger Retry-After when both limits are exceeded",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{
					ByEmail:       5,
					OldestByEmail: now.Add(-14 * time.Minute),
					ByIP:          20,
					OldestByIP:    now.Add(-5 * time.Minute),
				}
			},
			wantStatus:     http.StatusTooManyRequests,
			wantRetryAfter: 600,
		},
		{
			name: "rounds Retry-After up to whole seconds",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{ByEmail: 5, OldestByEmail: now.Add(-5*time.Minute - 300*time.Millisecond)}
			},
			wantStatus:     http.StatusTooManyRequests,
			wantRetryAfter: 600,
		},
		{
			name: "returns Retry-After of at least one second when the window is almost over",
			counts: func(now time.Time) domain.LoginAttemptCounts {
				return domain.LoginAttemptCounts{ByEmail: 5, OldestByEmail: now.Add(-window + 200*time.Millisecond)}
			},
			wantStatus:     http.StatusTooManyRequests,
			wantRetryAfter: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := &authTestLoginAttemptRepo{counts: tt.counts(time.Now())}
			h := newAuthHandlerWithDeps(t, authTestDeps{admin: &authTestAdminRepo{user: owner}, attempts: attempts})
			rec := httptest.NewRecorder()

			h.HandleLogin(rec, newLoginRequest(owner.Email, testAuthPassword))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus == http.StatusOK {
				if rec.Header().Get("Retry-After") != "" {
					t.Fatalf("Retry-After = %q, want absent on success", rec.Header().Get("Retry-After"))
				}
				return
			}
			assertAuthErrorCode(t, rec, TooManyRequestsCode)
			seconds, err := strconv.Atoi(rec.Header().Get("Retry-After"))
			if err != nil || seconds < 1 {
				t.Fatalf("Retry-After = %q, want integer >= 1", rec.Header().Get("Retry-After"))
			}
			if tt.wantRetryAfter != 0 && seconds != tt.wantRetryAfter {
				t.Fatalf("Retry-After = %d, want %d", seconds, tt.wantRetryAfter)
			}
			if attempts.recordCalls != 0 {
				t.Fatalf("RecordFailure calls = %d, want 0 while rate limited", attempts.recordCalls)
			}
		})
	}
}

func TestAuthHandler_HandleLogin_responseContainsNoSecrets(t *testing.T) {
	owner := authTestOwner(t)
	refresh := &authTestRefreshRepo{}
	attempts := &authTestLoginAttemptRepo{}
	h := newAuthHandlerWithDeps(t, authTestDeps{admin: &authTestAdminRepo{user: owner}, refresh: refresh, attempts: attempts})
	before := time.Now()
	rec := httptest.NewRecorder()

	h.HandleLogin(rec, newLoginRequest(owner.Email, testAuthPassword))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	t.Run("body has only token, expires_at and user", func(t *testing.T) {
		payload := decodeJSONObject(t, rec.Body.Bytes())
		assertJSONKeys(t, payload, "token", "expires_at", "user")
		assertJSONKeys(t, decodeJSONObject(t, payload["user"]), "id", "email", "role")
		for _, forbidden := range []string{"refresh_token", "password_hash", "password", owner.PasswordHash, testAuthPassword} {
			if strings.Contains(rec.Body.String(), forbidden) {
				t.Fatalf("body contains %q: %s", forbidden, rec.Body.String())
			}
		}
	})

	t.Run("expires_at is about one hour ahead", func(t *testing.T) {
		var payload loginResponse
		if err := decodeInto(rec, &payload); err != nil {
			t.Fatalf("decode: %v", err)
		}
		remaining := payload.ExpiresAt.Sub(before)
		if remaining < time.Hour-5*time.Second || remaining > time.Hour+5*time.Second {
			t.Fatalf("expires_at is %v from now, want about 1h", remaining)
		}
	})

	t.Run("refresh token travels only in the cookie and is stored hashed", func(t *testing.T) {
		c := findRefreshCookie(t, rec)
		assertRefreshCookieAttributes(t, c, refreshTokenMaxAge, "")
		if len(c.Value) != 64 || !reHex64.MatchString(c.Value) {
			t.Fatalf("cookie value = %q, want 64 hex chars", c.Value)
		}
		if strings.Contains(rec.Body.String(), c.Value) {
			t.Fatalf("body contains refresh token: %s", rec.Body.String())
		}
		if len(refresh.issueInputs) != 1 {
			t.Fatalf("Issue calls = %d, want 1", len(refresh.issueInputs))
		}
		stored := refresh.issueInputs[0].TokenHash
		if stored == c.Value {
			t.Fatal("stored token hash equals plaintext cookie value")
		}
		if stored != authTestTokenHash(c.Value) {
			t.Fatalf("stored hash = %q, want sha256(cookie value) = %q", stored, authTestTokenHash(c.Value))
		}
		if attempts.clearCalls != 1 {
			t.Fatalf("ClearByEmail calls = %d, want 1", attempts.clearCalls)
		}
	})
}

func TestAuthHandler_HandleLogin_logsContainNoSecrets(t *testing.T) {
	owner := authTestOwner(t)
	const marker = "repo failure marker"

	tests := []struct {
		name       string
		deps       func() authTestDeps
		password   string
		wantStatus int
		wantMarker bool
	}{
		{
			name:       "401 on wrong password",
			deps:       func() authTestDeps { return authTestDeps{admin: &authTestAdminRepo{user: owner}} },
			password:   "wrong-password",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "401 on unknown email",
			deps: func() authTestDeps {
				return authTestDeps{admin: &authTestAdminRepo{findErr: domain.ErrAdminUserNotFound}}
			},
			password:   testAuthPassword,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "500 when CountRecent fails",
			deps: func() authTestDeps {
				return authTestDeps{admin: &authTestAdminRepo{user: owner}, attempts: &authTestLoginAttemptRepo{countErr: errors.New(marker)}}
			},
			password:   testAuthPassword,
			wantStatus: http.StatusInternalServerError,
			wantMarker: true,
		},
		{
			name: "500 when FindByEmail fails",
			deps: func() authTestDeps {
				return authTestDeps{admin: &authTestAdminRepo{findErr: errors.New(marker)}}
			},
			password:   testAuthPassword,
			wantStatus: http.StatusInternalServerError,
			wantMarker: true,
		},
		{
			name: "500 when RecordFailure fails",
			deps: func() authTestDeps {
				return authTestDeps{admin: &authTestAdminRepo{user: owner}, attempts: &authTestLoginAttemptRepo{recordErr: errors.New(marker)}}
			},
			password:   "wrong-password",
			wantStatus: http.StatusInternalServerError,
			wantMarker: true,
		},
		{
			name: "500 when Issue fails",
			deps: func() authTestDeps {
				return authTestDeps{admin: &authTestAdminRepo{user: owner}, refresh: &authTestRefreshRepo{issueErr: errors.New(marker)}}
			},
			password:   testAuthPassword,
			wantStatus: http.StatusInternalServerError,
			wantMarker: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf := captureLog(t)
			h := newAuthHandlerWithDeps(t, tt.deps())
			rec := httptest.NewRecorder()

			h.HandleLogin(rec, newLoginRequest(owner.Email, tt.password))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			logText := logBuf.String()
			if tt.wantMarker && !strings.Contains(logText, marker) {
				t.Fatalf("log = %q, want to contain %q (log capture must observe the failure)", logText, marker)
			}
			assertNoSecretsInLog(t, logText, tt.password, owner.PasswordHash)
			if strings.Contains(rec.Body.String(), marker) {
				t.Fatalf("body leaks internal error: %s", rec.Body.String())
			}
		})
	}
}

func decodeInto(rec *httptest.ResponseRecorder, v *loginResponse) error {
	return json.Unmarshal(rec.Body.Bytes(), v)
}
