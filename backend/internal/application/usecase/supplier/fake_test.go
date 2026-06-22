package usecase

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/cergijame101007/satehits/internal/application"
	"github.com/cergijame101007/satehits/internal/domain"
)

type passThroughTxManager struct{}

func (passThroughTxManager) DoInTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

var _ application.TxManager = passThroughTxManager{}

// commitFailTxManager は fn 成功後に commit 失敗を返す TxManager フェイク
type commitFailTxManager struct{}

func (commitFailTxManager) DoInTx(ctx context.Context, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	return errors.New("commit failed")
}

var _ application.TxManager = commitFailTxManager{}

// fakeSupplierRepo は domain.SupplierRepository のテスト用フェイク
type fakeSupplierRepo struct {
	list        func(ctx context.Context, activeOnly bool) ([]domain.Supplier, error)
	getByID     func(ctx context.Context, id int64) (domain.Supplier, error)
	create      func(ctx context.Context, in domain.CreateSupplierInput) (domain.Supplier, error)
	update      func(ctx context.Context, id int64, in domain.UpdateSupplierInput) (domain.Supplier, error)
	delete      func(ctx context.Context, id int64) error
	reorder     func(ctx context.Context, ids []int64) error
	updateImage func(ctx context.Context, id int64, url string) (domain.Supplier, error)
}

func (f *fakeSupplierRepo) List(ctx context.Context, activeOnly bool) ([]domain.Supplier, error) {
	if f.list != nil {
		return f.list(ctx, activeOnly)
	}
	return nil, nil
}

func (f *fakeSupplierRepo) GetByID(ctx context.Context, id int64) (domain.Supplier, error) {
	if f.getByID != nil {
		return f.getByID(ctx, id)
	}
	return domain.Supplier{}, domain.ErrSupplierNotFound
}

func (f *fakeSupplierRepo) Create(ctx context.Context, in domain.CreateSupplierInput) (domain.Supplier, error) {
	if f.create != nil {
		return f.create(ctx, in)
	}
	return domain.Supplier{}, nil
}

func (f *fakeSupplierRepo) Update(ctx context.Context, id int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
	if f.update != nil {
		return f.update(ctx, id, in)
	}
	return domain.Supplier{}, nil
}

func (f *fakeSupplierRepo) Delete(ctx context.Context, id int64) error {
	if f.delete != nil {
		return f.delete(ctx, id)
	}
	return nil
}

func (f *fakeSupplierRepo) Reorder(ctx context.Context, ids []int64) error {
	if f.reorder != nil {
		return f.reorder(ctx, ids)
	}
	return nil
}

func (f *fakeSupplierRepo) UpdateImageURL(ctx context.Context, id int64, url string) (domain.Supplier, error) {
	if f.updateImage != nil {
		return f.updateImage(ctx, id, url)
	}
	return domain.Supplier{}, nil
}

// fakeStorage は domain.ImageStorage のテスト用フェイク
type fakeStorage struct {
	saved     string
	deleted   []string
	saveErr   error
	deleteErr error
}

func (f *fakeStorage) Save(_ context.Context, key, _ string, r io.Reader, _ int64) (string, error) {
	if f.saveErr != nil {
		return "", f.saveErr
	}
	_, _ = io.Copy(io.Discard, r)
	f.saved = key
	return "https://cdn.example.com/" + key, nil
}

func (f *fakeStorage) Delete(_ context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, key)
	return nil
}

func (f *fakeStorage) KeyFromURL(url string) string {
	const prefix = "https://cdn.example.com/"
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	return strings.TrimPrefix(url, prefix)
}

func validCreateSupplierCommand() CreateSupplierCommand {
	return CreateSupplierCommand{
		Name:        "〇〇農園",
		Description: "無農薬野菜の生産者です。",
	}
}
