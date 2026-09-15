# テスト設計

## 1. テスト方針

### テストレベル

| レベル | 対象 | 目的 |
|--------|------|------|
| 単体テスト | 関数・メソッド単位 | ビジネスロジックの正確性を保証 |
| 結合テスト | 複数レイヤーの連携 | レイヤー間のデータフロー・エラーハンドリングを検証 |
| E2E テスト | ユーザー操作の一連の流れ | 主要なユースケースが正常に動作することを確認 |

### テストカバレッジ目標

| レイヤー / 対象 | カバレッジ目標 | 実測（2026-09-16） | 備考 |
|----------------|---------------|-------------------|------|
| Domain Layer（Entity, Domain Service） | 90% 以上 | `domain` 100%、`domain/service` 90.9% | ビジネスロジックの中核。最優先でテスト |
| Application Layer（UseCase） | 80% 以上 | `usecase/reservation` 86.6%、`usecase/auth` 93.3%、`usecase/schedule` 90.3%、`usecase/supplier` 98.5% | 正常系・異常系の主要パスをカバー |
| Infrastructure Layer（Repository） | 結合テストで担保 | `repository` 9.5%（`TEST_DATABASE_URL` なしの値） | モックではなく実 DB 接続でテスト（結合テストは `TEST_DATABASE_URL` 必須） |
| Presentation Layer（Handler） | 結合テストで担保 | `handler` 51.2% | HTTP リクエスト/レスポンスの検証 |
| フロントエンド（React コンポーネント） | 70% 以上 | 行: `components/react` 97.4%、`components/react/ui` 100%、`lib` 46.8%（全体 81.7%） | ユーザー操作・表示ロジックをテスト。`lib` は API クライアント等の未テスト分を含む |

計測方法（バックエンド）:

```bash
# パッケージ別のカバレッジを表示
cd backend && go test -cover ./...

# coverage.out / coverage.html を生成
make test-coverage
```

計測方法（フロントエンド）:

```bash
# @vitest/coverage-v8。対象は src/**/*.{ts,tsx}（テスト・src/test・env.d.ts を除く）。frontend/coverage/ に HTML も出力
cd frontend && bun run test:coverage
```

## 2. 使用ツール

### バックエンド（Go）

標準 `testing` + 手書き fake + `httptest`（方針は `docs/coding_rule/go_testing.md`）。

| ツール | 用途 |
|--------|------|
| Go 標準 `testing` パッケージ | テストの基盤（`testify` 等は導入しない） |
| 手書き fake | Repository / 外部 API のインターフェースをテストファイル内の構造体で差し替える |
| `httptest` | HTTP ハンドラー・ミドルウェアのテスト |

### フロントエンド（Astro + React）

| ツール | 用途 |
|--------|------|
| Vitest（jsdom） | テストランナー・アサーション |
| Testing Library（`@testing-library/react` / `jest-dom`） | React コンポーネントのレンダリング・DOM アサーション |
| `@testing-library/user-event` | ユーザー操作のシミュレーション |
| `vi.mock` | `src/lib/*.ts` の API 層をモック（実 HTTP は叩かない） |

#### React コンポーネントテストの方針

