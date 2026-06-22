package domain

import (
	"context"
	"io"
)

// ImageStorage は画像オブジェクトの保存・削除を抽象化するインターフェース。
// Infrastructure 層（R2 / MinIO 等の S3 互換ストレージ）が実装する（依存性逆転）。
type ImageStorage interface {
	// Save は key にオブジェクトを保存し、公開アクセス可能な URL を返す。
	Save(ctx context.Context, key, contentType string, r io.Reader, size int64) (url string, err error)
	// Delete は key のオブジェクトを削除する。存在しない場合もエラーにしない実装を推奨。
	Delete(ctx context.Context, key string) error
	// KeyFromURL は Save が返した公開 URL からストレージ上の key を逆算する。
	// 自身が払い出した URL でない場合は空文字を返す。
	KeyFromURL(url string) string
}
