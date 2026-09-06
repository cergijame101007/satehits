# シーケンス図

## 1. 残り食数確認（顧客）

```mermaid
sequenceDiagram
    participant Customer as 顧客
    participant Frontend as フロントエンド
    participant Handler as Handler
    participant UseCase as UseCase
    participant AvailService as AvailabilityService
    participant ScheduleRepo as ScheduleRepository
    participant ReservationRepo as ReservationRepository
    participant DB as Supabase

    Customer->>Frontend: 日付を選択
    Frontend->>Handler: GET /reservations/availability?date=2026-02-10
    Handler->>UseCase: GetAvailability(date)
    
    UseCase->>AvailService: GetAvailability(date)
    
    AvailService->>ScheduleRepo: FindByDate(date)
    ScheduleRepo->>DB: SELECT * FROM daily_schedules WHERE date = ?
    DB-->>ScheduleRepo: schedule (or null)
    ScheduleRepo-->>AvailService: Schedule row or null
    
    AvailService->>AvailService: その日の営業設定（有効なスケジュール）の解決（DB行を正、無ければドメイン既定で合成）
    
    AvailService->>ReservationRepo: SumReservedPeopleByDate(date)
    ReservationRepo->>DB: SELECT SUM(people) FROM reservations WHERE visit_date = ? AND status IN ('pending', 'approved')
    DB-->>ReservationRepo: 6
    ReservationRepo-->>AvailService: 6
    
    AvailService->>AvailService: available = capacity - reserved
    AvailService-->>UseCase: Availability{capacity: 10, reserved: 6, available: 4}
    
    UseCase-->>Handler: AvailabilityResponse
    Handler-->>Frontend: 200 OK { available: 4, schedule_type: "normal" }
    Frontend-->>Customer: 残り4食と表示
```

## 2. 予約申請成功（顧客）

```mermaid
sequenceDiagram
    participant Customer as 顧客
    participant Frontend as フロントエンド
    participant Handler as Handler
    participant UseCase as CreateUseCase
    participant ReCaptcha as reCAPTCHA
    participant AvailService as AvailabilityService
    participant Locker as VisitDateLocker
    participant ReservationRepo as ReservationRepository
    participant MailService as MailService
    participant DB as Supabase
    participant Resend as Resend

    Customer->>Frontend: 予約情報を入力して送信
    Frontend->>Frontend: Turnstile実行
    Frontend->>Handler: POST /reservations {name, people, date, time, ...}
    
    Handler->>UseCase: Execute(request)
    
    UseCase->>ReCaptcha: Verify(token)
    ReCaptcha-->>UseCase: OK
    
    UseCase->>UseCase: バリデーション（人数、日付範囲、営業時間）
    
    UseCase->>AvailService: ResolveForDate(date) 楽観チェック
    AvailService-->>UseCase: OK（空きあり）

    Note over UseCase,DB: DoInTx（同一 visit_date を直列化）
    UseCase->>Locker: Lock(visit_date)
    Locker->>DB: SELECT pg_advisory_xact_lock(ns, YYYYMMDD)
    DB-->>Locker: acquired
    UseCase->>AvailService: ResolveForDate(date) 再確認
    AvailService->>ReservationRepo: SumReservedPeopleByDate(date)
    ReservationRepo->>DB: SELECT SUM(people) ...
    DB-->>ReservationRepo: reserved
    AvailService-->>UseCase: OK（空きあり）
    
    UseCase->>ReservationRepo: Create(reservation)
    ReservationRepo->>DB: INSERT INTO reservations
    DB-->>ReservationRepo: OK
    Note right of DB: COMMIT で advisory lock 解放
    ReservationRepo-->>UseCase: reservation
    
    UseCase->>MailService: SendReservationReceived(reservation)
    Note right of UseCase: インプロセス非同期キューへ投入（fire-and-forget）
    UseCase-->>Handler: ReservationResponse
    Handler-->>Frontend: 201 Created
    Frontend-->>Customer: 完了画面へ遷移

    MailService->>Resend: POST /emails（予約申請受付メール）
    Resend-->>MailService: OK
    Note right of MailService: 宛先: 顧客メールアドレス<br>失敗時はログのみ
```

## 3. 予約申請失敗（空きなし）

