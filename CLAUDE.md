# CLAUDE.md — さて、羊に戻るとしよう

飲食店「さて、羊に戻るとしよう」のホームページ兼テーブル予約管理システム。

## 1. プロジェクト概要

**顧客向け機能**
- 残り食数の確認・月間スケジュール（カレンダー）の閲覧
- オンライン予約の申請
- お取り引き先紹介ページの閲覧

**オーナー向け機能**
- 予約一覧の確認・承認・拒否
- 予約の手動登録（Instagram / 電話 / 知人経由）
- スケジュール設定（通常 / 朝営業 / イベント / 特別メニュー / 臨時休業）
- お取り引き先の管理

**技術スタック**

| レイヤー | 技術 |
|----------|------|
| フロントエンド | Astro 5 + React 19 (Islands Architecture), TypeScript, Tailwind CSS v4, Bun |
| バックエンド | Go 1.24, `net/http`, pgx/v5 |
| データベース | Supabase (PostgreSQL) |
| インフラ | Cloudflare Pages (frontend), Google Cloud Run (backend), Docker |
| メール配信 | Resend |
| CI/CD | GitHub Actions |

---

## 2. AI 利用ルール

**バックエンド（Go）: AI によるコード生成は原則しない**
- オーナー自身が実装する。Go の理解を深めることが目的
- AI はコードレビュー・質問回答・ヒント提示のみ行う
- AI がコードを提案する場合は「なぜその書き方が Go らしいか」を説明する

**フロントエンド: AI によるコード生成 OK**
- 設計方針（Islands Architecture、責務分離）に従うこと
- モックを API 呼び出しに置き換える作業は AI が担当してよい

**共通ルール**
- コミット: `type: 日本語の説明`（例: `feat: 予約フォームを追加`）
- コミットは1目的1コミット、絵文字・複数行説明なし
- コミュニケーションは日本語で

---

## 3. ディレクトリ構成

<!-- AUTO-UPDATE: DIRECTORY -->
<!-- ページ・コンポーネント・バックエンドのファイルが追加・削除されたらこのセクションを更新すること -->

```
satehits/
├── frontend/                    # Astro + React フロントエンド
│   └── src/
│       ├── components/
│       │   ├── astro/           # 静的コンポーネント（JS なし）
│       │   │   ├── Header.astro
│       │   │   ├── Footer.astro
│       │   │   ├── HeroSection.astro
│       │   │   ├── SectionReveal.astro
│       │   │   ├── AttentionSection.astro
│       │   │   └── AccordionItem.astro
│       │   └── react/           # React Islands（インタラクティブ）
│       │       ├── LoginForm.tsx
│       │       ├── ReservationForm.tsx
│       │       ├── ReservationTable.tsx
│       │       ├── ReservationCreateForm.tsx
│       │       ├── ScheduleCalendar.tsx
│       │       ├── DashboardSummary.tsx
│       │       └── SupplierManager.tsx
│       ├── layouts/
│       │   ├── BaseLayout.astro  # 顧客向けレイアウト
│       │   └── AdminLayout.astro # 管理者向けレイアウト（auth チェック付き）
│       ├── pages/               # ファイルベースルーティング
│       │   ├── index.astro
│       │   ├── reservation.astro
│       │   ├── reservation/complete.astro
│       │   ├── suppliers.astro
│       │   └── admin/
│       │       ├── index.astro
│       │       ├── login.astro
│       │       ├── reservations.astro
│       │       ├── reservations/new.astro
│       │       ├── schedules.astro
│       │       └── suppliers.astro
│       ├── types/
│       │   └── reservation.ts   # 型定義（フロント・バック共通）
│       ├── mocks/
│       │   └── reservation.ts   # API 未実装時のモックデータ
│       └── styles/
│           └── global.css       # Tailwind v4 カスタムテーマ
├── backend/                     # Go バックエンド
│   ├── cmd/
│   │   ├── api/main.go          # エントリーポイント・DI
│   │   └── migrate/main.go      # DB マイグレーション実行
│   ├── internal/
│   │   ├── domain/
│   │   │   └── reservation.go   # エンティティ + Repository インターフェース
│   │   ├── handler/
│   │   │   ├── reservation.go   # HTTP ハンドラー
│   │   │   └── response.go      # JSON レスポンスヘルパー
│   │   └── repository/
│   │       └── reservation.go   # PostgreSQL 実装
│   ├── docker/
│   │   ├── Dockerfile           # 本番用マルチステージビルド
│   │   └── Dockerfile.dev       # 開発用（Air ホットリロード）
│   └── schema.sql               # DB スキーマ
├── docs/                        # 設計ドキュメント（日本語）
├── .github/workflows/           # CI/CD
├── .cursor/rules/               # Cursor AI ルール
├── docker-compose.yml           # 開発用 Docker
├── docker-compose.prod.yml      # 本番用 Docker
├── Makefile                     # 開発コマンド
└── .env.example                 # 環境変数テンプレート
```

