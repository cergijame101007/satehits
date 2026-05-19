# テーブル設計

## 1. ER図（Mermaid）

```mermaid
erDiagram
    reservations {
        int id PK "予約ID"
        string name "予約者名"
        int people "人数"
        date visit_date "来店日"
        time visit_time "来店時間"
        string phone "電話番号"
        string email "メールアドレス"
        string note "備考"
        string status "ステータス"
        string source "予約経路"
        timestamp created_at "作成日時"
        timestamp updated_at "更新日時"
    }

    daily_schedules {
        date date PK "日付"
        string schedule_type "スケジュールタイプ"
        int capacity "提供可能数"
        string event_name "イベント名"
        string event_description "イベント説明"
        time open_time "開店時間"
        time last_order_time "ラストオーダー時間"
        time close_time "閉店時間"
        timestamp created_at "作成日時"
        timestamp updated_at "更新日時"
    }

    suppliers {
        int id PK "取引先ID"
        string name "取引先名"
        string description "説明文"
        string instagram_url "InstagramURL"
        string image_url "画像URL"
        int display_order "表示順"
        boolean is_active "表示フラグ"
        timestamp created_at "作成日時"
        timestamp updated_at "更新日時"
    }

    admin_users {
        int id PK "ユーザーID"
        string email "メールアドレス"
        string password_hash "パスワードハッシュ"
        string role "ロール"
        timestamp created_at "作成日時"
        timestamp updated_at "更新日時"
    }

    schema_migrations {
        bigint version PK "マイグレーション番号"
        timestamp applied_at "適用日時"
    }

    daily_schedules ||--o{ reservations : "date"
```

## 2. テーブル定義

### 2.1 reservations（予約）

予約情報を管理するテーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|----------|----------|------|------------|------|
| id | SERIAL | NO | auto | 予約ID（主キー） |
| name | TEXT | NO | - | 予約者名 |
| people | INTEGER | NO | - | 人数（1〜7） |
| visit_date | DATE | NO | - | 来店日 |
| visit_time | TIME | NO | - | 来店時間 |
| phone | TEXT | NO | - | 電話番号 |
| email | TEXT | NO | - | メールアドレス |
| note | TEXT | YES | NULL | 備考（魚の火入れ希望・テーブル席希望等） |
| status | TEXT | NO | 'pending' | ステータス |
| source | TEXT | NO | 'web' | 予約経路 |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

**制約:**
- `people`: CHECK (people >= 1 AND people <= 7)
- `status`: CHECK (status IN ('pending', 'approved', 'rejected', 'no_show'))
- `source`: CHECK (source IN ('web', 'instagram', 'phone', 'walk_in', 'other'))

**インデックス:**
- `idx_reservations_visit_date`: visit_date（日付検索用）
- `idx_reservations_status`: status（ステータス絞り込み用）

**sourceの値:**
| 値 | 説明 |
|----|------|
| web | Webサイトからの予約 |
| instagram | Instagram DMからの予約（オーナー登録） |
| phone | 電話からの予約（オーナー登録） |
| walk_in | 飛び込み・直接来店（オーナー登録） |
| other | その他（知人など、オーナー登録） |

### 2.2 daily_schedules（日別スケジュール）

日ごとのスケジュール・営業情報を管理するテーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|----------|----------|------|------------|------|
| date | DATE | NO | - | 日付（主キー） |
| schedule_type | TEXT | NO | - | スケジュールタイプ |
| capacity | INTEGER | NO | 10 | 提供可能数 |
| event_name | TEXT | YES | NULL | イベント名 |
| event_description | TEXT | YES | NULL | イベント説明（顧客に表示） |
| open_time | TIME | YES | NULL | 開店時間（`normal` / `morning` / `special_menu` / `event` で未指定時はアプリが店舗デフォルトを補完。`closed` は常に NULL） |
| last_order_time | TIME | YES | NULL | ラストオーダー時間（上記と同様） |
| close_time | TIME | YES | NULL | 閉店時間（上記と同様） |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

**制約:**
- `capacity`: CHECK (capacity >= 0)
- `schedule_type`: CHECK (schedule_type IN ('normal', 'morning', 'event', 'special_menu', 'closed'))

