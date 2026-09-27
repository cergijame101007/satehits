package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// docs/monitoring.md §4.6 L3: 予約申請の 400 はフィールド名・理由だけ記録し、入力値は出さない
func TestLogReservationValidationFailure(t *testing.T) {
	logBuf := captureLog(t)

	logReservationValidationFailure([]ErrorDetail{
		{Field: "visit_date", Message: "予約できない日付です"},
		{Field: "email", Message: "メールアドレスの形式が正しくありません"},
	})

	logText := logBuf.String()
	for _, want := range []string{"reservation validation failed", "visit_date", "email"} {
		if !strings.Contains(logText, want) {
			t.Errorf("log should contain %q: %s", want, logText)
		}
	}
	// メッセージ（利用者向けの文言）は冗長なので出さない
	if strings.Contains(logText, "予約できない日付です") {
		t.Errorf("log should not contain violation messages: %s", logText)
	}
}

func TestReservationHandler_logsMalformedRequest(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		wantReason  string
	}{
		{name: "logs content_type when the request is not JSON", contentType: "text/plain", body: "name=taro", wantReason: "reason=content_type"},
		{name: "logs decode when the JSON body is broken", contentType: mediaTypeJSON, body: `{"name":"山田`, wantReason: "reason=decode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf := captureLog(t)
			// 形式エラーは usecase を呼ぶ前に返るため usecase は nil でよい
			h := NewReservationHandler(nil, "/api/v1/reservations")
			req := httptest.NewRequest(http.MethodPost, "/api/v1/reservations", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			rec := httptest.NewRecorder()

			h.HandleReservations(rec, req)

			assertStatusAndCode(t, rec, http.StatusBadRequest, InvalidRequestCode)
			logText := logBuf.String()
			if !strings.Contains(logText, "reservation request malformed") || !strings.Contains(logText, tt.wantReason) {
				t.Errorf("log should contain the malformed reason %q: %s", tt.wantReason, logText)
			}
			if strings.Contains(logText, "山田") {
				t.Errorf("log should not contain the request body: %s", logText)
			}
		})
	}
}