> **注意**: `backend/` のディレクトリ構成は現在最小限。`docs/architecture.md` に定義された目標構成（`internal/presentation/`, `internal/application/`, `internal/domain/`, `internal/infrastructure/`）に向けて段階的に拡張する予定。

---

## 4. フロントエンド設計

### 設計意図 — Islands Architecture

ページの大部分を静的 HTML で配信し、インタラクティブな部分（Islands）だけ React でハイドレーションする。

- トップページ・取引先紹介など: `.astro` のみ → **JS 0KB 配信**
- 予約フォーム・管理画面など: React Islands → インタラクション実現
- Core Web Vitals に有利、Cloudflare Pages の静的配信と相性良

### 責務分離

| 場所 | 責務 |
|------|------|
| `.astro` ページ/レイアウト | ルーティング、静的 HTML の構造、SEO、OGP、レイアウト |
| `.tsx` React Islands | ユーザーインタラクション（フォーム入力・送信、テーブル操作、カレンダー編集） |
| `layouts/BaseLayout.astro` | 顧客向けページの HTML シェル（meta, font, global CSS） |
| `layouts/AdminLayout.astro` | 管理画面のシェル（ナビ、`localStorage` の `auth_token` チェック、ログアウト） |
| `types/reservation.ts` | 型定義。フロントとバックが共通で使う型はここに集約 |
| `mocks/reservation.ts` | API 未実装時のモックデータ・関数。API 実装後に `lib/api.ts` の呼び出しに置き換える |
| `styles/global.css` | Tailwind v4 `@theme`（カスタムカラー `#43676B`、Noto Serif JP、グラデーション、アニメーション） |

### ハイドレーション戦略

| ディレクティブ | タイミング | 使用箇所 |
|---------------|-----------|---------|
| `client:load` | ページ読み込み時に即時 | 予約フォーム、ログインフォーム、管理画面全般 |
| `client:visible` | 要素が画面内に入ったとき | 取引先カード（スクロール後に表示） |
| `client:idle` | ブラウザがアイドル状態のとき | ダッシュボードのウィジェット |

**パスエイリアス**: `@/*` → `src/*`（`tsconfig.json` で設定）

---

## 5. バックエンド設計

### 設計意図 — レイヤードアーキテクチャ + Repository + DI

| 観点 | 選択 | 理由 |
|------|------|------|
| 全体構成 | レイヤード | プロダクト規模に適切、学習コスト低い |
| データアクセス | Repository パターン | DB 変更への耐性、テスト時のモック化 |
| 依存管理 | DI（依存性注入） | テスト容易性、疎結合 |
| ビジネスロジック | UseCase + Domain Service | 責務の明確化、Handler の肥大化防止 |

採用しないもの: フル DDD（過剰）、Clean Architecture（ボイラープレート増）、CQRS（規模に不要）

### 各レイヤーの責務

```
Presentation  →  Application  →  Domain  ←  Infrastructure
(handler)        (usecase/dto)   (entity/    (repository実装
                                 repository  / 外部APIクライアント)
                                 interface
                                 / service)
```

