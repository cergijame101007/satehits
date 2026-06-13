package handler

import "net/http"

// RT Cookie の仕様
const (
	refreshTokenCookieName = "refresh_token"
	adminCookiePath        = "/api/v1/admin"
	refreshTokenMaxAge     = 30 * 24 * 60 * 60 // 30日（秒）
)

// login / refresh 成功時に RT を httpOnly Cookie でセット
func setRefreshTokenCookie(w http.ResponseWriter, token string, domain string) {
	cookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     adminCookiePath,
		MaxAge:   refreshTokenMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	if domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(w, cookie)
}

// logout 時の RT Cookie 削除（Max-Age=0）
func clearRefreshTokenCookie(w http.ResponseWriter, domain string) {
	cookie := &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     adminCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
	if domain != "" {
		cookie.Domain = domain
	}
	http.SetCookie(w, cookie)
}
