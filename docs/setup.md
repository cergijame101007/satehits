# 開発環境セットアップ

## 前提条件

- Docker & Docker Compose
- Go 1.24+（ローカル開発時）
- golangci-lint（ローカルでlint実行時）

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

`backend/.env` を編集して実際の値を設定：

| 変数名 | 説明 |
|--------|------|
| `DATABASE_URL` | Supabase の接続URL |
| `JWT_SECRET` | JWT署名用のシークレットキー |
| `RECAPTCHA_SECRET_KEY` | Google reCAPTCHA v3 のシークレットキー |
| `ENVIRONMENT` | `development` または `production` |
| `MIGRATIONS_DIR` | （任意）マイグレーション SQL のディレクトリで未設定時は `migrations`（実行時のカレントディレクトリ基準） |

フロントエンド用の `NEXT_PUBLIC_*` などは、リポジトリ直下の [`.env.example`](../.env.example) に記載があります。`frontend/` で `bun run dev` する場合は、必要な変数を `frontend/.env` などに置いてください。

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
| `make build` | バイナリをビルド |
| `make test` | テスト実行 |
| `make lint` | golangci-lint 実行 |
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

```bash
# golangci-lint のインストール（初回のみ）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# lint 実行
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
