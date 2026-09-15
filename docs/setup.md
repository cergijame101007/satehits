# 開発環境セットアップ

## 前提条件

- Docker & Docker Compose
- Go 1.24+（ローカル開発時）
- bun（フロントエンド開発時）
- curl（`make lint` 初回に golangci-lint バイナリを取得するため）

## セットアップ手順

### 1. リポジトリのクローン

```bash
git clone https://github.com/cergijame101007/satehits.git
cd satehits
```

### 2. 環境変数の設定

バックエンドは **`backend/.env`** を参照します（`docker compose` の `env_file` と、`go run` 実行時の `godotenv` 用）。

```bash
cp backend/.env.example backend/.env
```

`backend/.env` を編集して実際の値を設定する。変数の一覧・既定値・必須条件は [`backend/.env.example`](../backend/.env.example) のコメントを正とする。少なくとも次を満たさないと API が起動しない。

- `DATABASE_URL` と `CORS_ORIGINS` が設定されている
- `JWT_SECRET` が 32 バイト以上
- `ENVIRONMENT` が `development` 以外なら `TURNSTILE_SECRET_KEY` が設定されている

フロントエンドは `frontend/` 直下の環境変数ファイルを Astro が読む。`frontend/` で `bun run dev` する場合はテンプレートをコピーする。

```bash
cp frontend/.env.example frontend/.env
```

### 3. 開発サーバーの起動

```bash
# ビルド（イメージ未作成時）+ 起動
make dev-build

# 起動
make dev

# 停止
make dev-down
```

サーバーが `http://localhost:8080` で起動します。

### マイグレーション

`backend/.env` に `DATABASE_URL` が入っている前提です。

```bash
make migrate
```

リポジトリルートから `go run ./backend/cmd/migrate` などカレントが `backend/` でない場合は、`MIGRATIONS_DIR` に `backend/migrations` のようにパスを指定してください。

### シード（開発用）

`admin_users` にオーナー・開発者の初期ユーザを投入する（`backend/cmd/seed`）。

```bash
# backend/.env に SEED_ADMIN_PASSWORD を設定してから
make seed
```

| メール | ロール |
|--------|--------|
| `owner@example.com` | `owner` |
| `dev@example.com` | `developer` |

既存メールはスキップされる。パスワードを変えて再投入したい場合は該当行を削除してから再実行する。

## Makefile コマンド一覧

| コマンド | 説明 |
|----------|------|
| `make dev` | Docker で起動（ホットリロード対応）。ビルドはしない |
| `make dev-build` | 開発用 Docker イメージのビルド（初回・Dockerfile 変更時など） |
| `make dev-down` | Docker コンテナの停止 |
| `make migrate` | マイグレーション実行（一時コンテナで `go run ./cmd/migrate`） |
| `make seed` | 開発用シード投入（`go run ./cmd/seed`） |
| `make prod` | 本番用イメージのビルド＆ `docker-compose.prod.yml` で起動 |
| `make prod-build` | 本番用 Docker イメージのビルドのみ |
| `make prod-down` | 本番 compose の停止 |
| `make dev-front` | フロントエンド開発サーバー（`frontend/` で `bun run dev`） |
| `make run` | バックエンドをローカルで直接起動（Docker なし、`cd backend` 相当） |
| `make build` | バイナリをビルド（`backend/bin/api`） |
| `make test` | テスト実行 |
| `make test-coverage` | カバレッジ付きテスト（`backend/coverage.html` を生成） |
| `make test-integration` | `postgres-test` コンテナを起動して repository の DB 結合テストを実行 |
| `make outbox-flush` | ローカルの `POST /internal/outbox/flush` を呼ぶ（`OUTBOX_FLUSH_ENDPOINT_ENABLED=true` が必要） |
| `make lint` | golangci-lint 実行 |
| `make tools-install` | golangci-lint を `tools/bin` に取得 |
| `make update-holidays` | 祝日データ（内閣府 CSV）を取得して backend / frontend の同梱データを再生成（年 1 回。[`holidays.md`](./holidays.md)） |
| `make clean` | ビルド成果物の削除 |

## ローカル開発（Docker なし）

リポジトリルートから Makefile 経由で起動する場合（内部で `cd backend` します）：

```bash
# 依存関係のダウンロード
go mod download

# 直接実行
make run
```

手動で `backend/` に入って動かす場合：

```bash
cd backend
go mod download
go run ./cmd/api
```

いずれも **`backend/.env`** を置き、カレントが `backend/` であるか、環境変数で `DATABASE_URL` 等を渡してください。

## テスト

```bash
# 全テスト実行
make test

# カバレッジ付き
make test-coverage
```

## Lint

版は [`tools/golangci-lint.version`](../tools/golangci-lint.version)（CI と同期）。設定は [`backend/.golangci.yml`](../backend/.golangci.yml)。

```bash
make tools-install   # 初回など
make lint
```

## トラブルシューティング

### Docker コンテナが起動しない

```bash
# ログを確認
docker compose logs

# コンテナを削除して再ビルドして起動
make dev-down
make dev-build
make dev
```

### データベースに接続できない

- `backend/.env` の `DATABASE_URL` が正しいか確認（`docker compose` は `env_file: ./backend/.env` で読み込みます）
- Supabase のプロジェクトが起動しているか確認
- ネットワーク接続を確認
