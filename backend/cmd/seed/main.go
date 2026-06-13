package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

var adminUsers = []struct {
	email string
	role  string
}{
	{email: "owner@example.com", role: "owner"},
	{email: "dev@example.com", role: "developer"},
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found (or relying on system env)")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	password := os.Getenv("SEED_ADMIN_PASSWORD")
	if password == "" {
		log.Fatal("SEED_ADMIN_PASSWORD is not set")
	}

	ctx := context.Background()
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("parse DB URL: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	log.Println("Connected to Database!")

	if err := seedAdminUsers(ctx, db, password); err != nil {
		log.Fatalf("seed: %v", err)
	}

	log.Println("Seed completed")
}

func seedAdminUsers(ctx context.Context, db *sql.DB, plainPassword string) error {
	for _, u := range adminUsers {
		hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcryptCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		res, err := db.ExecContext(ctx, `
INSERT INTO admin_users (email, password_hash, role)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO NOTHING`,
			u.email, string(hash), u.role,
		)
		if err != nil {
			return fmt.Errorf("insert %s: %w", u.email, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("rows affected %s: %w", u.email, err)
		}
		if n == 1 {
			log.Printf("inserted admin_user %s (%s)", u.email, u.role)
			continue
		}
		log.Printf("skipped admin_user %s (already exists)", u.email)
	}
	return nil
}
