package handler

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"

	"github.com/cergijame101007/satehits/internal/application/usecase/schedule"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// スケジュール設定 POST のボディ上限（64KB）
const maxSetScheduleBodyBytes = 64 << 10

// SetScheduleRequest は日別スケジュール設定リクエストの DTO（OpenAPI SetScheduleRequest）
type SetScheduleRequest struct {
	Date             datetime.Date `json:"date"`
	ScheduleType     string        `json:"schedule_type"`
	Capacity         int           `json:"capacity"`
	EventName        string        `json:"event_name"`
	EventDescription string        `json:"event_description"`
	OpenTime         datetime.Time `json:"open_time"`
	LastOrderTime    datetime.Time `json:"last_order_time"`
	CloseTime        datetime.Time `json:"close_time"`
}

// ScheduleHandler はスケジュールに関するHTTPハンドラ（管理者向け）
//
// TODO: スケジュール管理 API は管理者 JWT 認証必須（OpenAPI BearerAuth）
// 認証は main のルート登録時にミドルウェアで行い、本ハンドラは業務処理のみ担当する
type ScheduleHandler struct {
	repo          domain.ScheduleRepository
	setSchedule   *usecase.SetScheduleUseCase
	schedulesPath string
}

// NewScheduleHandler はScheduleHandlerのインスタンスを作成する
// schedulesPath は net/http の ServeMux に登録する完全パス（例: /api/v1/admin/schedules）と一致させること
func NewScheduleHandler(
	repo domain.ScheduleRepository,
	setSchedule *usecase.SetScheduleUseCase,
	schedulesPath string,
) *ScheduleHandler {
	return &ScheduleHandler{
		repo:          repo,
		setSchedule:   setSchedule,
		schedulesPath: schedulesPath,
	}
}

// HandleSchedules はGET/POSTリクエストをルーティングする
func (h *ScheduleHandler) HandleSchedules(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.schedulesPath {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPost:
		h.handleSet(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleSet は日別スケジュールを Upsert する（新規 201 / 更新 200）
func (h *ScheduleHandler) handleSet(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSetScheduleBodyBytes)

	var request SetScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	result, err := h.setSchedule.Execute(r.Context(), usecase.SetScheduleCommand{
		Date:             request.Date,
		ScheduleType:     request.ScheduleType,
		Capacity:         request.Capacity,
		EventName:        request.EventName,
		EventDescription: request.EventDescription,
		OpenTime:         request.OpenTime,
		LastOrderTime:    request.LastOrderTime,
		CloseTime:        request.CloseTime,
	})
	if err != nil {
		var vErr *usecase.ValidationError
		if errors.As(err, &vErr) {
			details := make([]ErrorDetail, len(vErr.Violations))
			for i, v := range vErr.Violations {
				details[i] = ErrorDetail{Field: v.Field, Message: v.Message}
			}
			respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", details)
			return
		}
		log.Printf("Failed to set schedule: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	s := result.Schedule
	log.Printf("Saved Schedule date=%s scheduleType=%s capacity=%d inserted=%v openTime=%s lastOrderTime=%s closeTime=%s",
		s.Date, s.ScheduleType, s.Capacity, result.Inserted, s.OpenTime, s.LastOrderTime, s.CloseTime)

	status := http.StatusOK
	if result.Inserted {
		status = http.StatusCreated
	}
	respondWithJSON(w, status, s)
}
