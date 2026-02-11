package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/cergijame101007/satehits/internal/domain"
)

// ReservationRequest は予約作成リクエストのDTO
type ReservationRequest struct {
	Name   string `json:"name"`
	People int    `json:"people"`
}

// ReservationHandler は予約に関するHTTPハンドラ
type ReservationHandler struct {
	repo domain.ReservationRepository
}

// NewReservationHandler はReservationHandlerのインスタンスを作成する
func NewReservationHandler(repo domain.ReservationRepository) *ReservationHandler {
	return &ReservationHandler{repo: repo}
}

// HandleReservations はGET/POSTリクエストをルーティングする
func (h *ReservationHandler) HandleReservations(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/reservations" {
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
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if reservations == nil {
		reservations = []domain.Reservation{}
	}

	respondWithJSON(w, http.StatusOK, reservations)
}

// handleCreate は予約を作成する
func (h *ReservationHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var request ReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.Name == "" || request.People <= 0 {
		http.Error(w, "Name and valid people count required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(r.Context(), request.Name, request.People); err != nil {
		log.Printf("Failed to create reservation: %v", err)
		http.Error(w, "Failed to create reservation", http.StatusInternalServerError)
		return
	}

	log.Printf("Saved Reservation: Name=%s, People=%d", request.Name, request.People)

	respondWithJSON(w, http.StatusCreated, JSONResponse{
		Message: "Reservation created",
		Status:  "success",
	})
}
