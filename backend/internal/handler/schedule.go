package handler

import (
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/schedule"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// スケジュール設定 POST のボディ上限（64KB）
const maxSetScheduleBodyBytes = 64 << 10

// defaultSetScheduleCapacity は OpenAPI SetScheduleRequest.capacity の省略時デフォルト
const defaultSetScheduleCapacity = 10

// SetScheduleRequest は日別スケジュール設定リクエストの DTO（OpenAPI SetScheduleRequest）
type SetScheduleRequest struct {
	Date             datetime.Date `json:"date"`
	ScheduleType     string        `json:"schedule_type"`
	Capacity         *int          `json:"capacity"`
	EventName        string        `json:"event_name"`
	EventDescription string        `json:"event_description"`
	OpenTime         datetime.Time `json:"open_time"`
	LastOrderTime    datetime.Time `json:"last_order_time"`
	CloseTime        datetime.Time `json:"close_time"`
}

// ScheduleResponse は日別スケジュールのレスポンス DTO（OpenAPI ScheduleResponse）
// NOTE: 営業時刻に omitempty は付けない（datetime.Time は構造体のため omitempty が効かず、ゼロ値は MarshalJSON で null になる）
type ScheduleResponse struct {
	Date             datetime.Date `json:"date"`
	ScheduleType     string        `json:"schedule_type"`
	Capacity         int           `json:"capacity"`
	EventName        string        `json:"event_name,omitempty"`
	EventDescription string        `json:"event_description,omitempty"`
	OpenTime         datetime.Time `json:"open_time"`
	LastOrderTime    datetime.Time `json:"last_order_time"`
	CloseTime        datetime.Time `json:"close_time"`
	IsDefault        bool          `json:"is_default"`
	CreatedAt        *time.Time    `json:"created_at,omitempty"`
	UpdatedAt        *time.Time    `json:"updated_at,omitempty"`
}

// ScheduleListResponse は月間一覧のレスポンス DTO（OpenAPI ScheduleListResponse）
type ScheduleListResponse struct {
	Year      int                `json:"year"`
	Month     int                `json:"month"`
	Schedules []ScheduleResponse `json:"schedules"`
}

// ScheduleHandler — 管理者向けスケジュール HTTP ハンドラ
//
// TODO: 管理者 JWT 必須（OpenAPI BearerAuth）。認証は main のミドルウェアで行う
type ScheduleHandler struct {
	setSchedule   *usecase.SetScheduleUseCase
	listSchedules *usecase.ListSchedulesUseCase
	getSchedule   *usecase.GetScheduleUseCase
	schedulesPath string
}

// NewScheduleHandler はScheduleHandlerのインスタンスを作成する
// schedulesPath は net/http の ServeMux に登録する完全パス（例: /api/v1/admin/schedules）と一致させること
func NewScheduleHandler(
	setSchedule *usecase.SetScheduleUseCase,
	listSchedules *usecase.ListSchedulesUseCase,
	getSchedule *usecase.GetScheduleUseCase,
	schedulesPath string,
) *ScheduleHandler {
	return &ScheduleHandler{
		setSchedule:   setSchedule,
		listSchedules: listSchedules,
		getSchedule:   getSchedule,
		schedulesPath: schedulesPath,
	}
}

// HandleSchedules — /admin/schedules および /admin/schedules/{date} のルーティング
func (h *ScheduleHandler) HandleSchedules(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == h.schedulesPath {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleSet(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	prefix := h.schedulesPath + "/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}
	datePart := strings.TrimPrefix(path, prefix)
	if datePart == "" || strings.Contains(datePart, "/") {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	h.handleGetByDate(w, r, datePart)
}

func (h *ScheduleHandler) handleList(w http.ResponseWriter, r *http.Request) {
	year, month, details := parseYearMonthQuery(r)
	if len(details) > 0 {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", details)
		return
	}

	result, err := h.listSchedules.Execute(r.Context(), year, month)
	if writeScheduleUsecaseError(w, err, "Failed to list schedules") {
		return
	}

	schedules := make([]ScheduleResponse, len(result.Schedules))
	for i, item := range result.Schedules {
		schedules[i] = toScheduleResponse(item.Schedule, item.IsDefault)
	}
	respondWithJSON(w, http.StatusOK, ScheduleListResponse{
		Year:      result.Year,
		Month:     result.Month,
		Schedules: schedules,
	})
}

func (h *ScheduleHandler) handleGetByDate(w http.ResponseWriter, r *http.Request, dateStr string) {
	date, err := datetime.ParseDate(dateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	result, err := h.getSchedule.Execute(r.Context(), date)
	if writeScheduleUsecaseError(w, err, "Failed to get schedule") {
		return
	}

	respondWithJSON(w, http.StatusOK, toScheduleResponse(result.Schedule, result.IsDefault))
}

// handleSet — 日別スケジュール Upsert（新規 201 / 更新 200）
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
		Capacity:         resolveSetScheduleCapacity(request.Capacity),
		EventName:        request.EventName,
		EventDescription: request.EventDescription,
		OpenTime:         request.OpenTime,
		LastOrderTime:    request.LastOrderTime,
		CloseTime:        request.CloseTime,
	})
	if writeScheduleUsecaseError(w, err, "Failed to set schedule") {
		return
	}

	s := result.Schedule
	log.Printf("Saved Schedule date=%s scheduleType=%s capacity=%d inserted=%v openTime=%s lastOrderTime=%s closeTime=%s",
		s.Date, s.ScheduleType, s.Capacity, result.Inserted, s.OpenTime, s.LastOrderTime, s.CloseTime)

	status := http.StatusOK
	if result.Inserted {
		status = http.StatusCreated
	}
	respondWithJSON(w, status, toScheduleResponse(s, false))
}

func resolveSetScheduleCapacity(capacity *int) int {
	if capacity != nil {
		return *capacity
	}
	return defaultSetScheduleCapacity
}

func parseYearMonthQuery(r *http.Request) (year, month int, details []ErrorDetail) {
	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	monthStr := strings.TrimSpace(r.URL.Query().Get("month"))

	if yearStr == "" {
		details = append(details, ErrorDetail{Field: "year", Message: "年は必須です"})
	}
	if monthStr == "" {
		details = append(details, ErrorDetail{Field: "month", Message: "月は必須です"})
	}
	if len(details) > 0 {
		return 0, 0, details
	}

	var err error
	year, err = strconv.Atoi(yearStr)
	if err != nil {
		details = append(details, ErrorDetail{Field: "year", Message: "年は数値で指定してください"})
	}
	month, err = strconv.Atoi(monthStr)
	if err != nil {
		details = append(details, ErrorDetail{Field: "month", Message: "月は数値で指定してください"})
	}
	return year, month, details
}

func toScheduleResponse(s domain.Schedule, isDefault bool) ScheduleResponse {
	resp := ScheduleResponse{
		Date:             s.Date,
		ScheduleType:     s.ScheduleType,
		Capacity:         s.Capacity,
		EventName:        s.EventName,
		EventDescription: s.EventDescription,
		OpenTime:         s.OpenTime,
		LastOrderTime:    s.LastOrderTime,
		CloseTime:        s.CloseTime,
		IsDefault:        isDefault,
	}
	if !isDefault {
		if !s.CreatedAt.IsZero() {
			t := s.CreatedAt
			resp.CreatedAt = &t
		}
		if !s.UpdatedAt.IsZero() {
			t := s.UpdatedAt
			resp.UpdatedAt = &t
		}
	}
	return resp
}

func violationsToDetails(violations []usecase.FieldViolation) []ErrorDetail {
	details := make([]ErrorDetail, len(violations))
	for i, v := range violations {
		details[i] = ErrorDetail{Field: v.Field, Message: v.Message}
	}
	return details
}
