package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/domain"
)

// ReorderSuppliersCommand は並び順更新の入力
type ReorderSuppliersCommand struct {
	OrderedIDs []int64
}

// ReorderSuppliersUseCase は取引先の並び順更新
type ReorderSuppliersUseCase struct {
	repo      domain.SupplierRepository
	txManager application.TxManager
}

// NewReorderSuppliersUseCase は ReorderSuppliersUseCase を生成する
func NewReorderSuppliersUseCase(repo domain.SupplierRepository, txManager application.TxManager) *ReorderSuppliersUseCase {
	return &ReorderSuppliersUseCase{repo: repo, txManager: txManager}
}

// Execute は受領した ID 順に display_order を 1..N で更新する。
// 重複・空配列を検証し、更新はトランザクションで原子的に行う。
func (u *ReorderSuppliersUseCase) Execute(ctx context.Context, cmd ReorderSuppliersCommand) ([]domain.Supplier, error) {
	if violations := validateReorder(cmd.OrderedIDs); len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	err := u.txManager.DoInTx(ctx, func(txCtx context.Context) error {
		return u.repo.Reorder(txCtx, cmd.OrderedIDs)
	})
	if err != nil {
		return nil, err
	}

	suppliers, err := u.repo.List(ctx, false)
	if err != nil {
		return nil, err
	}
	if suppliers == nil {
		suppliers = []domain.Supplier{}
	}
	return suppliers, nil
}

func validateReorder(ids []int64) []FieldViolation {
	if len(ids) == 0 {
		return []FieldViolation{{Field: "order", Message: "並び順の指定が空です"}}
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return []FieldViolation{{Field: "order", Message: "IDが不正です"}}
		}
		if _, dup := seen[id]; dup {
			return []FieldViolation{{Field: "order", Message: "IDが重複しています"}}
		}
		seen[id] = struct{}{}
	}
	return nil
}
