package handler

import (
	"errors"
	"log"
	"net/http"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/schedule"
	"github.com/cergijame101007/satehits/internal/domain"
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
	if errors.Is(err, domain.ErrScheduleNotStored) {
		respondWithError(w, http.StatusNotFound, NotFoundCode, "この日は店舗定例のため削除できる設定がありません", nil)
		return true
	}
	log.Printf("%s: %v", failLog, err)
	respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
	return true
}
