package storage

import (
	"context"
	"errors"
	"io"
)

// ErrStorageNotConfigured はストレージ未設定時にアップロードを試みた場合のエラー
var ErrStorageNotConfigured = errors.New("image storage is not configured")

// NoOpStorage はストレージ未設定時のフォールバック実装。
// 画像アップロードは利用できないが、サーバ自体は起動できるようにするためのもの。
type NoOpStorage struct{}

// Save は常に ErrStorageNotConfigured を返す。
func (NoOpStorage) Save(_ context.Context, _, _ string, _ io.Reader, _ int64) (string, error) {
	return "", ErrStorageNotConfigured
}

// Delete は何もしない。
func (NoOpStorage) Delete(_ context.Context, _ string) error {
	return nil
}

// KeyFromURL は常に空文字を返す。
func (NoOpStorage) KeyFromURL(_ string) string {
	return ""
}
