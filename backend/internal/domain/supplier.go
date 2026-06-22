package domain

import (
	"context"
	"errors"
	"time"
)

// ErrSupplierNotFound は指定 ID の取引先が存在しない
var ErrSupplierNotFound = errors.New("supplier not found")

// Supplier はお取り引き先（取引先）のドメインエンティティ。
// InstagramURL / ImageURL は NULL 許容のためポインタで保持する。
type Supplier struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	InstagramURL *string   `json:"instagram_url"`
	ImageURL     *string   `json:"image_url"`
	DisplayOrder int       `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateSupplierInput は取引先作成の永続化入力。
// DisplayOrder が nil の場合はリポジトリ側で末尾（MAX+1）に採番する。
type CreateSupplierInput struct {
	Name         string
	Description  string
	InstagramURL *string
	ImageURL     *string
	DisplayOrder *int
	IsActive     bool
}

// UpdateSupplierInput は取引先更新の永続化入力（全項目を上書きする）。
type UpdateSupplierInput struct {
	Name         string
	Description  string
	InstagramURL *string
	ImageURL     *string
	DisplayOrder int
	IsActive     bool
}

// SupplierRepository は取引先データを永続化するためのインターフェース
type SupplierRepository interface {
	// List は取引先一覧を display_order ASC, id ASC で返す。
	// activeOnly が true の場合は is_active=true のみを返す（公開用）。
	List(ctx context.Context, activeOnly bool) ([]Supplier, error)
	GetByID(ctx context.Context, id int64) (Supplier, error)
	Create(ctx context.Context, in CreateSupplierInput) (Supplier, error)
	Update(ctx context.Context, id int64, in UpdateSupplierInput) (Supplier, error)
	Delete(ctx context.Context, id int64) error
	// Reorder は ids の並び順に従って display_order を 1..N で更新する。
	Reorder(ctx context.Context, ids []int64) error
	// UpdateImageURL は画像 URL のみを更新する。
	UpdateImageURL(ctx context.Context, id int64, imageURL string) (Supplier, error)
}
