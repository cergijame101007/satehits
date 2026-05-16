# Go テストのコーディングルール（バックエンド）

予約・スケジュール API のバックエンド（`backend/`）で `go test` を書くときの共通方針。

**読みやすさの目標:** `go test -v` のサブテスト名だけ見て、何を検証しているか分かること（英語の短い文章で書く）。

## 基本方針

| 項目 | ルール |
|------|--------|
| フレームワーク | 標準の `testing` のみ（`testify` 等は導入しない） |
| 配置 | 対象と同じパッケージの `*_test.go`（非公開関数の単体テスト可） |
| 構造 | **テーブル駆動** + `t.Run` サブテスト（Go の定番） |
| サブテスト名 | **英語の文章**（例: `rejects empty schedule_type`, `accepts valid command`） |
| `t.Parallel()` | **原則使わない**（出力が読みにくくなる。I/O や DB など遅いテストで必要なときだけ） |
| 成功時のログ | `t.Log` は原則不要（説明はサブテスト名に書く） |
| 失敗メッセージ | `got` / `want` を明示する |
| コメント | **日本語**（ドメイン制約・暫定ルール・参照ドキュメント） |

## テーブル駆動の型

```go
tests := []struct {
    name string // go test -v にそのまま出る。何を検証するか英語で書く
    // 入力・期待値
}{
    {name: "accepts valid command", ...},
    {name: "rejects empty schedule_type", wantField: "schedule_type"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // ...
    })
}
```

大きな `TestXxx` 関数の下に `t.Run` でケースを分けると、`-run` で絞り込みやすい。

## ヘルパー

- 検証の重複は `t.Helper()` 付きの `assertNoViolations` 等を **テストファイル内** に置く
- 正常系の入力組み立ては `validCreateXxxCommand()` にまとめる

## 実行

```bash
# 普段（失敗時のみ詳細）
go test ./...

# サブテスト名を確認したいとき
go test ./internal/application/usecase/schedule/... -v

# 1 ケースだけ
go test ./... -run 'TestValidateCreateSchedule_scheduleType/rejects_empty_schedule_type'
```

## バリデーションテストで押さえる観点

1. **正常系**（最小入力・代表入力）
2. **必須フィールド**（ゼロ値・空白のみ）
3. **列挙・形式**（`schedule_type`、メール、電話など）
4. **境界値**（min/max 文字数、人数、capacity）
5. **複合ルール**（営業時刻の前後関係、来店日範囲と曜日）
6. **ドキュメント準拠**（`docs/table_design.md`、`docs/openapi.yaml` の制約）

## 参照

- `backend/internal/application/usecase/schedule/create_schedule_validation_test.go`
- `backend/internal/application/usecase/reservation/create_reservation_validate_test.go`