```mermaid
sequenceDiagram
    participant Customer as 顧客
    participant Frontend as フロントエンド
    participant Handler as Handler
    participant UseCase as CreateUseCase
    participant ReCaptcha as reCAPTCHA
    participant AvailService as AvailabilityService

    Customer->>Frontend: 予約情報を入力して送信
    Frontend->>Handler: POST /reservations
    
    Handler->>UseCase: Execute(request)
    
    UseCase->>ReCaptcha: Verify(token)
    ReCaptcha-->>UseCase: OK
    
    UseCase->>UseCase: バリデーション OK
    
    UseCase->>AvailService: CanReserve(date, people)
    AvailService-->>UseCase: Error: CAPACITY_EXCEEDED
    
    UseCase-->>Handler: Error
    Handler-->>Frontend: 409 Conflict { code: "CAPACITY_EXCEEDED" }
    Frontend-->>Customer: エラーメッセージを表示
```

## 4. ログイン（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Handler as AuthHandler
    participant UseCase as LoginUseCase
    participant AdminUserRepo as AdminUserRepository
    participant RefreshRepo as RefreshTokenRepository
    participant JWT as JWTService
    participant DB as Supabase

    Owner->>Frontend: メールアドレス・パスワードを入力
    Frontend->>Handler: POST /admin/login {email, password} (credentials: include)
    
    Handler->>UseCase: Execute(email, password)
    
    UseCase->>AdminUserRepo: FindByEmail(email)
    AdminUserRepo->>DB: SELECT * FROM admin_users WHERE email = ?
    DB-->>AdminUserRepo: user
    AdminUserRepo-->>UseCase: AdminUser
    
    UseCase->>UseCase: bcrypt.Compare(password, hash)
    UseCase->>JWT: Generate(user_id, email, role)
    JWT-->>UseCase: access_token, expires_at
    UseCase->>UseCase: リフレッシュトークン生成 (crypto/rand)
    UseCase->>RefreshRepo: Issue(user_id, sha256(rt), expires_at=now+30d)
    RefreshRepo->>DB: INSERT INTO refresh_tokens
    DB-->>RefreshRepo: OK
    RefreshRepo-->>UseCase: OK
    
    UseCase-->>Handler: LoginResult{access_token, refresh_token, user}
    Handler-->>Frontend: 200 OK JSON {token, expires_at, user}<br/>Set-Cookie: refresh_token (HttpOnly)
    Frontend->>Frontend: アクセストークンをメモリに保持（localStorage は使わない）
    Frontend-->>Owner: ダッシュボードへ遷移
```

## 4.1. アクセストークン更新（リフレッシュ）

ページ再読み込みや AT 期限切れ（401）時に、Cookie の RT で新しい AT を取得する。

正常系は **検証 → 読み取り（ユーザー取得・AT 生成）→ 旧 RT の Revoke と新 RT の Issue を 1 トランザクション** の順で行う。副作用のない処理を先に済ませ、状態を変える Revoke / Issue を最後にまとめることで、途中失敗でセッションを壊さない。

```mermaid
sequenceDiagram
    participant Frontend as フロントエンド
    participant Handler as AuthHandler
    participant UseCase as RefreshUseCase
    participant AdminRepo as AdminUserRepository
    participant RefreshRepo as RefreshTokenRepository
    participant JWT as JWTService
    participant DB as Supabase

    Frontend->>Handler: POST /admin/refresh<br/>Cookie: refresh_token<br/>Origin: 許可オリジン (credentials: include)
    
    Handler->>Handler: Origin / Referer 検証
    Handler->>UseCase: Execute(refresh_token)
    
    UseCase->>RefreshRepo: FindByTokenHash(sha256(rt))
    RefreshRepo->>DB: SELECT WHERE token_hash = $1（revoked / 期限は絞らない）
    DB-->>RefreshRepo: row / none
    
    alt 該当行なし
        RefreshRepo-->>UseCase: not found
        UseCase-->>Handler: INVALID_TOKEN
        Handler-->>Frontend: 401 → /admin/login へ
    else 既に revoke 済み RT の再利用（revoked_at != NULL）
        RefreshRepo-->>UseCase: row
        UseCase->>RefreshRepo: RevokeAllByUser(user_id)
        UseCase-->>Handler: INVALID_TOKEN
        Handler-->>Frontend: 401
    else 期限切れ（expires_at <= now）
        RefreshRepo-->>UseCase: row
        UseCase-->>Handler: INVALID_TOKEN
        Handler-->>Frontend: 401 → /admin/login へ
    else 正常
        UseCase->>AdminRepo: FindByID(user_id)
        AdminRepo-->>UseCase: admin_user
        UseCase->>JWT: Generate(user_id, email, role)
        JWT-->>UseCase: 新 access_token
        UseCase->>RefreshRepo: 1 トランザクションで Revoke(旧 RT) → Issue(新 RT, expires_at=now+30d)
        RefreshRepo->>DB: BEGIN → UPDATE revoked_at → INSERT → COMMIT
        DB-->>RefreshRepo: OK
        UseCase-->>Handler: RefreshResult
        Handler-->>Frontend: 200 {token, expires_at}<br/>Set-Cookie: 新 refresh_token
        Frontend->>Frontend: メモリ上の AT を更新
    end
