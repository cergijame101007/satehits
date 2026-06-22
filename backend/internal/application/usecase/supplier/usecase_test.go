package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

// fakeSupplierRepo は domain.SupplierRepository のテスト用フェイク
type fakeSupplierRepo struct {
	getByID     func(ctx context.Context, id int64) (domain.Supplier, error)
	update      func(ctx context.Context, id int64, in domain.UpdateSupplierInput) (domain.Supplier, error)
	updateImage func(ctx context.Context, id int64, url string) (domain.Supplier, error)
}

func (f *fakeSupplierRepo) List(context.Context, bool) ([]domain.Supplier, error) {
	return nil, nil
}
func (f *fakeSupplierRepo) GetByID(ctx context.Context, id int64) (domain.Supplier, error) {
	return f.getByID(ctx, id)
}
func (f *fakeSupplierRepo) Create(context.Context, domain.CreateSupplierInput) (domain.Supplier, error) {
	return domain.Supplier{}, nil
}
func (f *fakeSupplierRepo) Update(ctx context.Context, id int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
	return f.update(ctx, id, in)
}
func (f *fakeSupplierRepo) Delete(context.Context, int64) error { return nil }
func (f *fakeSupplierRepo) Reorder(context.Context, []int64) error {
	return nil
}
func (f *fakeSupplierRepo) UpdateImageURL(ctx context.Context, id int64, url string) (domain.Supplier, error) {
	return f.updateImage(ctx, id, url)
}

// fakeStorage は domain.ImageStorage のテスト用フェイク
type fakeStorage struct {
	saved   string
	deleted []string
	saveErr error
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

func TestUpdateSupplierMergesProvidedFields(t *testing.T) {
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
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, InstagramURL: in.InstagramURL, DisplayOrder: in.DisplayOrder, IsActive: in.IsActive}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	// name のみ更新、他は維持されること
	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, Name: strptr("新名")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "新名" {
		t.Errorf("got name %q, want 新名", got.Name)
	}
	if got.Description != "旧説明" {
		t.Errorf("got description %q, want 旧説明 (unchanged)", got.Description)
	}
	if got.IsActive != true {
		t.Errorf("got is_active %v, want true (unchanged)", got.IsActive)
	}
}

func TestUpdateSupplierClearsInstagramWithEmptyString(t *testing.T) {
	current := domain.Supplier{ID: 1, Name: "名", Description: "説明", InstagramURL: strptr("https://instagram.com/old"), IsActive: true}
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, _ int64) (domain.Supplier, error) { return current, nil },
		update: func(_ context.Context, _ int64, in domain.UpdateSupplierInput) (domain.Supplier, error) {
			return domain.Supplier{ID: 1, Name: in.Name, Description: in.Description, InstagramURL: in.InstagramURL}, nil
		},
	}
	uc := NewUpdateSupplierUseCase(repo)

	got, err := uc.Execute(context.Background(), UpdateSupplierCommand{ID: 1, InstagramURL: strptr("")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.InstagramURL != nil {
		t.Errorf("got instagram_url %v, want nil (cleared)", *got.InstagramURL)
	}
}

func TestUploadImage(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		size        int64
		wantField   string // 空ならアップロード成功を期待
	}{
		{name: "accepts jpeg under limit", contentType: "image/jpeg", size: 1024},
		{name: "accepts png under limit", contentType: "image/png", size: 1024},
		{name: "accepts webp under limit", contentType: "image/webp", size: 1024},
		{name: "rejects unsupported content type", contentType: "image/gif", size: 1024, wantField: "image"},
		{name: "rejects oversized image", contentType: "image/jpeg", size: maxSupplierImageBytes + 1, wantField: "image"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeSupplierRepo{
				getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
					return domain.Supplier{ID: id, Name: "名", Description: "説明"}, nil
				},
				updateImage: func(_ context.Context, id int64, url string) (domain.Supplier, error) {
					return domain.Supplier{ID: id, Name: "名", Description: "説明", ImageURL: &url}, nil
				},
			}
			st := &fakeStorage{}
			uc := NewUploadImageUseCase(repo, st)

			got, err := uc.Execute(context.Background(), UploadImageCommand{
				SupplierID:  1,
				ContentType: tt.contentType,
				Data:        strings.NewReader("dummy"),
				Size:        tt.size,
			})

			if tt.wantField != "" {
				var vErr *ValidationError
				if !errors.As(err, &vErr) {
					t.Fatalf("got err %v, want ValidationError", err)
				}
				assertHasViolationField(t, vErr.Violations, tt.wantField)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ImageURL == nil {
				t.Fatal("got nil image_url, want set")
			}
		})
	}
}

func TestUploadImageDeletesOldImage(t *testing.T) {
	oldURL := "https://cdn.example.com/suppliers/1/old.jpg"
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明", ImageURL: &oldURL}, nil
		},
		updateImage: func(_ context.Context, id int64, url string) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明", ImageURL: &url}, nil
		},
	}
	st := &fakeStorage{}
	uc := NewUploadImageUseCase(repo, st)

	_, err := uc.Execute(context.Background(), UploadImageCommand{
		SupplierID:  1,
		ContentType: "image/jpeg",
		Data:        strings.NewReader("dummy"),
		Size:        1024,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(st.deleted) != 1 || st.deleted[0] != "suppliers/1/old.jpg" {
		t.Errorf("got deleted %v, want [suppliers/1/old.jpg]", st.deleted)
	}
}
