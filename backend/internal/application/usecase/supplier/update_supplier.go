package usecase

import (
	"context"
	"strings"

	"github.com/cergijame101007/satehits/internal/domain"
)

// UpdateSupplierCommand は取引先更新の入力（部分更新）。
// 各ポインタが nil のフィールドは既存値を維持する。
// InstagramURL / ImageURL は空文字を渡すと NULL クリアになる。
type UpdateSupplierCommand struct {
	ID           int64
	Name         *string
	Description  *string
	InstagramURL *string
	ImageURL     *string
	DisplayOrder *int
	IsActive     *bool
}

// UpdateSupplierUseCase は取引先更新
type UpdateSupplierUseCase struct {
	repo domain.SupplierRepository
}

// NewUpdateSupplierUseCase は UpdateSupplierUseCase を生成する
func NewUpdateSupplierUseCase(repo domain.SupplierRepository) *UpdateSupplierUseCase {
	return &UpdateSupplierUseCase{repo: repo}
}

// Execute は既存の取引先に指定フィールドをマージして更新する
func (u *UpdateSupplierUseCase) Execute(ctx context.Context, cmd UpdateSupplierCommand) (*domain.Supplier, error) {
	current, err := u.repo.GetByID(ctx, cmd.ID)
	if err != nil {
		return nil, err
	}

	name := current.Name
	if cmd.Name != nil {
		name = strings.TrimSpace(*cmd.Name)
	}
	description := current.Description
	if cmd.Description != nil {
		description = strings.TrimSpace(*cmd.Description)
	}
	instagramURL := current.InstagramURL
	if cmd.InstagramURL != nil {
		instagramURL = trimmedPtr(cmd.InstagramURL)
	}
	imageURL := current.ImageURL
	if cmd.ImageURL != nil {
		imageURL = trimmedPtr(cmd.ImageURL)
	}
	displayOrder := current.DisplayOrder
	if cmd.DisplayOrder != nil {
		displayOrder = *cmd.DisplayOrder
	}
	isActive := current.IsActive
	if cmd.IsActive != nil {
		isActive = *cmd.IsActive
	}

	violations := validateSupplierFields(name, description, instagramURL)
	if len(violations) > 0 {
		return nil, &ValidationError{Violations: violations}
	}

	updated, err := u.repo.Update(ctx, cmd.ID, domain.UpdateSupplierInput{
		Name:         name,
		Description:  description,
		InstagramURL: instagramURL,
		ImageURL:     imageURL,
		DisplayOrder: displayOrder,
		IsActive:     isActive,
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}