**schedule_typeの値:**
| 値 | 説明 | 予約 | デフォルト営業時間 |
|----|------|------|-------------------|
| normal | 通常営業（月火水） | 可 | 11:30-15:00 (LO 14:00) |
| morning | 朝営業（土日祝） | 可 | 8:30-15:00 (LO 14:00) |
| event | 店内イベント等（和紅茶をしばく会等）。名称・説明必須 | 可（注意書き表示） | 省略時は曜日別デフォルト |
| special_menu | 特別メニュー（リゾットランチ等） | 可（注意書き表示） | 通常と同じ |
| closed | 臨時休業 | 不可 | - |

**備考（営業可否・提供数の「正」）:**
- **該当日付に行が存在する場合** — その行の `schedule_type`・`capacity`・時刻などが**唯一の正**（顧客向け空き・カレンダー・予約可否はこれに従う）。
- **行が存在しない場合** — アプリケーションがドメイン既定（店の定例カレンダーに基づく曜日別の既定 `schedule_type` / `capacity` など）で**その日の営業設定（有効なスケジュール）**を合成する。臨時変更や例外は、オーナーが `PUT /admin/schedules/{date}` 等で行を作成し、`daily_schedules` に永続化する。

### 2.3 admin_users（管理者ユーザー）

管理画面にログインできるユーザーを管理するテーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|----------|----------|------|------------|------|
| id | SERIAL | NO | auto | ユーザーID（主キー） |
| email | TEXT | NO | - | メールアドレス（ユニーク） |
| password_hash | TEXT | NO | - | パスワードハッシュ（bcrypt） |
| role | TEXT | NO | - | ロール |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

**制約:**
- `email`: UNIQUE
- `role`: CHECK (role IN ('owner', 'developer'))

### 2.4 suppliers（お取り引き先）

お取り引き先の情報を管理するテーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|----------|----------|------|------------|------|
| id | SERIAL | NO | auto | 取引先ID（主キー） |
| name | TEXT | NO | - | 取引先名 |
| description | TEXT | NO | - | 説明文 |
| instagram_url | TEXT | YES | NULL | InstagramのURL |
| image_url | TEXT | YES | NULL | 画像URL |
| display_order | INTEGER | NO | 0 | 表示順（小さいほど上） |
| is_active | BOOLEAN | NO | true | 表示フラグ |
| created_at | TIMESTAMPTZ | NO | NOW() | 作成日時 |
| updated_at | TIMESTAMPTZ | NO | NOW() | 更新日時 |

**インデックス:**
- `idx_suppliers_display_order`: display_order（表示順ソート用）
- `idx_suppliers_is_active`: is_active（表示フィルタ用）

### 2.5 schema_migrations（スキーママイグレーション履歴）

`backend/cmd/migrate` が `migrations/*.sql` を適用した際に、適用済みのバージョン番号を記録するテーブル。`ensureSchemaMigrationsTable` で `CREATE TABLE IF NOT EXISTS` により初回接続時に自動作成される。業務テーブルとは外部キーで結ばない。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|----------|----------|------|------------|------|
| version | BIGINT | NO | - | マイグレーションファイル名の先頭番号（例: `000001_...sql` → `1`）。主キー |
| applied_at | TIMESTAMPTZ | NO | NOW() | 当該バージョンを適用した日時（行 INSERT 時） |

**備考:**
- 1ファイルの SQL をトランザクションで実行し、成功後に `INSERT INTO schema_migrations (version) VALUES (...)` で記録する（実装は `applyMigration`）。

## 3. DDL

