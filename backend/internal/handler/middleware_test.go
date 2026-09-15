package handler

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/cergijame101007/satehits/pkg/jwt"
)

// recordingHandler は RequireAuth の next。呼ばれたことと context の claims を記録する
type recordingHandler struct {
	called bool
	claims *jwt.Claims
}

func (h *recordingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	if c, ok := r.Context().Value(claimsCtxKey).(*jwt.Claims); ok {
		h.claims = c
	}
	w.WriteHeader(http.StatusOK)
}

func mustNoneAlgAuthToken(t *testing.T) string {
	t.Helper()
	now := time.Now()
	unsigned := gojwt.NewWithClaims(gojwt.SigningMethodNone, gojwt.MapClaims{
		"sub":   "1",
		"email": testAuthEmail,
		"role":  "owner",
		"iss":   testAuthIssuer,
		"aud":   testAuthAudience,
		"exp":   now.Add(time.Hour).Unix(),
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
	})
	token, err := unsigned.SignedString(gojwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build none token: %v", err)
	}
	return token
}

func mustTamperedAuthToken(t *testing.T, valid string, part int) string {
	t.Helper()
	parts := strings.Split(valid, ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts, want 3", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[part])
	if err != nil {
		t.Fatalf("decode part %d: %v", part, err)
	}
	raw[0] ^= 0xff
	parts[part] = base64.RawURLEncoding.EncodeToString(raw)
	return strings.Join(parts, ".")
}

func TestRequireAuth(t *testing.T) {
	secret := []byte(testAuthJWTSecret)
	otherSecret := []byte("another-secret-at-least-32-bytes-xx")
	svc := authTestJWTService()
	valid := mustAuthAccessToken(t, svc, 1, testAuthEmail, "owner")
	basic := base64.StdEncoding.EncodeToString([]byte(testAuthEmail + ":" + testAuthPassword))

	tests := []struct {
		name          string
		authorization string
		wantCode      string
	}{
		{name: "rejects missing Authorization header", authorization: "", wantCode: UnauthorizedCode},
		{name: "rejects Basic scheme", authorization: "Basic " + basic, wantCode: UnauthorizedCode},
		{name: "rejects bare token without scheme", authorization: valid, wantCode: UnauthorizedCode},
		{name: "rejects Bearer with empty token", authorization: "Bearer ", wantCode: UnauthorizedCode},
		{name: "rejects Bearer with whitespace token", authorization: "Bearer    ", wantCode: UnauthorizedCode},
		{name: "rejects malformed token", authorization: "Bearer not-a-jwt", wantCode: InvalidTokenCode},
		{name: "rejects token with tampered payload", authorization: "Bearer " + mustTamperedAuthToken(t, valid, 1), wantCode: InvalidTokenCode},
		{name: "rejects token with tampered signature", authorization: "Bearer " + mustTamperedAuthToken(t, valid, 2), wantCode: InvalidTokenCode},
		{
			name:          "rejects token signed with another secret",
			authorization: "Bearer " + mustSignedAuthToken(t, otherSecret, nil),
			wantCode:      InvalidTokenCode,
		},
		{name: "rejects alg=none token", authorization: "Bearer " + mustNoneAlgAuthToken(t), wantCode: InvalidTokenCode},
		{
			name: "rejects expired token",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.ExpiresAt = gojwt.NewNumericDate(time.Now().Add(-time.Second))
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects token not valid yet",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.NotBefore = gojwt.NewNumericDate(time.Now().Add(time.Minute))
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects token without iat",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.IssuedAt = nil
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects token without exp",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.ExpiresAt = nil
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects wrong issuer",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.Issuer = "evil-issuer"
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects wrong audience",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.Audience = gojwt.ClaimStrings{"evil-audience"}
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects non-numeric subject",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.Subject = "owner"
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects zero subject",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.Subject = "0"
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects negative subject",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.Subject = "-1"
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects role admin",
			authorization: "Bearer " + mustSignedAuthToken(t, secret, func(c *jwt.Claims) {
				c.Role = "admin"
			}),
			wantCode: InvalidTokenCode,
		},
		{
			name: "rejects forged token that only carries a real email",
			authorization: "Bearer " + mustSignedAuthToken(t, otherSecret, func(c *jwt.Claims) {
				c.Email = testAuthEmail
				c.Subject = "999"
			}),
			wantCode: InvalidTokenCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &recordingHandler{}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reservations", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()

			RequireAuth(svc)(next).ServeHTTP(rec, req)

			assertStatusAndCode(t, rec, http.StatusUnauthorized, tt.wantCode)
			if next.called {
				t.Fatal("next handler was called, want blocked")
			}
		})
	}

	t.Run("calls next with claims in context for a valid token", func(t *testing.T) {
		next := &recordingHandler{}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/reservations", nil)
		req.Header.Set("Authorization", "Bearer "+valid)
		rec := httptest.NewRecorder()

		RequireAuth(svc)(next).ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}
		if !next.called {
			t.Fatal("next handler was not called")
		}
		if next.claims == nil {
			t.Fatal("claims missing from context")
		}
		if next.claims.Subject != "1" || next.claims.Email != testAuthEmail || next.claims.Role != "owner" {
			t.Fatalf("claims = sub %q email %q role %q, want 1 / %s / owner", next.claims.Subject, next.claims.Email, next.claims.Role, testAuthEmail)
		}
	})
}

