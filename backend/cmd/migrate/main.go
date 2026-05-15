package main

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

type MigrationFile struct {
	path    string
	version int64
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Note: .env file not found (or relying on system env)")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := openAndPingDB(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Connected to Database!")

	if err := ensureSchemaMigrationsTable(ctx, db); err != nil {
		log.Fatalf("schema_migrations table: %v", err)
	}

	dir := os.Getenv("MIGRATIONS_DIR")
	if dir == "" {
		// デフォルトは backend 直下の migrations ディレクトリ
		dir = "migrations"
	}
	dir = filepath.Clean(dir)
	migrationFiles, err := loadMigrationFiles(dir)
	if err != nil {
		log.Fatal(err)
	}

	applied, err := appliedVersions(ctx, db)
	if err != nil {
		log.Fatalf("applied versions: %v", err)
	}

	if err := runMigrations(ctx, db, migrationFiles, applied); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	log.Println("Migrations completed")
}

func openAndPingDB(ctx context.Context, dbURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse DB URL: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return db, nil
}

func loadMigrationFiles(dir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory: %w", err)
	}

	// マイグレーションファイル数は既知なので、事前容量を確保
	migrationFiles := make([]MigrationFile, 0, len(entries))
	seen := make(map[int64]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// マイグレーションファイルは.sqlファイルのみを対象
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		path := filepath.Join(dir, name)
		version, err := versionFromPath(name)
		if err != nil {
			return nil, fmt.Errorf("invalid migration filename %s: %w", name, err)
		}
		if _, ok := seen[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %d: %s", version, name)
		}
		seen[version] = name
		migrationFiles = append(migrationFiles, MigrationFile{
			path:    path,
			version: version,
		})
	}
	slices.SortFunc(migrationFiles, func(i, j MigrationFile) int {
		return cmp.Compare(i.version, j.version)
	})
	return migrationFiles, nil
}

func runMigrations(ctx context.Context, db *sql.DB, migrationFiles []MigrationFile, applied map[int64]bool) error {
	for _, mf := range migrationFiles {
		if applied[mf.version] {
			log.Printf("Already applied, skipping: %s (version=%d)", filepath.Base(mf.path), mf.version)
			continue
		}
		log.Printf("Applying %s ...", filepath.Base(mf.path))
		body, err := os.ReadFile(mf.path)
		if err != nil {
			return fmt.Errorf("read %s: %w", mf.path, err)
		}
		if err := applyMigration(ctx, db, mf.version, body); err != nil {
			return fmt.Errorf("apply %s: %w", mf.path, err)
		}
		applied[mf.version] = true
		log.Printf("Applied migration version %d", mf.version)
	}
	return nil
}

// ensureSchemaMigrationsTable: schema_migrationsテーブルが存在するか確認
// 存在しない場合は作成する
func ensureSchemaMigrationsTable(ctx context.Context, db *sql.DB) error {
	// ロックは取らない:
	//   - 単一インスタンスでの起動時1回呼び出しを前提としているため
	//   - PostgresのAdvisory Lockに依存するとSQLite等への移行時に書き換えが必要になる
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version BIGINT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	return err
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[int64]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func versionFromPath(path string) (int64, error) {
	base := filepath.Base(path)
	i := strings.Index(base, "_")
	if i <= 0 {
		return 0, fmt.Errorf("no version prefix in %q", base)
	}
	v, err := strconv.ParseInt(base[:i], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse version in %q: %w", base, err)
	}
	return v, nil
}

// applyMigration は1ファイル分のSQLをトランザクションで実行する
func applyMigration(ctx context.Context, db *sql.DB, version int64, body []byte) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			log.Printf("transaction rollback: %v", rbErr)
		}
	}()

	// PostgreSQL は複数ステートメントを1回の Exec に渡すことが可能
	// strings.Split(";") による分割はリテラル内の ';' や DO $$ ... $$ 等で壊れるため行わない
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`,
		version,
	); err != nil {
		return err
	}
	return tx.Commit()
}
