package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestAuthHandler_HandleRefresh(t *testing.T) {
	owner := domain.AdminUser{ID: 1, Email: testAuthEmail, Role: "owner"}

	// 有効な RT を 1 件持つ依存一式
	seeded := func(edit func(rt *domain.RefreshToken)) (*authTestRefreshRepo, authTestDeps) {
		rt := authTestActiveRT(owner.ID)
		if edit != nil {
			edit(&rt)
		}
		refresh := &authTestRefreshRepo{}
		refresh.seed(rt)
		return refresh, authTestDeps{admin: &authTestAdminRepo{user: owner}, refresh: refresh}
	}

	t.Run("returns 405 for non-POST", func(t *testing.T) {
		_, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		req := httptest.NewRequest(http.MethodGet, testAuthRefreshPath, nil)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, req)

		assertStatusAndCode(t, rec, http.StatusMethodNotAllowed, InvalidRequestCode)
	})

	t.Run("returns 403 FORBIDDEN before reading the cookie when Origin is missing", func(t *testing.T) {
		refresh, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, "", testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusForbidden, ForbiddenCode)
		assertStrings(t, "repo calls", refresh.calls, nil)
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 403 when Origin is not allowed", func(t *testing.T) {
		refresh, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, "https://evil.example", testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusForbidden, ForbiddenCode)
		assertStrings(t, "repo calls", refresh.calls, nil)
	})

	t.Run("accepts allowed Referer when Origin is missing", func(t *testing.T) {
		_, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		req := newCookieAuthRequest(testAuthRefreshPath, "", testAuthRTPlain)
		req.Header.Set("Referer", testAuthOrigin+"/admin/reservations")
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 401 INVALID_TOKEN when cookie is missing", func(t *testing.T) {
		refresh, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, ""))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", refresh.calls, nil)
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 401 INVALID_TOKEN when cookie value is empty", func(t *testing.T) {
		refresh, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		req := newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, "")
		req.Header.Set("Cookie", refreshTokenCookieName+"=")
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, req)

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", refresh.calls, nil)
	})

	t.Run("returns 401 for unknown token without revoking any session", func(t *testing.T) {
		refresh, deps := seeded(nil)
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, strings.Repeat("f", 64)))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", refresh.calls, []string{"FindByTokenHash"})
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 401 and revokes all sessions when a revoked token is reused", func(t *testing.T) {
		refresh, deps := seeded(func(rt *domain.RefreshToken) {
			revokedAt := time.Now().Add(-time.Hour)
			rt.RevokedAt = &revokedAt
		})
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", refresh.calls, []string{"FindByTokenHash", "RevokeAllByUser"})
		assertInt64s(t, "RevokeAllByUser ids", refresh.revokeAllUserIDs, []int64{owner.ID})
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 401 without rotation when token expired 1ms ago", func(t *testing.T) {
		refresh, deps := seeded(func(rt *domain.RefreshToken) { rt.ExpiresAt = time.Now().Add(-time.Millisecond) })
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", refresh.calls, []string{"FindByTokenHash"})
		assertNoSetCookie(t, rec)
	})

	t.Run("rotates when token expires 1s from now", func(t *testing.T) {
		_, deps := seeded(func(rt *domain.RefreshToken) { rt.ExpiresAt = time.Now().Add(time.Second) })
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 401 and revokes all sessions when concurrent rotation already revoked the token", func(t *testing.T) {
		refresh, deps := seeded(nil)
		notActive := false
		refresh.revokeIfActiveResult = &notActive
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", refresh.calls, []string{"FindByTokenHash", "RevokeIfActive", "RevokeAllByUser"})
		assertInt64s(t, "RevokeAllByUser ids", refresh.revokeAllUserIDs, []int64{owner.ID})
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 500 without updating the cookie when Issue fails", func(t *testing.T) {
		refresh, deps := seeded(nil)
		refresh.issueErr = errors.New("insert failed")
		h := newAuthHandlerWithDeps(t, deps)
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusInternalServerError, InternalErrorCode)
		assertNoSetCookie(t, rec)
		if strings.Contains(rec.Body.String(), "insert failed") {
			t.Fatalf("body leaks internal error: %s", rec.Body.String())
		}
	})

	t.Run("rotates the refresh token and returns a new access token", func(t *testing.T) {
		refresh, deps := seeded(nil)
		tx := &authTestTxManager{}
		deps.tx = tx
		h := newAuthHandlerWithDeps(t, deps)
		before := time.Now()
		rec := httptest.NewRecorder()

		h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}

		// 平文ではなく SHA-256 ハッシュで照合する
		assertStrings(t, "FindByTokenHash keys", refresh.findHashes, []string{authTestTokenHash(testAuthRTPlain)})

		// 旧 RT の RevokeIfActive → 新 RT の Issue の順で、1 トランザクション内
		assertStrings(t, "repo calls", refresh.calls, []string{"FindByTokenHash", "RevokeIfActive", "Issue"})
		assertInt64s(t, "RevokeIfActive ids", refresh.revokeIfActiveIDs, []int64{42})
		if tx.calls != 1 {
			t.Fatalf("DoInTx calls = %d, want 1", tx.calls)
		}

		payload := decodeJSONObject(t, rec.Body.Bytes())
		assertJSONKeys(t, payload, "token", "expires_at")

		c := findRefreshCookie(t, rec)
		assertRefreshCookieAttributes(t, c, refreshTokenMaxAge, "")
		if c.Value == testAuthRTPlain {
			t.Fatal("cookie still carries the old refresh token")
		}
		if len(c.Value) != 64 || !reHex64.MatchString(c.Value) {
			t.Fatalf("cookie value = %q, want 64 hex chars", c.Value)
		}
		if strings.Contains(rec.Body.String(), c.Value) || strings.Contains(rec.Body.String(), testAuthRTPlain) {
			t.Fatalf("body contains a refresh token: %s", rec.Body.String())
		}

		issued := refresh.issueInputs[0]
		if issued.AdminUserID != owner.ID {
			t.Fatalf("Issue AdminUserID = %d, want %d", issued.AdminUserID, owner.ID)
		}
		if issued.TokenHash != authTestTokenHash(c.Value) {
			t.Fatalf("Issue TokenHash = %q, want sha256(new cookie value)", issued.TokenHash)
		}
		remaining := issued.ExpiresAt.Sub(before)
		if remaining < 30*24*time.Hour-5*time.Second || remaining > 30*24*time.Hour+5*time.Second {
			t.Fatalf("Issue ExpiresAt is %v from now, want about 30d", remaining)
		}

		var body loginResponse
		if err := decodeInto(rec, &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		atRemaining := body.ExpiresAt.Sub(before)
		if atRemaining < time.Hour-5*time.Second || atRemaining > time.Hour+5*time.Second {
			t.Fatalf("expires_at is %v from now, want about 1h", atRemaining)
		}
		claims, err := authTestJWTService().VerifyAccessToken(body.Token)
		if err != nil {
			t.Fatalf("issued access token does not verify: %v", err)
		}
		if claims.Subject != "1" || claims.Email != owner.Email || claims.Role != owner.Role {
			t.Fatalf("claims = sub %q email %q role %q, want 1 / %s / %s", claims.Subject, claims.Email, claims.Role, owner.Email, owner.Role)
		}
	})

	t.Run("logs contain no secrets", func(t *testing.T) {
		const marker = "repo failure marker"
		tests := []struct {
			name       string
			edit       func(refresh *authTestRefreshRepo)
			wantStatus int
			wantMarker bool
		}{
			{
				name: "401 on revoked token reuse",
				edit: func(refresh *authTestRefreshRepo) {
					revokedAt := time.Now()
					refresh.findByID(42).RevokedAt = &revokedAt
				},
				wantStatus: http.StatusUnauthorized,
			},
			{
				name:       "500 when FindByTokenHash fails",
				edit:       func(refresh *authTestRefreshRepo) { refresh.findErr = errors.New(marker) },
				wantStatus: http.StatusInternalServerError,
				wantMarker: true,
			},
			{
				name:       "500 when Issue fails",
				edit:       func(refresh *authTestRefreshRepo) { refresh.issueErr = errors.New(marker) },
				wantStatus: http.StatusInternalServerError,
				wantMarker: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				logBuf := captureLog(t)
				refresh, deps := seeded(nil)
				tt.edit(refresh)
				h := newAuthHandlerWithDeps(t, deps)
				rec := httptest.NewRecorder()

				h.HandleRefresh(rec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

				if rec.Code != tt.wantStatus {
					t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
				}
				logText := logBuf.String()
				if tt.wantMarker && !strings.Contains(logText, marker) {
					t.Fatalf("log = %q, want to contain %q", logText, marker)
				}
				assertNoSecretsInLog(t, logText, testAuthRTPlain, authTestTokenHash(testAuthRTPlain))
			})
		}
	})
}

