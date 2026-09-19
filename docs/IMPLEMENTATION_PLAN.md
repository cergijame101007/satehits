# 実装計画: 監視（ADR-012 / A: GCP 純正 + D: UptimeRobot）

> [monitoring.md](./monitoring.md) を実装に落とす手順書。**上から順に、1 ステップずつ**進める。
> 各ステップは「1 回の作業セッションで完了できる粒度」になっている。飛ばさない・まとめてやらない。

## 進め方のルール

- 各ステップの開始時に `docs/monitoring.md` の該当セクションを読み直す（D-NN は同書 §2）
- 完了条件をすべて満たし、**検証コマンドが通ってから**次へ進む
- ステップ内で設計と食い違いを見つけたら、勝手に設計を変えず `docs/monitoring.md` の修正を先に提案する
- 完了したステップはこのファイルのチェックボックスを `[x]` に更新する（進捗の共有メモリとして使う）
- コミットは 1 ステップ 1 コミット以上（`feat:` / `refactor:` / `docs:` / `ci:`。日本語一行。`CLAUDE.md` §2）。push・PR はユーザーの指示があるときだけ
- GCP・UptimeRobot への操作（Step 6〜8）は外に影響が出るので、実行前に何をするか一言添えて 1 つずつ行う

## 検証コマンド（全ステップ共通）

```bash
# リポジトリルートで
make lint && make test && make build
```

`make lint` は `tools/bin/golangci-lint` が無ければ `make tools-install` を先に実行する。CI と同じ `-race` で確認したいときは `cd backend && go test -race ./...`。

---

## Step 0: ベースライン確認と要確認事項の仮置き

- [x] `make lint && make test && make build` が現状の `develop` で通ることを確認する（通らなければ監視の作業に入る前に原因を報告する）
- [x] `grep -rn "log\." backend --include=*_test.go` で、テストが `log` パッケージの出力文字列に依存していないことを確認する（依存があれば Step 2 の影響範囲として控える）
- [x] 要確認事項（`monitoring.md` §2 末尾）の仮値を決める: 通知先メール = 開発者自身のアドレスで先行。UptimeRobot = 開発者アカウントで先行
- 完了条件: 検証コマンドが通る。上記 grep の結果（依存の有無）を報告に書く

## Step 1: 構造化ログ基盤 `internal/logging`（D-04 / D-05 / D-06）

- [x] `backend/internal/logging/logging.go` を作る: `Setup(environment string) *slog.Logger`、`NewCloudLoggingHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler`（`monitoring.md` §4.1 のキー変換表どおり。`WARN` → `WARNING`、`msg` → `message`、`level` → `severity`、`source` → `logging.googleapis.com/sourceLocation`、`AddSource: true`）
- [x] `development` は `slog.NewTextHandler(os.Stdout, ...)`、それ以外は `NewCloudLoggingHandler(os.Stdout, ...)`。どちらも `slog.SetDefault` する
- [x] `backend/internal/logging/logging_test.go`: `bytes.Buffer` に書いて `json.Unmarshal` し、(1) `severity=="WARNING"`、(2) `message`、(3) `logging.googleapis.com/sourceLocation` キーの存在、(4) 任意属性がトップレベルに出る、をテーブル駆動で検証（`docs/coding_rule/go_testing.md`）
- [x] `backend/cmd/api/main.go`: `cfg := config.Load()` の直後に `logger := logging.Setup(cfg.Environment)` を入れる。既存の `log.Println("Connected to Database!")` 等はそのまま（INFO で slog に流れる）
- 完了条件: 検証コマンドが通る。`cd backend && ENVIRONMENT=staging JWT_SECRET=$(head -c 48 /dev/zero | tr '\0' x) CORS_ORIGINS=http://localhost DATABASE_URL=postgres://invalid TURNSTILE_SECRET_KEY=x go run ./cmd/api 2>&1 | head -3` の出力が JSON で、`"severity"` と `"message"` キーを含む（DB 接続失敗で終了してよい。**このコマンドの環境変数はダミー値であり、`.env` は読まない**）

## Step 2: 既存ログ呼び出しのレベル付け（D-05、§4.2 の表）

