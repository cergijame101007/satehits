package handler

import (
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"

	usecase "github.com/cergijame101007/satehits/internal/application/usecase/supplier"
)

// 取引先 作成/更新 JSON ボディの上限（64KB）
const maxSupplierBodyBytes = 64 << 10

// 取引先画像アップロードの上限（5MB + multipart オーバーヘッド分の余裕）
const maxSupplierImageBodyBytes = 6 << 20

// CreateSupplierRequest は取引先作成リクエスト DTO（OpenAPI CreateSupplierRequest）
type CreateSupplierRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	InstagramURL *string `json:"instagram_url"`
	ImageURL     *string `json:"image_url"`
	DisplayOrder *int    `json:"display_order"`
	IsActive     *bool   `json:"is_active"`
}

// UpdateSupplierRequest は取引先更新リクエスト DTO（OpenAPI UpdateSupplierRequest、全項目任意）
type UpdateSupplierRequest struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	InstagramURL *string `json:"instagram_url"`
	ImageURL     *string `json:"image_url"`
	DisplayOrder *int    `json:"display_order"`
	IsActive     *bool   `json:"is_active"`
}

// ReorderSuppliersRequest は並び順更新リクエスト DTO
type ReorderSuppliersRequest struct {
	Order []int64 `json:"order"`
}

// SupplierImageResponse は画像アップロードのレスポンス DTO
type SupplierImageResponse struct {
	ImageURL string `json:"image_url"`
}

// AdminSupplierHandler は管理者向け取引先 HTTP ハンドラ
type AdminSupplierHandler struct {
	listSuppliers   *usecase.ListSuppliersUseCase
	getSupplier     *usecase.GetSupplierUseCase
	createSupplier  *usecase.CreateSupplierUseCase
	updateSupplier  *usecase.UpdateSupplierUseCase
	deleteSupplier  *usecase.DeleteSupplierUseCase
	reorderSupplier *usecase.ReorderSuppliersUseCase
	uploadImage     *usecase.UploadImageUseCase
	basePath        string
}

// NewAdminSupplierHandler は AdminSupplierHandler を生成する
func NewAdminSupplierHandler(
	listSuppliers *usecase.ListSuppliersUseCase,
	getSupplier *usecase.GetSupplierUseCase,
	createSupplier *usecase.CreateSupplierUseCase,
	updateSupplier *usecase.UpdateSupplierUseCase,
	deleteSupplier *usecase.DeleteSupplierUseCase,
	reorderSupplier *usecase.ReorderSuppliersUseCase,
	uploadImage *usecase.UploadImageUseCase,
	basePath string,
) *AdminSupplierHandler {
	return &AdminSupplierHandler{
		listSuppliers:   listSuppliers,
		getSupplier:     getSupplier,
		createSupplier:  createSupplier,
		updateSupplier:  updateSupplier,
		deleteSupplier:  deleteSupplier,
		reorderSupplier: reorderSupplier,
		uploadImage:     uploadImage,
		basePath:        basePath,
	}
}

