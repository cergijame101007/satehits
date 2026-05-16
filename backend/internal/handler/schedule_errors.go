package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/cergijame101007/satehits/internal/application/usecase/schedule"
)

// writeScheduleUsecaseError — スケジュール系ユースケースエラーを HTTP レスポンスへ変換（処理済みなら true）
func writeScheduleUsecaseError(w http.ResponseWriter, err error, failLog string) bool {
	if err == nil {
		return false
	}
	var vErr *usecase.ValidationError
	if errors.As(err, &vErr) {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", violationsToDetails(vErr.Violations))
		return true
	}
	log.Printf("%s: %v", failLog, err)
	respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
	return true
}
