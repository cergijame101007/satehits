# CLAUDE.md — さて、羊に戻るとしよう

飲食店「さて、羊に戻るとしよう」のホームページ兼テーブル予約管理システム。

## 1. プロジェクト概要

**顧客向け機能**
- 残り食数の確認・月間スケジュール（カレンダー）の閲覧
- オンライン予約の申請
- お取り引き先紹介ページの閲覧

**オーナー向け機能**
- 予約一覧の確認・承認・拒否
- 予約の手動登録（Instagram / 電話 / ウォークイン / その他）
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

- フロントエンド・バックエンドともに AI によるコード生成 OK
- 設計方針に従うこと（フロント: Islands Architecture・責務分離、バック: レイヤードアーキテクチャ・Repository・DI）
- モックを API 呼び出しに置き換える作業は AI が担当してよい

**共通ルール**
- コミット: `type: 日本語の説明`（例: `feat: 予約フォームを追加`）
- コミットは1目的1コミット、絵文字・複数行説明なし
- コミュニケーションは日本語で

**Docs 優先（Cursor / Copilot / Claude 共通）**

API・DB・バリデーション・ドメインルール・エラー形式に触れるときは、**コードを書く・直す・レビューする前に**関連 docs を読む。記憶や推測だけで実装しない。

| 作業 | 先に読むもの |
|------|----------------|
| HTTP（パス・メソッド・ステータス・ボディ・デフォルト値） | `docs/api_design.md` → `docs/openapi.yaml` |
| テーブル・NULL・CHECK・マイグレーション | `docs/table_design.md`、該当 `backend/migrations/*.sql` |
| 店舗・営業・用語 | `docs/domain_knowledge.md` |
| レイヤー責務・DI | `docs/architecture.md` |
| Go テスト | `docs/coding_rule/go_testing.md` |
| 画面・遷移 | `docs/screen_transition.md` |

Copilot 等の自動レビュー指摘は**仮説**とする。上表の docs と変更コードの**利用箇所**（handler・usecase・テスト）で確認してから直す。**docs と矛盾する指摘は採用しない**（必要なら docs 更新を先に提案）。戻り値が未使用なら過剰な refactor はしない。docs と実装がずれたら、どちらを正とするか決めてから片方を直す。

**バックエンド再発防止（docs 確認済みのみ追記）**

- `schedule_type` とイベント欄（正は `docs/api_design.md` の表）: `normal` / `morning` / `closed` はイベント欄を送ると 400（`VALIDATION_ERROR`）。`event` / `external_event` は名称必須・説明任意。`external_event` は capacity 0 固定・時刻 NULL。`special_menu` は任意（Copilot が「必須」と言っても docs 優先）
- Outbox Dispatcher（正は ADR-014）: claim → 送信 → 記録は lease 方式で**独立コミット**。送信を DB Tx の中に戻さない。`attempt_count` は claim 時に +1（失敗回数ではない）。401/403 は `ReleaseClaim` でバッチ中断
- 店舗定例の合成（正は `docs/domain_knowledge.md` §6 / `docs/holidays.md`）: `daily_schedules` 行なし日は **祝日 → 曜日** の順。祝日は曜日にかかわらず `normal`（日曜祝日も朝営業なし）。祝日データは同梱 CSV（`internal/domain/holiday`）で、外部へ取りに行かない。FE の `storeDefaultSchedule.ts` も同じデータ（`src/data/holidays.json`）で整合させる

---

## 3. ディレクトリ構成

<!-- AUTO-UPDATE: DIRECTORY -->
<!-- ページ・コンポーネント・バックエンドのファイルが追加・削除されたらこのセクションを更新すること -->

テストファイル（`*_test.go` / `*.test.ts(x)`）は対象ファイルの直後に置く。

