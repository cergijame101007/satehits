package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSaveSetsCacheControl は Save が PutObject リクエストに
// 長期キャッシュ用の Cache-Control ヘッダーを付与することを検証する。
// S3Storage は client を差し替えられない構造のため、httptest サーバーを
// S3 互換エンドポイントに見立てて実際に送られるリクエストヘッダーを検証する。
func TestSaveSetsCacheControl(t *testing.T) {
	var gotCacheControl string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCacheControl = r.Header.Get("Cache-Control")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewClient(Config{
		Endpoint:      srv.URL,
		Region:        "auto",
		Bucket:        "satehits-images",
		AccessKey:     "test",
		SecretKey:     "test",
		PublicBaseURL: "https://images.example.com",
	})

	_, err := s.Save(context.Background(), "suppliers/1/uuid.jpg", "image/jpeg", strings.NewReader("data"), 4)
	if err != nil {
		t.Fatalf("Save() err = %v", err)
	}
	want := "public, max-age=31536000, immutable"
	if gotCacheControl != want {
		t.Fatalf("Cache-Control = %q, want %q", gotCacheControl, want)
	}
}
