package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	authusecase "github.com/cergijame101007/satehits/internal/application/usecase/auth"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

const (
	testAuthPassword  = "password123"
	testAuthJWTSecret = "test-jwt-secret-minimum-32-bytes-long"
	testAuthLoginPath = "/api/v1/admin/login"
)

type authTestAdminRepo struct {
	user    domain.AdminUser
	findErr error
}

func (r *authTestAdminRepo) FindByEmail(context.Context, string) (domain.AdminUser, error) {
	if r.findErr != nil {
		return domain.AdminUser{}, r.findErr
	}
	return r.user, nil
}

func (r *authTestAdminRepo) FindByID(context.Context, int64) (domain.AdminUser, error) {
	return domain.AdminUser{}, errors.New("not implemented")
}

type authTestRefreshRepo struct{}

func (r *authTestRefreshRepo) Issue(_ context.Context, input domain.CreateRefreshTokenInput) (domain.RefreshToken, error) {
	return domain.RefreshToken{
		ID:          1,
		AdminUserID: input.AdminUserID,
		TokenHash:   input.TokenHash,
		ExpiresAt:   input.ExpiresAt,
	}, nil
}

func (r *authTestRefreshRepo) FindByTokenHash(context.Context, string) (domain.RefreshToken, error) {
	return domain.RefreshToken{}, errors.New("not implemented")
}

func (r *authTestRefreshRepo) Revoke(context.Context, int64) error { return nil }

func (r *authTestRefreshRepo) RevokeIfActive(context.Context, int64) (bool, error) {
	return true, nil
}

func (r *authTestRefreshRepo) RevokeAllByUser(context.Context, int64) error { return nil }

type authTestLoginAttemptRepo struct {
	counts    domain.LoginAttemptCounts
	countErr  error
	recordErr error
}

func (r *authTestLoginAttemptRepo) CountRecent(context.Context, string, string, time.Time) (domain.LoginAttemptCounts, error) {
	if r.countErr != nil {
		return domain.LoginAttemptCounts{}, r.countErr
	}
	return r.counts, nil
}

func (r *authTestLoginAttemptRepo) RecordFailure(context.Context, string, string, time.Time) error {
	return r.recordErr
}

func (r *authTestLoginAttemptRepo) ClearByEmail(context.Context, string) error {
	return nil
}

func mustAuthPasswordHash(t *testing.T, plain string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword: %v", err)
	}
	return string(hash)
}

func newAuthHandlerForTest(t *testing.T, admin *authTestAdminRepo, attempts *authTestLoginAttemptRepo) *AuthHandler {
	t.Helper()
	if attempts == nil {
		attempts = &authTestLoginAttemptRepo{}
	}
	jwtSvc := jwt.NewJWTService([]byte(testAuthJWTSecret), "satehits-api", "satehits-admin", time.Hour)
	loginUC := authusecase.NewLoginUseCase(
		admin,
		&authTestRefreshRepo{},
		attempts,
		jwtSvc,
		authusecase.LoginRateLimitPolicy{
			EmailMax: 5,
			IPMax:    20,
			Window:   15 * time.Minute,
		},
	)
	refreshUC := authusecase.NewRefreshUseCase(admin, &authTestRefreshRepo{}, jwtSvc, nil)
	logoutUC := authusecase.NewLogoutUseCase(&authTestRefreshRepo{})
	return NewAuthHandler(loginUC, refreshUC, logoutUC, []string{"http://localhost:4321"}, "", 1)
}

func assertAuthErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	var payload ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() err = %v; body = %s", err, rec.Body.String())
	}
	if payload.Error.Code != wantCode {
		t.Fatalf("error.code = %q, want %q; body = %s", payload.Error.Code, wantCode, rec.Body.String())
	}
}

func TestAuthHandler_HandleLogin(t *testing.T) {
	owner := domain.AdminUser{
		ID:           1,
		Email:        "owner@example.com",
		PasswordHash: mustAuthPasswordHash(t, testAuthPassword),
		Role:         "owner",
	}

	t.Run("returns 200 on valid credentials", func(t *testing.T) {
		h := newAuthHandlerForTest(t, &authTestAdminRepo{user: owner}, nil)
		body := `{"email":"owner@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, testAuthLoginPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.10")
		rec := httptest.NewRecorder()

		h.HandleLogin(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		var payload loginResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if payload.Token == "" {
			t.Fatal("token is empty")
		}
		if payload.User == nil || payload.User.Email != owner.Email {
			t.Fatalf("user = %#v, want email %q", payload.User, owner.Email)
		}
	})

	t.Run("returns 401 on wrong password", func(t *testing.T) {
		h := newAuthHandlerForTest(t, &authTestAdminRepo{user: owner}, nil)
		body := `{"email":"owner@example.com","password":"wrong"}`
		req := httptest.NewRequest(http.MethodPost, testAuthLoginPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.11:1234"
		rec := httptest.NewRecorder()

		h.HandleLogin(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401; body = %s", rec.Code, rec.Body.String())
		}
		assertAuthErrorCode(t, rec, UnauthorizedCode)
	})

	t.Run("returns 400 validation error when email is empty", func(t *testing.T) {
		h := newAuthHandlerForTest(t, &authTestAdminRepo{user: owner}, nil)
		body := `{"email":"","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, testAuthLoginPath, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.HandleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
		}
		assertAuthErrorCode(t, rec, ValidationErrorCode)
	})

	t.Run("returns 405 for non-POST", func(t *testing.T) {
		h := newAuthHandlerForTest(t, &authTestAdminRepo{user: owner}, nil)
		req := httptest.NewRequest(http.MethodGet, testAuthLoginPath, nil)
		rec := httptest.NewRecorder()

		h.HandleLogin(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", rec.Code)
		}
		assertAuthErrorCode(t, rec, InvalidRequestCode)
	})

	t.Run("returns 429 with Retry-After when rate limited", func(t *testing.T) {
		oldest := time.Now().Add(-5 * time.Minute)
		h := newAuthHandlerForTest(t, &authTestAdminRepo{user: owner}, &authTestLoginAttemptRepo{
			counts: domain.LoginAttemptCounts{
				ByEmail:       5,
				OldestByEmail: oldest,
			},
		})
		body := `{"email":"owner@example.com","password":"password123"}`
		req := httptest.NewRequest(http.MethodPost, testAuthLoginPath, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "203.0.113.99")
		rec := httptest.NewRecorder()

		h.HandleLogin(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429; body = %s", rec.Code, rec.Body.String())
		}
		assertAuthErrorCode(t, rec, TooManyRequestsCode)
		retryAfter := rec.Header().Get("Retry-After")
		if retryAfter == "" {
			t.Fatal("Retry-After header is empty")
		}
		seconds, err := strconv.Atoi(retryAfter)
		if err != nil || seconds < 1 {
			t.Fatalf("Retry-After = %q, want positive integer seconds", retryAfter)
		}
		var payload ErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		if !strings.Contains(payload.Error.Message, "分後") {
			t.Fatalf("message = %q, want minutes guidance", payload.Error.Message)
		}
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		h := newAuthHandlerForTest(t, &authTestAdminRepo{user: owner}, nil)
		req := httptest.NewRequest(http.MethodPost, testAuthLoginPath, strings.NewReader("{"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		h.HandleLogin(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
		}
		assertAuthErrorCode(t, rec, InvalidRequestCode)
	})
}
