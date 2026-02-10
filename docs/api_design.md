# API設計

## 1. 概要

- **ベースURL**: `/api/v1`
- **フォーマット**: JSON
- **認証**: JWT（管理者APIのみ）
- **文字コード**: UTF-8

## 2. エンドポイント一覧

### 顧客向けAPI（認証不要）

| メソッド | エンドポイント | 説明 |
|----------|----------------|------|
| GET | `/reservations/availability` | 残り食数を取得 |
| POST | `/reservations` | 予約を申請 |

### 管理者向けAPI（認証必要）

| メソッド | エンドポイント | 説明 |
|----------|----------------|------|
| POST | `/admin/login` | ログイン |
| POST | `/admin/logout` | ログアウト |
| GET | `/admin/reservations` | 予約一覧を取得 |
| PATCH | `/admin/reservations/{id}/status` | 予約ステータスを更新 |
| GET | `/admin/capacity/{date}` | 提供可能数を取得 |
| PUT | `/admin/capacity/{date}` | 提供可能数を設定 |

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

**成功時（200 OK）**
```json
{
  "date": "2025-02-10",
  "capacity": 10,
  "reserved": 6,
  "available": 4,
  "is_holiday": false
}
```

**定休日の場合（200 OK）**
```json
{
  "date": "2025-02-13",
  "capacity": 0,
  "reserved": 0,
  "available": 0,
  "is_holiday": true
}
```

#### ビジネスロジック
- `reserved` = 指定日のステータスが `approved` の予約の合計人数
- `available` = `capacity` - `reserved`
- `is_holiday` = 木曜または金曜の場合 `true`

---

### POST /reservations

予約を申請する。

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
| visit_date | 木・金（定休日）は不可 |
| visit_time | 営業時間内（月火水: 11:30-14:00、土日祝: 8:30-14:00） |
| phone | 電話番号形式 |
| email | メールアドレス形式 |
| note | 最大500文字 |

#### レスポンス

**成功時（201 Created）**
```json
{
  "id": 123,
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

**バリデーションエラー時（400 Bad Request）**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "入力内容に誤りがあります",
    "details": [
      { "field": "people", "message": "人数は1〜7名で指定してください" },
      { "field": "visit_date", "message": "定休日は予約できません" }
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

## 5. 管理者向けAPI詳細

### POST /admin/login

ログインしてJWTトークンを取得する。

#### リクエスト

```json
{
  "email": "owner@example.com",
  "password": "password123"
}
```

#### レスポンス

**成功時（200 OK）**
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

**認証失敗時（401 Unauthorized）**
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "メールアドレスまたはパスワードが正しくありません"
  }
}
```

#### JWTペイロード

```json
{
  "sub": 1,
  "email": "owner@example.com",
  "role": "owner",
  "exp": 1738425600,
  "iat": 1738382400
}
```

---

### POST /admin/logout

ログアウトする（トークンを無効化）。

#### レスポンス

**成功時（204 No Content）**

---

### GET /admin/reservations

予約一覧を取得する。

#### リクエスト

| パラメータ | 位置 | 型 | 必須 | 説明 |
|------------|------|-----|------|------|
| date | query | string | No | 日付で絞り込み（YYYY-MM-DD） |
| status | query | string | No | ステータスで絞り込み |

```
GET /api/v1/admin/reservations?date=2025-02-10
GET /api/v1/admin/reservations?date=2025-02-10&status=pending
```

#### レスポンス

**成功時（200 OK）**
```json
{
  "reservations": [
    {
      "id": 123,
      "name": "山田太郎",
      "people": 2,
      "visit_date": "2025-02-10",
      "visit_time": "12:00",
      "phone": "090-1234-5678",
      "email": "yamada@example.com",
      "note": "エビアレルギーあり",
      "status": "pending",
      "created_at": "2025-02-01T10:00:00+09:00",
      "updated_at": "2025-02-01T10:00:00+09:00"
    }
  ],
  "total": 1
}
```

---

### PATCH /admin/reservations/{id}/status

予約のステータスを更新する。

#### リクエスト

```json
{
  "status": "approved"
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| status | string | Yes | 新しいステータス（approved / rejected / no_show） |

#### ステータス遷移ルール

| 現在のステータス | 変更可能なステータス |
|------------------|----------------------|
| pending | approved, rejected |
| approved | no_show |
| rejected | （変更不可） |
| no_show | （変更不可） |

#### レスポンス

**成功時（200 OK）**
```json
{
  "id": 123,
  "status": "approved",
  "updated_at": "2025-02-01T12:00:00+09:00"
}
```

---

### GET /admin/capacity/{date}

指定日の提供可能数を取得する。

#### レスポンス

**成功時（200 OK）**
```json
{
  "date": "2025-02-10",
  "capacity": 10
}
```

**未設定の場合（200 OK）**
```json
{
  "date": "2025-02-10",
  "capacity": 10,
  "is_default": true
}
```

---

### PUT /admin/capacity/{date}

指定日の提供可能数を設定する。

#### リクエスト

```json
{
  "capacity": 8
}
```

| フィールド | 型 | 必須 | 説明 |
|------------|-----|------|------|
| capacity | integer | Yes | 提供可能数（0以上） |

#### レスポンス

**成功時（200 OK）**
```json
{
  "date": "2025-02-10",
  "capacity": 8,
  "updated_at": "2025-02-01T12:00:00+09:00"
}
```
