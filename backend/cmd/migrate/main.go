package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	// .envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found (or relying on system env)")
	}

	// DBのURLを.envファイルから取得
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// DB接続を開く
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Unable to parse DB URL: %v", err)
	}
	defer db.Close()

	// DB接続を確認
	if err := db.Ping(); err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	log.Println("✅ Connected to Database for Migration")

	// schema.sqlファイルを読み込む
	sqlBytes, err := os.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("Failed to read schema.sql: %v", err)
	}
	sqlQuery := string(sqlBytes)

	// SQLを実行
	log.Println("Executing migration...")
	_, err = db.Exec(sqlQuery)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// マイグレーションが成功したことを通知
	log.Println("✅ Migration completed successfully! Table created.")
}