- **配置**: 対象コンポーネントと同じディレクトリに `Foo.test.tsx` を置く（`src/components/react/`、`src/components/react/ui/`）。`src/lib` のロジックは `foo.test.ts`
- **セットアップ**: `vitest.config.ts`（`environment: jsdom`、`globals: true`、`passWithNoTests: false`）と `src/test/setup.ts`（`@testing-library/jest-dom/vitest` と `afterEach(cleanup)`）
- **API モック**: `vi.mock('@/lib/xxx', async (importOriginal) => ({ ...(await importOriginal()), listXxx: vi.fn() }))` の形で関数だけ差し替え、`XxxApiError` などのクラスは実物を使う（`instanceof` 分岐を検証するため）。`fetch` を直接モックする場合は `vi.stubGlobal('fetch', ...)` を使う
- **子コンポーネント**: 単体でテスト済みの重い子（`DatePickerField`、`TurnstileWidget`）は親のテストでは最小限のスタブに差し替えてよい
- **ブラウザ API**: `window.location` は `vi.stubGlobal('location', { href })`、`window.confirm` は `vi.spyOn`、日付は `vi.useFakeTimers({ toFake: ['Date'] })` + `vi.setSystemTime` で固定する（`setTimeout` は偽装しないので `user-event` / `waitFor` がそのまま動く）
- **検証する観点**: 表示（取得中 → 表示 / 空 / エラー）、バリデーション、API 呼び出し引数、成功時の遷移・表示、送信中の二重送信防止、確認モーダルの経路。スナップショットは使わず、`describe` / `it` は日本語で「何を検証するか」を書く
- **送信中の検証**: `src/test/deferred.ts` の `createDeferred()` で API の Promise を保留し、ボタンの無効化を確認してから resolve する
- **注意**: jsdom はフォーム送信時に `required` / `type="email"` の制約検証を行うため、コンポーネント側のバリデーションを検証するときは `fireEvent.submit` を使うか、ブラウザ検証は通る値を入力する

## 3. テストケース一覧

### 3.1 予約機能

#### 予約申請（顧客）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 正常な予約申請 | 単体 | 予約が作成され、ステータスが `pending` |
| 2 | Turnstile 検証失敗 | 単体 | `CAPTCHA_FAILED` エラー（400） |
| 3 | その日の営業設定（有効なスケジュール）において予約不可の日に予約 | 単体 | `VALIDATION_ERROR`（400、field `visit_date`「この日は予約できません」） |
| 4 | 残り食数を超える人数で予約 | 単体 | `CAPACITY_EXCEEDED` エラー |
| 5 | 過去日付に予約 | 単体 | バリデーションエラー |
| 6 | 2 週間以上先の日付に予約 | 単体 | バリデーションエラー |
| 7 | 8 名以上で予約 | 単体 | バリデーションエラー |
| 8 | 必須項目が未入力 | 単体 | バリデーションエラー |
| 9 | メールアドレスの形式不正 | 単体 | バリデーションエラー |
| 10 | 予約申請後に受付メールが送信される | 単体 | MailEnqueuer（Outbox）に作成済み予約で 1 回、Tx 内で enqueue される。enqueue 失敗は UseCase のエラー（実装: `TestCreateReservationUseCase_Execute_mailOutbox`） |
| 11 | 同一 visit_date の同時申請でオーバーブッキングしない | 単体 | Lock → 再 SUM → Create の順。満席時は `CAPACITY_EXCEEDED`（実 DB 並行は未実施） |

#### 予約承認（オーナー）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | pending → approved に遷移 | 単体 | ステータスが `approved` に更新 |
| 2 | 承認後に承認メールが送信される | 単体 | MailEnqueuer（Outbox）に enqueue される |
| 3 | rejected → approved への不正遷移 | 単体 | `VALIDATION_ERROR`（field `status`） |
| 4 | 存在しない予約 ID を承認 | 単体 / 単体（httptest） | `ErrReservationNotFound` がそのまま返り、handler は 404 `NOT_FOUND`（実装: `TestUpdateReservationStatusUseCaseReturnsNotFound`、`TestAdminReservationHandler_UpdateStatus`） |

#### 予約拒否（オーナー）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | pending → rejected に遷移 | 単体 | ステータスが `rejected` に更新 |
| 2 | 拒否後に拒否メールが送信される | 単体 | MailEnqueuer（Outbox）に enqueue される |
| 3 | approved → rejected への不正遷移 | 単体 / 単体（httptest） | `VALIDATION_ERROR`（field `status`）（実装: `TestUpdateReservationStatusUseCase/rejects_approved_to_rejected`、`TestAdminReservationHandler_UpdateStatus`。遷移表全体は `TestCanTransition`） |

