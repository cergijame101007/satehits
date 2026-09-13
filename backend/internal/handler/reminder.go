package handler

import (
	"log"
	"net/http"

	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
)

// PendingReminderHandler は内部運用向けの pending リマインド enqueue エンドポイント（UC-S04）。
// 認証はアプリ側では行わない（Cloud Run IAM。ADR-015 参照）
type PendingReminderHandler struct {
	enqueue *reservationusecase.EnqueuePendingRemindersUseCase
}

// NewPendingReminderHandler は PendingReminderHandler を生成する
func NewPendingReminderHandler(enqueue *reservationusecase.EnqueuePendingRemindersUseCase) *PendingReminderHandler {
	return &PendingReminderHandler{enqueue: enqueue}
}

// HandlePendingReminders は POST /internal/reminders/pending
func (h *PendingReminderHandler) HandlePendingReminders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondWithError(w, http.StatusMethodNotAllowed, InvalidRequestCode, "メソッドが許可されていません", nil)
		return
	}

	result, err := h.enqueue.Execute(r.Context())
	if err != nil {
		// 予約の個人情報はエラーに含まれない（ID と mail_type のみ）
		log.Printf("ERROR: enqueue pending reminders: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "リマインドの登録に失敗しました", nil)
		return
	}
	respondWithJSON(w, http.StatusOK, result)
}
