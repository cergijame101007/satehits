package handler

import (
	"net"
	"net/http"
	"strings"
)

// clientRemoteIP は Turnstile siteverify 用のクライアント IP を返す
func clientRemoteIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
