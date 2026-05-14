.PHONY: dev dev-build dev-down dev-front prod prod-build prod-down build run test test-coverage lint clean migrate

# ===== 開発環境（Docker） =====
dev: dev-build
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
# DATABASE_URL などは backend/.env（コンテナでは /app/.env）を godotenv が読む。
migrate:
	docker compose run --rm backend go run ./cmd/migrate

# ===== ローカル実行（Dockerなし） =====
run:
	cd backend && go run ./cmd/api

# ===== テスト =====
test:
	cd backend && go test -v ./...

test-coverage:
	cd backend && go test -v -race -coverprofile=coverage.out ./...
	cd backend && go tool cover -html=coverage.out -o coverage.html

# ===== Lint =====
lint:
	cd backend && golangci-lint run

# ===== クリーンアップ =====
clean:
	rm -rf backend/bin/ backend/tmp/ backend/coverage.out backend/coverage.html