- [x] `monitoring.md` §4.2 の表の全行を、表のとおり `slog.Error` / `slog.Warn` に書き換える（表にない行は触らない。メッセージ・属性キーも表に合わせる）
- [x] `backend/internal/handler/outbox.go` の `HandleFlush` の 500 分岐に `slog.Error("outbox flush failed", "err", err)` を追加する
- [x] `backend/cmd/api/main.go` の `log.Fatalf` 3 箇所を `slog.Error(...)` + `os.Exit(1)` にする（`config.Load` 内の `log.Fatal` は変えない。D-14）
- [x] 書き換えたファイルから未使用になった `"log"` import を外す（`goimports` が指摘する）
- [x] `mail/dispatcher_test.go` 等、ログ文言に依存するテストがあれば（Step 0 の grep 結果）新しいメッセージに合わせる
- 完了条件: 検証コマンドが通る。`grep -rn 'log\.Printf("ERROR\|log\.Printf("WARN' backend/internal backend/cmd/api` が 0 件。`grep -rn "slog\.Error" backend/internal/handler | wc -l` が 11 以上（表の handler 行 10 + outbox 1）

## Step 3: Recover ミドルウェア（D-07、§4.3）

- [x] `backend/internal/handler/recover.go`: `Recover(next http.Handler) http.Handler` と、`WriteHeader` / `Write` 済みを記録する最小の `ResponseWriter` ラッパー。`http.ErrAbortHandler` は再 panic。ERROR ログは `"panic"`, `"method"`, `"path"`, `"stack_trace"` 属性
- [x] `backend/internal/handler/recover_test.go`: §4.3 の 4 ケース（500 JSON / ErrAbortHandler 再 panic / 素通し / ヘッダ送信後は WriteHeader しない）。既存 `middleware_test.go` の `recordingHandler` の流儀に合わせる
- [x] `backend/cmd/api/main.go` の `srv.Handler` を `handler.Recover(handler.CORS(cfg.CORSOrigins)(http.DefaultServeMux))` にする
- 完了条件: 検証コマンドが通る。`go test ./internal/handler/ -run TestRecover -v` でサブテスト 4 件が PASS

## Step 4: `GET /healthz`（D-08 / D-09 / D-15、§4.4 / §4.5）

- [x] `backend/internal/handler/health.go`: `Pinger` インターフェース、`NewHealthHandler(db Pinger)`、`HandleHealth`（2 秒タイムアウト、200 `{"status":"ok","database":"ok"}` / 503 `{"status":"error","database":"error"}` + `slog.Warn`、405、`Cache-Control: no-store`）
- [x] `backend/internal/handler/health_test.go`: フェイク `Pinger` で 200 / 503 / タイムアウト時 503 / 405
- [x] `backend/internal/handler/method_not_allowed_test.go` の対象一覧に `/healthz` を追加
- [x] `backend/cmd/api/main.go`: `http.HandleFunc("/healthz", handler.NewHealthHandler(db).HandleHealth)` を `HandleRoot` の登録の直後に追加
- [x] `.github/workflows/deploy-backend.yml` のスモークテストの `curl` 先を `${PUBLIC_URL}/healthz` に変更
- [x] `docs/api_design.md` §2 に「運用向け（認証不要）」の小表を追加し `GET /healthz` を載せる（`POST /internal/outbox/flush` の記述位置に合わせる。openapi.yaml には載せない）
- [x] `CLAUDE.md` §6 のエンドポイント表に `GET /healthz` を追加
- 完了条件: 検証コマンドが通る。`make dev` で起動した API に `curl -si localhost:8080/healthz` が `200` と `{"status":"ok","database":"ok"}` を返す（実行した curl の結果を報告に書く。Docker が使えない環境なら「未確認」と明記する）

## Step 5: staging デプロイで構造化ログを確認

- [ ] Step 1〜4 を `develop` に push（ユーザーの指示があってから）し、`Backend Deploy (staging)` が成功することを確認する（スモークが `/healthz` で通る）
- [ ] Cloud Console → Logging で `resource.labels.service_name="satehits-api-stg"` を開き、アプリのログが `jsonPayload.message` / `severity` で表示されること、`severity=INFO` に起動ログ（`Connected to Database!`）が出ていることを確認する
- 完了条件: staging のログに `jsonPayload` としてパースされたエントリがあることを、ログ抜粋 1 行とともに報告に書く

## Step 6: `scripts/setup-monitoring.sh`（D-01〜D-03 / D-11〜D-13、§5）

