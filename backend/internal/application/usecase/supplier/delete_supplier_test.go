package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestDeleteSupplierUseCase_Execute_deletesByID(t *testing.T) {
	var deletedID int64
	repo := &fakeSupplierRepo{
		delete: func(_ context.Context, id int64) error {
			deletedID = id
			return nil
		},
	}
	uc := NewDeleteSupplierUseCase(repo)

	err := uc.Execute(context.Background(), 3)
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if deletedID != 3 {
		t.Fatalf("deletedID = %d, want 3", deletedID)
	}
}

func TestDeleteSupplierUseCase_Execute_propagatesRepoError(t *testing.T) {
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		delete: func(context.Context, int64) error {
			return repoErr
		},
	}
	uc := NewDeleteSupplierUseCase(repo)

	err := uc.Execute(context.Background(), 1)
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}

func TestDeleteSupplierUseCase_Execute_propagatesNotFound(t *testing.T) {
	repo := &fakeSupplierRepo{
		delete: func(context.Context, int64) error {
			return domain.ErrSupplierNotFound
		},
	}
	uc := NewDeleteSupplierUseCase(repo)

	err := uc.Execute(context.Background(), 99)
	if !errors.Is(err, domain.ErrSupplierNotFound) {
		t.Fatalf("err = %v, want ErrSupplierNotFound", err)
	}
}
