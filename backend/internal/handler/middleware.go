package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/cergijame101007/satehits/pkg/jwt"
)

// NOTE: 認証 / CORS / CSRF ミドルウェアは現状 package handler に同居
// respondWithError など非公開ヘルパー再利用のため、別パッケージ化は export か複製が必要
// internal/presentation/ 移行時は presentation/handler と presentation/middleware へ分割
// 参照: docs/architecture.md

type ctxKey string

const claimsCtxKey ctxKey = "admin_claims"

// RequireAuth は Bearer AT 検証、claims を context へ載せる JWT 認証ミドルウェア
// 保護対象 /admin/* ハンドラのラップ用
func RequireAuth(jwtSvc *jwt.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			const bearerPrefix = "Bearer "
			if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
				respondWithError(w, http.StatusUnauthorized, UnauthorizedCode, "認証が必要です", nil)
				return
			}

			token := strings.TrimSpace(authHeader[len(bearerPrefix):])
			if token == "" {
				respondWithError(w, http.StatusUnauthorized, UnauthorizedCode, "認証が必要です", nil)
				return
			}

			claims, err := jwtSvc.VerifyAccessToken(token)
			if err != nil {
				respondWithError(w, http.StatusUnauthorized, InvalidTokenCode, "トークンが無効です", nil)
				return
			}

			ctx := context.WithValue(r.Context(), claimsCtxKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CORS は credentials 付きリクエスト用ミドルウェア
// fetch(..., { credentials: 'include' }) 向け、許可リスト一致 Origin をそのまま反映（* は不可）
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if isAllowedOrigin(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// checkOrigin は refresh / logout の CSRF 緩和、Origin または Referer の許可リスト照合
// 許可時 true、拒否時 403 送出後 false
func checkOrigin(w http.ResponseWriter, r *http.Request, allowedOrigins []string) bool {
	origin := requestOrigin(r)
	if origin != "" && isAllowedOrigin(origin, allowedOrigins) {
		return true
	}

	respondWithError(w, http.StatusForbidden, ForbiddenCode, "リクエストが拒否されました", nil)
	return false
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

func requestOrigin(r *http.Request) string {
	origin := r.Header.Get("Origin")
	if origin != "" {
		return origin
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		return ""
	}

	u, err := url.Parse(referer)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}

	return u.Scheme + "://" + u.Host
}
