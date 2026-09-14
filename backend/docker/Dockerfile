# ビルドステージ
FROM golang:1.24 AS builder

WORKDIR /app

# 依存関係のインストール
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*

# go.mod/go.sumを先にコピー（キャッシュ効率化）
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーション・マイグレーション・シードをビルド
# 同じイメージを Cloud Run サービス（api）と Cloud Run Job（migrate / seed）で使い回す
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o migrate ./cmd/migrate \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o seed ./cmd/seed

# 実行ステージ
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata \
    && adduser -D -H app

WORKDIR /app

# ビルドしたバイナリとマイグレーション SQL をコピー
COPY --from=builder /app/main /app/migrate /app/seed ./
COPY --from=builder /app/migrations ./migrations

# 非 root で実行
USER app

# ポートを公開
EXPOSE 8080

# アプリケーションを実行（migrate / seed は --command で切り替える）
CMD ["./main"]
