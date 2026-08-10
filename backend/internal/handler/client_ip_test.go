package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name             string
		xff              string
		remoteAddr       string
		trustedProxyHops int
		want             string
	}{
		{
			name:             "uses single X-Forwarded-For entry",
			xff:              "203.0.113.10",
			remoteAddr:       "10.0.0.1:12345",
			trustedProxyHops: 1,
			want:             "203.0.113.10",
		},
		{
			name:             "uses rightmost entry with hops=1",
			xff:              "198.51.100.1, 203.0.113.50",
			remoteAddr:       "10.0.0.1:12345",
			trustedProxyHops: 1,
			want:             "203.0.113.50",
		},
		{
			name:             "ignores spoofed leftmost entries",
			xff:              "1.2.3.4, 5.6.7.8, 203.0.113.9",
			remoteAddr:       "10.0.0.1:12345",
			trustedProxyHops: 1,
			want:             "203.0.113.9",
		},
		{
			name:             "uses second-from-right when hops=2",
			xff:              "198.51.100.1, 203.0.113.20, 10.0.0.2",
			remoteAddr:       "10.0.0.1:12345",
			trustedProxyHops: 2,
			want:             "203.0.113.20",
		},
		{
			name:             "parses IPv6 from X-Forwarded-For",
			xff:              "2001:db8::1",
			remoteAddr:       "10.0.0.1:12345",
			trustedProxyHops: 1,
			want:             "2001:db8::1",
		},
		{
			name:             "falls back to RemoteAddr host without port",
			xff:              "",
			remoteAddr:       "203.0.113.77:54321",
			trustedProxyHops: 1,
			want:             "203.0.113.77",
		},
		{
			name:             "falls back to RemoteAddr IPv6 with brackets",
			xff:              "",
			remoteAddr:       "[2001:db8::2]:443",
			trustedProxyHops: 1,
			want:             "2001:db8::2",
		},
		{
			name:             "returns empty when no usable IP",
			xff:              "not-an-ip",
			remoteAddr:       "garbage",
			trustedProxyHops: 1,
			want:             "",
		},
		{
			name:             "defaults hops below 1 to 1",
			xff:              "198.51.100.1, 203.0.113.30",
			remoteAddr:       "10.0.0.1:1",
			trustedProxyHops: 0,
			want:             "203.0.113.30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			req.RemoteAddr = tt.remoteAddr

			got := clientIP(req, tt.trustedProxyHops)
			if got != tt.want {
				t.Fatalf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