| レイヤー | ディレクトリ | 責務 |
|---------|------------|------|
| Presentation | `internal/presentation/handler/` | HTTP リクエスト受付、レスポンス返却、ミドルウェア（JWT 認証・CORS・ロギング） |
| Application | `internal/application/usecase/` | ユースケースのオーケストレーション（reCAPTCHA検証 → バリデーション → 空き確認 → 保存 → メール送信） |
| Application | `internal/application/dto/` | リクエスト / レスポンスの変換 |
| Domain | `internal/domain/entity/` | ビジネスエンティティ、バリデーション、ステータス遷移ルール（例: `CanTransitionTo()`） |
| Domain | `internal/domain/repository/` | Repository インターフェース定義（DB に依存しない） |
| Domain | `internal/domain/service/` | 複数エンティティにまたがるドメインロジック（空き状況計算、営業時間判定） |
| Infrastructure | `internal/infrastructure/persistence/` | Repository 実装（Supabase / pgx） |
| Infrastructure | `internal/infrastructure/external/` | 外部 API クライアント（reCAPTCHA, Resend） |

**依存ルール**
- Domain 層は他のレイヤーに依存しない（最も安定）
- Infrastructure 層が Domain のインターフェースを実装（依存性逆転）
- `cmd/api/main.go` で全依存を解決・注入

**禁止事項**
- Web フレームワーク（Echo / Gin）は使用禁止。標準ライブラリ `net/http` のみ
- `interface{}` / `any` の多用禁止（静的型付けを最大化）
- `err != nil` のエラーハンドリングを省略しない

---

## 6. ページ・エンドポイント一覧

<!-- AUTO-UPDATE: ROUTES -->
<!-- ページやエンドポイントが追加・変更されたらこのセクションを更新すること -->

### フロントエンド ページ一覧

| URL | ファイル | レイアウト | React Island |
|-----|---------|-----------|-------------|
| `/` | `pages/index.astro` | BaseLayout | なし（静的） |
| `/reservation` | `pages/reservation.astro` | BaseLayout | `ReservationForm` |
| `/reservation/complete` | `pages/reservation/complete.astro` | BaseLayout | なし |
| `/suppliers` | `pages/suppliers.astro` | BaseLayout | なし（静的） |
| `/admin` | `pages/admin/index.astro` | AdminLayout | `DashboardSummary` |
| `/admin/login` | `pages/admin/login.astro` | なし | `LoginForm` |
| `/admin/reservations` | `pages/admin/reservations.astro` | AdminLayout | `ReservationTable` |
| `/admin/reservations/new` | `pages/admin/reservations/new.astro` | AdminLayout | `ReservationCreateForm` |
| `/admin/schedules` | `pages/admin/schedules.astro` | AdminLayout | `ScheduleCalendar` |
| `/admin/suppliers` | `pages/admin/suppliers.astro` | AdminLayout | `SupplierManager` |

### バックエンド API エンドポイント一覧

| メソッド | パス | 概要 | 実装状況 |
|---------|------|------|---------|
| `GET` | `/` | ヘルスチェック | 実装済み |
| `GET` | `/reservations` | 予約一覧取得 | 実装済み（最小限） |
| `POST` | `/reservations` | 予約作成 | 実装済み（最小限） |
| `GET` | `/api/v1/reservations/availability` | 日付別空き確認 | 未実装 |
| `GET` | `/api/v1/admin/reservations` | 管理者：予約一覧 | 未実装 |
| `POST` | `/api/v1/admin/reservations` | 管理者：予約手動登録 | 未実装 |
| `PATCH` | `/api/v1/admin/reservations/:id/status` | 予約ステータス更新 | 未実装 |
| `POST` | `/api/v1/admin/login` | 管理者ログイン | 未実装 |
| `POST` | `/api/v1/admin/logout` | 管理者ログアウト | 未実装 |
| `GET` | `/api/v1/admin/schedules` | 月間スケジュール取得 | 未実装 |
| `POST` | `/api/v1/admin/schedules` | スケジュール設定 | 未実装 |
| `GET` | `/api/v1/admin/suppliers` | 取引先一覧取得 | 未実装 |
| `POST` | `/api/v1/admin/suppliers` | 取引先作成 | 未実装 |
| `PUT` | `/api/v1/admin/suppliers/:id` | 取引先更新 | 未実装 |
| `DELETE` | `/api/v1/admin/suppliers/:id` | 取引先削除 | 未実装 |
| `PUT` | `/api/v1/admin/suppliers/order` | 取引先並び順更新 | 未実装 |

