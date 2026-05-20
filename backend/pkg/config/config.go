package config

import (
	"log"
	"os"
	"strings"
)

const minJWTSecretBytes = 32

// Config はバックエンドが起動時に必要とする環境変数を集約する
type Config struct {
	DatabaseURL     string
	JWTSecret       []byte
	CORSOrigins     []string
	CookieDomain    string
	RecaptchaSecret string
	Environment     string
	MigrationsDir   string
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

	return Config{
		DatabaseURL:     dbURL,
		JWTSecret:       []byte(jwtSecret),
		CORSOrigins:     corsOrigins,
		CookieDomain:    os.Getenv("COOKIE_DOMAIN"),
		RecaptchaSecret: os.Getenv("RECAPTCHA_SECRET_KEY"),
		Environment:     os.Getenv("ENVIRONMENT"),
		MigrationsDir:   migrationsDir,
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
