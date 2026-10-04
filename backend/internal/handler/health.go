package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// healthCheckTimeout は DB 接続確認に使う時間の上限（docs/monitoring.md §4.4）
const healthCheckTimeout = 2 * time.Second

const (
	healthStatusOK    = "ok"
	healthStatusError = "error"
)

// Pinger は DB 接続確認の最小インターフェース（*sql.DB が満たす）
type Pinger interface {
	PingContext(ctx context.Context) error
}

// healthResponse は GET /healthz のレスポンス
type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// HealthHandler は外形監視向けの死活確認エンドポイント
// 確認するのは自プロセスと DB だけ。外部 API の死活は混ぜない（外部障害で Down 扱いにしないため）
type HealthHandler struct {
	db      Pinger
	timeout time.Duration
}

// NewHealthHandler は HealthHandler を生成する
func NewHealthHandler(db Pinger) *HealthHandler {
	return &HealthHandler{db: db, timeout: healthCheckTimeout}
}

// HandleHealth は GET /healthz。DB に ping できれば 200、できなければ 503
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondMethodNotAllowed(w, http.MethodGet)
		return
	}
	w.Header().Set("Cache-Control", "no-store")

	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		// ERROR にしない。healthz の失敗は外形監視が通知する（ログベースアラートと二重にしない）
		slog.Warn("health check failed", "component", "database", "err", err)
		respondWithJSON(w, http.StatusServiceUnavailable, healthResponse{Status: healthStatusError, Database: healthStatusError})
		return
	}
	respondWithJSON(w, http.StatusOK, healthResponse{Status: healthStatusOK, Database: healthStatusOK})
}
