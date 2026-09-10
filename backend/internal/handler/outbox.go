package handler

import (
	"net/http"

	inframail "github.com/cergijame101007/satehits/internal/infrastructure/mail"
)

// OutboxHandler は内部運用向け Outbox flush エンドポイント
// 認証はアプリ側では行わない（Cloud Run IAM。ADR-008 参照）
type OutboxHandler struct {
	dispatcher *inframail.Dispatcher
}

// NewOutboxHandler は OutboxHandler を生成する
func NewOutboxHandler(dispatcher *inframail.Dispatcher) *OutboxHandler {
	return &OutboxHandler{dispatcher: dispatcher}
}

// HandleFlush は POST /internal/outbox/flush
func (h *OutboxHandler) HandleFlush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, InvalidRequestCode, "メソッドが許可されていません", nil)
		return
	}

	stats, err := h.dispatcher.ProcessPending(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "Outbox の処理に失敗しました", nil)
		return
	}
	respondWithJSON(w, http.StatusOK, stats)
}
