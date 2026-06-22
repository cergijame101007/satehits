package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestGetSupplierUseCase_Execute_returnsSupplier(t *testing.T) {
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "〇〇農園", Description: "説明"}, nil
		},
	}
	uc := NewGetSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.Name != "〇〇農園" {
		t.Fatalf("Name = %q, want 〇〇農園", got.Name)
	}
}

func TestGetSupplierUseCase_Execute_propagatesRepoError(t *testing.T) {
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		getByID: func(context.Context, int64) (domain.Supplier, error) {
			return domain.Supplier{}, repoErr
		},
	}
	uc := NewGetSupplierUseCase(repo)

	_, err := uc.Execute(context.Background(), 1)
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}

func TestGetSupplierUseCase_Execute_propagatesNotFound(t *testing.T) {
	repo := &fakeSupplierRepo{
		getByID: func(context.Context, int64) (domain.Supplier, error) {
			return domain.Supplier{}, domain.ErrSupplierNotFound
		},
	}
	uc := NewGetSupplierUseCase(repo)

	_, err := uc.Execute(context.Background(), 99)
	if !errors.Is(err, domain.ErrSupplierNotFound) {
		t.Fatalf("err = %v, want ErrSupplierNotFound", err)
	}
}