---

## 7. 開発コマンド

### Makefile（ルート）

```bash
make dev          # バックエンド開発サーバー起動（Docker + Air ホットリロード）
make dev-build    # バックエンド Docker イメージビルド
make dev-down     # バックエンド Docker 停止
make dev-front    # フロントエンド開発サーバー起動（bun run dev）
make prod         # 本番用 Docker 起動
make build        # Go バイナリビルド（bin/api）
make run          # Go サーバー直接起動
make test         # バックエンドテスト実行
make test-coverage # カバレッジ付きテスト
make lint         # golangci-lint
make clean        # ビルド成果物の削除
```

### フロントエンド（`frontend/`）

```bash
bun run dev       # 開発サーバー起動（localhost:4321）
bun run build     # 型チェック + 本番ビルド
bun run preview   # ビルド結果のプレビュー
bun run lint      # ESLint
bun run lint:fix  # ESLint 自動修正
bun run format    # Prettier フォーマット
bun run test      # Vitest（ウォッチモード）
bun run test:run  # Vitest（単発実行）
```

---

## 8. 環境変数

`.env.example` をコピーして `.env` を作成する。

```env
# バックエンド
DATABASE_URL=postgresql://user:password@host:5432/dbname
JWT_SECRET=your-jwt-secret
RECAPTCHA_SECRET_KEY=your-recaptcha-secret-key
ENVIRONMENT=development

# フロントエンド
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_RECAPTCHA_SITE_KEY=your-recaptcha-site-key
```

---

## 9. CI/CD

| ワークフロー | トリガー | ジョブ |
|------------|---------|-------|
| `.github/workflows/backend.yml` | `backend/**` の変更を `main`/`develop` に push/PR | lint → test → build → docker-build |
| `.github/workflows/frontend.yml` | `frontend/**` の変更を `main`/`develop` に push/PR | lint → test → build → アーティファクトアップロード |

- フロントエンドのビルド成果物は `frontend/dist/` にアップロードされ Cloudflare Pages へデプロイ
- バックエンドは Google Cloud Run へデプロイ

---

## 10. ドキュメント

詳細な設計はすべて `docs/` に記載されている。

| ファイル | 内容 |
|---------|------|
| [docs/setup.md](docs/setup.md) | 開発環境の構築手順 |
| [docs/domain_knowledge.md](docs/domain_knowledge.md) | 店舗情報、営業ルール、用語集 |
| [docs/architecture.md](docs/architecture.md) | レイヤー構成、責務分離、DI、エラーハンドリング戦略 |
| [docs/architecture_decision_records.md](docs/architecture_decision_records.md) | ADR（技術選定理由：Astro、Go、Cloudflare 等） |
| [docs/api_design.md](docs/api_design.md) | API 仕様の概要 |
| [docs/openapi.yaml](docs/openapi.yaml) | OpenAPI 3.0 形式の API 仕様 |
| [docs/sequence.md](docs/sequence.md) | 主要機能のシーケンス図 |
| [docs/screen_transition.md](docs/screen_transition.md) | 画面一覧・遷移図・ワイヤーフレーム |
| [docs/table_design.md](docs/table_design.md) | テーブル設計 |
| [docs/infrastructure.md](docs/infrastructure.md) | インフラ構成、環境設定、CI/CD 詳細 |
| [docs/test_design.md](docs/test_design.md) | テスト方針・テストケース一覧 |
| [docs/use_case.md](docs/use_case.md) | ユースケース一覧 |
| [docs/data_flow.md](docs/data_flow.md) | データフロー図 |