// login → refresh → 旧 RT 再利用 → 新 RT の順で、同じ repo に対して実行する
func TestAuthHandler_sessionLifecycle_reuseOfRotatedTokenRevokesEverything(t *testing.T) {
	owner := authTestOwner(t)
	refresh := &authTestRefreshRepo{}
	h := newAuthHandlerWithDeps(t, authTestDeps{admin: &authTestAdminRepo{user: owner}, refresh: refresh})

	loginRec := httptest.NewRecorder()
	h.HandleLogin(loginRec, newLoginRequest(owner.Email, testAuthPassword))
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status = %d; body = %s", loginRec.Code, loginRec.Body.String())
	}
	first := findRefreshCookie(t, loginRec).Value

	refreshRec := httptest.NewRecorder()
	h.HandleRefresh(refreshRec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, first))
	if refreshRec.Code != http.StatusOK {
		t.Fatalf("refresh status = %d; body = %s", refreshRec.Code, refreshRec.Body.String())
	}
	second := findRefreshCookie(t, refreshRec).Value
	if second == first {
		t.Fatal("refresh did not rotate the token")
	}

	reuseRec := httptest.NewRecorder()
	h.HandleRefresh(reuseRec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, first))
	assertStatusAndCode(t, reuseRec, http.StatusUnauthorized, InvalidTokenCode)
	assertInt64s(t, "RevokeAllByUser ids", refresh.revokeAllUserIDs, []int64{owner.ID})

	afterRec := httptest.NewRecorder()
	h.HandleRefresh(afterRec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, second))
	assertStatusAndCode(t, afterRec, http.StatusUnauthorized, InvalidTokenCode)
}
