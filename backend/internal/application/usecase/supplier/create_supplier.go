package usecase

import (
	"context"
	"strings"

	"github.com/cergijame101007/satehits/internal/domain"
)

// CreateSupplierCommand は取引先作成の入力。
// DisplayOrder 省略時は末尾、IsActive 省略時は true。
type CreateSupplierCommand struct {
	Name         string
	Description  string
	InstagramURL *string
	ImageURL     *string
	DisplayOrder *int
	IsActive     *bool
}

// CreateSupplierUseCase は取引先作成
type CreateSupplierUseCase struct {
	repo domain.SupplierRepository
}

// NewCreateSupplierUseCase は CreateSupplierUseCase を生成する
func NewCreateSupplierUseCase(repo domain.SupplierRepository) *CreateSupplierUseCase {
	return &CreateSupplierUseCase{repo: repo}
}

// Execute は入力検証および Repository への永続化
func (u *CreateSupplierUseCase) Execute(ctx context.Context, cmd CreateSupplierCommand) (*domain.Supplier, error) {
	violations := validateSupplierFields(cmd.Name, cmd.Description, cmd.InstagramURL)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	isActive := true
	if cmd.IsActive != nil {
		isActive = *cmd.IsActive
	}

	in := domain.CreateSupplierInput{
		Name:         strings.TrimSpace(cmd.Name),
		Description:  strings.TrimSpace(cmd.Description),
		InstagramURL: trimmedPtr(cmd.InstagramURL),
		ImageURL:     trimmedPtr(cmd.ImageURL),
		DisplayOrder: cmd.DisplayOrder,
		IsActive:     isActive,
	}
	s, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
