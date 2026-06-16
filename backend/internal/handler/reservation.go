package handler

import (
	"encoding/json"
	"errors"
	"log"
	"mime"
	"net/http"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/privacy"
)

// 予約作成 POST のボディ上限（64KB）
const maxCreateReservationBodyBytes = 64 << 10

// ReservationRequest は予約作成リクエストのDTO
type ReservationRequest struct {
	Name           string        `json:"name"`
	People         int           `json:"people"`
	VisitDate      datetime.Date `json:"visit_date"`
	VisitTime      datetime.Time `json:"visit_time"`
	Phone          string        `json:"phone"`
	Email          string        `json:"email"`
	Note           string        `json:"note"`
	TurnstileToken string        `json:"turnstile_token"`
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
//
// TODO: GET は認証なしで全件返却しており、OpenAPI の顧客向け /reservations（POST のみ）とも齟齬がある
// 本来は次のいずれかに揃える:
// - 一覧は GET /api/v1/admin/reservations 等へ移し JWT 必須にする（設計の管理者向け一覧と一致）
// - 顧客向けに一覧が必要なら、公開範囲・クエリ・認証を OpenAPI と設計書に書いたうえで実装する
// - 開発専用ならルート分離やビルドタグで本番から除外する
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
	if err != nil || mediaType != mediaTypeJSON {
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
		Name:           request.Name,
		People:         request.People,
		VisitDate:      request.VisitDate,
		VisitTime:      request.VisitTime,
		Phone:          request.Phone,
		Email:          request.Email,
		Note:           request.Note,
		TurnstileToken: request.TurnstileToken,
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
		if errors.Is(err, domain.ErrCaptchaFailed) {
			respondWithError(w, http.StatusBadRequest, CaptchaFailedCode, "認証に失敗しました。もう一度お試しください", nil)
			return
		}
		if errors.Is(err, domain.ErrReservationConflict) {
			respondWithError(w, http.StatusConflict, ReservationConflictCode, "同じ日時の予約が既に登録されています", nil)
			return
		}
		if errors.Is(err, domain.ErrCapacityExceeded) {
			respondWithError(w, http.StatusConflict, CapacityExceededCode, "この日の予約可能数を超えています", nil)
			return
		}
		log.Printf("Failed to create reservation: %v", err)
		respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
		return
	}

	log.Printf("Saved Reservation id=%s name=%s phone=%s email=%s people=%d visitDate=%s visitTime=%s status=%s source=%s",
		created.ID, privacy.MaskName(created.Name), privacy.MaskPhone(created.Phone), privacy.MaskEmail(created.Email),
		created.People, created.VisitDate, created.VisitTime, created.Status, created.Source)

	respondWithJSON(w, http.StatusCreated, created)
}
