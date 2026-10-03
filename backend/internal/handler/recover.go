package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover は panic を捕捉して 500 INTERNAL_ERROR を返し、スタック付きの ERROR ログを出すミドルウェア
// 最外周に置く（main.go）。net/http 標準の回復は平文ログを出して接続を切るだけで、アラートに乗らないため
// stack_trace キーは Error Reporting がスタックとして解釈する（docs/monitoring.md §4.3）
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &headerRecorder{ResponseWriter: w}
		defer func() {
			p := recover()
			if p == nil {
				return
			}
			// 接続を切る意図の panic は net/http の規約どおりそのまま伝える
			if err, ok := p.(error); ok && errors.Is(err, http.ErrAbortHandler) {
				panic(p)
			}
			slog.Error("panic recovered",
				"panic", fmt.Sprint(p),
				"method", r.Method,
				"path", truncateForLog(r.URL.Path),
				"stack_trace", string(debug.Stack()),
			)
			// ヘッダ送信後は追加のレスポンスを書けない
			if !rec.wroteHeader {
				respondWithError(rec, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
			}
		}()
		next.ServeHTTP(rec, r)
	})
}

// headerRecorder はレスポンスヘッダを送信済みかどうかを記録する ResponseWriter
type headerRecorder struct {
	http.ResponseWriter
	wroteHeader bool
}

func (h *headerRecorder) WriteHeader(status int) {
	h.wroteHeader = true
	h.ResponseWriter.WriteHeader(status)
}

func (h *headerRecorder) Write(b []byte) (int, error) {
	h.wroteHeader = true
	return h.ResponseWriter.Write(b)
}

// Unwrap は http.ResponseController が元の ResponseWriter（Flusher 等）へ到達できるようにする
func (h *headerRecorder) Unwrap() http.ResponseWriter {
	return h.ResponseWriter
}