#### 予約一覧（オーナー）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 日付指定で予約一覧取得 | 単体 / 単体（httptest） | 該当日の予約リストが返る。date / status / source はそのまま Repository に渡り、不正な status / source は `VALIDATION_ERROR`（field `status` / `source`）、date 形式不正は 400（field `date`）（実装: `TestListReservationsUseCase_Execute_filters`、`TestListReservationsUseCase_Execute_validation`、`TestAdminReservationHandler_List`） |
| 2 | 予約がない日付で取得 | 単体 / 単体（httptest） | 空のリストが返る（JSON は `{"reservations":[],"total":0}`）（実装: `TestListReservationsUseCase_Execute_results`、`TestAdminReservationHandler_List`） |
| 3 | 認証なしでアクセス | 単体（httptest） | 401 Unauthorized |

### 3.2 スケジュール機能

#### スケジュール取得

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 月間スケジュール取得 | 単体 | 指定月のスケジュールリストが返る |
| 2 | スケジュール未設定の日付 | 単体 | ドメイン既定で合成したその日の営業設定（有効なスケジュール）が返る |
| 2-1 | スケジュール未設定の祝日（木・金・日曜と重なる場合を含む） | 単体 | 曜日にかかわらず `normal`（提供数 10・11:30-15:00）として合成される |
| 3 | 残り食数の計算 | 単体 | 提供可能数 - 予約済み（pending + approved）人数 |

#### スケジュール設定（オーナー）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 通常営業の設定 | 単体 | スケジュールが保存される |
| 2 | イベント営業の設定（イベント名・説明付き） | 単体 | イベント情報を含めて保存 |
| 3 | 臨時休業の設定 | 単体 | 提供可能数 0 で保存 |
| 4 | 既存スケジュールの上書き（Upsert） | 単体 | 既存データが更新される（`SetScheduleResult.Inserted` が false）（実装: `TestSetScheduleUseCase_Execute_upsert`） |
| 5 | 認証なしでアクセス | 単体（httptest） | 401 Unauthorized |

#### スケジュール削除（オーナー）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 設定済みスケジュールの削除 | 単体 | デフォルトスケジュールに戻る |
| 2 | 未設定の日付を削除 | 単体 | 404 `NOT_FOUND` |

### 3.3 取引先機能

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 取引先の新規作成 | 単体 | 取引先が保存される |
| 2 | 取引先の更新 | 単体 | 指定した項目が更新される |
| 3 | 取引先の削除 | 単体 | 取引先が削除される |
| 4 | 取引先一覧の取得（顧客向け: 表示中のみ） | 単体 | `is_active = true` の取引先のみ返る |
| 5 | 取引先一覧の取得（管理者: 全件） | 単体 | 全取引先が返る |
| 6 | 必須項目（名前）未入力で作成 | 単体 | バリデーションエラー |
| 7 | 認証なしでCUD操作 | 単体（httptest） | 401 Unauthorized |

