package usecase

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/google/uuid"

	"github.com/cergijame101007/satehits/internal/domain"
)

// maxSupplierImageBytes は取引先画像の最大サイズ（5MB）
const maxSupplierImageBytes = 5 << 20

// allowedImageContentTypes は許可する画像 MIME とファイル拡張子の対応
var allowedImageContentTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// UploadImageCommand は取引先画像アップロードの入力
type UploadImageCommand struct {
	SupplierID  int64
	ContentType string
	Data        io.Reader
	Size        int64
}

// UploadImageUseCase は取引先画像のアップロード
type UploadImageUseCase struct {
	repo    domain.SupplierRepository
	storage domain.ImageStorage
}

// NewUploadImageUseCase は UploadImageUseCase を生成する
func NewUploadImageUseCase(repo domain.SupplierRepository, storage domain.ImageStorage) *UploadImageUseCase {
	return &UploadImageUseCase{repo: repo, storage: storage}
}

// Execute は画像をストレージへ保存し、取引先の image_url を更新する。
// 差し替え時の旧画像削除は best-effort（失敗してもアップロードは成功扱い）。
func (u *UploadImageUseCase) Execute(ctx context.Context, cmd UploadImageCommand) (*domain.Supplier, error) {
	ext, ok := allowedImageContentTypes[cmd.ContentType]
	if !ok {
		return nil, &ValidationError{Violations: []FieldViolation{
			{Field: "image", Message: "対応していない画像形式です（JPEG / PNG / WebP のみ）"},
		}}
	}
	if cmd.Size > maxSupplierImageBytes {
		return nil, &ValidationError{Violations: []FieldViolation{
			{Field: "image", Message: "画像サイズは5MB以内にしてください"},
		}}
	}

	// 取引先の存在確認と旧画像 URL の取得
	current, err := u.repo.GetByID(ctx, cmd.SupplierID)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf("suppliers/%d/%s.%s", cmd.SupplierID, uuid.NewString(), ext)
	publicURL, err := u.storage.Save(ctx, key, cmd.ContentType, cmd.Data, cmd.Size)
	if err != nil {
		return nil, fmt.Errorf("save image: %w", err)
	}

	updated, err := u.repo.UpdateImageURL(ctx, cmd.SupplierID, publicURL)
	if err != nil {
		return nil, err
	}

	// 旧画像の削除（best-effort）
	if current.ImageURL != nil && *current.ImageURL != publicURL {
		if oldKey := u.storage.KeyFromURL(*current.ImageURL); oldKey != "" {
			if delErr := u.storage.Delete(ctx, oldKey); delErr != nil {
				log.Printf("failed to delete old supplier image: key=%s err=%v", oldKey, delErr)
			}
		}
	}

	return &updated, nil
}
