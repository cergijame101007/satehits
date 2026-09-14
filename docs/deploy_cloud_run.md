# Cloud Run デプロイ手順

バックエンド（Go API）を Google Cloud Run に継続デプロイ（CD）するための手順書。オーナー / 開発者が **初回デプロイまで** 辿れることを目的とする。フロントエンド（Cloudflare Pages）は対象外（`docs/infrastructure.md` §10）。

- workflow: `.github/workflows/deploy-backend.yml`
- GCP 初期セットアップ: `scripts/setup-gcp.sh`
- Cloud Scheduler（Outbox flush）: `scripts/setup-scheduler.sh`
- 設計判断: ADR-011（CI/CD）、ADR-014（Outbox）、ADR-015（flush エンドポイントの配置）

## 1. 全体像

```mermaid
flowchart LR
    Push[push to main<br>backend/**] --> Auth[WIF 認証<br>鍵ファイルなし]
    Auth --> Build[docker build 1 回<br>api / migrate / seed を同梱]
    Build --> AR[(Artifact Registry<br>api:sha, api:latest)]
    AR --> Migrate[Cloud Run Job<br>satehits-migrate]
    Migrate -->|成功時のみ| Public[Cloud Run<br>satehits-api 公開]
    Migrate -->|成功時のみ| Private[Cloud Run<br>satehits-api-internal private]
    Public --> Smoke[curl GET /]
    Scheduler[Cloud Scheduler] -->|OIDC 1 分ごと| Private
```

| 段階 | 内容 |
|------|------|
| build once | `backend/docker/Dockerfile` でイメージを 1 回ビルドし、`api:<git sha>` と `api:latest` の 2 タグで Artifact Registry に push。同じイメージにサービス本体（`/app/main`）、マイグレーション（`/app/migrate` + `/app/migrations`）、シード（`/app/seed`）を同梱 |
| migrate job | Cloud Run **Job** `satehits-migrate` を同じイメージで `--command=/app/migrate` として deploy → `execute --wait`。失敗したらサービスのデプロイに進まない |
| 2 サービス | 同一イメージを **公開** `satehits-api` と **private** `satehits-api-internal` の 2 つとしてデプロイ（ADR-015）。flush エンドポイントの有効・無効と `RESEND_API_KEY` の有無はデプロイ設定で固定（ADR-014）。手動で env を変えない（env / secrets は毎回全置換） |
| スモーク | 公開サービスの URL に `GET /` して 200 を確認 |

**認証に Workload Identity Federation（WIF）を選んだ理由**: GitHub Actions の OIDC トークンを GCP が直接信頼するため、サービスアカウントの **鍵ファイル（JSON）を作らない・GitHub に置かない**。漏えい時のローテーションや有効期限の管理が不要になる。信頼はリポジトリ単位（`assertion.repository == "owner/name"`）に限定する。

## 2. 前提

| 項目 | 内容 |
|------|------|
| GCP プロジェクト | 課金有効。`gcloud` は Cloud Shell で実行する想定（ローカルでも可） |
| Supabase | 接続文字列は **Session pooler（IPv4）** のものを使う。Cloud Run からの直結（Direct connection）は IPv6 のため到達できない。Transaction pooler は prepared statement が使えず pgx の既定と相性が悪い |
| Cloudflare R2 | 取引先画像用バケット、S3 API トークン（Access Key / Secret Key）、公開 URL（カスタムドメインまたは `r2.dev`） |
| Cloudflare Turnstile | サイトキー（フロント）とシークレットキー（バックエンド）。本番では `TURNSTILE_SECRET_KEY` 必須（未設定なら起動失敗） |
| Resend | API キー。送信元ドメイン `satehits.com` の SPF / DKIM / DMARC は `docs/infrastructure.md` §11 |
| GitHub | リポジトリの Settings を変更できる権限（Environments / Secrets / Variables） |

## 3. 手順

### (1) GCP 初期セットアップ

Cloud Shell で 1 回実行する（冪等なので再実行しても壊れない）。

```bash
PROJECT_ID=<your-project-id> bash scripts/setup-gcp.sh
# 任意: REGION=asia-northeast1 GITHUB_REPO=cergijame101007/satehits
```

やること: API 有効化、Artifact Registry `satehits`、ランタイム SA `satehits-run-sa`（`roles/secretmanager.secretAccessor`）、デプロイ SA `github-deployer`（`roles/run.admin`、`roles/artifactregistry.writer`、ランタイム SA への `roles/iam.serviceAccountUser`）、Workload Identity Pool / Provider `github`、Secret Manager の空枠。最後に GitHub へ登録する値を表示する。

