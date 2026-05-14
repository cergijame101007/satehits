package handler

import (
	"encoding/json"
	"log"
	"net/http"
)

// エラーコード
const (
	InvalidRequestCode   = "INVALID_REQUEST"
	ValidationErrorCode  = "VALIDATION_ERROR"
	UnauthorizedCode     = "UNAUTHORIZED"
	InvalidTokenCode     = "INVALID_TOKEN"
	ForbiddenCode        = "FORBIDDEN"
	NotFoundCode         = "NOT_FOUND"
	CapacityExceededCode = "CAPACITY_EXCEEDED"
	InternalErrorCode    = "INTERNAL_ERROR"
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"レスポンスの生成に失敗しました"}}`)); err != nil {
			log.Printf("Failed to write error fallback response: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
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
