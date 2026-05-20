# API設計

## 1. 概要

- **ベースURL**: `/api/v1`
- **フォーマット**: JSON
- **認証**: JWT（管理者APIのみ）
- **文字コード**: UTF-8
- **機械可読な定義**: フィールド単位の完全なスキーマ・バリデーション・レスポンス型は **[docs/openapi.yaml](openapi.yaml)**（OpenAPI 3.0.3）を正とする。本書は人間向けの概要と代表例を示す。

## 2. エンドポイント一覧

### 顧客向けAPI（認証不要）

| メソッド | エンドポイント | 説明 |
|----------|----------------|------|
| GET | `/reservations/availability` | 指定日の残り食数を取得 |
| POST | `/reservations` | 予約を申請 |
| GET | `/schedules` | 月間スケジュールを取得（カレンダー用） |
| GET | `/suppliers` | お取り引き先一覧を取得 |

### 管理者向けAPI（認証必要）

| メソッド | エンドポイント | 説明 |
|----------|----------------|------|
| POST | `/admin/login` | ログイン |
| POST | `/admin/refresh` | アクセストークン更新（リフレッシュトークン） |
| POST | `/admin/logout` | ログアウト |
| GET | `/admin/reservations` | 予約一覧を取得 |
| POST | `/admin/reservations` | 予約を登録（オーナー手動・Instagram/電話等） |
| PATCH | `/admin/reservations/{id}/status` | 予約ステータスを更新 |
| GET | `/admin/schedules` | 指定期間のスケジュール一覧を取得 |
| GET | `/admin/schedules/{date}` | 日別スケジュールを取得 |
| PUT | `/admin/schedules/{date}` | 日別スケジュールを設定 |
| DELETE | `/admin/schedules/{date}` | 日別スケジュールを削除（デフォルトに戻す） |
| GET | `/admin/suppliers` | 取引先一覧を取得（非表示含む） |
| POST | `/admin/suppliers` | 取引先を登録 |
| GET | `/admin/suppliers/{id}` | 取引先を取得 |
| PUT | `/admin/suppliers/{id}` | 取引先を更新 |
| DELETE | `/admin/suppliers/{id}` | 取引先を削除 |
| PUT | `/admin/suppliers/order` | 取引先の表示順を更新 |

## 3. 共通仕様

### リクエストヘッダー

| ヘッダー | 値 | 必須 | 説明 |
|----------|-----|------|------|
| Content-Type | application/json | Yes | リクエストボディの形式 |
| Authorization | Bearer {token} | 管理者APIのみ | JWTトークン |

