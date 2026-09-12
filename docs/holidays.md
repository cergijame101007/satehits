# 祝日データの運用

店舗定例の合成（`daily_schedules` に行が無い日の営業設定）で使う「国民の祝日・休日」のデータについて、出典・仕組み・年次更新手順をまとめる。

## 1. 目的と合成ルール

店の定例（[`domain_knowledge.md`](./domain_knowledge.md) §3・§4・§6）では祝日は曜日にかかわらず通常営業（11:30〜15:00、朝営業なし）である。`daily_schedules` に行が無い日は次の順で営業設定を合成する。

| 優先 | 条件 | 合成結果 |
|------|------|----------|
| 1 | 国民の祝日・休日 | `normal`（提供数 10、11:30-15:00）。木・金の祝日は営業日、日曜の祝日は朝営業なし |
| 2 | 木・金 | `closed` |
| 3 | 日 | `morning`（8:30-15:00） |
| 4 | 月火水土 | `normal` |

- `daily_schedules` に行がある日は従来どおり**行が優先**（祝日でもオーナー登録が正）
- `event` 登録時に営業時刻を省略した場合の補完も同じ判定を使う（祝日でない日曜のみ朝営業、それ以外は通常営業）
- 同梱データに対象年が無い場合はエラーにせず「祝日なし」として曜日ルールだけで解決する

## 2. データの出典と同梱先

| 項目 | 内容 |
|------|------|
| 出典 | 内閣府「国民の祝日」CSV: <https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv>（Shift_JIS、`国民の祝日・休日月日,国民の祝日・休日名称`） |
| backend | `backend/internal/domain/holiday/syukujitsu.csv`（UTF-8 / LF に変換したもの。`//go:embed` で読み込み、1 列目の日付のみ判定に使う） |
| frontend | `frontend/src/data/holidays.json`（`["1955-01-01", ...]` の日付配列。管理画面のスケジュール削除確認モーダルで「店舗定例に戻すとこうなる」プレビューに使う） |

両方とも `scripts/update-holidays.sh` が同一の CSV から生成するため、backend の合成結果と frontend のプレビューがずれない。リクエストのたびに外部へ取りに行くことはしない。

backend では祝日集合をパッケージグローバルで参照せず、`service.NationalHolidayChecker` として `StoreCalendar` に注入する（`cmd/api/main.go`）。service 層のテストは stub の祝日集合で書き、同梱データには依存しない。

## 3. 年次更新手順

内閣府 CSV は翌年分が例年 **2 月頃**に公開される（予約は 2 週間先までなので、運用上は 2 週間先の祝日が分かっていればよい）。**毎年 1 回、翌年分の公開後に次を実行してコミットする。**

```bash
make update-holidays      # ダウンロード → UTF-8 変換 → backend CSV / frontend JSON を再生成
git diff --stat           # 差分を確認（翌年分の行が増えていること）
git add backend/internal/domain/holiday/syukujitsu.csv frontend/src/data/holidays.json
git commit -m "chore: 祝日データを 20XX 年分まで更新"
```

必要なコマンド: `curl`, `iconv`, `awk`, `sort`（macOS / Linux 標準）。

更新後は `cd backend && go test ./internal/domain/...` と `cd frontend && bun run test:run` で祝日ケースのテストが通ることを確認する。

### 更新忘れの検知

バックエンドは起動時に同梱データの最終年を確認し、次のいずれかなら警告を出す。

- 最終年が現在年より前（当年の祝日すら同梱されていない）
- 最終年が現在年と同じ（翌年分が未同梱）で、かつ **3 月以降**（翌年分は 2 月頃に公開されるため、それより前は更新しようがなく警告しない）

```
WARNING: bundled holiday data ends at 2027; run `make update-holidays` to bundle next year's holidays (docs/holidays.md)
```

警告が出ても動作は止まらない（対象年の祝日が無いだけ）。Cloud Run のログで見かけたら上記手順で更新する。

## 4. 実装の場所

| レイヤー | ファイル | 役割 |
|----------|----------|------|
| backend Domain | `backend/internal/domain/holiday/holiday.go` | CSV パース（ヘッダ・BOM・不正行スキップ）、`Set.IsNationalHoliday(date)`、`Embedded().LastYear()` / `NeedsUpdate(now)` |
| backend Domain | `backend/internal/domain/service/store_calendar.go` | `StoreCalendar`（`NationalHolidayChecker` を注入）。`DefaultSchedule` が祝日 → 曜日の順で合成、`ApplyEventDefaultBusinessHours` が event の朝／通常を判定、`BookingWindowMinutes` が予約受付時間帯を算出 |
| backend Domain | `backend/internal/domain/service/schedule_resolver.go` | 行が無い日を `StoreCalendar.DefaultSchedule` で合成 |
| backend 起動 | `backend/cmd/api/main.go` | `holiday.Embedded()` を `StoreCalendar` に注入、鮮度警告のログ出力 |
| frontend | `frontend/src/lib/holidays.ts` | `isNationalHoliday(dateStr)` |
| frontend | `frontend/src/lib/storeDefaultSchedule.ts` | 店舗定例プレビュー（祝日 → 曜日） |
| スクリプト | `scripts/update-holidays.sh` / `Makefile` (`update-holidays`) | 年次更新 |
