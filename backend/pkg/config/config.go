package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	minJWTSecretBytes   = 32
	defaultMailFrom     = "さて、羊に戻るとしよう <noreply@satehits.com>"
	defaultMailQueueSize = 100
)

// Config はバックエンドが起動時に必要とする環境変数を集約する
type Config struct {
	DatabaseURL     string
	JWTSecret       []byte
	CORSOrigins     []string
	CookieDomain    string
	TurnstileSecret string
	Environment     string
	MigrationsDir   string
	ResendAPIKey    string
	MailFromAddress string
	MailQueueSize   int
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
		DatabaseURL:     dbURL,
		JWTSecret:       []byte(jwtSecret),
		CORSOrigins:     corsOrigins,
		CookieDomain:    os.Getenv("COOKIE_DOMAIN"),
		TurnstileSecret: turnstileSecret,
		Environment:     environment,
		MigrationsDir:   migrationsDir,
		ResendAPIKey:    strings.TrimSpace(os.Getenv("RESEND_API_KEY")),
		MailFromAddress: mailFrom,
		MailQueueSize:   mailQueueSize,
	}
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
