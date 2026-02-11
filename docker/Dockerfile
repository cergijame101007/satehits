# ビルドステージ
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 依存関係のインストール
RUN apk --no-cache add ca-certificates tzdata

# go.mod/go.sumを先にコピー（キャッシュ効率化）
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピー
COPY . .

# アプリケーションをビルド
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api

# 実行ステージ
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# ビルドしたバイナリをコピー
COPY --from=builder /app/main .

# ポートを公開
EXPOSE 8080

# アプリケーションを実行
CMD ["./main"]
