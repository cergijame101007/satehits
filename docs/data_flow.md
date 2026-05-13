# データフロー図

## 1. システム全体のデータフロー

```mermaid
flowchart TB
    subgraph External[外部]
        Customer[顧客]
        Owner[オーナー]
        ReCaptcha[reCAPTCHA]
    end

    subgraph Frontend[フロントエンド - Cloudflare Pages]
        CustomerUI[顧客画面]
        AdminUI[管理画面]
    end

    subgraph Backend[バックエンド - Cloud Run]
        API[Go API Server]
    end

    subgraph Database[データベース - Supabase]
        DB[(PostgreSQL)]
    end

    subgraph Mail[メール配信]
        Resend[Resend]
    end

    Customer -->|予約情報入力| CustomerUI
    CustomerUI -->|reCAPTCHAトークン| ReCaptcha
    ReCaptcha -->|検証結果| API
    CustomerUI -->|予約申請・空き確認・スケジュール| API

    Owner -->|ログイン・予約管理・スケジュール設定| AdminUI
    AdminUI -->|JWT認証付きリクエスト| API

    API -->|CRUD| DB
    DB -->|データ| API
    API -->|レスポンス| CustomerUI
    API -->|レスポンス| AdminUI
    API -->|メール送信| Resend
    Resend -->|受付・承認・拒否メール| Customer
```

## 2. 予約申請のデータフロー

```mermaid
flowchart LR
    subgraph Input[入力データ]
        Name[名前]
        People[人数]
        Date[来店日]
        Time[来店時間]
        Phone[電話番号]
        Email[メールアドレス]
        Note[備考]
        Token[reCAPTCHAトークン]
    end

    subgraph Validation[バリデーション]
        V1[人数: 1-7]
        V2[日付: 翌日〜14日後]
        V3[営業可否<br/>その日の営業設定（有効なスケジュール）]
        V4[営業時間チェック]
        V5[空き確認]
    end

    subgraph Process[処理]
        ReCaptcha[reCAPTCHA検証]
        AvailService[AvailabilityService]
        Create[予約作成]
        SendMail[受付メール送信]
    end

    subgraph Output[出力データ]
        Reservation[予約レコード]
        Response[APIレスポンス]
    end

    Input --> ReCaptcha
    ReCaptcha --> Validation
    V5 --> AvailService
    AvailService --> Create
    Create --> Reservation
    Reservation --> SendMail
    SendMail --> Response
```

## 3. 残り食数の計算フロー

`daily_schedules` に該当日の行がある場合はその行を正とし、**行がない場合**はドメイン既定（店の定例に基づく曜日別の既定など）で**その日の営業設定（有効なスケジュール）**を合成する。`AvailabilityService` はその内容から `capacity`・休業相当かどうかを決め、`reservations` の承認済人数と組み合わせて残りを算出する。

```mermaid
flowchart TB
    subgraph Input[入力]
        Date[指定日付]
    end

    subgraph Resolve[その日の営業設定（有効なスケジュール）の解決]
        RowExists{daily_schedules に<br/>該当日の行がある?}
        FromRow[行の schedule_type / capacity を採用]
        FromDefault[無い場合はドメイン既定で合成]
    end

    subgraph Effective[当該日の capacity・種別]
        Eff[その日の営業設定<br/>（有効なスケジュール）]
    end

    subgraph ReservationData[予約データ集計]
        Reservations[reservations]
        SumPeople[承認済み予約の人数合計]
    end

    subgraph Calculation[計算]
        Calc[残り食数 = 提供可能数 - 承認済み人数]
    end

    subgraph Output[出力]
        Result[AvailabilityResponse]
    end

    Date --> RowExists
    RowExists -->|Yes| FromRow --> Eff
    RowExists -->|No| FromDefault --> Eff
    Eff --> Calc
    Reservations --> SumPeople --> Calc
    Calc --> Result
```

## 4. 認証フロー

```mermaid
flowchart TB
    subgraph Login[ログイン]
        Email[メールアドレス]
        Password[パスワード]
    end

    subgraph Verify[検証]
        FindUser[ユーザー検索]
        CompareHash[パスワード照合]
    end

    subgraph Token[トークン生成]
        GenerateJWT[JWT生成]
        SetExpiry[有効期限設定]
    end

    subgraph Response[レスポンス]
        JWTToken[JWTトークン]
        UserInfo[ユーザー情報]
    end

    Email --> FindUser
    Password --> CompareHash
    FindUser --> CompareHash
    CompareHash -->|OK| GenerateJWT
    GenerateJWT --> SetExpiry
    SetExpiry --> JWTToken
    SetExpiry --> UserInfo
```

