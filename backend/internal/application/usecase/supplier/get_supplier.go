package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/domain"
)

// GetSupplierUseCase は取引先1件取得（管理用）
type GetSupplierUseCase struct {
	repo domain.SupplierRepository
}

// NewGetSupplierUseCase は GetSupplierUseCase を生成する
func NewGetSupplierUseCase(repo domain.SupplierRepository) *GetSupplierUseCase {
	return &GetSupplierUseCase{repo: repo}
}

// Execute は ID で取引先を1件返す
func (u *GetSupplierUseCase) Execute(ctx context.Context, id int64) (*domain.Supplier, error) {
	s, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