```

## 4.2. ログアウト（オーナー）

Cookie の RT をサーバ側で revoke し、Cookie を削除する。有効な AT（`Authorization: Bearer`）も必須（`api_design.md` 参照）。

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Handler as AuthHandler
    participant Middleware as AuthMiddleware
    participant UseCase as LogoutUseCase
    participant RefreshRepo as RefreshTokenRepository
    participant DB as Supabase

    Owner->>Frontend: ログアウトを実行
    Frontend->>Middleware: POST /admin/logout<br/>Authorization: Bearer {AT}<br/>Cookie: refresh_token<br/>Origin: 許可オリジン (credentials: include)

    Middleware->>Middleware: JWT 検証（AT）
    alt AT 無し・無効・期限切れ
        Middleware-->>Frontend: 401 { code: "UNAUTHORIZED" or "INVALID_TOKEN" }
        Frontend-->>Owner: ログイン画面へ
    else AT 有効
        Middleware->>Handler: Request（ユーザー context 付き）
        Handler->>Handler: Origin / Referer 検証
        alt Origin / Referer 不一致
            Handler-->>Frontend: 403 { code: "FORBIDDEN" }
        else 検証 OK
            Handler->>UseCase: Execute(refresh_token)
            UseCase->>RefreshRepo: FindByTokenHash(sha256(rt))
            RefreshRepo->>DB: SELECT WHERE token_hash = $1（revoked は絞らない）
            DB-->>RefreshRepo: row / none
            UseCase->>RefreshRepo: Revoke(該当 RT) ※未登録・失効済みでも成功扱い
            RefreshRepo->>DB: UPDATE refresh_tokens SET revoked_at = NOW()
            DB-->>RefreshRepo: OK
            UseCase-->>Handler: OK
            Handler-->>Frontend: 204 No Content<br/>Set-Cookie: refresh_token (Max-Age=0 で削除)
            Frontend->>Frontend: メモリ上の AT を破棄
            Frontend-->>Owner: ログイン画面へ遷移
        end
    end
```

## 5. 予約一覧取得（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Middleware as AuthMiddleware
    participant Handler as ReservationHandler
    participant UseCase as ListUseCase
    participant ReservationRepo as ReservationRepository
    participant DB as Supabase

    Owner->>Frontend: 予約一覧画面を開く
    Frontend->>Middleware: GET /admin/reservations?date=2026-02-10 (with JWT)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Handler: Request (with user context)
    
    Handler->>UseCase: Execute(date, status, source)
    UseCase->>ReservationRepo: FindByDate(date)
    ReservationRepo->>DB: SELECT * FROM reservations WHERE visit_date = ?
    DB-->>ReservationRepo: [reservation1, reservation2, ...]
    ReservationRepo-->>UseCase: []*Reservation
    
    UseCase-->>Handler: ReservationListResponse
    Handler-->>Frontend: 200 OK {reservations: [...], total: 5}
    Frontend-->>Owner: 予約一覧を表示
```

## 6. 予約承認（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Middleware as AuthMiddleware
    participant Handler as ReservationHandler
    participant UseCase as UpdateStatusUseCase
    participant ReservationRepo as ReservationRepository
    participant MailService as MailService
    participant DB as Supabase
    participant Resend as Resend

    Owner->>Frontend: 「承認」ボタンをクリック
    Frontend->>Middleware: PATCH /admin/reservations/123/status (with JWT)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Handler: Request
    
    Handler->>UseCase: Execute(id=123, status="approved")
    
    UseCase->>ReservationRepo: FindByID(123)
    ReservationRepo->>DB: SELECT * FROM reservations WHERE id = 123
    DB-->>ReservationRepo: reservation
    ReservationRepo-->>UseCase: Reservation{status: "pending"}
    
    UseCase->>UseCase: reservation.CanTransitionTo("approved") → true
    UseCase->>UseCase: reservation.TransitionTo("approved")
    
    UseCase->>ReservationRepo: Update(reservation)
    ReservationRepo->>DB: UPDATE reservations SET status = 'approved' WHERE id = 123
    DB-->>ReservationRepo: OK
    
    UseCase->>MailService: SendReservationApproved(reservation)
    Note right of UseCase: インプロセス非同期キューへ投入
    UseCase-->>Handler: UpdateStatusResponse
    Handler-->>Frontend: 200 OK {id: 123, status: "approved"}
    Frontend-->>Owner: ステータス更新を反映

    MailService->>Resend: POST /emails（予約承認メール）
    Resend-->>MailService: OK
    Note right of MailService: 宛先: 顧客メールアドレス<br>内容: 予約確定、来店日時、キャンセルポリシー<br>失敗時はログのみ
```