## 5. ステータス遷移

```mermaid
stateDiagram-v2
    [*] --> pending: 予約申請

    pending --> approved: オーナー承認
    pending --> rejected: オーナー拒否

    approved --> no_show: 無断キャンセル記録

    rejected --> [*]
    no_show --> [*]
    approved --> [*]: 来店完了（ステータス変更なし）
```

## 6. スケジュールタイプと予約可否

**その日の営業設定（有効なスケジュール）**（DB 行または行なし時の合成結果）に応じて予約可否を決める。`closed` や `capacity: 0` に相当する日、および**既定解決の結果として休業相当となった日**は予約不可となる。

```mermaid
flowchart TB
    subgraph ScheduleType[その日の営業設定（有効なスケジュール）に応じた区分]
        Normal[normal: 通常営業]
        Morning[morning: 朝営業]
        Event[event: イベント]
        Special[special_menu: 特別メニュー]
        Closed[closed: 臨時休業等]
        NoBook[その他 予約不可と解決された日]
    end

    subgraph Reservable[予約可否]
        Yes[予約可能]
        YesWithNote[予約可能（注意書き表示）]
        No[予約不可]
    end

    Normal --> Yes
    Morning --> Yes
    Event --> YesWithNote
    Special --> YesWithNote
    Closed --> No
    NoBook --> No
```

## 7. API と テーブルの関係

```mermaid
flowchart TB
    subgraph CustomerAPI[顧客向けAPI]
        GetAvailability[GET /reservations/availability]
        CreateReservation[POST /reservations]
        GetSchedules[GET /schedules]
        GetSuppliers[GET /suppliers]
    end

    subgraph AdminAPI[管理者向けAPI]
        Login[POST /admin/login]
        GetReservations[GET /admin/reservations]
        CreateByAdmin[POST /admin/reservations]
        UpdateStatus[PATCH /admin/reservations/:id/status]
        SetSchedule[PUT /admin/schedules/:date]
        ManageSuppliers[Suppliers CRUD Operations]
    end

    subgraph Tables[テーブル]
        ReservationsTable[(reservations)]
        SchedulesTable[(daily_schedules)]
        SuppliersTable[(suppliers)]
        AdminUsersTable[(admin_users)]
    end

    GetAvailability --> SchedulesTable
    GetAvailability --> ReservationsTable
    CreateReservation --> ReservationsTable
    GetSchedules --> SchedulesTable
    GetSuppliers --> SuppliersTable

    Login --> AdminUsersTable
    GetReservations --> ReservationsTable
    CreateByAdmin --> ReservationsTable
    UpdateStatus --> ReservationsTable
    SetSchedule --> SchedulesTable
    ManageSuppliers --> SuppliersTable
```

## 8. 予約経路によるデータフローの違い

```mermaid
flowchart TB
    subgraph WebReservation[Webからの予約]
        WebInput[顧客入力]
        ReCaptcha[reCAPTCHA検証]
        WebValidation[厳密なバリデーション]
        WebCreate[予約作成 source=web, status=pending]
        WebMail[受付メール送信]
    end

    subgraph AdminReservation[オーナー登録の予約]
        AdminInput[オーナー入力]
        AdminValidation[簡易バリデーション]
        AdminCreate[予約作成 source=instagram等, status=approved]
    end

    subgraph DB[データベース]
        ReservationsTable[(reservations)]
    end

    subgraph MailService[メール配信]
        Resend[Resend]
    end

    WebInput --> ReCaptcha --> WebValidation --> WebCreate --> ReservationsTable
    WebCreate --> WebMail --> Resend
    AdminInput --> AdminValidation --> AdminCreate --> ReservationsTable
```

## 9. 取引先データフロー

```mermaid
flowchart TB
    subgraph AdminOperations[管理者操作]
        Create[取引先登録]
        Update[取引先編集]
        Delete[取引先削除]
        Reorder[表示順変更]
    end

    subgraph Database[データベース]
        SuppliersTable[(suppliers)]
    end

    subgraph CustomerView[顧客向け表示]
        API[GET /suppliers]
        Filter[is_active=true のみ]
        Sort[display_order順]
        Display[取引先紹介ページ]
    end

    Create --> SuppliersTable
    Update --> SuppliersTable
    Delete --> SuppliersTable
    Reorder --> SuppliersTable

    SuppliersTable --> API --> Filter --> Sort --> Display
```
