package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

// logoutFixture は RequireAuth でラップした HandleLogout と、有効な AT / RT を持つ依存一式
type logoutFixture struct {
	handler http.Handler
	refresh *authTestRefreshRepo
	auth    *AuthHandler
	at      string
}

func newLogoutFixture(t *testing.T, edit func(deps *authTestDeps)) *logoutFixture {
	t.Helper()
	owner := domain.AdminUser{ID: 1, Email: testAuthEmail, Role: "owner"}
	refresh := &authTestRefreshRepo{}
	refresh.seed(authTestActiveRT(owner.ID))
	deps := authTestDeps{admin: &authTestAdminRepo{user: owner}, refresh: refresh}
	if edit != nil {
		edit(&deps)
	}
	svc := authTestJWTService()
	auth := newAuthHandlerWithDeps(t, deps)
	return &logoutFixture{
		handler: RequireAuth(svc)(http.HandlerFunc(auth.HandleLogout)),
		refresh: refresh,
		auth:    auth,
		at:      mustAuthAccessToken(t, svc, owner.ID, owner.Email, owner.Role),
	}
}

func (f *logoutFixture) request(authorization, origin, cookieValue string) *http.Request {
	req := newCookieAuthRequest(testAuthLogoutPath, origin, cookieValue)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	return req
}

func TestAuthHandler_HandleLogout(t *testing.T) {
	t.Run("returns 401 UNAUTHORIZED without access token and touches nothing", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("", testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, UnauthorizedCode)
		assertStrings(t, "repo calls", f.refresh.calls, nil)
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 401 INVALID_TOKEN with tampered access token", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+mustTamperedAuthToken(t, f.at, 2), testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusUnauthorized, InvalidTokenCode)
		assertStrings(t, "repo calls", f.refresh.calls, nil)
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 403 FORBIDDEN when Origin is missing", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, "", testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusForbidden, ForbiddenCode)
		assertStrings(t, "repo calls", f.refresh.calls, nil)
		assertNoSetCookie(t, rec)
	})

	t.Run("returns 403 when Origin is not allowed", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, "https://evil.example", testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusForbidden, ForbiddenCode)
		assertStrings(t, "repo calls", f.refresh.calls, nil)
	})

	t.Run("returns 204 with allowed Referer when Origin is missing", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		req := f.request("Bearer "+f.at, "", testAuthRTPlain)
		req.Header.Set("Referer", testAuthOrigin+"/admin/")
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("returns 405 for non-POST", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		req := httptest.NewRequest(http.MethodGet, testAuthLogoutPath, nil)
		req.Header.Set("Authorization", "Bearer "+f.at)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, req)

		assertStatusAndCode(t, rec, http.StatusMethodNotAllowed, InvalidRequestCode)
	})

	t.Run("returns 204, revokes the token by hash and deletes the cookie", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, testAuthOrigin, testAuthRTPlain))

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("body = %s, want empty", rec.Body.String())
		}
		assertStrings(t, "FindByTokenHash keys", f.refresh.findHashes, []string{authTestTokenHash(testAuthRTPlain)})
		assertInt64s(t, "Revoke ids", f.refresh.revokeIDs, []int64{42})
		if f.refresh.findByID(42).RevokedAt == nil {
			t.Fatal("refresh token is still active after logout")
		}
		c := findRefreshCookie(t, rec)
		assertRefreshCookieAttributes(t, c, -1, "")
		if c.Value != "" {
			t.Fatalf("cookie value = %q, want empty", c.Value)
		}
		if raw := rec.Header().Get("Set-Cookie"); !strings.Contains(raw, "Max-Age=0") {
			t.Fatalf("Set-Cookie = %q, want Max-Age=0", raw)
		}
	})

	t.Run("adds Domain to the deletion cookie only when configured", func(t *testing.T) {
		f := newLogoutFixture(t, func(deps *authTestDeps) { deps.cookieDomain = "api.example.com" })
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, testAuthOrigin, testAuthRTPlain))

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
		assertRefreshCookieAttributes(t, findRefreshCookie(t, rec), -1, "api.example.com")
	})

	t.Run("returns 204 without cookie and does not call Revoke", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, testAuthOrigin, ""))

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
		assertStrings(t, "repo calls", f.refresh.calls, nil)
		assertRefreshCookieAttributes(t, findRefreshCookie(t, rec), -1, "")
	})

	t.Run("returns 204 for unknown token and does not call Revoke", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, testAuthOrigin, strings.Repeat("f", 64)))

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
		assertStrings(t, "repo calls", f.refresh.calls, []string{"FindByTokenHash"})
		assertRefreshCookieAttributes(t, findRefreshCookie(t, rec), -1, "")
	})

	t.Run("returns 500 without deleting the cookie when the repository fails", func(t *testing.T) {
		const marker = "repo failure marker"
		logBuf := captureLog(t)
		f := newLogoutFixture(t, nil)
		f.refresh.revokeErr = errors.New(marker)
		rec := httptest.NewRecorder()

		f.handler.ServeHTTP(rec, f.request("Bearer "+f.at, testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, rec, http.StatusInternalServerError, InternalErrorCode)
		assertNoSetCookie(t, rec)
		if strings.Contains(rec.Body.String(), marker) {
			t.Fatalf("body leaks internal error: %s", rec.Body.String())
		}
		logText := logBuf.String()
		if !strings.Contains(logText, marker) {
			t.Fatalf("log = %q, want to contain %q", logText, marker)
		}
		assertNoSecretsInLog(t, logText, testAuthRTPlain, authTestTokenHash(testAuthRTPlain), f.at)
	})

	t.Run("rejects refresh with the same token after logout", func(t *testing.T) {
		f := newLogoutFixture(t, nil)
		logoutRec := httptest.NewRecorder()
		f.handler.ServeHTTP(logoutRec, f.request("Bearer "+f.at, testAuthOrigin, testAuthRTPlain))
		if logoutRec.Code != http.StatusNoContent {
			t.Fatalf("logout status = %d; body = %s", logoutRec.Code, logoutRec.Body.String())
		}

		refreshRec := httptest.NewRecorder()
		f.auth.HandleRefresh(refreshRec, newCookieAuthRequest(testAuthRefreshPath, testAuthOrigin, testAuthRTPlain))

		assertStatusAndCode(t, refreshRec, http.StatusUnauthorized, InvalidTokenCode)
		assertNoSetCookie(t, refreshRec)
	})
}