### (2) Secret Manager に値を投入

枠は (1) で作られている。値だけ追加する（履歴に残さないため `echo` ではなく `printf` / ファイル経由を推奨）。

```bash
PROJECT_ID=<your-project-id>

# Supabase の Session pooler（IPv4）接続文字列
printf '%s' 'postgresql://postgres.xxxx:PASSWORD@aws-0-ap-northeast-1.pooler.supabase.com:5432/postgres?sslmode=require' \
  | gcloud secrets versions add DATABASE_URL --project="$PROJECT_ID" --data-file=-

# JWT 署名鍵（32 バイト以上）
openssl rand -base64 48 | tr -d '\n' \
  | gcloud secrets versions add JWT_SECRET --project="$PROJECT_ID" --data-file=-

printf '%s' '<turnstile secret key>' | gcloud secrets versions add TURNSTILE_SECRET_KEY --project="$PROJECT_ID" --data-file=-
printf '%s' '<resend api key>'      | gcloud secrets versions add RESEND_API_KEY       --project="$PROJECT_ID" --data-file=-
printf '%s' '<r2 access key id>'    | gcloud secrets versions add STORAGE_ACCESS_KEY   --project="$PROJECT_ID" --data-file=-
printf '%s' '<r2 secret access key>'| gcloud secrets versions add STORAGE_SECRET_KEY   --project="$PROJECT_ID" --data-file=-
```

`JWT_SECRET` を差し替えると発行済みのアクセストークンは全て無効になる（管理者は再ログイン）。

### (3) GitHub の Secrets / Variables

Settings → Environments → `production`（次項で作成）に登録する。値は (1) の最後に表示される。

| 名前 | 種別 | 例 | 必須 |
|------|------|----|------|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Secret | `projects/123456789012/locations/global/workloadIdentityPools/github/providers/github` | 必須 |
| `GCP_DEPLOY_SERVICE_ACCOUNT` | Secret | `github-deployer@PROJECT.iam.gserviceaccount.com` | 必須 |
| `GCP_PROJECT_ID` | Variable | `satehits-prod` | 必須 |
| `GCP_RUNTIME_SERVICE_ACCOUNT` | Variable | `satehits-run-sa@PROJECT.iam.gserviceaccount.com` | 必須 |
| `GCP_REGION` | Variable | `asia-northeast1` | 任意（既定 `asia-northeast1`） |
| `GCP_ARTIFACT_REPO` | Variable | `satehits` | 任意（既定 `satehits`） |
| `CORS_ORIGINS` | Variable | `https://satehits.com,https://www.satehits.com` | 任意（既定は左の値） |
| `COOKIE_DOMAIN` | Variable | `api.satehits.com` | 任意（カスタムドメインを付けたら設定。空なら Cookie の Domain 属性なし） |
| `STORAGE_ENDPOINT` | Variable | `https://<account id>.r2.cloudflarestorage.com` | 画像アップロードを使うなら必須 |
| `STORAGE_BUCKET` | Variable | `satehits-images` | 同上 |
| `STORAGE_PUBLIC_BASE_URL` | Variable | `https://images.satehits.com` | 同上 |

`STORAGE_*` と Secret Manager の `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` が揃わない場合、画像アップロード API は NoOp になる（`docs/infrastructure.md` §6）。

### (4) `production` environment の作成

Settings → Environments → New environment → `production`。workflow の `deploy` job はこの environment を参照するため、上の Secrets / Variables はここに置く。Required reviewers（承認者）を付けると main への push 後に手動承認を挟める（任意）。

### (5) 初回デプロイ

`main` への push（`backend/**` または workflow 自身の変更）で自動実行される。手動で走らせる場合は Actions → **Backend Deploy** → Run workflow（`workflow_dispatch`）。

確認ポイント:

- `Run migrations` step が成功していること（Cloud Run Job `satehits-migrate` の実行ログは Cloud Console → Cloud Run → Jobs）
- `Smoke test` step が `{"message":"Welcome to the Go API","status":"success"}` を出力していること
- 公開サービスの URL は `Deploy public service` step の `url` 出力（ログに出る）

### (6) Cloud Scheduler（Outbox flush）

private サービスがデプロイされてから 1 回実行する。