### 3.4 認証機能

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 正しい認証情報でログイン | 単体 | JWT トークンが返る |
| 2 | 存在しないメールアドレス | 単体 | `UNAUTHORIZED` エラー（失敗試行を記録） |
| 3 | パスワード不一致 | 単体 | `UNAUTHORIZED` エラー（失敗試行を記録） |
| 4 | 有効な JWT でアクセス | 単体（httptest） | 認証成功、リクエスト続行 |
| 5 | 期限切れの JWT でアクセス | 単体（httptest） | 401 Unauthorized |
| 6 | 不正な JWT でアクセス | 単体（httptest） | 401 Unauthorized |
| 7 | ログアウト | 単体 | トークン無効化 |
| 8 | メール単位の失敗上限超過 | 単体 | 429 `TOO_MANY_REQUESTS`、`FindByEmail` 未呼出、`Retry-After` 付与 |
| 9 | IP 単位の失敗上限超過 | 単体 | 429 `TOO_MANY_REQUESTS`、認証処理未実行 |
| 10 | ログイン成功後 | 単体 | 当該メールの失敗試行がクリアされる |
| 11 | 制限中の再試行 | 単体 / 結合（PostgreSQL） | 失敗行が増えず、窓経過後に解除される（実装: 失敗行は `TestLoginUseCase_Execute`・`TestAuthHandler_HandleLogin_rateLimitThresholds` で `RecordFailure` 未呼出、窓は `TestLoginAttemptRepository_CountRecent` で `since` より前の行を数えないこと） |
| 12 | 未登録メールとパスワード不一致の応答 | 単体 | HTTP ステータス・`code`・`message` が完全一致（`details` なし）。429 も同様 |
| 13 | 未登録メールの所要時間 | 単体 | ダミー bcrypt 比較により、登録済みメールの誤パスワードと中央値の比が 3 倍以内 |
| 14 | パスワード比較方式 | 単体 | bcrypt 比較。保存ハッシュが平文と同じ文字列でも認証失敗 |
| 15 | レートリミット境界 | 単体 | メール `max-1` / IP `max-1` は通過、`max` で 429。両方超過時は `Retry-After` が大きい方。端数秒は切り上げ、最小 1 |
| 16 | ログイン応答の機密 | 単体 | JSON に RT・`password_hash` を含めない。RT は Cookie のみ、DB には SHA-256 ハッシュ |
| 17 | 認証系ログの機密 | 単体 | login / refresh / logout の 401・500 でパスワード平文・bcrypt ハッシュ・RT 平文・RT ハッシュ・AT がログに出ない |
| 18 | `RequireAuth` の拒否 | 単体 | ヘッダ無し・`Basic`・空 `Bearer` は 401 `UNAUTHORIZED`。改ざん・別 secret・`alg=none`・期限切れ・`nbf` 未来・`iss` / `aud` 不一致・`sub` 不正・`role` 不正は 401 `INVALID_TOKEN`。いずれも next ハンドラ未呼出 |
| 19 | AT 時刻境界 | 単体 | `exp = now+2s` 有効、`exp = now` / `now-1s` 無効、`nbf = now+2s` 無効。`expires_at - now ≈ 1h` |
| 20 | refresh の CSRF 緩和 | 単体 | Origin 無し・許可外は 403 `FORBIDDEN`（Cookie 検査より先。repo 未呼出）。Referer のみ許可は通過 |
| 21 | refresh の RT 検証 | 単体 | Cookie 無し・空・未知 RT は 401 `INVALID_TOKEN`。失効済み RT の再利用は 401 + `RevokeAllByUser`。`expires_at = now-1ms` は 401 かつローテーションなし、`now+1s` は 200 |
| 22 | RT ローテーション | 単体 | `FindByTokenHash(sha256)` → `RevokeIfActive(旧)` → `Issue(新)` の順、1 トランザクション。新 RT ≠ 旧 RT、Cookie 値 = 新 RT、`Issue` の `expires_at ≈ now+30d`。`RevokeIfActive` が false なら 401 + `RevokeAllByUser`、`Issue` 失敗は 500 で Cookie 未更新 |
| 23 | セッションの連続動作 | 単体 | login → refresh → 旧 RT 再利用で 401 + 全失効 → 新 RT も 401。logout 後の同一 RT で refresh は 401 |
| 24 | logout | 単体 | AT 無し 401 `UNAUTHORIZED`、AT 改ざん 401 `INVALID_TOKEN`、Origin 無し / 許可外 403、Referer のみ許可は 204。成功時 `Revoke(当該 RT)` と Cookie 削除（`Max-Age=0`・`HttpOnly`・`Secure`・`SameSite=None`・`Path=/api/v1/admin`・`Domain` は設定時のみ）。Cookie 無し・未知 RT でも 204（`Revoke` 未呼出）。repo 失敗は 500 で Cookie 未削除 |
| 25 | RT Cookie 属性 | 単体 | 発行時 `Max-Age=2592000`、削除時 `Max-Age=0`。`HttpOnly` / `Secure` / `SameSite=None` / `Path=/api/v1/admin`、`Domain` は `COOKIE_DOMAIN` 設定時のみ |
| 26 | CORS | 単体 | 許可 Origin のみ反映（`*` は返さない）。許可外・Origin 無しはヘッダ無し。OPTIONS は 204 で next 未呼出 |

