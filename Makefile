.PHONY: dev dev-build dev-down dev-front prod prod-build prod-down build run test test-coverage test-integration outbox-flush lint tools-install clean migrate seed update-holidays

# ===== 開発ツール =====
TOOLS_DIR := $(CURDIR)/tools
TOOLS_BIN := $(TOOLS_DIR)/bin
GOLANGCI_LINT_VERSION := $(shell tr -d '[:space:]' < $(TOOLS_DIR)/golangci-lint.version)
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint

$(GOLANGCI_LINT):
	@mkdir -p $(TOOLS_BIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(TOOLS_BIN) $(GOLANGCI_LINT_VERSION)

tools-install: $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) version

# ===== 開発環境（Docker） =====
dev:
	docker compose up

dev-build:
	docker compose build

dev-down:
	docker compose down

# ===== フロントエンド開発 =====
dev-front:
	cd frontend && bun run dev

# ===== 本番シミュレーション =====
prod: prod-build
	docker compose -f docker-compose.prod.yml up

prod-build:
	docker compose -f docker-compose.prod.yml build

prod-down:
	docker compose -f docker-compose.prod.yml down

# ===== バックエンドビルド =====
build:
	cd backend && go build -o bin/api ./cmd/api

# ===== マイグレーション =====
# DATABASE_URL / MIGRATIONS_DIR などは docker compose の env_file（./backend/.env）で注入
migrate:
	docker compose run --rm backend go run ./cmd/migrate

seed:
	docker compose run --rm backend go run ./cmd/seed

# ===== ローカル実行（Dockerなし） =====
run:
	cd backend && go run ./cmd/api

# ===== テスト =====
test:
	cd backend && go test -v ./...

test-coverage:
	cd backend && go test -v -race -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

# Outbox 等の repository integration test（postgres-test が必要）
TEST_DATABASE_URL ?= postgres://satehits:satehits@localhost:5433/satehits_test?sslmode=disable
test-integration:
	docker compose up -d --wait postgres-test
	cd backend && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -count=1 ./internal/repository/ -run 'TestEmailOutbox|TestRefreshTokenRepository'

# ローカルで Outbox flush を手動実行（OUTBOX_FLUSH_ENDPOINT_ENABLED=true が必要）
outbox-flush:
	curl -sS -X POST http://localhost:8080/internal/outbox/flush

# ===== 祝日データの年次更新 =====
# 内閣府 CSV → backend の embed 用 CSV と frontend の JSON を再生成（docs/holidays.md）
update-holidays:
	./scripts/update-holidays.sh

# ===== Lint =====
# 版は tools/golangci-lint.version（CI と同期）
lint: $(GOLANGCI_LINT)
	cd backend && $(GOLANGCI_LINT) run

# ===== クリーンアップ =====
clean:
	rm -rf backend/bin/ backend/tmp/ backend/coverage.out backend/coverage.html
