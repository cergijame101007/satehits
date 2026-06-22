package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestListSuppliersUseCase_Execute_passesActiveOnlyFlag(t *testing.T) {
	tests := []struct {
		name       string
		activeOnly bool
	}{
		{name: "lists all for admin", activeOnly: false},
		{name: "lists active only for public", activeOnly: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotActiveOnly bool
			repo := &fakeSupplierRepo{
				list: func(_ context.Context, activeOnly bool) ([]domain.Supplier, error) {
					gotActiveOnly = activeOnly
					return []domain.Supplier{{ID: 1, Name: "A", Description: "説明", IsActive: true}}, nil
				},
			}
			uc := NewListSuppliersUseCase(repo)

			got, err := uc.Execute(context.Background(), tt.activeOnly)
			if err != nil {
				t.Fatalf("Execute() err = %v, want nil", err)
			}
			if gotActiveOnly != tt.activeOnly {
				t.Fatalf("activeOnly = %v, want %v", gotActiveOnly, tt.activeOnly)
			}
			if len(got) != 1 {
				t.Fatalf("len = %d, want 1", len(got))
			}
		})
	}
}

func TestListSuppliersUseCase_Execute_returnsEmptySliceWhenNil(t *testing.T) {
	repo := &fakeSupplierRepo{
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			return nil, nil
		},
	}
	uc := NewListSuppliersUseCase(repo)

	got, err := uc.Execute(context.Background(), false)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("got = nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

func TestListSuppliersUseCase_Execute_preservesRepositoryOrder(t *testing.T) {
	repo := &fakeSupplierRepo{
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			return []domain.Supplier{
				{ID: 2, Name: "B", Description: "説明", DisplayOrder: 1},
				{ID: 1, Name: "A", Description: "説明", DisplayOrder: 2},
			}, nil
		},
	}
	uc := NewListSuppliersUseCase(repo)

	got, err := uc.Execute(context.Background(), false)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 1 {
		t.Fatalf("order = [%d, %d], want [2, 1]", got[0].ID, got[1].ID)
	}
}

func TestListSuppliersUseCase_Execute_propagatesRepoError(t *testing.T) {
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			return nil, repoErr
		},
	}
	uc := NewListSuppliersUseCase(repo)

	_, err := uc.Execute(context.Background(), false)
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}
