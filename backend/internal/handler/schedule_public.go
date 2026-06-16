package handler

import (
	"net/http"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/domain/service"
)

// DaySchedule は顧客向け月間スケジュールの1日分（OpenAPI DaySchedule）
type DaySchedule struct {
	Date             string  `json:"date"`
	ScheduleType     *string `json:"schedule_type"`
	Capacity         int     `json:"capacity"`
	Available        int     `json:"available"`
	EventName        string  `json:"event_name,omitempty"`
	EventDescription string  `json:"event_description,omitempty"`
	IsHoliday        bool    `json:"is_holiday"`
}

// MonthlyScheduleResponse は顧客向け月間スケジュール一覧（OpenAPI MonthlyScheduleResponse）
type MonthlyScheduleResponse struct {
	Year      int           `json:"year"`
	Month     int           `json:"month"`
	Schedules []DaySchedule `json:"schedules"`
}

// PublicScheduleHandler は顧客向け公開スケジュール HTTP ハンドラ
type PublicScheduleHandler struct {
	getAvailability *usecase.GetAvailabilityUseCase
	path            string
}

// NewPublicScheduleHandler は PublicScheduleHandler のインスタンスを作成する
func NewPublicScheduleHandler(getAvailability *usecase.GetAvailabilityUseCase, path string) *PublicScheduleHandler {
	return &PublicScheduleHandler{getAvailability: getAvailability, path: path}
}

// HandlePublicSchedules は GET /schedules を処理する
func (h *PublicScheduleHandler) HandlePublicSchedules(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.path {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	year, month, details := parseYearMonthQuery(r)
	if len(details) > 0 {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", details)
		return
	}

	result, err := h.getAvailability.ExecuteMonth(r.Context(), year, month)
	if writeAvailabilityUsecaseError(w, err, "Failed to list public schedules") {
		return
	}

	schedules := make([]DaySchedule, len(result.Availabilities))
	for i, a := range result.Availabilities {
		schedules[i] = toDaySchedule(a)
	}

	respondWithJSON(w, http.StatusOK, MonthlyScheduleResponse{
		Year:      result.Year,
		Month:     result.Month,
		Schedules: schedules,
	})
}

func toDaySchedule(a service.Availability) DaySchedule {
	resp := DaySchedule{
		Date:             a.Date.String(),
		Capacity:         a.Capacity,
		Available:        a.Available,
		EventName:        a.EventName,
		EventDescription: a.EventDescription,
		IsHoliday:        a.IsHoliday,
	}
	if a.IsHoliday {
		resp.ScheduleType = nil
		return resp
	}
	st := a.ScheduleType
	resp.ScheduleType = &st
	return resp
}
