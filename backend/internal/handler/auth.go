package handler

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"strconv"
	"time"

	authusecase "github.com/cergijame101007/satehits/internal/application/usecase/auth"
	"github.com/cergijame101007/satehits/internal/domain"
)

// loginRequest は POST /admin/login のリクエストボディ
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginUserResponse は LoginResponse.user
type loginUserResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// loginResponse は login / refresh 共通のレスポンス（refresh では user を省略）
type loginResponse struct {
	Token     string             `json:"token"`
	ExpiresAt time.Time          `json:"expires_at"`
	User      *loginUserResponse `json:"user,omitempty"`
}

// AuthHandler は管理者認証（login / refresh / logout）の HTTP ハンドラ
// AT 検証は RequireAuth ミドルウェアの責務、Cookie・CSRF・レスポンス整形に集中
type AuthHandler struct {
	login            *authusecase.LoginUseCase
	refresh          *authusecase.RefreshUseCase
	logout           *authusecase.LogoutUseCase
	corsOrigins      []string
	cookieDomain     string
	trustedProxyHops int
}

// NewAuthHandler は AuthHandler 生成
func NewAuthHandler(
	login *authusecase.LoginUseCase,
	refresh *authusecase.RefreshUseCase,
	logout *authusecase.LogoutUseCase,
	corsOrigins []string,
	cookieDomain string,
	trustedProxyHops int,
) *AuthHandler {
	return &AuthHandler{
		login:            login,
		refresh:          refresh,
		logout:           logout,
		corsOrigins:      corsOrigins,
		cookieDomain:     cookieDomain,
		trustedProxyHops: trustedProxyHops,
	}
}

// HandleLogin は POST /admin/login
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, InvalidRequestCode, "許可されていないメソッドです", nil)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != mediaTypeJSON {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	result, err := h.login.Execute(r.Context(), authusecase.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
		ClientIP: clientIP(r, h.trustedProxyHops),
	})
	if err != nil {
		var vErr *authusecase.ValidationError
		if errors.As(err, &vErr) {
			details := make([]ErrorDetail, len(vErr.Violations))
			for i, v := range vErr.Violations {
				details[i] = ErrorDetail{Field: v.Field, Message: v.Message}
			}
			respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", details)
			return
		}
		var rateErr *authusecase.RateLimitedError
		if errors.As(err, &rateErr) {
			seconds := int(rateErr.RetryAfter.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			respondWithError(w, http.StatusTooManyRequests, TooManyRequestsCode, rateErr.Error(), nil)
			return
		}
		if errors.Is(err, domain.ErrAdminUserUnauthorized) {
			respondWithError(w, http.StatusUnauthorized, UnauthorizedCode, "メールアドレスまたはパスワードが正しくありません", nil)
			return
		}
		log.Printf("Login failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, h.cookieDomain)
	respondWithJSON(w, http.StatusOK, loginResponse{
		Token:     result.AccessToken,
		ExpiresAt: result.ExpiresAt,
		User: &loginUserResponse{
			ID:    result.User.ID,
			Email: result.User.Email,
			Role:  result.User.Role,
		},
	})
}

// HandleRefresh は POST /admin/refresh
func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, InvalidRequestCode, "許可されていないメソッドです", nil)
		return
	}

	if !checkOrigin(w, r, h.corsOrigins) {
		return
	}

	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil || cookie.Value == "" {
		respondWithError(w, http.StatusUnauthorized, InvalidTokenCode, "セッションが無効です。再度ログインしてください", nil)
		return
	}

	result, err := h.refresh.Execute(r.Context(), authusecase.RefreshCommand{RefreshToken: cookie.Value})
	if err != nil {
		if errors.Is(err, domain.ErrRefreshTokenInvalid) || errors.Is(err, domain.ErrRefreshTokenNotFound) {
			respondWithError(w, http.StatusUnauthorized, InvalidTokenCode, "セッションが無効です。再度ログインしてください", nil)
			return
		}
		log.Printf("Refresh failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	setRefreshTokenCookie(w, result.RefreshToken, h.cookieDomain)
	respondWithJSON(w, http.StatusOK, loginResponse{
		Token:     result.AccessToken,
		ExpiresAt: result.ExpiresAt,
	})
}

// HandleLogout は POST /admin/logout（RequireAuth でラップ前提）
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, InvalidRequestCode, "許可されていないメソッドです", nil)
		return
	}

	if !checkOrigin(w, r, h.corsOrigins) {
		return
	}

	var refreshTokenValue string
	if cookie, err := r.Cookie(refreshTokenCookieName); err == nil {
		refreshTokenValue = cookie.Value
	}

	if err := h.logout.Execute(r.Context(), authusecase.LogoutCommand{RefreshToken: refreshTokenValue}); err != nil {
		log.Printf("Logout failed: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	clearRefreshTokenCookie(w, h.cookieDomain)
	w.WriteHeader(http.StatusNoContent)
}
