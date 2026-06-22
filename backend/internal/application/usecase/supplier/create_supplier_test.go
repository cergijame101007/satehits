package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestCreateSupplierUseCase_Execute_createsWithDefaults(t *testing.T) {
	var captured domain.CreateSupplierInput
	repo := &fakeSupplierRepo{
		create: func(_ context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
			captured = in
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, IsActive: in.IsActive}, nil
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), validCreateSupplierCommand())
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.ID != 1 {
		t.Fatalf("ID = %d, want 1", got.ID)
	}
	if !captured.IsActive {
		t.Fatal("IsActive = false, want true (default)")
	}
	if captured.DisplayOrder != nil {
		t.Fatalf("DisplayOrder = %v, want nil", captured.DisplayOrder)
	}
}

func TestCreateSupplierUseCase_Execute_trimsFields(t *testing.T) {
	var captured domain.CreateSupplierInput
	repo := &fakeSupplierRepo{
		create: func(_ context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
			captured = in
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description}, nil
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	cmd := validCreateSupplierCommand()
	cmd.Name = "  〇〇農園  "
	cmd.Description = "  説明文  "
	instagram := "  https://instagram.com/example  "
	cmd.InstagramURL = &instagram

	_, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if captured.Name != "〇〇農園" {
		t.Fatalf("Name = %q, want 〇〇農園", captured.Name)
	}
	if captured.Description != "説明文" {
		t.Fatalf("Description = %q, want 説明文", captured.Description)
	}
	if captured.InstagramURL == nil || *captured.InstagramURL != "https://instagram.com/example" {
		t.Fatalf("InstagramURL = %v, want trimmed url", captured.InstagramURL)
	}
}

func TestCreateSupplierUseCase_Execute_respectsIsActiveFalse(t *testing.T) {
	var captured domain.CreateSupplierInput
	repo := &fakeSupplierRepo{
		create: func(_ context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
			captured = in
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, IsActive: in.IsActive}, nil
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	isActive := false
	cmd := validCreateSupplierCommand()
	cmd.IsActive = &isActive

	_, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if captured.IsActive {
		t.Fatal("IsActive = true, want false")
	}
}

func TestCreateSupplierUseCase_Execute_rejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name      string
		cmd       CreateSupplierCommand
		wantField string
	}{
		{
			name: "rejects empty name",
			cmd: CreateSupplierCommand{Name: "  ", Description: "説明"},
			wantField: "name",
		},
		{
			name: "rejects empty description",
			cmd: CreateSupplierCommand{Name: "〇〇農園", Description: "  "},
			wantField: "description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewCreateSupplierUseCase(&fakeSupplierRepo{})
			_, err := uc.Execute(context.Background(), tt.cmd)
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			assertHasViolationField(t, vErr.Violations, tt.wantField)
		})
	}
}

func TestCreateSupplierUseCase_Execute_passesDisplayOrder(t *testing.T) {
	var captured domain.CreateSupplierInput
	repo := &fakeSupplierRepo{
		create: func(_ context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
			captured = in
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description}, nil
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	displayOrder := 3
	cmd := validCreateSupplierCommand()
	cmd.DisplayOrder = &displayOrder

	_, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if captured.DisplayOrder == nil || *captured.DisplayOrder != 3 {
		t.Fatalf("DisplayOrder = %v, want 3", captured.DisplayOrder)
	}
}

func TestCreateSupplierUseCase_Execute_trimsImageURL(t *testing.T) {
	var captured domain.CreateSupplierInput
	repo := &fakeSupplierRepo{
		create: func(_ context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
			captured = in
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description}, nil
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	imageURL := "  https://cdn.example.com/a.jpg  "
	cmd := validCreateSupplierCommand()
	cmd.ImageURL = &imageURL

	_, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if captured.ImageURL == nil || *captured.ImageURL != "https://cdn.example.com/a.jpg" {
		t.Fatalf("ImageURL = %v, want trimmed url", captured.ImageURL)
	}
}

func TestCreateSupplierUseCase_Execute_clearsEmptyImageURL(t *testing.T) {
	var captured domain.CreateSupplierInput
	repo := &fakeSupplierRepo{
		create: func(_ context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
			captured = in
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description}, nil
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	empty := "   "
	cmd := validCreateSupplierCommand()
	cmd.ImageURL = &empty

	_, err := uc.Execute(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if captured.ImageURL != nil {
		t.Fatalf("ImageURL = %v, want nil", captured.ImageURL)
	}
}

func TestCreateSupplierUseCase_Execute_propagatesRepoError(t *testing.T) {
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		create: func(context.Context, domain.CreateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{}, repoErr
		},
	}
	uc := NewCreateSupplierUseCase(repo)

	_, err := uc.Execute(context.Background(), validCreateSupplierCommand())
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}
