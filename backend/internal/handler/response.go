package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const mediaTypeJSON = "application/json"

// エラーコード
const (
	InvalidRequestCode      = "INVALID_REQUEST"
	ValidationErrorCode     = "VALIDATION_ERROR"
	UnauthorizedCode        = "UNAUTHORIZED"
	InvalidTokenCode        = "INVALID_TOKEN"
	ForbiddenCode           = "FORBIDDEN"
	NotFoundCode            = "NOT_FOUND"
	CapacityExceededCode    = "CAPACITY_EXCEEDED"
	ReservationConflictCode = "RESERVATION_CONFLICT"
	CaptchaFailedCode       = "CAPTCHA_FAILED"
	TooManyRequestsCode     = "TOO_MANY_REQUESTS"
	InternalErrorCode       = "INTERNAL_ERROR"
)

// JSONResponse はAPIレスポンスの共通構造体
type JSONResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// ErrorResponse はエラーレスポンスの共通構造体
type ErrorResponse struct {
	Error errorResponseBody `json:"error"`
}

// errorResponseBody はエラーレスポンスのボディ
type errorResponseBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// ErrorDetail は API の error.details 用のフィールドごとの詳細
// usecase の FieldViolation と同じ形の別定義
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// respondWithJSON はJSONレスポンスを返すヘルパー関数
func respondWithJSON[T any](w http.ResponseWriter, status int, payload T) {
	res, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		w.Header().Set("Content-Type", mediaTypeJSON)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"レスポンスの生成に失敗しました"}}`)); err != nil {
			log.Printf("Failed to write error fallback response: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", mediaTypeJSON)
	w.WriteHeader(status)
	if _, err := w.Write(res); err != nil {
		// ヘッダ送信後のため追加のレスポンスは不可
		log.Printf("Failed to write response: %v", err)
		return
	}
}

func respondWithError(w http.ResponseWriter, status int, code string, message string, details []ErrorDetail) {
	payload := ErrorResponse{
		Error: errorResponseBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	respondWithJSON(w, status, payload)
}

// respondMethodNotAllowed は Allow ヘッダに受け付けるメソッドを載せ、405 INVALID_REQUEST を返す
// OPTIONS は CORS ミドルウェアが先に 204 を返すため allowed に含めない
func respondMethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	respondWithError(w, http.StatusMethodNotAllowed, InvalidRequestCode, "許可されていないメソッドです", nil)
}

// respondNotFound はルーティングに一致しないパスへ 404 NOT_FOUND を返す
// 個別リソースが無い場合（予約・取引先など）は各 handler のメッセージを使う
func respondNotFound(w http.ResponseWriter) {
	respondWithError(w, http.StatusNotFound, NotFoundCode, "リソースが見つかりません", nil)
}
