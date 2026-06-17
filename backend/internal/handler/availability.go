package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

// AvailabilityResponse は OpenAPI AvailabilityResponse に対応する DTO
type AvailabilityResponse struct {
	Date             datetime.Date `json:"date"`
	Capacity         int           `json:"capacity"`
	Reserved         int           `json:"reserved"`
	Available        int           `json:"available"`
	ScheduleType     *string       `json:"schedule_type"`
	EventName        string        `json:"event_name,omitempty"`
	EventDescription string        `json:"event_description,omitempty"`
	IsHoliday        bool          `json:"is_holiday"`
}

// AvailabilityListResponse は月間空き状況一覧の DTO
type AvailabilityListResponse struct {
	Year           int                    `json:"year"`
	Month          int                    `json:"month"`
	Availabilities []AvailabilityResponse `json:"availabilities"`
}

// AvailabilityHandler は残り食数取得の HTTP ハンドラ
type AvailabilityHandler struct {
	getAvailability *usecase.GetAvailabilityUseCase
	path            string
}

// NewAvailabilityHandler は AvailabilityHandler のインスタンスを作成する
func NewAvailabilityHandler(getAvailability *usecase.GetAvailabilityUseCase, path string) *AvailabilityHandler {
	return &AvailabilityHandler{getAvailability: getAvailability, path: path}
}

// HandleAvailability は GET /reservations/availability を処理する
func (h *AvailabilityHandler) HandleAvailability(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.path {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	yearStr := strings.TrimSpace(r.URL.Query().Get("year"))
	monthStr := strings.TrimSpace(r.URL.Query().Get("month"))
	dateStr := strings.TrimSpace(r.URL.Query().Get("date"))

	hasYearMonth := yearStr != "" || monthStr != ""
	hasDate := dateStr != ""

	if hasYearMonth && hasDate {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", []ErrorDetail{
			{Field: "date", Message: "date と year/month は同時に指定できません"},
		})
		return
	}

	switch {
	case hasYearMonth:
		h.handleAvailabilityMonth(w, r)
	case hasDate:
		h.handleAvailabilityDate(w, r, dateStr)
	default:
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", []ErrorDetail{
			{Field: "date", Message: "日付または年月を指定してください"},
		})
	}
}

func (h *AvailabilityHandler) handleAvailabilityDate(w http.ResponseWriter, r *http.Request, dateStr string) {
	date, err := datetime.ParseDate(dateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", []ErrorDetail{
			{Field: "date", Message: "日付の形式が正しくありません"},
		})
		return
	}

	avail, err := h.getAvailability.Execute(r.Context(), date)
	if writeAvailabilityUsecaseError(w, err, "Failed to get availability") {
		return
	}

	respondWithJSON(w, http.StatusOK, toAvailabilityResponse(*avail))
}

func (h *AvailabilityHandler) handleAvailabilityMonth(w http.ResponseWriter, r *http.Request) {
	year, month, details := parseYearMonthQuery(r)
	if len(details) > 0 {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", details)
		return
	}

	result, err := h.getAvailability.ExecuteMonth(r.Context(), year, month)
	if writeAvailabilityUsecaseError(w, err, "Failed to list availability") {
		return
	}

	resp := AvailabilityListResponse{
		Year:           result.Year,
		Month:          result.Month,
		Availabilities: make([]AvailabilityResponse, len(result.Availabilities)),
	}
	for i, a := range result.Availabilities {
		resp.Availabilities[i] = toAvailabilityResponse(a)
	}
	respondWithJSON(w, http.StatusOK, resp)
}

func toAvailabilityResponse(a service.Availability) AvailabilityResponse {
	resp := AvailabilityResponse{
		Date:             a.Date,
		Capacity:         a.Capacity,
		Reserved:         a.Reserved,
		Available:        a.Available,
		EventName:        a.EventName,
		EventDescription: a.EventDescription,
		IsHoliday:        a.IsHoliday,
	}
	if a.IsHoliday && a.ScheduleType != domain.ScheduleTypeExternalEvent {
		resp.ScheduleType = nil
		return resp
	}
	st := a.ScheduleType
	resp.ScheduleType = &st
	return resp
}

func writeAvailabilityUsecaseError(w http.ResponseWriter, err error, failLog string) bool {
	if err == nil {
		return false
	}
	var vErr *usecase.ValidationError
	if errors.As(err, &vErr) {
		details := make([]ErrorDetail, len(vErr.Violations))
		for i, v := range vErr.Violations {
			details[i] = ErrorDetail{Field: v.Field, Message: v.Message}
		}
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", details)
		return true
	}
	log.Printf("%s: %v", failLog, err)
	respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
	return true
}
