# 開発環境セットアップ

## 前提条件

- Docker & Docker Compose
- Go 1.22+（ローカル開発時）
- golangci-lint（ローカルでlint実行時）

## セットアップ手順

### 1. リポジトリのクローン

```bash
git clone https://github.com/cergijame101007/satehits.git
cd satehits
```

### 2. 環境変数の設定

```bash
cp .env.example .env
```

`.env` を編集して実際の値を設定：

| 変数名 | 説明 |
|--------|------|
| `DATABASE_URL` | Supabase の接続URL |
| `JWT_SECRET` | JWT署名用のシークレットキー |
| `RECAPTCHA_SECRET_KEY` | Google reCAPTCHA v3 のシークレットキー |
| `ENVIRONMENT` | `development` または `production` |

### 3. 開発サーバーの起動

```bash
# Docker で起動（ホットリロード対応）
make dev

# 停止
make dev-down
```

サーバーが `http://localhost:8080` で起動します。

## Makefile コマンド一覧

| コマンド | 説明 |
|----------|------|
| `make dev` | Docker でビルド＆起動（ホットリロード対応） |
| `make dev-up` | Docker で起動のみ（ビルド済みの場合） |
| `make dev-build` | Docker イメージのビルドのみ |
| `make dev-down` | Docker コンテナの停止 |
| `make run` | ローカルで直接起動（Docker なし） |
| `make build` | バイナリをビルド |
| `make test` | テスト実行 |
| `make lint` | golangci-lint 実行 |
| `make clean` | ビルド成果物の削除 |

## ローカル開発（Docker なし）

Docker を使わずに直接実行する場合：

```bash
# 依存関係のダウンロード
go mod download

# 直接実行
make run

# または
go run ./cmd/api
```

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
docker compose -f docker/docker-compose.local.yml logs

# コンテナを削除して再ビルド
make dev-down
make dev
```

### データベースに接続できない

- `.env` の `DATABASE_URL` が正しいか確認
- Supabase のプロジェクトが起動しているか確認
- ネットワーク接続を確認
