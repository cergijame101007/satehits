package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cergijame101007/satehits/internal/domain"
)

func TestUploadImageUseCase_Execute_acceptsAllowedFormats(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		size        int64
		wantField   string
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
					t.Fatalf("err = %v, want ValidationError", err)
				}
				assertHasViolationField(t, vErr.Violations, tt.wantField)
				return
			}
			if err != nil {
				t.Fatalf("Execute() err = %v, want nil", err)
			}
			if got.ImageURL == nil {
				t.Fatal("ImageURL = nil, want set")
			}
		})
	}
}

func TestUploadImageUseCase_Execute_deletesOldImage(t *testing.T) {
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
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if len(st.deleted) != 1 || st.deleted[0] != "suppliers/1/old.jpg" {
		t.Fatalf("deleted = %v, want [suppliers/1/old.jpg]", st.deleted)
	}
}

func TestUploadImageUseCase_Execute_propagatesNotFound(t *testing.T) {
	repo := &fakeSupplierRepo{
		getByID: func(context.Context, int64) (domain.Supplier, error) {
			return domain.Supplier{}, domain.ErrSupplierNotFound
		},
	}
	uc := NewUploadImageUseCase(repo, &fakeStorage{})

	_, err := uc.Execute(context.Background(), UploadImageCommand{
		SupplierID:  99,
		ContentType: "image/jpeg",
		Data:        strings.NewReader("dummy"),
		Size:        1024,
	})
	if !errors.Is(err, domain.ErrSupplierNotFound) {
		t.Fatalf("err = %v, want ErrSupplierNotFound", err)
	}
}

func TestUploadImageUseCase_Execute_acceptsImageAtSizeLimit(t *testing.T) {
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明"}, nil
		},
		updateImage: func(_ context.Context, id int64, url string) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明", ImageURL: &url}, nil
		},
	}
	uc := NewUploadImageUseCase(repo, &fakeStorage{})

	got, err := uc.Execute(context.Background(), UploadImageCommand{
		SupplierID:  1,
		ContentType: "image/jpeg",
		Data:        strings.NewReader("dummy"),
		Size:        maxSupplierImageBytes,
	})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if got.ImageURL == nil {
		t.Fatal("ImageURL = nil, want set")
	}
}

func TestUploadImageUseCase_Execute_propagatesUpdateImageRepoError(t *testing.T) {
	repoErr := errors.New("db error")
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明"}, nil
		},
		updateImage: func(context.Context, int64, string) (domain.Supplier, error) {
			return domain.Supplier{}, repoErr
		},
	}
	uc := NewUploadImageUseCase(repo, &fakeStorage{})

	_, err := uc.Execute(context.Background(), UploadImageCommand{
		SupplierID:  1,
		ContentType: "image/jpeg",
		Data:        strings.NewReader("dummy"),
		Size:        1024,
	})
	if !errors.Is(err, repoErr) {
		t.Fatalf("err = %v, want %v", err, repoErr)
	}
}

func TestUploadImageUseCase_Execute_succeedsWhenOldImageDeleteFails(t *testing.T) {
	oldURL := "https://cdn.example.com/suppliers/1/old.jpg"
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明", ImageURL: &oldURL}, nil
		},
		updateImage: func(_ context.Context, id int64, url string) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明", ImageURL: &url}, nil
		},
	}
	st := &fakeStorage{deleteErr: errors.New("delete failed")}
	uc := NewUploadImageUseCase(repo, st)

	got, err := uc.Execute(context.Background(), UploadImageCommand{
		SupplierID:  1,
		ContentType: "image/jpeg",
		Data:        strings.NewReader("dummy"),
		Size:        1024,
	})
	if err != nil {
		t.Fatalf("Execute() err = %v, want nil (best-effort delete)", err)
	}
	if got.ImageURL == nil {
		t.Fatal("ImageURL = nil, want set")
	}
}

func TestUploadImageUseCase_Execute_skipsDeleteForUnrecognizedOldURL(t *testing.T) {
	oldURL := "https://other-cdn.example.com/old.jpg"
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
		t.Fatalf("Execute() err = %v, want nil", err)
	}
	if len(st.deleted) != 0 {
		t.Fatalf("deleted = %v, want none", st.deleted)
	}
}

func TestUploadImageUseCase_Execute_propagatesStorageError(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	repo := &fakeSupplierRepo{
		getByID: func(_ context.Context, id int64) (domain.Supplier, error) {
			return domain.Supplier{ID: id, Name: "名", Description: "説明"}, nil
		},
	}
	st := &fakeStorage{saveErr: storageErr}
	uc := NewUploadImageUseCase(repo, st)

	_, err := uc.Execute(context.Background(), UploadImageCommand{
		SupplierID:  1,
		ContentType: "image/jpeg",
		Data:        io.NopCloser(strings.NewReader("dummy")),
		Size:        1024,
	})
	if err == nil {
		t.Fatal("err = nil, want storage error")
	}
}
