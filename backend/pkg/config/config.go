package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	minJWTSecretBytes    = 32
	defaultMailFrom      = "さて、羊に戻るとしよう <noreply@satehits.com>"
	defaultMailQueueSize = 100

	defaultLoginRateLimitEmailMax      = 5
	defaultLoginRateLimitIPMax         = 20
	defaultLoginRateLimitWindowMinutes = 15
	defaultTrustedProxyHops            = 1
)

// Config はバックエンドが起動時に必要とする環境変数を集約する
type Config struct {
	DatabaseURL      string
	JWTSecret        []byte
	CORSOrigins      []string
	CookieDomain     string
	TurnstileSecret  string
	Environment      string
	MigrationsDir    string
	ResendAPIKey     string
	MailFromAddress  string
	MailQueueSize    int
	TrustedProxyHops int
	LoginRateLimit   LoginRateLimitConfig
	Storage          StorageConfig
}

// LoginRateLimitConfig はログイン失敗のレートリミットしきい値
type LoginRateLimitConfig struct {
	EmailMax      int
	IPMax         int
	WindowMinutes int
}

// StorageConfig は取引先画像などを保存する S3 互換ストレージ（R2 / MinIO）の設定。
// 必須項目が揃っていない場合は Enabled=false となり、画像アップロードは無効化される。
type StorageConfig struct {
	Enabled       bool
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string
}

// Load は環境変数を読み込み、必須項目の検証に失敗したら log.Fatal する
func Load() Config {
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minJWTSecretBytes {
		log.Fatalf("JWT_SECRET must be at least %d bytes, got %d", minJWTSecretBytes, len(jwtSecret))
	}

	corsOrigins := parseCSV(os.Getenv("CORS_ORIGINS"))
	if len(corsOrigins) == 0 {
		log.Fatal("CORS_ORIGINS is not set")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	migrationsDir := os.Getenv("MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	environment := os.Getenv("ENVIRONMENT")
	turnstileSecret := os.Getenv("TURNSTILE_SECRET_KEY")
	if environment != "development" && turnstileSecret == "" {
		log.Fatal("TURNSTILE_SECRET_KEY is required when ENVIRONMENT is not development")
	}

	mailFrom := strings.TrimSpace(os.Getenv("MAIL_FROM_ADDRESS"))
	if mailFrom == "" {
		mailFrom = defaultMailFrom
	}

	mailQueueSize := defaultMailQueueSize
	if raw := strings.TrimSpace(os.Getenv("MAIL_QUEUE_SIZE")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			log.Fatalf("MAIL_QUEUE_SIZE must be a positive integer, got %q", raw)
		}
		mailQueueSize = n
	}

	return Config{
		DatabaseURL:      dbURL,
		JWTSecret:        []byte(jwtSecret),
		CORSOrigins:      corsOrigins,
		CookieDomain:     os.Getenv("COOKIE_DOMAIN"),
		TurnstileSecret:  turnstileSecret,
		Environment:      environment,
		MigrationsDir:    migrationsDir,
		ResendAPIKey:     strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		MailFromAddress:  mailFrom,
		MailQueueSize:    mailQueueSize,
		TrustedProxyHops: positiveIntEnv("TRUSTED_PROXY_HOPS", defaultTrustedProxyHops),
		LoginRateLimit: LoginRateLimitConfig{
			EmailMax:      positiveIntEnv("LOGIN_RATE_LIMIT_EMAIL_MAX", defaultLoginRateLimitEmailMax),
			IPMax:         positiveIntEnv("LOGIN_RATE_LIMIT_IP_MAX", defaultLoginRateLimitIPMax),
			WindowMinutes: positiveIntEnv("LOGIN_RATE_LIMIT_WINDOW_MINUTES", defaultLoginRateLimitWindowMinutes),
		},
		Storage: loadStorageConfig(),
	}
}

func positiveIntEnv(key string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		log.Fatalf("%s must be a positive integer, got %q", key, raw)
	}
	return n
}

// loadStorageConfig は STORAGE_* 環境変数を読み込む。
// 必須項目（endpoint / bucket / access key / secret key / public base url）が
// すべて揃っている場合のみ Enabled=true となる。
func loadStorageConfig() StorageConfig {
	region := strings.TrimSpace(os.Getenv("STORAGE_REGION"))
	if region == "" {
		region = "auto"
	}
	cfg := StorageConfig{
		Endpoint:      strings.TrimSpace(os.Getenv("STORAGE_ENDPOINT")),
		Region:        region,
		Bucket:        strings.TrimSpace(os.Getenv("STORAGE_BUCKET")),
		AccessKey:     strings.TrimSpace(os.Getenv("STORAGE_ACCESS_KEY")),
		SecretKey:     strings.TrimSpace(os.Getenv("STORAGE_SECRET_KEY")),
		PublicBaseURL: strings.TrimSpace(os.Getenv("STORAGE_PUBLIC_BASE_URL")),
	}
	cfg.Enabled = cfg.Endpoint != "" && cfg.Bucket != "" &&
		cfg.AccessKey != "" && cfg.SecretKey != "" && cfg.PublicBaseURL != ""
	return cfg
}

func parseCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
