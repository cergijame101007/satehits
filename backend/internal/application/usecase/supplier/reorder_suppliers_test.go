package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestReorderSuppliersUseCase_Execute_reordersInTransaction(t *testing.T) {
	var reorderedIDs []int64
	repo := &fakeSupplierRepo{
		reorder: func(_ context.Context, ids []int64) error {
			reorderedIDs = ids
			return nil
		},
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			return []domain.Supplier{
				{ID: 2, Name: "B", Description: "説明", DisplayOrder: 1},
				{ID: 1, Name: "A", Description: "説明", DisplayOrder: 2},
			}, nil
		},
	}
	uc := NewReorderSuppliersUseCase(repo, passThroughTxManager{})

	got, err := uc.Execute(context.Background(), ReorderSuppliersCommand{OrderedIDs: []int64{2, 1}})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if len(reorderedIDs) != 2 || reorderedIDs[0] != 2 || reorderedIDs[1] != 1 {
		t.Fatalf("reorderedIDs = %v, want [2 1]", reorderedIDs)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestReorderSuppliersUseCase_Execute_rejectsInvalidOrder(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
	}{
		{name: "rejects empty order", ids: []int64{}},
		{name: "rejects duplicate ids", ids: []int64{1, 2, 1}},
		{name: "rejects non-positive id", ids: []int64{1, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewReorderSuppliersUseCase(&fakeSupplierRepo{}, passThroughTxManager{})
			_, err := uc.Execute(context.Background(), ReorderSuppliersCommand{OrderedIDs: tt.ids})
			var vErr *ValidationError
			if !errors.As(err, &vErr) {
				t.Fatalf("err = %v, want ValidationError", err)
			}
			assertHasViolationField(t, vErr.Violations, "order")
		})
	}
}

func TestReorderSuppliersUseCase_Execute_propagatesReorderError(t *testing.T) {
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		reorder: func(context.Context, []int64) error {
			return repoErr
		},
	}
	uc := NewReorderSuppliersUseCase(repo, passThroughTxManager{})

	_, err := uc.Execute(context.Background(), ReorderSuppliersCommand{OrderedIDs: []int64{1, 2}})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}

func TestReorderSuppliersUseCase_Execute_skipsListWhenCommitFails(t *testing.T) {
	listCalled := false
	repo := &fakeSupplierRepo{
		reorder: func(context.Context, []int64) error { return nil },
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			listCalled = true
			return nil, nil
		},
	}
	uc := NewReorderSuppliersUseCase(repo, commitFailTxManager{})

	_, err := uc.Execute(context.Background(), ReorderSuppliersCommand{OrderedIDs: []int64{1, 2}})
	if err == nil {
		t.Fatal("err = nil, want commit failure")
	}
	if listCalled {
		t.Fatal("List called, want skipped when transaction fails")
	}
}

func TestReorderSuppliersUseCase_Execute_propagatesListError(t *testing.T) {
	listErr := errors.New("list failed")
	repo := &fakeSupplierRepo{
		reorder: func(context.Context, []int64) error { return nil },
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			return nil, listErr
		},
	}
	uc := NewReorderSuppliersUseCase(repo, passThroughTxManager{})

	_, err := uc.Execute(context.Background(), ReorderSuppliersCommand{OrderedIDs: []int64{1, 2}})
	if !errors.Is(err, listErr) {
		t.Fatalf("err = %v, want %v", err, listErr)
	}
}

func TestReorderSuppliersUseCase_Execute_returnsEmptySliceWhenListNil(t *testing.T) {
	repo := &fakeSupplierRepo{
		reorder: func(context.Context, []int64) error { return nil },
		list: func(context.Context, bool) ([]domain.Supplier, error) {
			return nil, nil
		},
	}
	uc := NewReorderSuppliersUseCase(repo, passThroughTxManager{})

	got, err := uc.Execute(context.Background(), ReorderSuppliersCommand{OrderedIDs: []int64{1}})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got == nil {
		t.Fatal("got = nil, want empty slice")
	}
}
