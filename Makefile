.PHONY: dev dev-up dev-build dev-down build run test lint docker-build docker-build-dev clean

# 開発環境（Docker）
dev: dev-build dev-up

dev-build:
	docker compose -f docker/docker-compose.local.yml build

dev-up:
	docker compose -f docker/docker-compose.local.yml up

dev-down:
	docker compose -f docker/docker-compose.local.yml down

# ビルド
build:
	go build -o bin/api ./cmd/api

# ローカル実行（Dockerなし）
run:
	go run ./cmd/api

# テスト
test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run

# Dockerイメージビルド
docker-build:
	docker build -f docker/Dockerfile -t satehits-api .

docker-build-dev:
	docker build -f docker/Dockerfile.dev -t satehits-api:dev .

# クリーンアップ
clean:
	rm -rf bin/ tmp/ coverage.out coverage.html