- [ ] `scripts/setup-monitoring.sh` を `setup-scheduler.sh` と同じ流儀で作る（§5.3 の環境変数・手順・冪等性）。P1 / P2 / P3 の JSON は §5.2 の値を bash 変数展開の heredoc で `mktemp` に書く
- [ ] `bash -n scripts/setup-monitoring.sh` と `shellcheck`（あれば）が通る
- [ ] `gcloud alpha monitoring policies --help` / `gcloud beta monitoring channels --help` で使用サブコマンドの存在を確認し、alpha が無ければ beta / GA に置き換えてスクリプトのコメントに残す
- [ ] `docs/deploy_cloud_run.md` §3 に `(9) 監視（Cloud Monitoring）` を追加: production の実行例、staging の実行例（`SERVICE_SUFFIX=-stg JOB_NAME=outbox-flush-stg CREATE_ABSENT_POLICY=false`）、`docs/monitoring.md` への参照
- 完了条件: `bash -n` が通る。`PROJECT_ID` と `NOTIFICATION_EMAIL` 未設定で実行すると `is required` で即終了する。**GCP へはまだ実行しない**

## Step 7: staging でアラートを発火確認（§7.2）

- [ ] Cloud Shell で `PROJECT_ID=... NOTIFICATION_EMAIL=<開発者> SERVICE_SUFFIX=-stg JOB_NAME=outbox-flush-stg CREATE_ABSENT_POLICY=false bash scripts/setup-monitoring.sh` を実行し、チャネル 1 つ・ポリシー 2 つ（P1 / P2）ができる
- [ ] §7.2 の `entries:write` で P1 を発火させ、メールが届く。届いたインシデントを閉じる
- [ ] §7.2 の手順で P2 を発火させ、メールが届く。`setup-scheduler.sh` を再実行して Scheduler を戻す（staging の Scheduler を作っていない場合は P2 の確認をスキップし、その旨を報告に書く）
- [ ] 確認後、staging のポリシー 2 つを `gcloud alpha monitoring policies delete` で消す（チャネルは残す。production で再利用）
- 完了条件: P1 のメール受信時刻と、削除後に `gcloud alpha monitoring policies list --filter='displayName:satehits'` が空であることを報告に書く

## Step 8: production のアラートと UptimeRobot（§5 / §6）

- [ ] 通知先メールアドレスを確定する（要確認事項）。未確定なら開発者アドレスで進め、確定後にチャネルの `email_address` を更新する手順を報告に残す
- [ ] Cloud Shell で `PROJECT_ID=... NOTIFICATION_EMAIL=... bash scripts/setup-monitoring.sh` を実行し、P1 / P2 / P3 ができる（production は main へのマージ後・初回デプロイ後に行う。`docs/deploy_cloud_run.md` §6）
- [ ] UptimeRobot で §6 の 2 モニターと Alert Contact を作る。URL を一時的に `/healthz-nope` にして Down メールを確認し、戻す
- [ ] P3 を営業時間外に `gcloud scheduler jobs pause outbox-flush` → 10 分待ってメール → `resume` で確認する
- 完了条件: production のポリシー 3 つが `enabled`、UptimeRobot の 2 モニターが Up。実施した確認（メール受信）を報告に書く

## Step 9: docs の仕上げ

- [ ] `docs/architecture_decision_records.md` ADR-012: ステータスを「採用（実装済み）」にし、Consequences の「(現状)」を実測に置き換える（メール受信までの所要時間など）。更新履歴に 1 行追加
- [ ] `docs/infrastructure.md` §8 を実装後の状態に書き換える（表の「実装中」を外し、P1〜P3 と UptimeRobot の一覧、`docs/monitoring.md` への参照）。§9 コスト表に Monitoring / UptimeRobot の行を追加。§11 の「監視・アラートは未決定（ADR-014）」を「`docs/monitoring.md` §7」への参照に置き換える
- [ ] `docs/architecture_decision_records.md` ADR-014 の Consequences「監視・アラートは未決定」を `docs/monitoring.md` 参照に更新
- [ ] `docs/architecture.md` §9 に「500 を返す分岐は `slog.Error` を出す」規約（`monitoring.md` §8）への参照を 1 行追加
- [ ] `docs/test_design.md` §3 に Recover / healthz / logging のテストケースを追加
- [ ] `CLAUDE.md`: §3 のツリーに `internal/logging/`、`handler/recover.go`、`handler/health.go`（と各 `_test.go`）、`scripts/setup-monitoring.sh` を追加。§2 の再発防止に「500 を返す分岐は `slog.Error`。`log.Printf` は INFO 扱いでアラートに乗らない」を 1 行追加。§2 の「実装タスクは `docs/IMPLEMENTATION_PLAN.md` を上から順に」の一文は、本計画が完了したら外す
- [ ] `docs-update` スキルで CLAUDE.md・docs と実装のずれを最終確認する
- 完了条件: `grep -rn "未決定" docs/infrastructure.md docs/architecture_decision_records.md | grep -i "監視"` が 0 件。検証コマンドが通る