### 3.5 メール送信（Outbox）

| # | テストケース | テスト種別 | 期待結果 |
|---|-------------|-----------|----------|
| 1 | 受付メール enqueue | 単体 | Outbox に `reservation_received` が同一 Tx で記録される（実装: `TestCreateReservationUseCase_Execute_mailOutbox`） |
| 2 | 承認・拒否メール enqueue | 単体 | 件名・本文・reason が Outbox に入る |
| 3 | 重複 enqueue | 単体 / 結合 | `ErrMailAlreadyEnqueued`、予約操作は成功、警告ログ |
| 4 | Dispatcher 成功 | 単体 | `sent`、Idempotency-Key = `mail_type/reservation_id`、送信 ctx にタイムアウト、claim に lease 期限 |
| 5 | 一時失敗 | 単体 | claim 時の `attempt_count` に応じた `next_attempt_at` 後退（1m/5m/15m/1h/4h） |
| 6 | 恒久失敗 / 上限 | 単体 | `failed`（6 回目の失敗、または `ErrMailPermanent`） |
| 7 | 認証エラー | 単体 | `ReleaseClaim` で試行を戻し、`halted=auth_error` で残りの行を claim しない |
| 8 | 記録失敗の継続 | 単体 | mark の DB エラーは `errors` に数え、次の行を処理する |
| 9 | 時間予算 | 単体 | 残り予算 < 送信タイムアウトで `halted=time_budget`、次の claim をしない |
| 10 | claim 失敗 | 単体 | DB エラーはそのまま返す（flush は 500） |
| 11 | Claim と lease | 結合（PostgreSQL） | claim で `attempt_count + 1`、lease 中は再 claim されない、lease 切れで attempt 2 として再 claim |
| 12 | Claim 直列化 | 結合（PostgreSQL） | 同一 reservation の先行 pending（lease 中）がある間は後続を取らない。先行が `sent` / `failed` なら取る |
| 13 | mark 系 | 結合（PostgreSQL） | `MarkRetry` は attempt を増やさない、`ReleaseClaim` は attempt を戻す、`sent` 後の mark はエラー |
| 14 | Resend ステータス分類 | 単体 | 5xx/429/409 concurrent は一時、401/403 は `ErrMailAuth`、他 4xx は `ErrMailPermanent` |
| 15 | flush エンドポイント | 単体 | POST 200、GET 405 |

## 4. フロントエンドテストケース

### 4.1 React コンポーネントテスト