```
satehits/
├── frontend/                                   # Astro + React フロントエンド
│   ├── .env.example                            # 環境変数テンプレート（PUBLIC_ のみクライアント公開）
│   ├── .gitignore                              # frontend 固有の除外設定
│   ├── .prettierrc                             # Prettier 設定
│   ├── README.md                               # フロントエンドの README
│   ├── astro.config.mjs                        # Astro 設定（静的出力・React 統合・Tailwind Vite プラグイン）
│   ├── bun.lock                                # Bun のロックファイル
│   ├── eslint.config.mjs                       # ESLint 設定
│   ├── package.json                            # 依存パッケージと bun run スクリプト
│   ├── tsconfig.json                           # TypeScript 設定（パスエイリアス `@/*`）
│   ├── vitest.config.ts                        # Vitest 設定（jsdom・カバレッジ）
│   ├── public/                                 # そのまま配信する静的ファイル
│   │   ├── _headers                            # Cloudflare Pages のレスポンスヘッダ設定
│   │   ├── favicon.ico                         # ファビコン（ICO）
│   │   ├── favicon.svg                         # ファビコン（SVG）
│   │   ├── robots.txt                          # クローラ設定
│   │   └── images/
│   │       ├── top-icon.svg                    # ロゴアイコン
│   │       └── top-view-high.jpg               # ヒーロー画像
│   └── src/
│       ├── env.d.ts                            # Astro クライアント型の参照
│       ├── components/
│       │   ├── astro/                          # 静的コンポーネント（JS なし）
│       │   │   ├── AccordionItem.astro         # 開閉式セクション（details ベース）
│       │   │   ├── AttentionSection.astro      # トップ「ご来店の方へ」（PublicScheduleCalendar を client:load で埋め込む）
│       │   │   ├── Footer.astro                # 顧客向けフッター
│       │   │   ├── Header.astro                # 顧客向けヘッダー・ナビ（予約する / お取り引き先）
│       │   │   ├── HeroSection.astro           # トップのヒーロー
│       │   │   └── SectionReveal.astro         # スクロール時のフェードイン演出
│       │   └── react/                          # React Islands（インタラクティブ）
│       │       ├── DashboardSummary.tsx        # 管理ダッシュボード（今日・明日のサマリー・未対応注意帯・クイックリンク）
│       │       ├── DashboardSummary.test.tsx
│       │       ├── DatePickerField.tsx         # ポップアップ日付ピッカー（予約登録フォーム用）
│       │       ├── DatePickerField.test.tsx
│       │       ├── LoginForm.tsx               # 管理者ログインフォーム
│       │       ├── LoginForm.test.tsx
│       │       ├── MonthCalendar.tsx           # 月カレンダー表示（予約一覧・スケジュール設定・日付ピッカー共通）
│       │       ├── MonthCalendar.test.tsx
│       │       ├── PublicScheduleCalendar.tsx  # トップの営業カレンダー（公開スケジュール）
│       │       ├── PublicScheduleCalendar.test.tsx
│       │       ├── ReservationCreateForm.tsx   # 管理者の予約手動登録フォーム（提供数超過の確認付き）
│       │       ├── ReservationCreateForm.test.tsx
│       │       ├── ReservationCreateForm.capacityConfirm.test.tsx # 提供数超過確認のテスト
│       │       ├── ReservationForm.tsx         # 顧客向け予約申請フォーム（Turnstile）
│       │       ├── ReservationForm.test.tsx
│       │       ├── ReservationTable.tsx        # 管理者の予約一覧（カレンダー・カード・承認 / 拒否 / キャンセル）
│       │       ├── ReservationTable.test.tsx
│       │       ├── ScheduleCalendar.tsx        # 管理者のスケジュール設定カレンダー
│       │       ├── ScheduleCalendar.test.tsx
│       │       ├── SupplierList.tsx            # 公開の取引先一覧
│       │       ├── SupplierList.test.tsx
│       │       ├── SupplierManager.tsx         # 管理者の取引先管理（作成・編集・削除・並び替え・画像）
│       │       ├── SupplierManager.test.tsx
│       │       ├── TurnstileWidget.tsx         # Cloudflare Turnstile ウィジェット
│       │       ├── TurnstileWidget.test.tsx
│       │       └── ui/                         # 汎用 UI 部品
│       │           ├── Alert.tsx               # エラー / 警告 / 情報の帯
│       │           ├── Alert.test.tsx
│       │           ├── Button.tsx              # ボタン（variant / size）
│       │           ├── Button.test.tsx
│       │           ├── ButtonLink.tsx          # ボタン見た目のリンク
│       │           ├── ButtonLink.test.tsx
│       │           ├── Card.tsx                # カード枠
│       │           ├── Card.test.tsx
│       │           ├── ConfirmModal.tsx        # 確認ダイアログ（理由入力欄の差し込み可）
│       │           ├── ConfirmModal.test.tsx
│       │           ├── Modal.tsx               # モーダルの土台（Esc・オーバーレイで閉じる）
│       │           ├── Modal.test.tsx
│       │           ├── Textarea.tsx            # テキストエリア
│       │           └── Textarea.test.tsx
│       ├── data/
│       │   └── holidays.json                   # 祝日データ（make update-holidays で生成。docs/holidays.md）
│       ├── layouts/
│       │   ├── AdminLayout.astro               # 管理画面シェル（ナビ・未対応バッジ・ensureAccessToken による認証確認・ログアウト）
│       │   └── BaseLayout.astro                # 顧客向けページの HTML シェル（meta・font・global CSS）
│       ├── lib/                                # API クライアント・共通ロジック・表示テーマ
│       │   ├── adminReservation.ts             # 管理者予約 API（一覧・手動登録・ステータス更新）
│       │   ├── api.ts                          # 保護 API 用 fetch（Bearer 付与・401 時 refresh リトライ）
│       │   ├── auth.ts                         # ログイン / refresh / ログアウト、AT のメモリ保持
│       │   ├── availability.ts                 # 空き状況 API
│       │   ├── calendarTheme.ts                # スケジュール種別の色・ラベル
│       │   ├── calendarTheme.test.ts
│       │   ├── calendarUtils.ts                # 日付フォーマット等のカレンダー補助
│       │   ├── cx.ts                           # className 結合
│       │   ├── cx.test.ts
│       │   ├── holidays.ts                     # 祝日判定（data/holidays.json）
│       │   ├── holidays.test.ts
│       │   ├── japaneseMonth.ts                # 和風月名（睦月〜師走）
│       │   ├── pendingReservations.ts          # 未対応件数の取得・バッジ表示・更新通知
│       │   ├── pendingReservations.test.ts
│       │   ├── publicSchedule.ts               # 公開スケジュール API
│       │   ├── publicScheduleCell.ts           # 公開カレンダーのセル表示判定（休・イベントバー）
│       │   ├── publicScheduleCell.test.ts
│       │   ├── publicSuppliers.ts              # 公開取引先 API
│       │   ├── reservation.ts                  # 顧客向け予約申請 API
│       │   ├── reservationStatusTheme.ts       # 予約ステータスのラベル・色
│       │   ├── schedule.ts                     # 管理者スケジュール API・編集可能な種別
│       │   ├── schedule.test.ts
│       │   ├── storeDefaultSchedule.ts         # 店舗定例プレビュー（祝日 → 曜日）
│       │   ├── storeDefaultSchedule.test.ts
│       │   ├── storeHours.ts                   # カレンダーヘッダーの店舗定例表示
│       │   ├── suppliers.ts                    # 管理者取引先 API・画像の制約
│       │   ├── suppliers.test.ts
│       │   ├── useMonthCalendar.ts             # 月カレンダーのスケジュール・予約サマリー取得フック
│       │   ├── useMonthCalendar.test.ts
│       │   └── ui/
│       │       ├── buttonStyles.ts             # Button / ButtonLink 共通の className
│       │       ├── inputStyles.ts              # input / select / textarea 共通の className
│       │       └── types.ts                    # ButtonVariant / ButtonSize 型
│       ├── pages/                              # ファイルベースルーティング
│       │   ├── index.astro                     # トップ
│       │   ├── reservation.astro               # 予約申請
│       │   ├── reservation/
│       │   │   └── complete.astro              # 予約申請完了
│       │   ├── suppliers.astro                 # お取り引き先紹介
│       │   └── admin/
│       │       ├── index.astro                 # ダッシュボード
│       │       ├── login.astro                 # ログイン
│       │       ├── reservations.astro          # 予約一覧
│       │       ├── reservations/
│       │       │   └── new.astro               # 予約登録
│       │       ├── schedules.astro             # スケジュール設定
│       │       └── suppliers.astro             # 取引先管理
│       ├── styles/
│       │   └── global.css                      # Tailwind v4 カスタムテーマ
│       ├── test/
│       │   ├── deferred.ts                     # 送信中状態を検証するための Promise ヘルパー
│       │   └── setup.ts                        # Vitest セットアップ（jest-dom, cleanup）
│       └── types/
│           ├── reservation.ts                  # 予約・スケジュール・空き状況の型（API と対応）
│           └── supplier.ts                     # 取引先の型
├── backend/                                    # Go バックエンド
│   ├── .air.toml                               # Air（ホットリロード）設定
│   ├── .dockerignore                           # Docker ビルドの除外設定
│   ├── .env.example                            # 環境変数テンプレート
│   ├── .golangci.yml                           # golangci-lint 設定
│   ├── go.mod                                  # Go モジュール定義
│   ├── go.sum                                  # 依存のチェックサム
│   ├── cmd/
│   │   ├── api/main.go                         # API サーバーのエントリーポイント・DI・ルーティング
│   │   ├── migrate/main.go                     # DB マイグレーション実行（make migrate）
│   │   └── seed/main.go                        # 開発用シード投入（make seed）
│   ├── docker/
│   │   ├── Dockerfile                          # 本番用マルチステージビルド
│   │   └── Dockerfile.dev                      # 開発用（Air ホットリロード）
│   ├── internal/
│   │   ├── application/
│   │   │   ├── transaction.go                  # TxManager インターフェース（トランザクション境界）
│   │   │   ├── visit_date_lock.go              # VisitDateLocker インターフェース（来店日単位の直列化）
│   │   │   └── usecase/                        # ユースケース（入出力は Command / Query / Result 型）
│   │   │       ├── auth/
│   │   │       │   ├── login.go                # ログイン
│   │   │       │   ├── login_test.go
│   │   │       │   ├── login_security_test.go  # ログインのセキュリティ観点テスト
│   │   │       │   ├── login_rate_limit.go     # ログイン失敗のレートリミット
│   │   │       │   ├── login_rate_limit_test.go
│   │   │       │   ├── login_validate.go       # ログイン入力の検証
│   │   │       │   ├── login_validate_test.go
│   │   │       │   ├── logout.go               # ログアウト（RT 失効）
│   │   │       │   ├── logout_test.go
│   │   │       │   ├── refresh.go              # AT 更新・RT ローテーション
│   │   │       │   ├── refresh_test.go
│   │   │       │   ├── refresh_token_hash.go   # RT 平文のハッシュ化
│   │   │       │   └── refresh_token_hash_test.go
│   │   │       ├── reservation/
│   │   │       │   ├── admin_reservation_validate.go      # 管理者予約系の入力検証（ステータス・経路・理由）
│   │   │       │   ├── admin_reservation_validate_test.go
│   │   │       │   ├── captcha_verifier.go     # CaptchaVerifier インターフェースと開発用 NoOp
│   │   │       │   ├── create_admin_reservation.go        # 管理者の予約手動登録
│   │   │       │   ├── create_reservation.go   # 顧客の予約申請（Turnstile → 検証 → 空き確認 → 保存 → Outbox）
│   │   │       │   ├── create_reservation_test.go
│   │   │       │   ├── create_reservation_validate.go     # 予約申請の入力検証
│   │   │       │   ├── create_reservation_validate_test.go
│   │   │       │   ├── get_availability.go     # 日別・月次の空き状況
│   │   │       │   ├── get_availability_test.go
│   │   │       │   ├── list_reservations.go    # 管理者の予約一覧
│   │   │       │   ├── list_reservations_test.go
│   │   │       │   ├── mail_notifier.go        # MailEnqueuer インターフェースと NoOp
│   │   │       │   ├── update_reservation_status.go       # 予約ステータス更新（承認・拒否・キャンセル等）
│   │   │       │   ├── update_reservation_status_test.go
│   │   │       │   └── update_reservation_status_mail_test.go # ステータス更新時のメール記録テスト
│   │   │       ├── schedule/
│   │   │       │   ├── delete_schedule.go      # 日別設定の削除（店舗定例に戻す）
│   │   │       │   ├── delete_schedule_test.go
│   │   │       │   ├── get_schedule.go         # 日別スケジュール取得
│   │   │       │   ├── get_schedule_test.go
│   │   │       │   ├── list_schedules.go       # 月間スケジュール取得
│   │   │       │   ├── list_schedules_test.go
│   │   │       │   ├── set_schedule.go         # 日別スケジュール設定（Upsert）
│   │   │       │   ├── set_schedule_test.go
│   │   │       │   ├── set_schedule_validation.go         # スケジュール設定の入力検証
│   │   │       │   └── set_schedule_validation_test.go
│   │   │       └── supplier/
│   │   │           ├── supplier.go             # 取引先ユースケース共通のエラー型
│   │   │           ├── fake_test.go            # テスト用 TxManager フェイク
│   │   │           ├── create_supplier.go      # 取引先作成
│   │   │           ├── create_supplier_test.go
│   │   │           ├── delete_supplier.go      # 取引先削除
│   │   │           ├── delete_supplier_test.go
│   │   │           ├── get_supplier.go         # 取引先 1 件取得
│   │   │           ├── get_supplier_test.go
│   │   │           ├── list_suppliers.go       # 取引先一覧（公開・管理共通）
│   │   │           ├── list_suppliers_test.go
│   │   │           ├── reorder_suppliers.go    # 並び順更新
│   │   │           ├── reorder_suppliers_test.go
│   │   │           ├── update_supplier.go      # 取引先更新（部分更新）
│   │   │           ├── update_supplier_test.go
│   │   │           ├── upload_image.go         # 取引先画像アップロード
│   │   │           ├── upload_image_test.go
│   │   │           ├── validate.go             # 作成・更新共通の入力検証
│   │   │           └── validate_test.go
│   │   ├── datetime/
│   │   │   ├── date.go                         # 日付型 Date（JSON・SQL 対応）
│   │   │   ├── date_test.go
│   │   │   ├── time.go                         # 時刻型 Time（HH:MM。JSON・SQL 対応）
│   │   │   └── time_test.go
│   │   ├── domain/                             # フラットな domain パッケージ（エンティティ・Repository インターフェース・sentinel エラー）
│   │   │   ├── admin_user.go                   # 管理者ユーザー
│   │   │   ├── captcha.go                      # CAPTCHA 検証失敗エラー
│   │   │   ├── email_outbox.go                 # メール Outbox 行・MailType・Repository
│   │   │   ├── email_outbox_test.go
│   │   │   ├── login_attempt.go                # ログイン失敗試行の集計・Repository
│   │   │   ├── mail.go                         # MailSender インターフェース・送信エラー
│   │   │   ├── refresh_token.go                # リフレッシュトークン・Repository
│   │   │   ├── reservation.go                  # 予約・許容値・遷移ルール（CanTransition）・Repository
│   │   │   ├── reservation_test.go
│   │   │   ├── schedule.go                     # 日別スケジュール・Repository
│   │   │   ├── storage.go                      # ImageStorage インターフェース
│   │   │   ├── supplier.go                     # 取引先・Repository
│   │   │   ├── holiday/                        # 祝日判定（内閣府 CSV を go:embed。docs/holidays.md）
│   │   │   │   ├── holiday.go                  # CSV パースと祝日判定
│   │   │   │   ├── holiday_test.go
│   │   │   │   └── syukujitsu.csv              # 同梱の祝日データ
│   │   │   └── service/                        # 複数エンティティにまたがるドメインロジック
│   │   │       ├── availability.go             # 残り食数・予約可否の算出
│   │   │       ├── availability_test.go
│   │   │       ├── business_hours.go           # schedule_type ごとの既定営業時刻
│   │   │       ├── business_hours_test.go
│   │   │       ├── schedule_resolver.go        # 保存済み設定と店舗定例から有効スケジュールを解決
│   │   │       ├── schedule_resolver_test.go
│   │   │       └── store_calendar.go           # 店舗定例の合成（祝日 → 曜日）
│   │   ├── handler/                            # HTTP ハンドラ・リクエスト / レスポンス型・ミドルウェア
│   │   │   ├── admin_reservation.go            # 管理者予約 API
│   │   │   ├── admin_reservation_test.go
│   │   │   ├── admin_supplier.go               # 管理者取引先 API
│   │   │   ├── auth.go                         # login / refresh / logout
│   │   │   ├── auth_test.go
│   │   │   ├── auth_login_security_test.go     # ログインのセキュリティ観点テスト
│   │   │   ├── auth_logout_test.go
│   │   │   ├── auth_refresh_test.go
│   │   │   ├── auth_testutil_test.go           # 認証ハンドラテストの共通フェイク
│   │   │   ├── availability.go                 # 空き状況 API
│   │   │   ├── availability_test.go
│   │   │   ├── client_ip.go                    # X-Forwarded-For からのクライアント IP 取得
│   │   │   ├── client_ip_test.go
│   │   │   ├── cookie.go                       # RT Cookie の発行・削除
│   │   │   ├── cookie_test.go
│   │   │   ├── method_not_allowed_test.go      # 405 / 404 の JSON レスポンステスト
│   │   │   ├── middleware.go                   # JWT 認証（RequireAuth）・CORS
│   │   │   ├── middleware_test.go
│   │   │   ├── outbox.go                       # 内部 Outbox flush API
│   │   │   ├── outbox_test.go
│   │   │   ├── reservation.go                  # 顧客の予約申請 API
│   │   │   ├── response.go                     # JSON レスポンス・エラーレスポンスの共通処理
│   │   │   ├── root.go                         # GET / ヘルスチェックと未知パスの 404
│   │   │   ├── schedule.go                     # 管理者スケジュール API
│   │   │   ├── schedule_test.go
│   │   │   ├── schedule_errors.go              # スケジュール系ユースケースエラーの HTTP 変換
│   │   │   ├── schedule_public.go              # 顧客向け月間スケジュール API
│   │   │   ├── schedule_public_test.go
│   │   │   └── supplier.go                     # 公開取引先 API・取引先レスポンス型
│   │   ├── infrastructure/
│   │   │   ├── external/                       # 外部 API クライアント
│   │   │   │   ├── resend/
│   │   │   │   │   ├── client.go               # Resend メール送信クライアント
│   │   │   │   │   ├── client_test.go
│   │   │   │   │   └── noop.go                 # API キー未設定時のログのみ実装
│   │   │   │   ├── storage/
│   │   │   │   │   ├── noop.go                 # ストレージ未設定時のフォールバック
│   │   │   │   │   └── s3.go                   # S3 互換ストレージ（R2 / MinIO）
│   │   │   │   └── turnstile/
│   │   │   │       └── verifier.go             # Cloudflare Turnstile の siteverify クライアント
│   │   │   └── mail/                           # メールテンプレート・Outbox enqueue・Dispatcher
│   │   │       ├── backoff_test.go             # 再送間隔・失敗確定判定のテスト
│   │   │       ├── dispatcher.go               # Outbox の pending 行を送信（lease 方式）
│   │   │       ├── dispatcher_test.go
│   │   │       ├── outbox_enqueuer.go          # 予約関連メールを Outbox へ記録
│   │   │       ├── outbox_enqueuer_test.go
│   │   │       ├── templates.go                # 受付・承認・拒否メールの本文
│   │   │       └── templates_test.go
│   │   ├── privacy/
│   │   │   ├── mask.go                         # ログ向けの個人情報マスキング
│   │   │   └── mask_test.go
│   │   └── repository/                         # Repository 実装（PostgreSQL。#46 で infrastructure/repository へ移動予定）
│   │       ├── admin_user.go                   # 管理者ユーザー
│   │       ├── email_outbox.go                 # メール Outbox（claim は FOR UPDATE SKIP LOCKED）
│   │       ├── email_outbox_integration_test.go
│   │       ├── login_attempt.go                # ログイン失敗試行
│   │       ├── login_attempt_integration_test.go
│   │       ├── refresh_token.go                # リフレッシュトークン
│   │       ├── refresh_token_integration_test.go
│   │       ├── reservation.go                  # 予約
│   │       ├── schedule.go                     # 日別スケジュール
│   │       ├── schedule_event_text_test.go     # イベント欄の NULL 変換テスト
│   │       ├── supplier.go                     # 取引先
│   │       ├── transaction.go                  # TxManager 実装（ctx に Tx を載せる）
│   │       ├── transaction_test.go
│   │       ├── visit_date_lock.go              # 来店日単位の advisory lock
│   │       └── visit_date_lock_test.go
│   ├── migrations/                             # DB マイグレーション SQL（make migrate）
│   │   ├── 000001_init_reservations.sql        # reservations
│   │   ├── 000002_daily_schedules.sql          # daily_schedules
│   │   ├── 000003_reservations_updated_at_trigger.sql # reservations.updated_at トリガー
│   │   ├── 000004_suppliers.sql                # suppliers
│   │   ├── 000005_admin_users.sql              # admin_users
│   │   ├── 000006_refresh_tokens.sql           # refresh_tokens
│   │   ├── 000007_external_event_schedule_type.sql # schedule_type に external_event を追加
│   │   ├── 000008_login_attempts.sql           # login_attempts
│   │   ├── 000009_email_outbox.sql             # email_outbox
│   │   └── 000010_refresh_tokens_expires_at_index.sql # refresh_tokens.expires_at のインデックス
│   └── pkg/
│       ├── config/
│       │   └── config.go                       # 環境変数の読み込み・検証
│       └── jwt/
│           ├── jwt.go                          # 管理者 AT（JWT HS256）の発行・検証
│           └── jwt_test.go
├── docs/                                       # 設計ドキュメント（一覧は §10）
├── scripts/
│   ├── setup-gcp.sh                            # GCP 初期セットアップ（Artifact Registry / SA / WIF / Secret Manager 枠）
│   ├── setup-scheduler.sh                      # Cloud Scheduler（Outbox flush）セットアップ
│   └── update-holidays.sh                      # 祝日データの年次更新（make update-holidays）
├── tools/
│   ├── .gitignore                              # make tools-install で入れるバイナリ（bin/）を除外
│   └── golangci-lint.version                   # golangci-lint のバージョン固定
├── .github/
│   ├── pull_request_template.md                # PR テンプレート
│   └── workflows/
│       ├── backend.yml                         # バックエンド CI
│       ├── deploy-backend.yml                  # Cloud Run への CD
│       └── frontend.yml                        # フロントエンド CI
├── .cursor/rules/                              # Cursor AI ルール
│   ├── code-docs-hygiene.mdc                   # コードと docs の整合
│   ├── git-branch.mdc                          # ブランチ運用
│   ├── git-commit.mdc                          # コミットメッセージ
│   ├── post-implementation.mdc                 # 実装後の確認
│   └── pull-request.mdc                        # PR 作成
├── .cursorignore                               # Cursor の読み込み除外
├── .gitignore                                  # リポジトリ全体の除外設定
├── CLAUDE.md                                   # このファイル
├── PROJECT_RULES.md                            # AI の役割・対話方針
├── README.md                                   # リポジトリの概要
├── docker-compose.yml                          # 開発用 Docker
├── docker-compose.prod.yml                     # 本番用 Docker
└── Makefile                                    # 開発コマンド
```

> **注意**: `backend/` の移行予定は `docs/architecture.md` §7 を正とする。方針はレイヤード + Repository + DI に絞り、DDD 由来の要素は目標から外した（詳細は `docs/architecture.md` §7、ADR-005）。

---

## 4. フロントエンド設計

### 設計意図 — Islands Architecture

ページの大部分を静的 HTML で配信し、インタラクティブな部分（Islands）だけ React でハイドレーションする。

- トップページ: 本文は `.astro` の静的 HTML。営業カレンダー（`PublicScheduleCalendar`）だけ Island
- お取り引き先紹介: 本文は静的 HTML。取引先一覧（`SupplierList`）は `client:visible` の Island
- 予約フォーム・管理画面など: React Islands → インタラクション実現
- Core Web Vitals に有利、Cloudflare Pages の静的配信と相性良

### 責務分離

| 場所 | 責務 |
|------|------|
| `.astro` ページ/レイアウト | ルーティング、静的 HTML の構造、SEO、OGP、レイアウト |
| `.tsx` React Islands | ユーザーインタラクション（フォーム入力・送信、テーブル操作、カレンダー編集） |
| `layouts/BaseLayout.astro` | 顧客向けページの HTML シェル（meta, font, global CSS） |
| `layouts/AdminLayout.astro` | 管理画面のシェル（ナビ、`ensureAccessToken()` による認証確認（AT はメモリ保持、RT は httpOnly Cookie）、未対応バッジ、ログアウト） |
| `types/*.ts` | 型定義（`reservation`, `supplier`）。フロントとバックが共通で使う型はここに集約 |
| `lib/*.ts` | API クライアント（`api`, `auth`, `adminReservation`, `reservation`, `availability`, `schedule`, `publicSchedule`, `suppliers`, `publicSuppliers`, `pendingReservations`）、カレンダー共通ロジック（`useMonthCalendar`, `calendarUtils`, `holidays`, `storeDefaultSchedule`, `storeHours`, `publicScheduleCell`, `japaneseMonth`）、表示テーマ（`calendarTheme`, `reservationStatusTheme`, `ui/*`）、`cx` |
| `styles/global.css` | Tailwind v4 `@theme`（カスタムカラー `#43676B`、Noto Serif JP、グラデーション、アニメーション） |
| `*.test.ts(x)` | テスト。対象と同じディレクトリに置く（`components/react/Foo.test.tsx`、`lib/foo.test.ts`）。コンポーネントは React Testing Library で描画し、`lib/*` の API 層は `vi.mock` で差し替える（方針は `docs/test_design.md`） |

### ハイドレーション戦略

| ディレクティブ | タイミング | 使用箇所 |
|---------------|-----------|---------|
| `client:load` | ページ読み込み時に即時 | トップページの `PublicScheduleCalendar`（`AttentionSection.astro` 経由）、予約フォーム、ログインフォーム、ダッシュボードの `DashboardSummary`、管理画面全般 |
| `client:visible` | 要素が画面内に入ったとき | お取り引き先ページの `SupplierList` |

`client:idle` は未使用（採用是非は #39）。

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
(handler)        (usecase)       (domain      (repository 実装
                                 / service)   / external / mail)
```

| レイヤー | ディレクトリ | 責務 |
|---------|------------|------|
| Presentation | `internal/handler/` | HTTP リクエスト受付、リクエスト / レスポンス型への変換、レスポンス返却、ミドルウェア（JWT 認証・CORS）。ミドルウェアは同居しており #47 で `internal/middleware` に分離予定 |
| Application | `internal/application/usecase/` | ユースケースのオーケストレーション（Turnstile 検証 → バリデーション → 空き確認 → 保存 → Outbox enqueue）。入出力は usecase の Command / Result 型と handler のリクエスト / レスポンス型で、`dto` 層は置かない |
| Domain | `internal/domain/` | フラットな `domain` パッケージ。エンティティ、Repository インターフェース、sentinel エラー、ステータス遷移ルール（`domain.CanTransition()`） |
| Domain | `internal/domain/service/` | 複数エンティティにまたがるドメインロジック（空き状況計算、店舗定例の合成、営業時間判定） |
| Infrastructure | `internal/repository/` | Repository 実装（PostgreSQL）。#46 で `internal/infrastructure/repository/` へ移動予定 |
| Infrastructure | `internal/infrastructure/external/` | 外部 API クライアント（Turnstile, Resend） |
| Infrastructure | `internal/infrastructure/mail/` | メールテンプレート・Outbox enqueue・Dispatcher |

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
| `/` | `pages/index.astro` | BaseLayout | `PublicScheduleCalendar`（`AttentionSection.astro` 内、`client:load`） |
| `/reservation` | `pages/reservation.astro` | BaseLayout | `ReservationForm` |
| `/reservation/complete` | `pages/reservation/complete.astro` | BaseLayout | なし |
| `/suppliers` | `pages/suppliers.astro` | BaseLayout | `SupplierList` |
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
| `POST` | `/api/v1/reservations` | 顧客：予約申請（Turnstile） | 実装済み |
| `GET` | `/api/v1/reservations/availability` | 日付別・月次空き確認 | 実装済み |
| `GET` | `/api/v1/schedules` | 顧客：月間スケジュール | 実装済み |
| `POST` | `/api/v1/admin/login` | 管理者ログイン | 実装済み |
| `POST` | `/api/v1/admin/refresh` | AT 更新 | 実装済み |
| `POST` | `/api/v1/admin/logout` | 管理者ログアウト | 実装済み |
| `GET` | `/api/v1/admin/reservations` | 管理者：予約一覧 | 実装済み |
| `POST` | `/api/v1/admin/reservations` | 管理者：予約手動登録 | 実装済み |
| `PATCH` | `/api/v1/admin/reservations/{id}/status` | 予約ステータス更新 | 実装済み |
| `GET` | `/api/v1/admin/schedules` | 月間スケジュール取得 | 実装済み |
| `GET` | `/api/v1/admin/schedules/{date}` | 日別スケジュール取得 | 実装済み |
| `PUT` | `/api/v1/admin/schedules/{date}` | 日別スケジュール設定（Upsert） | 実装済み |
| `DELETE` | `/api/v1/admin/schedules/{date}` | スケジュール削除（店舗定例に戻す） | 実装済み |
| `GET` | `/api/v1/suppliers` | 取引先一覧（公開） | 実装済み |
| `GET` | `/api/v1/admin/suppliers` | 取引先一覧取得 | 実装済み |
| `POST` | `/api/v1/admin/suppliers` | 取引先作成 | 実装済み |
| `GET` | `/api/v1/admin/suppliers/:id` | 取引先取得 | 実装済み |
| `PUT` | `/api/v1/admin/suppliers/:id` | 取引先更新 | 実装済み |
| `DELETE` | `/api/v1/admin/suppliers/:id` | 取引先削除 | 実装済み |
| `PUT` | `/api/v1/admin/suppliers/order` | 取引先並び順更新 | 実装済み |
| `POST` | `/api/v1/admin/suppliers/:id/image` | 取引先画像アップロード | 実装済み |
| `POST` | `/internal/outbox/flush` | Outbox 送信処理（内部・IAM 保護） | 実装済み |

---

## 7. 開発コマンド

### Makefile（ルート）

```bash
make dev          # バックエンド開発サーバー起動（Docker + Air）
make dev-build    # バックエンド Docker イメージビルド
make dev-down     # バックエンド Docker 停止
make dev-front    # フロントエンド開発サーバー起動（bun run dev）
make prod         # 本番用 Docker 起動
make prod-build   # 本番用 Docker イメージビルド
make prod-down    # 本番用 Docker 停止
make migrate      # DB マイグレーション実行（cmd/migrate）
make seed         # 開発用シード投入（cmd/seed。SEED_ADMIN_PASSWORD が必要）
make build        # Go バイナリビルド（bin/api）
make run          # Go サーバー直接起動
make test         # バックエンドテスト実行
make test-integration # Outbox 等の DB 結合テスト（postgres-test）
make test-coverage # カバレッジ付きテスト
make lint         # golangci-lint
make outbox-flush # ローカルで Outbox flush を手動実行
make tools-install # lint ツールのインストール
make update-holidays # 祝日データ（内閣府 CSV）を取得し backend/frontend の同梱データを再生成（年 1 回・docs/holidays.md）
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
bun run test:coverage # Vitest + カバレッジ計測（v8。frontend/coverage/ に出力）
```

---

## 8. 環境変数

テンプレートはバックエンドとフロントエンドで分かれている。それぞれコピーして使う。

```bash
cp backend/.env.example backend/.env    # docker compose の env_file と go run 時の godotenv が読む
cp frontend/.env.example frontend/.env  # Astro が frontend/ 直下から読む（PUBLIC_ のみクライアントに公開）
```

変数の一覧・既定値・必須条件は `backend/.env.example` / `frontend/.env.example` のコメントを正とし、ここには一覧を持たない（定義元は `backend/pkg/config/config.go`）。本番（Cloud Run / Cloudflare Pages）の設定は `docs/deploy_cloud_run.md`。

---

## 9. CI/CD

| ワークフロー | トリガー | ジョブ |
|------------|---------|-------|
| `.github/workflows/backend.yml` | `backend/**` の変更を `main`/`develop` に push/PR | lint / test / build / docker-build（並列） |
| `.github/workflows/frontend.yml` | `frontend/**` の変更を `main`/`develop` に push/PR | lint / test / build（並列。build でアーティファクトアップロード） |
| `.github/workflows/deploy-backend.yml` | `backend/**` の変更を `develop` に push → **staging**、`main` に push → **production** / `workflow_dispatch`（環境を choice） | WIF 認証 → docker build/push → Cloud Run Job で migrate → `satehits-api` / `satehits-api-internal`（staging は `-stg`）デプロイ → スモーク |

- フロントエンドのビルド成果物は `frontend/dist/` にアップロードされ Cloudflare Pages へデプロイ
- バックエンドは Google Cloud Run へデプロイ（手順は `docs/deploy_cloud_run.md`）

---

## 10. ドキュメント

詳細な設計はすべて `docs/` に記載されている。

| ファイル | 内容 |
|---------|------|
| [docs/setup.md](docs/setup.md) | 開発環境の構築手順 |
| [docs/domain_knowledge.md](docs/domain_knowledge.md) | 店舗情報、営業ルール、用語集 |
| [docs/architecture.md](docs/architecture.md) | レイヤー構成、責務分離、DI、エラーハンドリング戦略 |
| [docs/architecture_decision_records.md](docs/architecture_decision_records.md) | ADR（想定規模 ADR-000、FE/BE/DB/認証/CI、監視・個人情報は一部未決定） |
| [docs/api_design.md](docs/api_design.md) | API 仕様の概要 |
| [docs/openapi.yaml](docs/openapi.yaml) | OpenAPI 3.0 形式の API 仕様 |
| [docs/sequence.md](docs/sequence.md) | 主要機能のシーケンス図 |
| [docs/screen_transition.md](docs/screen_transition.md) | 画面一覧・遷移図・各画面の要素と操作 |
| [docs/table_design.md](docs/table_design.md) | テーブル設計 |
| [docs/infrastructure.md](docs/infrastructure.md) | インフラ構成、環境設定、CI/CD 詳細 |
| [docs/deploy_cloud_run.md](docs/deploy_cloud_run.md) | Cloud Run への CD 手順（GCP セットアップ、Secrets/Variables、初回デプロイ、ロールバック） |
| [docs/test_design.md](docs/test_design.md) | テスト方針・テストケース一覧 |
| [docs/coding_rule/go_testing.md](docs/coding_rule/go_testing.md) | Go テストの書き方 |
| [docs/use_case.md](docs/use_case.md) | ユースケース一覧 |
| [docs/data_flow.md](docs/data_flow.md) | データフロー図 |
| [docs/holidays.md](docs/holidays.md) | 祝日データの出典・合成ルール・年次更新手順（毎年 2 月頃に `make update-holidays`） |