func TestRequestOrigin(t *testing.T) {
	tests := []struct {
		name    string
		origin  string
		referer string
		want    string
	}{
		{name: "prefers Origin header", origin: testAuthOrigin, referer: "https://evil.example/", want: testAuthOrigin},
		{name: "falls back to Referer scheme and host", referer: testAuthOrigin + "/admin/reservations?x=1", want: testAuthOrigin},
		{name: "keeps port from Referer", referer: "https://example.com:8443/path", want: "https://example.com:8443"},
		{name: "returns empty without both headers", want: ""},
		{name: "returns empty for Referer without scheme", referer: "localhost:4321/admin", want: ""},
		{name: "returns empty for relative Referer", referer: "/admin", want: ""},
		{name: "returns empty for malformed Referer", referer: "http://[::1", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, testAuthRefreshPath, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			if got := requestOrigin(req); got != tt.want {
				t.Fatalf("requestOrigin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckOrigin(t *testing.T) {
	allowed := []string{"https://satehits.com", testAuthOrigin}

	tests := []struct {
		name    string
		origin  string
		referer string
		wantOK  bool
	}{
		{name: "allows listed Origin", origin: "https://satehits.com", wantOK: true},
		{name: "allows listed Referer when Origin is missing", referer: testAuthOrigin + "/admin/", wantOK: true},
		{name: "rejects when both headers are missing", wantOK: false},
		{name: "rejects unlisted Origin", origin: "https://evil.example", wantOK: false},
		{name: "rejects unlisted Origin even if Referer is listed", origin: "https://evil.example", referer: testAuthOrigin + "/", wantOK: false},
		{name: "rejects null Origin", origin: "null", wantOK: false},
		{name: "rejects Origin with trailing slash", origin: "https://satehits.com/", wantOK: false},
		{name: "rejects Origin with different scheme", origin: "http://satehits.com", wantOK: false},
		{name: "rejects Origin with different port", origin: "http://localhost:3000", wantOK: false},
		{name: "rejects subdomain of listed Origin", origin: "https://evil.satehits.com", wantOK: false},
		{name: "rejects unlisted Referer", referer: "https://evil.example/admin", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, testAuthRefreshPath, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			rec := httptest.NewRecorder()

			got := checkOrigin(rec, req, allowed)

			if got != tt.wantOK {
				t.Fatalf("checkOrigin() = %v, want %v", got, tt.wantOK)
			}
			if tt.wantOK {
				if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
					t.Fatalf("response written on allow: status=%d body=%s", rec.Code, rec.Body.String())
				}
				return
			}
			assertStatusAndCode(t, rec, http.StatusForbidden, ForbiddenCode)
		})
	}
}

func TestCORS(t *testing.T) {
	allowed := []string{testAuthOrigin}
	corsHeaders := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Expose-Headers",
	}

	tests := []struct {
		name        string
		method      string
		origin      string
		allowed     []string
		wantHeaders bool
		wantStatus  int
		wantNext    bool
	}{
		{name: "echoes listed Origin with credentials", method: http.MethodGet, origin: testAuthOrigin, allowed: allowed, wantHeaders: true, wantStatus: http.StatusOK, wantNext: true},
		{name: "adds no CORS headers for unlisted Origin", method: http.MethodGet, origin: "https://evil.example", allowed: allowed, wantHeaders: false, wantStatus: http.StatusOK, wantNext: true},
		{name: "adds no CORS headers without Origin", method: http.MethodGet, origin: "", allowed: allowed, wantHeaders: false, wantStatus: http.StatusOK, wantNext: true},
		{name: "answers preflight with 204 without calling next", method: http.MethodOptions, origin: testAuthOrigin, allowed: allowed, wantHeaders: true, wantStatus: http.StatusNoContent, wantNext: false},
		{name: "answers preflight from unlisted Origin with 204 and no CORS headers", method: http.MethodOptions, origin: "https://evil.example", allowed: allowed, wantHeaders: false, wantStatus: http.StatusNoContent, wantNext: false},
		{name: "does not treat wildcard entry as a match", method: http.MethodGet, origin: "https://evil.example", allowed: []string{"*"}, wantHeaders: false, wantStatus: http.StatusOK, wantNext: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := &recordingHandler{}
			req := httptest.NewRequest(tt.method, testAuthRefreshPath, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			CORS(tt.allowed)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if next.called != tt.wantNext {
				t.Fatalf("next called = %v, want %v", next.called, tt.wantNext)
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "*" {
				t.Fatal("Access-Control-Allow-Origin = *, want never wildcard with credentials")
			}
			if !tt.wantHeaders {
				for _, h := range corsHeaders {
					if v := rec.Header().Get(h); v != "" {
						t.Fatalf("%s = %q, want absent", h, v)
					}
				}
				return
			}
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != tt.origin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, tt.origin)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
			}
			if got := rec.Header().Get("Access-Control-Expose-Headers"); !strings.Contains(got, "Retry-After") {
				t.Fatalf("Access-Control-Expose-Headers = %q, want Retry-After", got)
			}
			if got := rec.Header().Get("Vary"); !strings.Contains(got, "Origin") {
				t.Fatalf("Vary = %q, want Origin", got)
			}
		})
	}
}