```sql
-- マイグレーション履歴
CREATE TABLE IF NOT EXISTS schema_migrations (
    version     BIGINT PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 予約テーブル
CREATE TABLE IF NOT EXISTS reservations (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    people      INTEGER NOT NULL CHECK (people >= 1 AND people <= 7),
    visit_date  DATE NOT NULL,
    visit_time  TIME NOT NULL,
    phone       TEXT NOT NULL,
    email       TEXT NOT NULL,
    note        TEXT,
    status      TEXT NOT NULL DEFAULT 'pending' 
                CHECK (status IN ('pending', 'approved', 'rejected', 'no_show')),
    source      TEXT NOT NULL DEFAULT 'web'
                CHECK (source IN ('web', 'instagram', 'phone', 'walk_in', 'other')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_reservations_visit_date ON reservations(visit_date);
CREATE INDEX idx_reservations_status ON reservations(status);
CREATE INDEX idx_reservations_visit_date_status ON reservations(visit_date, status);

-- 日別スケジュールテーブル
CREATE TABLE IF NOT EXISTS daily_schedules (
    date              DATE PRIMARY KEY,
    schedule_type     TEXT NOT NULL 
                      CHECK (schedule_type IN ('normal', 'morning', 'event', 'special_menu', 'closed')),
    capacity          INTEGER NOT NULL DEFAULT 10 CHECK (capacity >= 0),
    event_name        TEXT,
    event_description TEXT,
    open_time         TIME,
    last_order_time   TIME,
    close_time        TIME,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- お取り引き先テーブル
CREATE TABLE IF NOT EXISTS suppliers (
    id             SERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL,
    instagram_url  TEXT,
    image_url      TEXT,
    display_order  INTEGER NOT NULL DEFAULT 0,
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_suppliers_display_order ON suppliers(display_order);
CREATE INDEX idx_suppliers_is_active ON suppliers(is_active);

-- 管理者ユーザーテーブル
CREATE TABLE IF NOT EXISTS admin_users (
    id             SERIAL PRIMARY KEY,
    email          TEXT NOT NULL UNIQUE,
    password_hash  TEXT NOT NULL,
    role           TEXT NOT NULL CHECK (role IN ('owner', 'developer')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- updated_at自動更新用のトリガー関数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- トリガー設定
CREATE TRIGGER update_reservations_updated_at
    BEFORE UPDATE ON reservations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_daily_schedules_updated_at
    BEFORE UPDATE ON daily_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_suppliers_updated_at
    BEFORE UPDATE ON suppliers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_admin_users_updated_at
    BEFORE UPDATE ON admin_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

## 4. ステータス遷移図

```mermaid
stateDiagram-v2
    [*] --> pending: 予約申請

    pending --> approved: オーナー承認
    pending --> rejected: オーナー拒否

    approved --> no_show: 無断キャンセル

    rejected --> [*]
    no_show --> [*]
    approved --> [*]: 来店完了（状態変更なし）
```

## 5. サンプルデータ

```sql
-- 管理者ユーザー（パスワードは別途ハッシュ化が必要）
INSERT INTO admin_users (email, password_hash, role) VALUES
('owner@example.com', '$2a$10$xxxxx', 'owner'),
('dev@example.com', '$2a$10$xxxxx', 'developer');

-- 日別スケジュール
INSERT INTO daily_schedules (date, schedule_type, capacity, event_name, event_description) VALUES
('2026-02-01', 'morning', 10, NULL, NULL),  -- 日曜・朝営業
('2026-02-02', 'normal', 10, NULL, NULL),   -- 月曜・通常
('2026-02-07', 'morning', 10, NULL, NULL),  -- 土曜・朝営業
('2026-02-11', 'event', 0, '和紅茶をしばく会', '和紅茶をしばく会 入門編@WINE LAB. 通常のランチ営業はおやすみです。'),
('2026-02-23', 'special_menu', 10, 'リゾットランチ', '本日はリゾットランチの日です。');

-- 予約サンプル
INSERT INTO reservations (name, people, visit_date, visit_time, phone, email, note, status, source) VALUES
('山田太郎', 2, '2026-02-10', '12:00', '090-1234-5678', 'yamada@example.com', NULL, 'approved', 'web'),
('佐藤花子', 4, '2026-02-10', '12:30', '080-9876-5432', 'sato@example.com', '魚の火入れ希望', 'pending', 'instagram'),
('鈴木一郎', 3, '2026-02-10', '13:00', '070-1111-2222', 'suzuki@example.com', NULL, 'approved', 'phone');

-- お取り引き先サンプル
INSERT INTO suppliers (name, description, instagram_url, image_url, display_order, is_active) VALUES
('〇〇農園', '富山県で無農薬野菜を栽培されています。旬の野菜を直接仕入れています。', 'https://www.instagram.com/example_farm/', NULL, 1, true),
('△△和紅茶園', '国産和紅茶を生産されています。当店の和紅茶はこちらから仕入れています。', 'https://www.instagram.com/example_tea/', NULL, 2, true),
('□□水産', '富山湾の新鮮な魚を毎朝届けていただいています。', 'https://www.instagram.com/example_fish/', NULL, 3, true);
```