| # | コンポーネント | テストケース | 期待結果 |
|---|--------------|-------------|----------|
| 1 | ReservationForm | 必須項目未入力で送信 | バリデーションエラーが表示される |
| 2 | ReservationForm | 正常な入力で送信 | API が呼ばれ、完了画面に遷移 |
| 3 | ReservationForm | 予約不可日がカレンダーで選択不可 | `GET /reservations/availability`（月次）の `is_holiday` / `available` に従う |
| 4 | ReservationForm | 残り食数が表示される | 日付選択後に残り食数を表示 |
| 5 | LoginForm | 正しい認証情報で送信 | ダッシュボードに遷移 |
| 6 | LoginForm | 認証失敗 | エラーメッセージが表示される |
| 6a | LoginForm | 429 `TOO_MANY_REQUESTS` | サーバの待ち時間入りメッセージが表示される |
| 7 | ReservationTable | 予約一覧が表示される | 予約データがカードで表示される |
| 8 | ReservationTable | 承認ボタンクリック | ステータスが更新される |
| 9 | ReservationTable | 拒否ボタンクリック | ステータスが更新される |
| 10 | ScheduleCalendar | 月間スケジュールが表示される | 各日のスケジュールタイプが描画される |
| 11 | SupplierManager | 取引先の追加・編集・削除 | CRUD 操作が正常に動作する |
| 12 | SupplierManager | ドラッグ＆ドロップで並び替え | 新しい順序で並び順更新 API が呼ばれる。失敗時は一覧を取り直す |
| 13 | ReservationCreateForm | 提供数超過の登録 | `window.confirm` で確認し、キャンセルなら登録しない。空き取得失敗時も `window.confirm('空き状況を確認できませんでした…このまま登録しますか？')` で確認する。`is_holiday` の日は確認なし |
| 14 | ScheduleCalendar | 保存・定例に戻す | `PUT` / `DELETE` が呼ばれ、外部イベントは capacity 0、戻り先の定例をプレビュー表示 |
| 15 | DashboardSummary | 今日・明日のサマリー | 残り提供数・承認待ち／承認済み件数・直近 3 件。片方の API 失敗でも他方は表示 |
| 16 | PublicScheduleCalendar / SupplierList | 公開ページの表示 | 取得中 → 表示 / 空 / エラーの切り替え、休・Event バー・Event Info |
| 17 | TurnstileWidget | `window.turnstile` をモック | `render` に siteKey が渡り、callback のトークンが `onToken` に伝播、unmount で `remove` |
| 18 | ui/* | Alert / Button / ButtonLink / Card / ConfirmModal / Modal / Textarea | props に応じた描画、`onClick` / `onClose` / `disabled`、Modal のフォーカストラップ・Escape・入力中にフォーカスを奪わない |

実装済みのテストファイルは各コンポーネントと同じディレクトリの `*.test.tsx` を参照（`cd frontend && bun run test:run` で全件実行）。

## 5. E2E テストシナリオ

> **未実装**（手動確認用シナリオ。Playwright 等は未導入）

### 5.1 顧客の予約フロー

```
1. トップページにアクセス
2. 「予約する」ボタンをクリック
3. カレンダーから来店日を選択
4. 残り食数が表示されることを確認
5. 来店時間を選択
6. 人数を選択
7. 名前・電話番号・メールアドレスを入力
8. Turnstile を通過
9. 「予約を申請する」ボタンをクリック
10. 完了ページが表示されることを確認
11. 予約内容が正しく表示されることを確認
```

### 5.2 オーナーの予約管理フロー

```
1. /admin/login にアクセス
2. メールアドレスとパスワードを入力してログイン
3. ダッシュボードが表示されることを確認
4. 予約一覧ページに遷移
5. 日付を選択して予約一覧を表示
6. 申請中の予約を承認
7. ステータスが「承認済み」に変わることを確認
8. 別の予約を拒否
9. ステータスが「拒否」に変わることを確認
```

### 5.3 オーナーのスケジュール設定フロー

```
1. ログイン済みの状態でスケジュール設定ページに遷移
2. カレンダーから日付を選択
3. スケジュールタイプを「イベント」に変更
4. イベント名・説明・提供可能数を入力
5. 「保存する」ボタンをクリック
6. カレンダー上にイベントが反映されることを確認
```

### 5.4 オーナーの取引先管理フロー

```
1. ログイン済みの状態で取引先管理ページに遷移
2. 「取引先を追加」ボタンをクリック
3. 取引先情報を入力して保存
4. 一覧に追加されることを確認
5. 取引先を編集して保存
6. 変更が反映されることを確認
7. 取引先を削除
8. 一覧から消えることを確認
```

### 5.5 未認証アクセスの検証

```
1. 未ログイン状態で /admin にアクセス
2. ログインページにリダイレクトされることを確認
3. 未ログイン状態で /admin/reservations にアクセス
4. ログインページにリダイレクトされることを確認
```

## 6. テスト実行方法

### バックエンド

```bash
# 全テスト実行
cd backend && go test ./...

# カバレッジ付き
cd backend && go test -cover ./...

# 特定パッケージのテスト
cd backend && go test ./internal/domain/service/...

# 詳細出力
cd backend && go test -v ./...
```

### フロントエンド

```bash
# 全テスト実行（単発）
cd frontend && bun run test:run

# カバレッジ付き（frontend/coverage/ に出力）
cd frontend && bun run test:coverage

# ウォッチモード（開発中）
cd frontend && bun run test

# 特定ファイルのテスト
cd frontend && bunx vitest run src/components/react/ReservationForm.test.tsx
```
