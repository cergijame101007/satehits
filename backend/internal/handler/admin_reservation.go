package handler

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"
	"strings"

	"github.com/google/uuid"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/privacy"
)

const maxAdminReservationBodyBytes = 64 << 10

// AdminReservationRequest は管理者手動予約登録リクエスト DTO
type AdminReservationRequest struct {
	Name      string        `json:"name"`
	People    int           `json:"people"`
	VisitDate datetime.Date `json:"visit_date"`
	VisitTime datetime.Time `json:"visit_time"`
	Phone     string        `json:"phone"`
	Email     string        `json:"email"`
	Note      string        `json:"note"`
	Source    string        `json:"source"`
	Status    string        `json:"status"`
}

// UpdateStatusRequest は予約ステータス更新リクエスト DTO
type UpdateStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ReservationResponse は予約レスポンス DTO（OpenAPI ReservationResponse）
type ReservationResponse struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	People    int           `json:"people"`
	VisitDate datetime.Date `json:"visit_date"`
	VisitTime datetime.Time `json:"visit_time"`
	Phone     string        `json:"phone"`
	Email     string        `json:"email"`
	Note      string        `json:"note"`
	Status    string        `json:"status"`
	Source    string        `json:"source"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// ReservationListResponse は予約一覧レスポンス DTO
type ReservationListResponse struct {
	Reservations []ReservationResponse `json:"reservations"`
	Total        int                   `json:"total"`
}

// UpdateStatusResponse はステータス更新レスポンス DTO
type UpdateStatusResponse struct {
	ID        uuid.UUID `json:"id"`
	Status    string    `json:"status"`
	UpdatedAt string    `json:"updated_at"`
}

// AdminReservationHandler は管理者向け予約 HTTP ハンドラ
type AdminReservationHandler struct {
	listReservations      *usecase.ListReservationsUseCase
	createAdmin           *usecase.CreateAdminReservationUseCase
	updateStatus          *usecase.UpdateReservationStatusUseCase
	adminReservationsPath string
}

// NewAdminReservationHandler は AdminReservationHandler を生成する
func NewAdminReservationHandler(
	listReservations *usecase.ListReservationsUseCase,
	createAdmin *usecase.CreateAdminReservationUseCase,
	updateStatus *usecase.UpdateReservationStatusUseCase,
	adminReservationsPath string,
) *AdminReservationHandler {
	return &AdminReservationHandler{
		listReservations:      listReservations,
		createAdmin:           createAdmin,
		updateStatus:          updateStatus,
		adminReservationsPath: adminReservationsPath,
	}
}

// HandleAdminReservations は /admin/reservations および /admin/reservations/{id}/status をルーティングする
func (h *AdminReservationHandler) HandleAdminReservations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == h.adminReservationsPath {
		switch r.Method {
		case http.MethodGet:
			h.handleList(w, r)
		case http.MethodPost:
			h.handleCreate(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	prefix := h.adminReservationsPath + "/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}

	remainder := strings.TrimPrefix(path, prefix)
	if remainder == "" {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(remainder, "/")
	if len(parts) == 2 && parts[1] == "status" {
		if r.Method != http.MethodPatch {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleUpdateStatus(w, r, parts[0])
		return
	}

	http.NotFound(w, r)
}

func (h *AdminReservationHandler) handleList(w http.ResponseWriter, r *http.Request) {
	var dateFilter *datetime.Date
	dateStr := strings.TrimSpace(r.URL.Query().Get("date"))
	if dateStr != "" {
		date, err := datetime.ParseDate(dateStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", []ErrorDetail{
				{Field: "date", Message: "日付の形式が正しくありません"},
			})
			return
		}
		dateFilter = &date
	}

	result, err := h.listReservations.Execute(r.Context(), usecase.ListReservationsQuery{
		Date:   dateFilter,
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Source: strings.TrimSpace(r.URL.Query().Get("source")),
	})
	if writeReservationUsecaseError(w, err, "Failed to list reservations") {
		return
	}

	responses := make([]ReservationResponse, len(result.Reservations))
	for i, item := range result.Reservations {
		responses[i] = toReservationResponse(item)
	}
	respondWithJSON(w, http.StatusOK, ReservationListResponse{
		Reservations: responses,
		Total:        result.Total,
	})
}

func (h *AdminReservationHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != mediaTypeJSON {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAdminReservationBodyBytes)

	var request AdminReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	created, err := h.createAdmin.Execute(r.Context(), usecase.CreateAdminReservationCommand{
		Name:      request.Name,
		People:    request.People,
		VisitDate: request.VisitDate,
		VisitTime: request.VisitTime,
		Phone:     request.Phone,
		Email:     request.Email,
		Note:      request.Note,
		Source:    request.Source,
		Status:    request.Status,
	})
	if writeReservationUsecaseError(w, err, "Failed to create admin reservation") {
		return
	}

	log.Printf("Saved Admin Reservation id=%s name=%s phone=%s email=%s people=%d visitDate=%s visitTime=%s status=%s source=%s",
		created.ID, privacy.MaskName(created.Name), privacy.MaskPhone(created.Phone), privacy.MaskEmail(created.Email),
		created.People, created.VisitDate, created.VisitTime, created.Status, created.Source)

	respondWithJSON(w, http.StatusCreated, toReservationResponse(*created))
}

func (h *AdminReservationHandler) handleUpdateStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != mediaTypeJSON {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAdminReservationBodyBytes)

	var request UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	updated, err := h.updateStatus.Execute(r.Context(), usecase.UpdateReservationStatusCommand{
		ID:     id,
		Status: request.Status,
		Reason: request.Reason,
	})
	if writeReservationUsecaseError(w, err, "Failed to update reservation status") {
		return
	}

	log.Printf("Updated Reservation Status id=%s status=%s", updated.ID, updated.Status)

	respondWithJSON(w, http.StatusOK, UpdateStatusResponse{
		ID:        updated.ID,
		Status:    updated.Status,
		UpdatedAt: updated.UpdatedAt.Format(timeRFC3339),
	})
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func toReservationResponse(r domain.Reservation) ReservationResponse {
	return ReservationResponse{
		ID:        r.ID,
		Name:      r.Name,
		People:    r.People,
		VisitDate: r.VisitDate,
		VisitTime: r.VisitTime,
		Phone:     r.Phone,
		Email:     r.Email,
		Note:      r.Note,
		Status:    r.Status,
		Source:    r.Source,
		CreatedAt: r.CreatedAt.Format(timeRFC3339),
		UpdatedAt: r.UpdatedAt.Format(timeRFC3339),
	}
}

func writeReservationUsecaseError(w http.ResponseWriter, err error, logPrefix string) bool {
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
	if errors.Is(err, domain.ErrReservationConflict) {
		respondWithError(w, http.StatusConflict, ReservationConflictCode, "同じ日時の予約が既に登録されています", nil)
		return true
	}
	if errors.Is(err, domain.ErrReservationNotFound) {
		respondWithError(w, http.StatusNotFound, NotFoundCode, "予約が見つかりません", nil)
		return true
	}

	log.Printf("%s: %v", logPrefix, err)
	respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
	return true
}
