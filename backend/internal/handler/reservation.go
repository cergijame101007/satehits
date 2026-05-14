package handler

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"

	"github.com/cergijame101007/satehits/internal/application/usecase"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/privacy"
)

// 予約作成 POST のボディ上限（64KB）
const maxCreateReservationBodyBytes = 64 << 10

// ReservationRequest は予約作成リクエストのDTO
type ReservationRequest struct {
	Name      string        `json:"name"`
	People    int           `json:"people"`
	VisitDate datetime.Date `json:"visit_date"`
	VisitTime datetime.Time `json:"visit_time"`
	Phone     string        `json:"phone"`
	Email     string        `json:"email"`
	Note      string        `json:"note"`
	// NOTE: Status/Sourceはサーバー側で管理するのでクライアントには返さない
	// 管理者用 API では Status/Source を指定する
}

// ReservationHandler は予約に関するHTTPハンドラ
type ReservationHandler struct {
	repo              domain.ReservationRepository
	createReservation *usecase.CreateReservationUseCase
	reservationsPath  string
}

// NewReservationHandler はReservationHandlerのインスタンスを作成する
// reservationsPath は net/http の ServeMux に登録する完全パス（例: /api/v1/reservations）と一致させること
func NewReservationHandler(
	repo domain.ReservationRepository,
	createReservation *usecase.CreateReservationUseCase,
	reservationsPath string,
) *ReservationHandler {
	return &ReservationHandler{
		repo:              repo,
		createReservation: createReservation,
		reservationsPath:  reservationsPath,
	}
}

// HandleReservations はGET/POSTリクエストをルーティングする
func (h *ReservationHandler) HandleReservations(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.reservationsPath {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleList は予約一覧を取得する
func (h *ReservationHandler) handleList(w http.ResponseWriter, r *http.Request) {
	reservations, err := h.repo.GetAll(r.Context())
	if err != nil {
		log.Printf("Failed to get reservations: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	if reservations == nil {
		reservations = []domain.Reservation{}
	}

	respondWithJSON(w, http.StatusOK, reservations)
}

// handleCreate は予約を作成する
func (h *ReservationHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCreateReservationBodyBytes)

	var request ReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	created, err := h.createReservation.Execute(r.Context(), usecase.CreateReservationCommand{
		Name:      request.Name,
		People:    request.People,
		VisitDate: request.VisitDate,
		VisitTime: request.VisitTime,
		Phone:     request.Phone,
		Email:     request.Email,
		Note:      request.Note,
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
		log.Printf("Failed to create reservation: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	log.Printf("Saved Reservation id=%d name=%s phone=%s email=%s people=%d visitDate=%s visitTime=%s status=%s source=%s",
		created.ID, privacy.MaskName(created.Name), privacy.MaskPhone(created.Phone), privacy.MaskEmail(created.Email),
		created.People, created.VisitDate, created.VisitTime, created.Status, created.Source)

	respondWithJSON(w, http.StatusCreated, created)
}