// HandleAdminSuppliers は /admin/suppliers 配下をルーティングする。
//   - /admin/suppliers           GET（一覧）/ POST（作成）
//   - /admin/suppliers/order     PUT（並び順更新）
//   - /admin/suppliers/{id}      GET / PUT / DELETE
//   - /admin/suppliers/{id}/image POST（画像アップロード）
func (h *AdminSupplierHandler) HandleAdminSuppliers(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == h.basePath {
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

	prefix := h.basePath + "/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}
	remainder := strings.TrimPrefix(path, prefix)
	if remainder == "" {
		http.NotFound(w, r)
		return
	}

	// /admin/suppliers/order（"order" は数値 ID より先に判定する）
	if remainder == "order" {
		if r.Method != http.MethodPut {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleReorder(w, r)
		return
	}

	parts := strings.Split(remainder, "/")
	switch {
	case len(parts) == 1:
		h.handleByID(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "image":
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		h.handleUploadImage(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (h *AdminSupplierHandler) handleList(w http.ResponseWriter, r *http.Request) {
	suppliers, err := h.listSuppliers.Execute(r.Context(), false)
	if writeSupplierUsecaseError(w, err, "Failed to list suppliers") {
		return
	}
	responses := make([]SupplierResponse, len(suppliers))
	for i, s := range suppliers {
		responses[i] = toSupplierResponse(s)
	}
	respondWithJSON(w, http.StatusOK, AdminSupplierListResponse{Suppliers: responses})
}

func (h *AdminSupplierHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSupplierBodyBytes)

	var req CreateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	created, err := h.createSupplier.Execute(r.Context(), usecase.CreateSupplierCommand{
		Name:         req.Name,
		Description:  req.Description,
		InstagramURL: req.InstagramURL,
		ImageURL:     req.ImageURL,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	})
	if writeSupplierUsecaseError(w, err, "Failed to create supplier") {
		return
	}

	log.Printf("Created supplier id=%d name=%s", created.ID, created.Name)
	respondWithJSON(w, http.StatusCreated, toSupplierResponse(*created))
}

func (h *AdminSupplierHandler) handleByID(w http.ResponseWriter, r *http.Request, idStr string) {
	id, ok := parseSupplierID(w, idStr)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r, id)
	case http.MethodPut:
		h.handleUpdate(w, r, id)
	case http.MethodDelete:
		h.handleDelete(w, r, id)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminSupplierHandler) handleGet(w http.ResponseWriter, r *http.Request, id int64) {
	s, err := h.getSupplier.Execute(r.Context(), id)
	if writeSupplierUsecaseError(w, err, "Failed to get supplier") {
		return
	}
	respondWithJSON(w, http.StatusOK, toSupplierResponse(*s))
}

func (h *AdminSupplierHandler) handleUpdate(w http.ResponseWriter, r *http.Request, id int64) {
	if !requireJSON(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSupplierBodyBytes)

	var req UpdateSupplierRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	updated, err := h.updateSupplier.Execute(r.Context(), usecase.UpdateSupplierCommand{
		ID:           id,
		Name:         req.Name,
		Description:  req.Description,
		InstagramURL: req.InstagramURL,
		ImageURL:     req.ImageURL,
		DisplayOrder: req.DisplayOrder,
		IsActive:     req.IsActive,
	})
	if writeSupplierUsecaseError(w, err, "Failed to update supplier") {
		return
	}

	log.Printf("Updated supplier id=%d name=%s isActive=%v", updated.ID, updated.Name, updated.IsActive)
	respondWithJSON(w, http.StatusOK, toSupplierResponse(*updated))
}

func (h *AdminSupplierHandler) handleDelete(w http.ResponseWriter, r *http.Request, id int64) {
	err := h.deleteSupplier.Execute(r.Context(), id)
	if writeSupplierUsecaseError(w, err, "Failed to delete supplier") {
		return
	}
	log.Printf("Deleted supplier id=%d", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminSupplierHandler) handleReorder(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxSupplierBodyBytes)

	var req ReorderSuppliersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return
	}

	suppliers, err := h.reorderSupplier.Execute(r.Context(), usecase.ReorderSuppliersCommand{OrderedIDs: req.Order})
	if writeSupplierUsecaseError(w, err, "Failed to reorder suppliers") {
		return
	}

	responses := make([]SupplierResponse, len(suppliers))
	for i, s := range suppliers {
		responses[i] = toSupplierResponse(s)
	}
	respondWithJSON(w, http.StatusOK, AdminSupplierListResponse{Suppliers: responses})
}

func (h *AdminSupplierHandler) handleUploadImage(w http.ResponseWriter, r *http.Request, idStr string) {
	id, ok := parseSupplierID(w, idStr)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSupplierImageBodyBytes)
	if err := r.ParseMultipartForm(maxSupplierImageBodyBytes); err != nil {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "画像の読み込みに失敗しました", nil)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("image")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, ValidationErrorCode, "入力内容に誤りがあります", []ErrorDetail{
			{Field: "image", Message: "画像ファイルを指定してください"},
		})
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")

	updated, err := h.uploadImage.Execute(r.Context(), usecase.UploadImageCommand{
		SupplierID:  id,
		ContentType: contentType,
		Data:        file,
		Size:        header.Size,
	})
	if writeSupplierUsecaseError(w, err, "Failed to upload supplier image") {
		return
	}

	imageURL := ""
	if updated.ImageURL != nil {
		imageURL = *updated.ImageURL
	}
	log.Printf("Uploaded supplier image id=%d url=%s", id, imageURL)
	respondWithJSON(w, http.StatusOK, SupplierImageResponse{ImageURL: imageURL})
}

// requireJSON は Content-Type が application/json か検証する。不正なら 400 を返し false。
func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != mediaTypeJSON {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return false
	}
	return true
}

func parseSupplierID(w http.ResponseWriter, idStr string) (int64, bool) {
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		respondWithError(w, http.StatusBadRequest, InvalidRequestCode, "リクエスト形式が不正です", nil)
		return 0, false
	}
	return id, true
}
