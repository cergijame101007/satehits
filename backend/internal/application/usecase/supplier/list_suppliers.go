package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/domain"
)

// ListSuppliersUseCase は取引先一覧取得（公開・管理共通）
type ListSuppliersUseCase struct {
	repo domain.SupplierRepository
}

// NewListSuppliersUseCase は ListSuppliersUseCase を生成する
func NewListSuppliersUseCase(repo domain.SupplierRepository) *ListSuppliersUseCase {
	return &ListSuppliersUseCase{repo: repo}
}

// Execute は取引先一覧を返す。activeOnly が true なら is_active=true のみ（公開用）。
func (u *ListSuppliersUseCase) Execute(ctx context.Context, activeOnly bool) ([]domain.Supplier, error) {
	suppliers, err := u.repo.List(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	if suppliers == nil {
		suppliers = []domain.Supplier{}
	}
	return suppliers, nil
}