```bash
PROJECT_ID=<your-project-id> bash scripts/setup-scheduler.sh
```

Scheduler 用 SA を作り、`satehits-api-internal` への `roles/run.invoker` を付与し、1 分ごとに `POST /internal/outbox/flush` を OIDC で呼ぶジョブを作る。詳細は `docs/infrastructure.md` §11。

### (7) 管理者ユーザーの作成

`admin_users` は空なので、`cmd/seed`（イメージに `/app/seed` として同梱）を Cloud Run Job で 1 回実行する。

```bash
PROJECT_ID=<your-project-id>
REGION=asia-northeast1
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/satehits/api:latest"

gcloud run jobs deploy satehits-seed \
  --project="$PROJECT_ID" --region="$REGION" \
  --image="$IMAGE" \
  --command=/app/seed \
  --set-secrets=DATABASE_URL=DATABASE_URL:latest \
  --set-env-vars="SEED_ADMIN_PASSWORD=<初期パスワード>" \
  --service-account="satehits-run-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --max-retries=0 --task-timeout=5m --quiet

gcloud run jobs execute satehits-seed --project="$PROJECT_ID" --region="$REGION" --wait

# パスワードを env に残さないため、実行後は Job を削除する
gcloud run jobs delete satehits-seed --project="$PROJECT_ID" --region="$REGION" --quiet
```

注意:

- seed が投入するメールアドレスは **`owner@example.com` / `dev@example.com` 固定**（`backend/cmd/seed/main.go`）。本番の実アドレスにしたい場合は、seed 実行後に SQL で `email` を更新するか、seed をアドレス指定できるよう直す（未対応）
- 同じパスワードが両ユーザーに設定される。初回ログイン後に変更する運用（パスワード変更 API は未実装のため、当面は SQL で `password_hash` を更新する）
- `ON CONFLICT (email) DO NOTHING` なので再実行しても上書きされない

### (8) カスタムドメインと Cloudflare Pages

1. Cloud Run → `satehits-api` → カスタムドメインに `api.satehits.com` を割り当て、表示された CNAME を Cloudflare DNS に追加する（Cloudflare のプロキシは **DNS only** にする。Cloudflare の証明書と Cloud Run の証明書が二重になり、Cookie / CORS の切り分けが難しくなるため）
2. Cloudflare Pages の環境変数に `PUBLIC_API_URL=https://api.satehits.com`、`PUBLIC_TURNSTILE_SITE_KEY` を設定して再ビルド
3. GitHub Variables の `COOKIE_DOMAIN` を `api.satehits.com` にして再デプロイ（`workflow_dispatch`）

カスタムドメインを付けない間は `*.run.app` の URL を `PUBLIC_API_URL` に使う。`COOKIE_DOMAIN` は空のままでよい。

## 4. Cloud Run の設定一覧

| 名前 | 種別 | 認証 | env（非秘密） | secrets（Secret Manager） | スケール | timeout |
|------|------|------|---------------|---------------------------|----------|---------|
| `satehits-api` | Service | `--allow-unauthenticated` | `ENVIRONMENT=production`, `CORS_ORIGINS`, `COOKIE_DOMAIN`, `MAIL_FROM_ADDRESS`, `OUTBOX_BATCH_SIZE=20`, `OUTBOX_FLUSH_TIME_BUDGET_SECONDS=120`, **`OUTBOX_FLUSH_ENDPOINT_ENABLED=false`**, `TRUSTED_PROXY_HOPS=1`, `STORAGE_ENDPOINT`, `STORAGE_REGION=auto`, `STORAGE_BUCKET`, `STORAGE_PUBLIC_BASE_URL` | `DATABASE_URL`, `JWT_SECRET`, `TURNSTILE_SECRET_KEY`, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY` | cpu 1 / 256Mi / min 0 / max 2 / concurrency 80 / CPU はリクエスト中のみ | 60s |
| `satehits-api-internal` | Service | `--no-allow-unauthenticated`（Scheduler SA に `roles/run.invoker`） | 同上、ただし **`OUTBOX_FLUSH_ENDPOINT_ENABLED=true`** | 同上 + **`RESEND_API_KEY`** | cpu 1 / 256Mi / min 0 / max 1 / concurrency 80 | 300s（`OUTBOX_FLUSH_TIME_BUDGET_SECONDS` 120 < Scheduler attempt-deadline 180 < 300） |
| `satehits-migrate` | Job | ランタイム SA | `MIGRATIONS_DIR=/app/migrations` | `DATABASE_URL` | tasks 1 / max-retries 0 | 10m |
| `satehits-seed` | Job（手動・実行後削除） | ランタイム SA | `SEED_ADMIN_PASSWORD` | `DATABASE_URL` | tasks 1 / max-retries 0 | 5m |

共通: ポート `8080`、ランタイム SA `satehits-run-sa`、イメージ `asia-northeast1-docker.pkg.dev/PROJECT/satehits/api:<git sha>`。env / secrets は **overwrite（全置換）** で、workflow にない値はデプロイのたびに消える。設定変更はコンソールではなく workflow か GitHub Variables で行う。

ログイン失敗のレートリミット（`LOGIN_RATE_LIMIT_*`）は既定値（5 / 20 / 15 分）を使う。変えたくなったら workflow の `env_vars` に追加する。

## 5. 運用

### ロールバック

Cloud Run はリビジョンを保持するので、直前のリビジョンにトラフィックを戻す。

```bash
gcloud run revisions list --service=satehits-api --project="$PROJECT_ID" --region="$REGION"
gcloud run services update-traffic satehits-api \
  --project="$PROJECT_ID" --region="$REGION" \
  --to-revisions=<前のリビジョン名>=100