### エラーレスポンス

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "エラーメッセージ"
  }
}
```

バリデーションエラー時は `details` 配列を含む形式（`ValidationErrorResponse`）になる。型定義は OpenAPI を参照。

### エラーコード一覧

| HTTPステータス | コード | 説明 |
|----------------|--------|------|
| 400 | INVALID_REQUEST | リクエスト形式が不正 |
| 400 | VALIDATION_ERROR | バリデーションエラー |
| 401 | UNAUTHORIZED | 認証が必要 |
| 401 | INVALID_TOKEN | トークンが無効 |
| 403 | FORBIDDEN | アクセス権限がない |
| 404 | NOT_FOUND | リソースが見つからない |
| 409 | CAPACITY_EXCEEDED | 予約可能数を超過 |
| 500 | INTERNAL_ERROR | サーバー内部エラー |

---

## 4. 顧客向けAPI詳細

### GET /reservations/availability

指定日の残り食数を取得する。

#### リクエスト

| パラメータ | 位置 | 型 | 必須 | 説明 |
|------------|------|-----|------|------|
| date | query | string | Yes | 日付（YYYY-MM-DD形式） |

```
GET /api/v1/reservations/availability?date=2025-02-10
```

#### レスポンス

**成功時（200 OK）** — 必須フィールドは `date`, `capacity`, `reserved`, `available`, `is_holiday`。イベント日などでは `schedule_type`, `event_name`, `event_description` 等が付く（OpenAPI `AvailabilityResponse`）。

```json
{
  "date": "2025-02-10",
  "capacity": 10,
  "reserved": 6,
  "available": 4,
  "schedule_type": "normal",
  "is_holiday": false
}
```

**定休日など休業相当の例（200 OK）**

```json
{
  "date": "2025-02-13",
  "capacity": 0,
  "reserved": 0,
  "available": 0,
  "schedule_type": null,
  "is_holiday": true
}
```

#### ビジネスロジック（概要）

- `reserved`: その日付の、ステータスが `approved` の予約の人数合計。
- `available`: `capacity - reserved`（負にならないようクリップする等の詳細は実装・OpenAPIの例に従う）。
- `is_holiday` / `schedule_type` / `capacity`: **`daily_schedules` の該当日行を正**とし、行が無い日はドメイン既定で合成した**その日の営業設定（有効なスケジュール）**に基づく（[docs/domain_knowledge.md](domain_knowledge.md) も参照）。

---

### POST /reservations

Web からの予約を申請する。リクエストボディに **`status` や `source` は含めない**（サーバが `pending` / `web` として扱う想定。悪意あるクライアントによる上書きを防ぐ）。

#### リクエスト

```json
{
  "name": "山田太郎",
  "people": 2,
  "visit_date": "2025-02-10",
  "visit_time": "12:00",
  "phone": "090-1234-5678",
  "email": "yamada@example.com",
  "note": "エビアレルギーあり",
  "recaptcha_token": "xxxxx"
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| name | string | Yes | 予約者名 |
| people | integer | Yes | 人数（1〜7） |
| visit_date | string | Yes | 来店日（YYYY-MM-DD） |
| visit_time | string | Yes | 来店時間（HH:MM） |
| phone | string | Yes | 電話番号 |
| email | string | Yes | メールアドレス |
| note | string | No | 備考 |
| recaptcha_token | string | Yes | reCAPTCHAトークン |

#### バリデーションルール

| フィールド | ルール |
|------------|--------|
| name | 1〜100文字 |
| people | 1〜7の整数 |
| visit_date | 翌日〜14日後の範囲内 |
| visit_date | その日の営業設定（有効なスケジュール）において予約不可の日は不可（`daily_schedules` の行を正、無ければドメイン既定で解決。`closed` や休業相当の日を含む） |
| visit_time | 営業時間内（月火水: 11:30-14:00、土日祝: 8:30-14:00） |
| phone | 電話番号形式 |
| email | メールアドレス形式 |
| note | 最大500文字 |

#### レスポンス

**成功時（201 Created）** — `ReservationResponse` に従い、少なくとも `id`, `name`, `people`, `visit_date`, `visit_time`, `phone`, `email`, `status`, `source`, `created_at` を返す。

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "山田太郎",
  "people": 2,
  "visit_date": "2025-02-10",
  "visit_time": "12:00",
  "phone": "090-1234-5678",
  "email": "yamada@example.com",
  "note": "エビアレルギーあり",
  "status": "pending",
  "created_at": "2025-02-01T10:00:00+09:00"
}
```

`id` は UUID 文字列（RFC 4122）。

**二重予約時（409 Conflict）** — 同一電話番号で同一来店日時に `pending` / `approved` の予約が既にある場合（DB 部分ユニーク `idx_reservations_unique_active`）。

```json
{
  "error": {
    "code": "RESERVATION_CONFLICT",
    "message": "同じ日時の予約が既に登録されています"
  }
}
```

**バリデーションエラー時（400 Bad Request）**

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "入力内容に誤りがあります",
    "details": [
      { "field": "people", "message": "人数は1〜7名で指定してください" },
      { "field": "visit_date", "message": "この日は予約できません" }
    ]
  }
}
```

**予約可能数超過時（409 Conflict）**

```json
{
  "error": {
    "code": "CAPACITY_EXCEEDED",
    "message": "この日の予約可能数を超えています"
  }
}
```

---

### GET /schedules

指定月のスケジュール一覧を取得する（顧客向けカレンダー用）。

#### リクエスト

| パラメータ | 位置 | 型 | 必須 | 説明 |
|------------|------|-----|------|------|
| year | query | integer | Yes | 年 |
| month | query | integer | Yes | 月（1〜12） |

```
GET /api/v1/schedules?year=2025&month=2
```

#### レスポンス

**成功時（200 OK）** — `MonthlyScheduleResponse`（`year`, `month`, `schedules`）。各要素は `DaySchedule`（`date`, `schedule_type`, `capacity`, `available`, `is_holiday` 等）。

---

### GET /suppliers

表示対象のお取り引き先一覧を取得する。

#### レスポンス

**成功時（200 OK）** — `SupplierListResponse`。各要素は `PublicSupplier`（`id`, `name`, `description` 必須、`instagram_url`, `image_url` は任意）。

---

## 5. 管理者向けAPI詳細

### 認証方式（概要）

管理者 API は **アクセストークン（AT）** と **リフレッシュトークン（RT）** の二層で認証する。