## 7. 予約拒否（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Middleware as AuthMiddleware
    participant Handler as ReservationHandler
    participant UseCase as UpdateStatusUseCase
    participant ReservationRepo as ReservationRepository
    participant MailService as MailService
    participant DB as Supabase
    participant Resend as Resend

    Owner->>Frontend: 「拒否」ボタンをクリック（任意で理由入力）
    Frontend->>Middleware: PATCH /admin/reservations/123/status {status, reason?} (with JWT)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Handler: Request
    
    Handler->>UseCase: Execute(id=123, status="rejected", reason?)
    
    UseCase->>ReservationRepo: FindByID(123)
    ReservationRepo->>DB: SELECT * FROM reservations WHERE id = 123
    DB-->>ReservationRepo: reservation
    ReservationRepo-->>UseCase: Reservation{status: "pending"}
    
    UseCase->>UseCase: reservation.CanTransitionTo("rejected") → true
    UseCase->>UseCase: reservation.TransitionTo("rejected")
    
    UseCase->>ReservationRepo: Update(reservation)
    ReservationRepo->>DB: UPDATE reservations SET status = 'rejected' WHERE id = 123
    DB-->>ReservationRepo: OK
    
    UseCase->>MailService: SendReservationRejected(reservation, reason?)
    Note right of UseCase: reason あり時のみメール本文に「理由：」行を含める
    UseCase-->>Handler: UpdateStatusResponse
    Handler-->>Frontend: 200 OK {id: 123, status: "rejected"}
    Frontend-->>Owner: ステータス更新を反映

    MailService->>Resend: POST /emails（予約拒否メール）
    Resend-->>MailService: OK
    Note right of MailService: 宛先: 顧客メールアドレス<br>内容: 予約不可の旨、Instagramへの誘導<br>失敗時はログのみ
```

## 8. メール送信シーケンス

予約関連メールは、予約 INSERT / ステータス更新と**同一トランザクション**で `email_outbox` に送信意図を記録する。HTTP レスポンスは DB コミット後に返却し、メール送信自体は予約成立の必須条件としない。Cloud Scheduler（1 分ごと）が private Cloud Run の `POST /internal/outbox/flush` を呼び、Dispatcher が `MailSender`（本番は Resend）へ送信する。失敗時は指数バックオフで再送し、上限到達または恒久エラーで `failed` にする（ADR-014）。

### 8.1 予約申請受付メール（顧客へ）

```mermaid
sequenceDiagram
    participant UseCase as CreateReservationUseCase
    participant DB as PostgreSQL
    participant Flush as OutboxFlush
    participant Sender as MailSender
    participant Resend as Resend API

    UseCase->>DB: BEGIN
    UseCase->>DB: INSERT reservations
    UseCase->>DB: INSERT email_outbox (reservation_received)
    UseCase->>DB: COMMIT
    Note over Flush: Cloud Scheduler → POST /internal/outbox/flush
    Flush->>DB: ClaimNextPending (FOR UPDATE SKIP LOCKED)
    Flush->>Sender: Send (Idempotency-Key=mail_type/reservation_id)
    Sender->>Resend: POST /emails
    Resend-->>Sender: 200 OK
    Flush->>DB: MarkSent
