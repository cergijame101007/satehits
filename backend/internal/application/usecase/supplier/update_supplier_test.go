package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestUpdateSupplierUseCase_Execute_mergesProvidedFields(t *testing.T) {
	current := domain.Supplier{
		ID:           1,
		Name:         "旧名",
		Description:  "旧説明",
		InstagramURL: strptr("https://instagram.com/old"),
		DisplayOrder: 5,
		IsActive:     true,
	}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{
				ID: 1, Name: in.Name, Description: in.Description,
				InstagramURL: in.InstagramURL, DisplayOrder: in.DisplayOrder, IsActive: in.IsActive,
			}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, Name: strptr("新名")})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.Name != "新名" {
		t.Fatalf("Name = %q, want 新名", got.Name)
	}
	if got.Description != "旧説明" {
		t.Fatalf("Description = %q, want 旧説明 (unchanged)", got.Description)
	}
	if !got.IsActive {
		t.Fatalf("IsActive = %v, want true (unchanged)", got.IsActive)
	}
}

func TestUpdateSupplierUseCase_Execute_clearsInstagramWithEmptyString(t *testing.T) {
	current := domain.Supplier{
		ID: 1, Name: "名", Description: "説明",
		InstagramURL: strptr("https://instagram.com/old"), IsActive: true,
	}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, InstagramURL: in.InstagramURL}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, InstagramURL: strptr("")})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.InstagramURL != nil {
		t.Fatalf("InstagramURL = %v, want nil (cleared)", *got.InstagramURL)
	}
}

func TestUpdateSupplierUseCase_Execute_rejectsInvalidMergedFields(t *testing.T) {
	current := domain.Supplier{ID: 1, Name: "旧名", Description: "旧説明", IsActive: true}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
	}
	uc := NewUpdateSupplierUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, Name: strptr("  ")})
	var vErr *ValidationError
	if !errors.As(err, &vErr) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	assertHasViolationField(t, vErr.Violations, "name")
}

func TestUpdateSupplierUseCase_Execute_updatesDescriptionOnly(t *testing.T) {
	current := domain.Supplier{ID: 1, Name: "旧名", Description: "旧説明", IsActive: true}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, IsActive: in.IsActive}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, Description: strptr("新説明")})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.Description != "新説明" {
		t.Fatalf("Description = %q, want 新説明", got.Description)
	}
	if got.Name != "旧名" {
		t.Fatalf("Name = %q, want 旧名 (unchanged)", got.Name)
	}
}

func TestUpdateSupplierUseCase_Execute_updatesIsActiveOnly(t *testing.T) {
	current := domain.Supplier{ID: 1, Name: "名", Description: "説明", IsActive: true}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, IsActive: in.IsActive}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	isActive := false
	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, IsActive: &isActive})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.IsActive {
		t.Fatal("IsActive = true, want false")
	}
}

func TestUpdateSupplierUseCase_Execute_updatesDisplayOrderOnly(t *testing.T) {
	current := domain.Supplier{ID: 1, Name: "名", Description: "説明", DisplayOrder: 1, IsActive: true}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, DisplayOrder: in.DisplayOrder, IsActive: in.IsActive}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	displayOrder := 9
	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, DisplayOrder: &displayOrder})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.DisplayOrder != 9 {
		t.Fatalf("DisplayOrder = %d, want 9", got.DisplayOrder)
	}
}

func TestUpdateSupplierUseCase_Execute_clearsImageURLWithEmptyString(t *testing.T) {
	oldURL := "https://cdn.example.com/old.jpg"
	current := domain.Supplier{ID: 1, Name: "名", Description: "説明", ImageURL: &oldURL, IsActive: true}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, ImageURL: in.ImageURL}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, ImageURL: strptr("")})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.ImageURL != nil {
		t.Fatalf("ImageURL = %v, want nil (cleared)", *got.ImageURL)
	}
}

func TestUpdateSupplierUseCase_Execute_propagatesUpdateRepoError(t *testing.T) {
	current := domain.Supplier{ID: 1, Name: "名", Description: "説明", IsActive: true}
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(context.Context, int64, domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{}, repoErr
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, Name: strptr("新名")})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}

func TestUpdateSupplierUseCase_Execute_propagatesNotFound(t *testing.T) {
	repo := &fakeSupplierRepo{
		getByID: func(context.Context, int64) (domain.Supplier, error) {
			return domain.Supplier{}, domain.ErrSupplierNotFound
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	_, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 99, Name: strptr("新名")})
	if !errors.Is(err, domain.ErrSupplierNotFound) {
		t.Fatalf("err = %v, want ErrSupplierNotFound", err)
	}
}