| トークン | 形式 | 有効期限 | 保存場所（クライアント） | 用途 |
|----------|------|----------|--------------------------|------|
| AT | JWT（HS256） | 1時間 | JS メモリ（React state 等）。`localStorage` には保存しない | `Authorization: Bearer {AT}` で保護 API を呼ぶ |
| RT | 不透明ランダム文字列（DB に SHA-256 ハッシュのみ保存） | 発行から30日（スライディング。下記） | `httpOnly` Cookie | `POST /admin/refresh`・`POST /admin/logout` のみ |

**環境変数**（`pkg/config.Load()` で読み込み。`JWT_SECRET` は起動時に 32 バイト未満なら fatal）:

| 変数 | 説明 |
|------|------|
| `JWT_SECRET` | AT 署名用（32 バイト以上。例: `openssl rand -base64 48`） |
| `CORS_ORIGINS` | 許可オリジン（カンマ区切り。例: `https://satehits.com,http://localhost:4321`） |
| `COOKIE_DOMAIN` | RT Cookie の `Domain`。開発時は空、本番は API ホスト名 |

**AT の JWT クレーム**（検証時は `alg=HS256` のみ許可、`iss` / `aud` / 時刻クレームを必須検証）:

```json
{
  "sub": "1",
  "email": "owner@example.com",
  "role": "owner",
  "iss": "satehits-api",
  "aud": "satehits-admin",
  "exp": 1738425600,
  "iat": 1738422000,
  "nbf": 1738422000,
  "jti": "550e8400-e29b-41d4-a716-446655440000"
}
```

- `sub`: `admin_users.id`（文字列）
- `role`: `owner` または `developer`

**RT Cookie 仕様**（ログイン成功時および `refresh` 成功時に `Set-Cookie`）:

| 属性 | 値 |
|------|-----|
| 名前 | `refresh_token` |
| `HttpOnly` | 必須 |
| `Secure` | 必須（本番 HTTPS） |
| `SameSite` | `None`（Cloudflare Pages ↔ Cloud Run のクロスオリジン） |
| `Path` | `/api/v1/admin` |
| `Max-Age` | 2592000（30日） |
| `Domain` | `COOKIE_DOMAIN` が空なら省略、設定時のみ付与 |

JSON ボディに RT は含めない（XSS で読まれないようにする）。

**CSRF 緩和**: RT を読む `POST /admin/refresh` と `POST /admin/logout` では、サーバが `Origin` または `Referer` を `CORS_ORIGINS` の許可リストと照合する。不一致は **403 Forbidden**（`FORBIDDEN`）。

**RT ローテーション**: `refresh` 成功時に旧 RT を DB で revoke し、新 RT を Cookie で返す。既に revoke 済みの RT が再送された場合は漏洩疑いとして当該ユーザの全 RT を revoke し **401**（`INVALID_TOKEN`）。

**RT 有効期限（スライディング）**: ログインおよび `refresh` 成功時に、DB の `expires_at` と Cookie の `Max-Age` を **その時点から30日後** に設定する。30日以内に一度でも `refresh` が成功すれば期限は延びる。**連続30日間** 一度も `refresh` されなかった RT は無効となり、次回の管理画面アクセスではパスワードによる再ログインが必要（初回ログインから30日で固定切れにはしない）。

**保護 API**: 上記以外の `/admin/*` は `Authorization: Bearer {AT}` 必須。ヘッダ無し → `UNAUTHORIZED`、検証失敗・期限切れ → `INVALID_TOKEN`。

---

### POST /admin/login

メールアドレスとパスワードで認証し、AT を JSON で返し、RT を Cookie で返す。

#### リクエスト

```json
{
  "email": "owner@example.com",
  "password": "password123"
}
```

#### レスポンス

**成功時（200 OK）** — ボディは OpenAPI `LoginResponse`（`refresh_token` は含めない）。`Set-Cookie` で RT を付与。

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-02-01T22:00:00+09:00",
  "user": {
    "id": 1,
    "email": "owner@example.com",
    "role": "owner"
  }
}
```

レスポンスヘッダ例:

```
Set-Cookie: refresh_token=<opaque>; HttpOnly; Secure; SameSite=None; Path=/api/v1/admin; Max-Age=2592000
```

**認証失敗時（401 Unauthorized）**

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "メールアドレスまたはパスワードが正しくありません"
  }
}
```

---

### POST /admin/refresh

Cookie の RT で新しい AT を発行する。成功時は RT をローテーションし、新 RT を `Set-Cookie` で返す。

#### リクエスト

