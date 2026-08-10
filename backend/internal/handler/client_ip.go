package handler

import (
	"net"
	"net/http"
	"strings"
)

// clientIP は信頼するプロキシ段数に基づきクライアント IP を返す。
// Cloud Run は観測した実 IP を X-Forwarded-For の末尾に追加するため、
// 右から trustedProxyHops 番目を採用する。取れなければ RemoteAddr、それも無効なら空文字。
func clientIP(r *http.Request, trustedProxyHops int) string {
	if trustedProxyHops < 1 {
		trustedProxyHops = 1
	}

	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		idx := len(parts) - trustedProxyHops
		if idx >= 0 {
			if ip := parseIPToken(parts[idx]); ip != "" {
				return ip
			}
		}
		// 段数不足や無効値のときは右端から有効な IP を探す
		for i := len(parts) - 1; i >= 0; i-- {
			if ip := parseIPToken(parts[i]); ip != "" {
				return ip
			}
		}
	}

	return parseIPToken(r.RemoteAddr)
}

func parseIPToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	host := raw
	if h, _, err := net.SplitHostPort(raw); err == nil {
		host = h
	}
	// IPv6 リテラルがブラケット付きで来る場合
	host = strings.Trim(host, "[]")

	ip := net.ParseIP(host)
	if ip == nil {
		return ""
	}
	return ip.String()
}
