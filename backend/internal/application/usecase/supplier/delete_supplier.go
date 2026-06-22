package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/domain"
)

// DeleteSupplierUseCase は取引先削除
type DeleteSupplierUseCase struct {
	repo domain.SupplierRepository
}

// NewDeleteSupplierUseCase は DeleteSupplierUseCase を生成する
func NewDeleteSupplierUseCase(repo domain.SupplierRepository) *DeleteSupplierUseCase {
	return &DeleteSupplierUseCase{repo: repo}
}

// Execute は ID で取引先を削除する
func (u *DeleteSupplierUseCase) Execute(ctx context.Context, id int64) error {
	return u.repo.Delete(ctx, id)
}