- ボディ: なし
- `Cookie: refresh_token=<opaque>` 必須
- フロントは `fetch(..., { credentials: 'include' })` で呼ぶ
- `Origin` または `Referer` が許可オリジンであること（CSRF 緩和）

#### レスポンス

**成功時（200 OK）** — ボディは `LoginResponse` と同型（`token`, `expires_at`。`user` は省略可）。

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2025-02-01T23:00:00+09:00"
}
```

**トークン無効・期限切れ・再利用検知時（401 Unauthorized）**

```json
{
  "error": {
    "code": "INVALID_TOKEN",
    "message": "セッションが無効です。再度ログインしてください"
  }
}
```

**Origin / Referer 不一致時（403 Forbidden）**

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "リクエストが拒否されました"
  }
}
```

---

### POST /admin/logout

Cookie の RT を revoke し、同名 Cookie を削除する。リクエストには有効な AT（`Authorization: Bearer`）も付与する。

#### リクエスト

- `Authorization: Bearer {AT}` 必須
- `Cookie: refresh_token=<opaque>`
- `Origin` / `Referer` 検証（`refresh` と同様）

#### レスポンス

**成功時（204 No Content）** — `Set-Cookie` で `refresh_token` を `Max-Age=0` にして削除。

**認証エラー時（401 Unauthorized）** — AT 無し・無効時は `UNAUTHORIZED` / `INVALID_TOKEN`。

---

### GET /admin/reservations

予約一覧を取得する。

#### リクエスト

| パラメータ | 位置 | 型 | 必須 | 説明 |
|------------|------|-----|------|------|
| date | query | string | No | 日付で絞り込み（YYYY-MM-DD） |
| status | query | string | No | ステータスで絞り込み（`pending` / `approved` / `rejected` / `cancelled` / `no_show`） |
| source | query | string | No | 予約経路で絞り込み（`web` / `instagram` / `phone` / `walk_in` / `other`） |

```
GET /api/v1/admin/reservations?date=2025-02-10
GET /api/v1/admin/reservations?date=2025-02-10&status=pending
```

#### レスポンス

**成功時（200 OK）** — `ReservationListResponse`。各予約は `ReservationResponse`（`status`, `source` を含む）。

```json
{
  "reservations": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "山田太郎",
      "people": 2,
      "visit_date": "2025-02-10",
      "visit_time": "12:00",
      "phone": "090-1234-5678",
      "email": "yamada@example.com",
      "note": "エビアレルギーあり",
      "status": "pending",
      "source": "web",
      "created_at": "2025-02-01T10:00:00+09:00",
      "updated_at": "2025-02-01T10:00:00+09:00"
    }
  ],
  "total": 1
}
```

---

### POST /admin/reservations

Instagram・電話・知人経由など、オーナーが手動で予約を登録する。Web 申請（`POST /reservations`）とは異なり、**`source` を必ず指定**し、**`status` を省略可能**（省略時は `approved`）。

#### 列挙値（OpenAPI と同一）

- **status（任意）**: `pending`, `approved`, `rejected`, `cancelled`, `no_show`
- **source（必須）**: `web`, `instagram`, `phone`, `walk_in`, `other`

#### リクエスト

```json
{
  "name": "鈴花子",
  "people": 2,
  "visit_date": "2025-02-12",
  "visit_time": "11:30",
  "phone": "080-0000-0000",
  "email": "suzuki@example.com",
  "note": "電話予約",
  "source": "phone",
  "status": "approved"
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| name | string | Yes | 予約者名 |
| people | integer | Yes | 人数（1〜7） |
| visit_date | string | Yes | 来店日 |
| visit_time | string | Yes | 来店時間 |
| phone | string | Yes | 電話番号 |
| email | string | Yes | メールアドレス |
| note | string | No | 備考 |
| source | string | Yes | 予約経路 |
| status | string | No | 省略時は `approved` |

#### レスポンス

**成功時（201 Created）** — `ReservationResponse`（公開申請と同型）。

---

### PATCH /admin/reservations/{id}/status

予約の**ステータスだけ**を更新する（サブリソース `/status`）。

#### リクエスト

```json
{
  "status": "approved"
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| status | string | Yes | `approved` / `rejected` / `cancelled` / `no_show`（`UpdateStatusRequest`） |

パス `{id}` は予約 UUID。形式不正は 400。

#### ステータス遷移ルール

| 現在のステータス | 変更可能なステータス |
|------------------|----------------------|
| pending | approved, rejected, cancelled |
| approved | no_show, cancelled |
| rejected | （変更不可） |
| cancelled | （変更不可） |
| no_show | （変更不可） |

`cancelled` は Web キャンセル機能ではなく、顧客からのメール連絡を受けたオーナーが管理画面から手動で設定する。

#### レスポンス

**成功時（200 OK）**

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "approved",
  "updated_at": "2025-02-01T12:00:00+09:00"
}
```

---

### GET /admin/schedules

指定期間（月）のスケジュール一覧を取得する。

#### リクエスト

| パラメータ | 位置 | 型 | 必須 | 説明 |
|------------|------|-----|------|------|
| year | query | integer | Yes | 年 |
| month | query | integer | Yes | 月（1〜12） |

```
GET /api/v1/admin/schedules?year=2025&month=2
```

#### レスポンス

**成功時（200 OK）** — `ScheduleListResponse`（`schedules` は `ScheduleResponse` の配列）。

**バリデーションエラー時（400 Bad Request）** — `year` / `month` クエリの欠落・空白のみ・非数値は handler で `VALIDATION_ERROR`（`details[].field` は `year` または `month`）。範囲外（年 2000〜2100、月 1〜12）はユースケース層で同じ形式。

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "入力内容に誤りがあります",
    "details": [
      { "field": "year", "message": "年は必須です" },
      { "field": "month", "message": "月は必須です" }
    ]
  }
}
```

