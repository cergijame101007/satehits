package handler

import (
	"errors"
	"log"
	"net/http"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/supplier"
	"github.com/cergijame101007/satehits/internal/domain"
)

// PublicSupplierResponse は公開取引先レスポンス DTO（OpenAPI PublicSupplier）
type PublicSupplierResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	InstagramURL *string `json:"instagram_url"`
	ImageURL     *string `json:"image_url"`
}

// PublicSupplierListResponse は公開取引先一覧レスポンス DTO（OpenAPI SupplierListResponse）
type PublicSupplierListResponse struct {
	Suppliers []PublicSupplierResponse `json:"suppliers"`
}

// SupplierResponse は管理者向け取引先レスポンス DTO（OpenAPI SupplierResponse）
type SupplierResponse struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	InstagramURL *string `json:"instagram_url"`
	ImageURL     *string `json:"image_url"`
	DisplayOrder int     `json:"display_order"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// AdminSupplierListResponse は管理者向け取引先一覧レスポンス DTO（OpenAPI AdminSupplierListResponse）
type AdminSupplierListResponse struct {
	Suppliers []SupplierResponse `json:"suppliers"`
}

// PublicSupplierHandler は公開取引先一覧 HTTP ハンドラ
type PublicSupplierHandler struct {
	listSuppliers       *usecase.ListSuppliersUseCase
	publicSuppliersPath string
}

// NewPublicSupplierHandler は PublicSupplierHandler を生成する
func NewPublicSupplierHandler(listSuppliers *usecase.ListSuppliersUseCase, publicSuppliersPath string) *PublicSupplierHandler {
	return &PublicSupplierHandler{
		listSuppliers:       listSuppliers,
		publicSuppliersPath: publicSuppliersPath,
	}
}

// HandlePublicSuppliers は GET /api/v1/suppliers をルーティングする（is_active=true のみ）
func (h *PublicSupplierHandler) HandlePublicSuppliers(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.publicSuppliersPath {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// 公開一覧は常に activeOnly=true 固定。クエリパラメータでの絞り込みは受け付けない。
	suppliers, err := h.listSuppliers.Execute(r.Context(), true)
	if writeSupplierUsecaseError(w, err, "Failed to list public suppliers") {
		return
	}

	responses := make([]PublicSupplierResponse, len(suppliers))
	for i, s := range suppliers {
		responses[i] = toPublicSupplierResponse(s)
	}
	respondWithJSON(w, http.StatusOK, PublicSupplierListResponse{Suppliers: responses})
}

func toPublicSupplierResponse(s domain.Supplier) PublicSupplierResponse {
	return PublicSupplierResponse{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		InstagramURL: s.InstagramURL,
		ImageURL:     s.ImageURL,
	}
}

func toSupplierResponse(s domain.Supplier) SupplierResponse {
	return SupplierResponse{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		InstagramURL: s.InstagramURL,
		ImageURL:     s.ImageURL,
		DisplayOrder: s.DisplayOrder,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt.Format(timeRFC3339),
		UpdatedAt:    s.UpdatedAt.Format(timeRFC3339),
	}
}

// writeSupplierUsecaseError は取引先ユースケースのエラーを HTTP レスポンスに変換する。
// エラーを処理した場合 true を返す。
func writeSupplierUsecaseError(w http.ResponseWriter, err error, logPrefix string) bool {
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
	if errors.Is(err, domain.ErrSupplierNotFound) {
		respondWithError(w, http.StatusNotFound, NotFoundCode, "取引先が見つかりません", nil)
		return true
	}

	log.Printf("%s: %v", logPrefix, err)
	respondWithError(w, http.StatusInternalServerError, InternalErrorCode, "サーバー内部でエラーが発生しました", nil)
	return true
}