```

### 8.2 予約承認メール（顧客へ）

```mermaid
sequenceDiagram
    participant UseCase as UpdateStatusUseCase
    participant DB as PostgreSQL
    participant Flush as OutboxFlush
    participant Sender as MailSender
    participant Resend as Resend API

    UseCase->>DB: BEGIN
    UseCase->>DB: UPDATE reservations status=approved
    UseCase->>DB: INSERT email_outbox (reservation_approved)
    UseCase->>DB: COMMIT
    Note over Flush: Cloud Scheduler → POST /internal/outbox/flush
    Flush->>DB: ClaimNextPending
    Note right of Flush: 同一 reservation に先行 pending があればスキップ
    Flush->>Sender: Send (Idempotency-Key)
    Sender->>Resend: POST /emails
    Resend-->>Sender: 200 OK
    Flush->>DB: MarkSent
```

### 8.3 予約拒否メール（顧客へ）

拒否理由 `reason` は DB カラムに残さず、enqueue 時に本文へ埋め込んで Outbox に保存する。フローは 8.2 と同様（`mail_type=reservation_rejected`）。

## 9. スケジュール設定（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Middleware as AuthMiddleware
    participant Handler as ScheduleHandler
    participant UseCase as SetScheduleUseCase
    participant ScheduleRepo as ScheduleRepository
    participant DB as Supabase

    Owner->>Frontend: スケジュール情報を入力して保存
    Frontend->>Middleware: PUT /admin/schedules/2026-02-11 (with JWT)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Handler: Request {schedule_type: "event", event_name: "和紅茶をしばく会", ...}
    
    Handler->>UseCase: Execute(date, request)
    
    UseCase->>UseCase: バリデーション
    UseCase->>UseCase: Scheduleエンティティ生成
    
    UseCase->>ScheduleRepo: Upsert(schedule)
    ScheduleRepo->>DB: INSERT INTO daily_schedules ... ON CONFLICT (date) DO UPDATE
    DB-->>ScheduleRepo: OK
    ScheduleRepo-->>UseCase: schedule
    
    UseCase-->>Handler: ScheduleResponse
    Handler-->>Frontend: 200 OK {date, schedule_type, event_name, ...}
    Frontend-->>Owner: 設定完了を表示
```

## 10. 予約登録（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Middleware as AuthMiddleware
    participant Handler as ReservationHandler
    participant UseCase as CreateByAdminUseCase
    participant ReservationRepo as ReservationRepository
    participant DB as Supabase

    Owner->>Frontend: Instagram経由の予約情報を入力
    Frontend->>Middleware: POST /admin/reservations (with JWT)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Handler: Request {name, people, date, source: "instagram", status: "approved"}
    
    Handler->>UseCase: Execute(request)
    
    UseCase->>UseCase: バリデーション（reCAPTCHA不要）
    UseCase->>UseCase: Reservationエンティティ生成（source, statusを設定）
    
    UseCase->>ReservationRepo: Create(reservation)
    ReservationRepo->>DB: INSERT INTO reservations (source='instagram', status='approved')
    DB-->>ReservationRepo: OK (id=124)
    ReservationRepo-->>UseCase: reservation
    
    UseCase-->>Handler: ReservationResponse
    Handler-->>Frontend: 201 Created
    Frontend-->>Owner: 登録完了を表示
```

## 11. 認証エラー

```mermaid
sequenceDiagram
    participant Client as クライアント
    participant Middleware as AuthMiddleware
    participant Handler as Handler

    Client->>Middleware: GET /admin/reservations (トークンなし or 無効)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Middleware: 検証失敗
    
    Middleware-->>Client: 401 Unauthorized { code: "UNAUTHORIZED" }
```

## 12. 取引先登録（オーナー）

```mermaid
sequenceDiagram
    participant Owner as オーナー
    participant Frontend as フロントエンド
    participant Middleware as AuthMiddleware
    participant Handler as SupplierHandler
    participant UseCase as CreateSupplierUseCase
    participant SupplierRepo as SupplierRepository
    participant DB as Supabase

    Owner->>Frontend: 取引先情報を入力して保存
    Frontend->>Middleware: POST /admin/suppliers (with JWT)
    
    Middleware->>Middleware: JWT検証
    Middleware->>Handler: Request {name, description, instagram_url, ...}
    
    Handler->>UseCase: Execute(request)
    
    UseCase->>UseCase: バリデーション
    UseCase->>UseCase: Supplierエンティティ生成
    
    UseCase->>SupplierRepo: Create(supplier)
    SupplierRepo->>DB: INSERT INTO suppliers
    DB-->>SupplierRepo: OK (id=1)
    SupplierRepo-->>UseCase: supplier
    
    UseCase-->>Handler: SupplierResponse
    Handler-->>Frontend: 201 Created
    Frontend-->>Owner: 登録完了を表示
```
