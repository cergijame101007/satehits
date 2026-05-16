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

// スケジュール作成 POST のボディ上限（64KB）
const maxCreateScheduleBodyBytes = 64 << 10

// ScheduleRequest はスケジュール作成リクエストのDTO
type ScheduleRequest struct {
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
	repo              domain.ScheduleRepository
	createSchedule *usecase.CreateScheduleUseCase
	schedulesPath  string
}

// NewScheduleHandler はScheduleHandlerのインスタンスを作成する
// schedulesPath は net/http の ServeMux に登録する完全パス（例: /api/v1/admin/schedules）と一致させること
func NewScheduleHandler(
	repo domain.ScheduleRepository,
	createSchedule *usecase.CreateScheduleUseCase,
	schedulesPath string,
) *ScheduleHandler {
	return &ScheduleHandler{
		repo:              repo,
		createSchedule: createSchedule,
		schedulesPath:  schedulesPath,
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
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreate はスケジュールを作成する
func (h *ScheduleHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCreateScheduleBodyBytes)

	var request ScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	created, err := h.createSchedule.Execute(r.Context(), usecase.CreateScheduleCommand{
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
		log.Printf("Failed to create schedule: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	log.Printf("Saved Schedule date=%s scheduleType=%s capacity=%d eventName=%s eventDescription=%s openTime=%s lastOrderTime=%s closeTime=%s",
		created.Date, created.ScheduleType, created.Capacity, created.EventName, created.EventDescription, created.OpenTime, created.LastOrderTime, created.CloseTime)

	respondWithJSON(w, http.StatusCreated, created)
}