package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetRefreshTokenCookie(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		wantDomain string
	}{
		{name: "omits Domain when not configured", domain: "", wantDomain: ""},
		{name: "sets Domain when configured", domain: "api.example.com", wantDomain: "api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			setRefreshTokenCookie(rec, testAuthRTPlain, tt.domain)

			c := findRefreshCookie(t, rec)
			assertRefreshCookieAttributes(t, c, refreshTokenMaxAge, tt.wantDomain)
			if c.Value != testAuthRTPlain {
				t.Fatalf("Value = %q, want %q", c.Value, testAuthRTPlain)
			}
			raw := rec.Header().Get("Set-Cookie")
			for _, attr := range []string{"Max-Age=2592000", "HttpOnly", "Secure", "SameSite=None", "Path=/api/v1/admin"} {
				if !strings.Contains(raw, attr) {
					t.Fatalf("Set-Cookie = %q, want to contain %q", raw, attr)
				}
			}
			if tt.wantDomain == "" && strings.Contains(raw, "Domain=") {
				t.Fatalf("Set-Cookie = %q, want no Domain attribute", raw)
			}
		})
	}
}

func TestClearRefreshTokenCookie(t *testing.T) {
	tests := []struct {
		name       string
		domain     string
		wantDomain string
	}{
		{name: "omits Domain when not configured", domain: "", wantDomain: ""},
		{name: "sets Domain when configured", domain: "api.example.com", wantDomain: "api.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			clearRefreshTokenCookie(rec, tt.domain)

			c := findRefreshCookie(t, rec)
			// net/http は Max-Age=0 を MaxAge=-1（即時削除）として解釈する
			assertRefreshCookieAttributes(t, c, -1, tt.wantDomain)
			if c.Value != "" {
				t.Fatalf("Value = %q, want empty", c.Value)
			}
			raw := rec.Header().Get("Set-Cookie")
			for _, attr := range []string{"Max-Age=0", "HttpOnly", "Secure", "SameSite=None", "Path=/api/v1/admin"} {
				if !strings.Contains(raw, attr) {
					t.Fatalf("Set-Cookie = %q, want to contain %q", raw, attr)
				}
			}
			if tt.wantDomain == "" && strings.Contains(raw, "Domain=") {
				t.Fatalf("Set-Cookie = %q, want no Domain attribute", raw)
			}
		})
	}
}
