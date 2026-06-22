// Package storage は S3 互換オブジェクトストレージ（本番 Cloudflare R2 / ローカル MinIO）の
// domain.ImageStorage 実装を提供する。R2 と MinIO はどちらも S3 API 互換のため、
// エンドポイントと認証情報の差し替えだけで同一コードパスで扱える。
package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config は S3 互換ストレージの接続設定
type Config struct {
	Endpoint      string // S3 互換エンドポイント（R2 / MinIO）
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string // 公開配信ベース URL（例: https://images.example.com、末尾スラッシュは無視）
}

// S3Storage は domain.ImageStorage の S3 互換実装
type S3Storage struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

// NewClient は S3Storage を生成する。
// UsePathStyle を有効にすることで MinIO（http://host:9000/bucket/key）にも対応する。
func NewClient(cfg Config) *S3Storage {
	creds := credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
	client := s3.New(s3.Options{
		Region:       cfg.Region,
		Credentials:  creds,
		BaseEndpoint: aws.String(cfg.Endpoint),
		UsePathStyle: true,
	})
	return &S3Storage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}
}

// Save は key にオブジェクトを保存し、公開 URL を返す。
func (s *S3Storage) Save(ctx context.Context, key, contentType string, r io.Reader, _ int64) (string, error) {
	// PutObject は再試行のため seek 可能な body を要求するため、一旦メモリに読み出す
	// （サイズはハンドラ/ユースケースで上限済み）。
	body, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(body),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(int64(len(body))),
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}
	return s.publicBaseURL + "/" + key, nil
}

// Delete は key のオブジェクトを削除する。
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// KeyFromURL は Save が返した公開 URL から key を逆算する。
func (s *S3Storage) KeyFromURL(url string) string {
	prefix := s.publicBaseURL + "/"
	if !strings.HasPrefix(url, prefix) {
		return ""
	}
	return strings.TrimPrefix(url, prefix)
}