```

private 側も同じように戻す。マイグレーションは前進のみ（down なし）なので、**スキーマ変更を含むデプロイのロールバックはコードが旧スキーマで動くかを先に確認する**。加算的な変更（列追加・テーブル追加）にとどめる方針。

次のデプロイ（main への push）で自動的に新しいリビジョンへ 100% 戻る。

### ログ

- Cloud Console → Cloud Run → サービス → ログ。または:

```bash
gcloud run services logs read satehits-api --project="$PROJECT_ID" --region="$REGION" --limit=100
gcloud run jobs executions list --job=satehits-migrate --project="$PROJECT_ID" --region="$REGION"
```

- 起動失敗（`log.Fatal`）は `JWT_SECRET must be at least 32 bytes` / `TURNSTILE_SECRET_KEY is required` / `RESEND_API_KEY is required when ENVIRONMENT=production and OUTBOX_FLUSH_ENDPOINT_ENABLED=true` のように原因がログに出る（`backend/pkg/config/config.go`）
- Outbox の `failed` 行は SQL で確認する（`docs/infrastructure.md` §11）

### 失敗したマイグレーション

各 SQL ファイルは 1 トランザクションで適用され、成功したものだけ `schema_migrations` に記録される。途中で失敗した場合:

1. Job の実行ログで失敗したファイルと SQL エラーを確認する
2. SQL を修正して main に push する（失敗したファイルは記録されていないので、次回の Job が再適用する）。**適用済みファイルは編集しない**（記録済みで再実行されない）。直す場合は新しい番号のファイルを追加する
3. サービスは前のリビジョンのまま（migrate 失敗時はサービスのデプロイに進まない）

手動で再実行するだけなら `gcloud run jobs execute satehits-migrate --wait`。

### コスト

- **CPU always allocated は使わない**（`--cpu-throttling`、リクエスト処理中のみ課金）
- `--min-instances=0`（アイドル時は 0 台。コールドスタートは数百 ms 〜 1 秒程度）
- Scheduler の 1 分ごとの flush は private サービスを毎分起こす。無料枠内の見込みだが billing account 単位で共有されるため保証はない（`docs/infrastructure.md` §11）
- Artifact Registry はイメージが sha ごとに溜まる。容量課金（数十 MB / イメージ）なので、気になったら古いタグを削除するか cleanup policy を設定する

## 6. 既知の制約

- `main` ブランチは現状 initial commit のままなので、**develop → main のマージが初回デプロイ**になる。マージ前に §3 (1)〜(4) を済ませておく
- サーバーのポートは `:8080` 固定（`backend/cmd/api/main.go`）。Cloud Run の既定 `PORT=8080` と一致しているので問題ないが、`--port` を変えても効かない
- migrate はロックを取らない（`backend/cmd/migrate/main.go`）。workflow の `concurrency` で直列化しているので通常は問題ないが、手動 `execute` を同時に走らせない
- `docker build` はキャッシュなしで毎回フルビルド（数分）。速度が問題になったら `docker/build-push-action` + GHA キャッシュに切り替える
- seed のメールアドレスが `example.com` 固定（§3 (7)）