---

### GET /admin/schedules/{date}

指定日のスケジュールを取得する。

#### レスポンス

**成功時（200 OK）** — `ScheduleResponse`（`date`, `schedule_type`, `capacity` 必須。`event_name`, `open_time` 等は OpenAPI 参照）。

---

### PUT /admin/schedules/{date}

指定日のスケジュールを設定する（`daily_schedules` への Upsert。初回・更新とも同一エンドポイント）。

#### リクエスト

```
PUT /api/v1/admin/schedules/2025-02-11
```

パス `{date}` は `YYYY-MM-DD`。ボディは `SetScheduleRequest`（**`date` フィールドは含めない**）。

**必須は `schedule_type`**。`capacity` は省略時デフォルト（OpenAPI上は説明で「10」）。

| `schedule_type` | `event_name` / `event_description` | 時刻（`open_time` 等） |
|-----------------|-------------------------------------|-------------------------|
| `normal` / `morning` / `special_menu` | 任意（`special_menu` はメニュー名など） | 省略時は店舗デフォルトを適用 |
| `event` | **両方必須** | **任意**（省略時は曜日別の店舗デフォルトを適用） |
| `closed` | 不要 | 送信しても保存しない（常に NULL） |

```json
{
  "schedule_type": "normal",
  "capacity": 10
}
```

イベント日の例（時刻省略・平日なら 11:30–15:00 が入る）:

```json
{
  "schedule_type": "event",
  "capacity": 10,
  "event_name": "和紅茶をしばく会",
  "event_description": "和紅茶をしばく会 入門編@店内開催"
}
```

#### レスポンス

**成功時（200 OK）** — `ScheduleResponse`（新規作成・更新のいずれも 200）

---

### DELETE /admin/schedules/{date}

指定日のスケジュールを削除し、デフォルト設定に戻す。

#### レスポンス

**成功時（204 No Content）**

---

### GET /admin/suppliers

全取引先を取得する（非表示 `is_active: false` も含む）。

#### レスポンス

**成功時（200 OK）** — `AdminSupplierListResponse`（`SupplierResponse` の配列）。

---

### POST /admin/suppliers

取引先を登録する。

#### リクエスト

`CreateSupplierRequest`。**必須は `name`, `description`**。`instagram_url`, `image_url`, `display_order`, `is_active` は任意（省略時の挙動は OpenAPI の description に従う）。

#### レスポンス

**成功時（201 Created）** — `SupplierResponse`

---

### GET /admin/suppliers/{id}

取引先を1件取得する。

#### レスポンス

**成功時（200 OK）** — `SupplierResponse`

---

### PUT /admin/suppliers/{id}

取引先を更新する。`UpdateSupplierRequest` の各フィールドはすべて任意（部分更新）。

#### レスポンス

**成功時（200 OK）** — `SupplierResponse`

---

### DELETE /admin/suppliers/{id}

取引先を削除する。

#### レスポンス

**成功時（204 No Content）**

---

### PUT /admin/suppliers/order

取引先の表示順を一括更新する。

#### リクエスト

```json
{
  "order": [3, 1, 2]
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| order | integer[] | Yes | 取引先IDを表示順に並べた配列 |

#### レスポンス

**成功時（200 OK）** — `AdminSupplierListResponse`
