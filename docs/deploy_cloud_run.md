# Cloud Run デプロイ手順

バックエンド（Go API）を Google Cloud Run に継続デプロイ（CD）するための手順書。オーナー / 開発者が **初回デプロイまで** 辿れることを目的とする。フロントエンド（Cloudflare Pages）は対象外（`docs/infrastructure.md` §10）。

- workflow: `.github/workflows/deploy-backend.yml`
- GCP 初期セットアップ: `scripts/setup-gcp.sh`
- Cloud Scheduler（Outbox flush）: `scripts/setup-scheduler.sh`
- 設計判断: ADR-011（CI/CD）、ADR-014（Outbox）、ADR-015（flush エンドポイントの配置）

## 1. 全体像

### ブランチ運用と環境

```mermaid
flowchart LR
    Feature[feature/*] -->|PR| Develop[develop]
    Develop -->|push = マージ| Staging[staging へデプロイ]
    Develop -->|PR| Main[main]
    Main -->|push = マージ| Production[production へデプロイ]
    Hotfix[hotfix/*<br>main から切る] -->|PR| Main
    Main -.->|hotfix を戻す PR| Develop
```

| ブランチ | 役割 | デプロイ |
|----------|------|----------|
| `feature/*` | 作業ブランチ。`develop` へ PR | なし（`backend.yml` の CI のみ） |
| `develop` | 統合ブランチ。PR マージ = `develop` への push | **staging** へ自動デプロイ |
| `main` | リリースブランチ。`develop` からの PR マージ | **production** へ自動デプロイ |
| `hotfix/*` | 緊急修正。`main` から切り `main` へ PR。マージ後 `main` → `develop` の PR で戻す | `main` マージで production |

- **PR を開いただけではデプロイしない**（`pull_request` トリガーなし）。fork からの PR に Secrets を晒さないため
- `main` はブランチ保護（PR 必須・`Backend CI` / `Frontend CI` の成功必須・直接 push 禁止）を推奨
- GitHub Environment `production` に **Required reviewers** を付けると、`main` へのマージ後に承認するまでデプロイが待機する（手動承認ゲート）
- 手動デプロイは Actions → **Backend Deploy** → Run workflow → `environment` を choice で選ぶ（既定 staging）

### 環境対応表

GCP プロジェクトと Artifact Registry は 1 つを共用する。環境の違いは Cloud Run のリソース名と Secret Manager の名前のサフィックスだけ。

| 項目 | staging | production |
|------|---------|------------|
| GitHub Environment | `staging` | `production` |
| トリガー | `develop` への push | `main` への push |
| アプリの `ENVIRONMENT` | `staging` | `production` |
| Cloud Run 公開サービス | `satehits-api-stg` | `satehits-api` |
| Cloud Run private サービス | `satehits-api-internal-stg` | `satehits-api-internal` |
| Cloud Run migrate Job | `satehits-migrate-stg` | `satehits-migrate` |
| Secret Manager | `DATABASE_URL_STG`, `JWT_SECRET_STG`, `TURNSTILE_SECRET_KEY_STG`, `RESEND_API_KEY_STG`, `STORAGE_ACCESS_KEY_STG`, `STORAGE_SECRET_KEY_STG` | `DATABASE_URL`, `JWT_SECRET`, `TURNSTILE_SECRET_KEY`, `RESEND_API_KEY`, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY` |
| イメージタグ | `api:<git sha>`, `api:latest-stg` | `api:<git sha>`, `api:latest` |
| Supabase | **現行の開発用プロジェクトを当面再利用**（`DATABASE_URL_STG` に開発用の Session pooler 接続文字列） | 後で **新規 Supabase プロジェクト（Tokyo）** を作り、その接続文字列を `DATABASE_URL` に入れる |
| 公開サービスの max-instances | 1 | 1（ピーク同時 10 未満想定・ADR-000。足りなくなったら workflow で上げる） |
| Cloud Scheduler（flush） | 任意（`outbox-flush-stg`） | 必須（`outbox-flush`） |

GitHub 側は `staging` / `production` の両 Environment に **同じ名前** の Secrets / Variables を置く。workflow は Environment を切り替えるだけで同じキー名を参照する（§3 (3)）。

### デプロイの流れ（1 環境分）

```mermaid
flowchart LR
    Push[push to develop / main<br>backend/**] --> Auth[WIF 認証<br>鍵ファイルなし]
    Auth --> Build[docker build 1 回<br>api / migrate / seed を同梱]
    Build --> AR[(Artifact Registry<br>api:sha, api:latest / latest-stg)]
    AR --> Migrate[Cloud Run Job<br>satehits-migrate / -stg]
    Migrate -->|成功時のみ| Public[Cloud Run<br>satehits-api / -stg 公開]
    Migrate -->|成功時のみ| Private[Cloud Run<br>satehits-api-internal / -stg private]
    Public --> Smoke[curl GET /]
    Scheduler[Cloud Scheduler] -->|OIDC 1 分ごと| Private
```

| 段階 | 内容 |
|------|------|
| build once | `backend/docker/Dockerfile` でイメージを 1 回ビルドし、`api:<git sha>` と `api:latest`（staging は `api:latest-stg`）の 2 タグで Artifact Registry に push。同じイメージにサービス本体（`/app/main`）、マイグレーション（`/app/migrate` + `/app/migrations`）、シード（`/app/seed`）を同梱 |
| migrate job | Cloud Run **Job** `satehits-migrate`（staging は `-stg`）を同じイメージで `--command=/app/migrate` として deploy → `execute --wait`。失敗したらサービスのデプロイに進まない |
| 2 サービス | 同一イメージを **公開** `satehits-api` と **private** `satehits-api-internal`（staging は `-stg`）の 2 つとしてデプロイ（ADR-015）。flush エンドポイントの有効・無効と `RESEND_API_KEY` の有無はデプロイ設定で固定（ADR-014）。手動で env を変えない（env / secrets は毎回全置換） |
| スモーク | 公開サービスの URL に `GET /` して 200 を確認 |

**認証に Workload Identity Federation（WIF）を選んだ理由**: GitHub Actions の OIDC トークンを GCP が直接信頼するため、サービスアカウントの **鍵ファイル（JSON）を作らない・GitHub に置かない**。漏えい時のローテーションや有効期限の管理が不要になる。信頼はリポジトリ単位（`assertion.repository == "owner/name"`）に限定する。

## 2. 前提

| 項目 | 内容 |
|------|------|
| GCP プロジェクト | 課金有効。`gcloud` は Cloud Shell で実行する想定（ローカルでも可）。staging / production で 1 つを共用 |
| Supabase | 接続文字列は **Session pooler（IPv4）** のものを使う。Cloud Run からの直結（Direct connection）は IPv6 のため到達できない。Transaction pooler は prepared statement が使えず pgx の既定と相性が悪い。staging は当面開発用プロジェクトを再利用、production は新規プロジェクト（Tokyo） |
| Cloudflare R2 | 取引先画像用バケット、S3 API トークン（Access Key / Secret Key）、公開 URL（カスタムドメインまたは `r2.dev`）。staging 用に別バケット（例 `satehits-images-stg`）を用意すると本番画像と混ざらない |
| Cloudflare Turnstile | サイトキー（フロント）とシークレットキー（バックエンド）。`ENVIRONMENT` が `development` 以外（staging も）では `TURNSTILE_SECRET_KEY` 必須（未設定なら起動失敗） |
| Resend | API キー。送信元ドメイン `satehits.com` の SPF / DKIM / DMARC は `docs/infrastructure.md` §11 |
| GitHub | リポジトリの Settings を変更できる権限（Environments / Secrets / Variables / Branch protection） |

## 3. 手順

### (1) GCP 初期セットアップ

Cloud Shell で 1 回実行する（冪等なので再実行しても壊れない）。

```bash
PROJECT_ID=<your-project-id> bash scripts/setup-gcp.sh
# 任意: REGION=asia-northeast1 GITHUB_REPO=cergijame101007/satehits
```

やること: API 有効化、Artifact Registry `satehits`、ランタイム SA `satehits-run-sa`（`roles/secretmanager.secretAccessor`）、デプロイ SA `github-deployer`（`roles/run.admin`、`roles/artifactregistry.writer`、ランタイム SA への `roles/iam.serviceAccountUser`）、Workload Identity Pool / Provider `github`、Secret Manager の空枠（production 6 つ + staging `_STG` 6 つ = 12）。最後に GitHub へ登録する値を表示する。

### (2) Secret Manager に値を投入

枠は (1) で作られている。値だけ追加する（履歴に残さないため `echo` ではなく `printf` / ファイル経由を推奨）。**production と staging の両方**に入れる。

```bash
PROJECT_ID=<your-project-id>

# ---- production（サフィックスなし）----
# Supabase（production 用プロジェクト）の Session pooler（IPv4）接続文字列
printf '%s' 'postgresql://postgres.xxxx:PASSWORD@aws-0-ap-northeast-1.pooler.supabase.com:5432/postgres?sslmode=require' \
  | gcloud secrets versions add DATABASE_URL --project="$PROJECT_ID" --data-file=-

# JWT 署名鍵（32 バイト以上。環境ごとに別の鍵を生成する）
openssl rand -base64 48 | tr -d '\n' \
  | gcloud secrets versions add JWT_SECRET --project="$PROJECT_ID" --data-file=-

printf '%s' '<turnstile secret key>' | gcloud secrets versions add TURNSTILE_SECRET_KEY --project="$PROJECT_ID" --data-file=-
printf '%s' '<resend api key>'      | gcloud secrets versions add RESEND_API_KEY       --project="$PROJECT_ID" --data-file=-
printf '%s' '<r2 access key id>'    | gcloud secrets versions add STORAGE_ACCESS_KEY   --project="$PROJECT_ID" --data-file=-
printf '%s' '<r2 secret access key>'| gcloud secrets versions add STORAGE_SECRET_KEY   --project="$PROJECT_ID" --data-file=-

# ---- staging（_STG サフィックス）----
# Supabase は当面、現行の開発用プロジェクトの Session pooler 接続文字列を使う
printf '%s' 'postgresql://postgres.yyyy:PASSWORD@aws-0-ap-northeast-1.pooler.supabase.com:5432/postgres?sslmode=require' \
  | gcloud secrets versions add DATABASE_URL_STG --project="$PROJECT_ID" --data-file=-
openssl rand -base64 48 | tr -d '\n' \
  | gcloud secrets versions add JWT_SECRET_STG --project="$PROJECT_ID" --data-file=-
printf '%s' '<turnstile secret key (staging)>' | gcloud secrets versions add TURNSTILE_SECRET_KEY_STG --project="$PROJECT_ID" --data-file=-
printf '%s' '<resend api key>'                 | gcloud secrets versions add RESEND_API_KEY_STG       --project="$PROJECT_ID" --data-file=-
printf '%s' '<r2 access key id (staging)>'     | gcloud secrets versions add STORAGE_ACCESS_KEY_STG   --project="$PROJECT_ID" --data-file=-
printf '%s' '<r2 secret access key (staging)>' | gcloud secrets versions add STORAGE_SECRET_KEY_STG   --project="$PROJECT_ID" --data-file=-
```

`JWT_SECRET` / `JWT_SECRET_STG` を差し替えると、その環境で発行済みのアクセストークンは全て無効になる（管理者は再ログイン）。

**各環境の 6 つすべてに値（バージョン）が必要。** workflow は環境ごとに 6 つの Secret を毎回参照するため、バージョンが 1 つも無い Secret があるとデプロイが `Secret version not found` で失敗する。R2 をまだ使わない場合は `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY`（と `_STG`）にダミー値（例 `unused`）を入れておく（`STORAGE_ENDPOINT` 等の Variables が空なら画像アップロードは NoOp のまま）。同様に Resend 未契約なら `RESEND_API_KEY` にもダミー値を入れる（private サービスは起動するが、flush 時の送信は認証エラーで中断し `pending` のまま残る）。

**staging の `RESEND_API_KEY_STG` は実キーを入れることを推奨する。** `backend/pkg/config/config.go` の「flush 有効なのにキー未設定なら起動失敗」チェックは `ENVIRONMENT=production` のときだけ発火する。staging（`ENVIRONMENT=staging`）で `RESEND_API_KEY_STG` が **空文字** だと NoOp Sender が動き、**メールを送らずに `sent` を記録する**（送信されたように見えるが届かない）。Resend の同じキーを staging にも入れるか、staging では予約のメールアドレスをテスト用に限定する運用にする。ダミー値（`unused` 等）なら NoOp ではなく認証エラーで中断するので `sent` にはならない。

### (3) GitHub の Secrets / Variables

Settings → Environments → `staging` と `production`（次項で作成）の **両方** に、同じ名前で登録する。値は (1) の最後に表示される。

| 名前 | 種別 | 例 | 必須 | 環境で値が変わるか |
|------|------|----|------|--------------------|
| `GCP_WORKLOAD_IDENTITY_PROVIDER` | Secret | `projects/123456789012/locations/global/workloadIdentityPools/github/providers/github` | 必須 | 同じ |
| `GCP_DEPLOY_SERVICE_ACCOUNT` | Secret | `github-deployer@PROJECT.iam.gserviceaccount.com` | 必須 | 同じ |
| `GCP_PROJECT_ID` | Variable | `satehits-prod` | 必須 | 同じ |
| `GCP_RUNTIME_SERVICE_ACCOUNT` | Variable | `satehits-run-sa@PROJECT.iam.gserviceaccount.com` | 必須 | 同じ |
| `GCP_REGION` | Variable | `asia-northeast1` | 任意（既定 `asia-northeast1`） | 同じ |
| `GCP_ARTIFACT_REPO` | Variable | `satehits` | 任意（既定 `satehits`） | 同じ |
| `CORS_ORIGINS` | Variable | production: `https://satehits.com,https://www.satehits.com` / staging: `https://develop.satehits.pages.dev` | staging では必須（既定は production の値） | **違う** |
| `COOKIE_DOMAIN` | Variable | production: `api.satehits.com` / staging: 空 | 任意（カスタムドメインを付けたら設定。空なら Cookie の Domain 属性なし） | **違う** |
| `STORAGE_ENDPOINT` | Variable | `https://<account id>.r2.cloudflarestorage.com` | 画像アップロードを使うなら必須 | 同じでもよい |
| `STORAGE_BUCKET` | Variable | production: `satehits-images` / staging: `satehits-images-stg` | 同上 | **違う** |
| `STORAGE_PUBLIC_BASE_URL` | Variable | production: `https://images.satehits.com` / staging: `https://pub-xxxx.r2.dev` | 同上 | **違う** |

- staging の `CORS_ORIGINS` は Cloudflare Pages の `develop` ブランチ preview URL（例 `https://develop.satehits.pages.dev`）を想定。Pages の preview はコミットごとの URL（`https://<hash>.satehits.pages.dev`）も発行されるが、ブランチ別名の URL を使えば固定できる。ローカルから staging API を叩きたいときは `,http://localhost:4321` を足す
- `STORAGE_*` と Secret Manager の `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` が揃わない場合、画像アップロード API は NoOp になる（`docs/infrastructure.md` §6）

### (4) `staging` / `production` environment の作成

Settings → Environments → New environment で `staging` と `production` を作る。workflow の `deploy` job はブランチ（または `workflow_dispatch` の choice）に応じてどちらかを参照するため、上の Secrets / Variables はそれぞれの Environment に置く。

- `production` には **Required reviewers**（承認者）を付けると、`main` へのマージ後に手動承認を挟める（推奨）
- Deployment branches を `production` は `main` のみ、`staging` は `develop` のみに制限すると、`workflow_dispatch` で別ブランチから誤って本番へデプロイすることを防げる（任意）

### (5) 初回デプロイ

まず **staging** から。`develop` への push（`backend/**` または workflow 自身の変更）で自動実行される。手動なら Actions → **Backend Deploy** → Run workflow → `environment: staging`。

確認ポイント:

- `Resolve target environment` step のログに `target: staging (services: satehits-api-stg / ...)` と出ていること
- `Run migrations` step が成功していること（Cloud Run Job `satehits-migrate-stg` の実行ログは Cloud Console → Cloud Run → Jobs）
- `Smoke test` step が `{"message":"Welcome to the Go API","status":"success"}` を出力していること
- 公開サービスの URL は `Deploy public service` step の `url` 出力（ログに出る）

staging で問題なければ `develop` → `main` の PR をマージして **production** へ。`production` に Required reviewers を付けていれば Actions 画面で承認する。

### (6) Cloud Scheduler（Outbox flush）

private サービスがデプロイされてから実行する。production は必須、staging は任意（staging でメール送信まで確認したい場合のみ）。

```bash
# production
PROJECT_ID=<your-project-id> bash scripts/setup-scheduler.sh

# staging（任意）: private サービス名と Scheduler ジョブ名を staging 用にする。SA（scheduler-sa）は共用でよい
PROJECT_ID=<your-project-id> PRIVATE_SERVICE=satehits-api-internal-stg JOB_NAME=outbox-flush-stg \
  bash scripts/setup-scheduler.sh
```

Scheduler 用 SA を作り、private サービスへの `roles/run.invoker` を付与し、1 分ごとに `POST /internal/outbox/flush` を OIDC で呼ぶジョブを作る。詳細は `docs/infrastructure.md` §11。staging の Scheduler を作らない場合、staging の `email_outbox` は `pending` のまま溜まる（動作確認には `gcloud run services proxy` 等で private サービスを手動で叩く）。

### (7) 管理者ユーザーの作成

`admin_users` は空なので、`cmd/seed`（イメージに `/app/seed` として同梱）を Cloud Run Job で 1 回実行する。**staging で先に試してから production で実行する**（staging は開発用 Supabase を再利用しているので、既にユーザーがあれば `ON CONFLICT DO NOTHING` で何も起きない）。

```bash
PROJECT_ID=<your-project-id>
REGION=asia-northeast1

# staging: SUFFIX=-stg SECRET_SUFFIX=_STG IMAGE_TAG=latest-stg
# production: SUFFIX=  SECRET_SUFFIX=    IMAGE_TAG=latest
SUFFIX=-stg
SECRET_SUFFIX=_STG
IMAGE_TAG=latest-stg
IMAGE="${REGION}-docker.pkg.dev/${PROJECT_ID}/satehits/api:${IMAGE_TAG}"

gcloud run jobs deploy "satehits-seed${SUFFIX}" \
  --project="$PROJECT_ID" --region="$REGION" \
  --image="$IMAGE" \
  --command=/app/seed \
  --set-secrets="DATABASE_URL=DATABASE_URL${SECRET_SUFFIX}:latest" \
  --set-env-vars="SEED_ADMIN_PASSWORD=<初期パスワード>" \
  --service-account="satehits-run-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
  --max-retries=0 --task-timeout=5m --quiet

gcloud run jobs execute "satehits-seed${SUFFIX}" --project="$PROJECT_ID" --region="$REGION" --wait

# パスワードを env に残さないため、実行後は Job を削除する
gcloud run jobs delete "satehits-seed${SUFFIX}" --project="$PROJECT_ID" --region="$REGION" --quiet
```

注意:

- seed が投入するメールアドレスは **`owner@example.com` / `dev@example.com` 固定**（`backend/cmd/seed/main.go`）。本番の実アドレスにしたい場合は、seed 実行後に SQL で `email` を更新するか、seed をアドレス指定できるよう直す（未対応）
- 同じパスワードが両ユーザーに設定される。初回ログイン後に変更する運用（パスワード変更 API は未実装のため、当面は SQL で `password_hash` を更新する）
- `ON CONFLICT (email) DO NOTHING` なので再実行しても上書きされない

### (8) カスタムドメインと Cloudflare Pages

production:

1. Cloud Run → `satehits-api` → カスタムドメインに `api.satehits.com` を割り当て、表示された CNAME を Cloudflare DNS に追加する（Cloudflare のプロキシは **DNS only** にする。Cloudflare の証明書と Cloud Run の証明書が二重になり、Cookie / CORS の切り分けが難しくなるため）
2. Cloudflare Pages の Production 環境変数に `PUBLIC_API_URL=https://api.satehits.com`、`PUBLIC_TURNSTILE_SITE_KEY` を設定して再ビルド
3. GitHub `production` Environment の `COOKIE_DOMAIN` を `api.satehits.com` にして再デプロイ（`workflow_dispatch` → production）

staging:

- カスタムドメインは付けず `satehits-api-stg` の `*.run.app` URL をそのまま使う。Cloudflare Pages の Preview 環境変数（`develop` ブランチ）に `PUBLIC_API_URL=https://satehits-api-stg-xxxx.a.run.app` を設定する
- `COOKIE_DOMAIN` は空のまま（`*.run.app` は Public Suffix なので Domain 属性を付けるとブラウザが拒否する）

カスタムドメインを付けない間は production も `*.run.app` の URL を `PUBLIC_API_URL` に使い、`COOKIE_DOMAIN` は空のままでよい。

## 4. Cloud Run の設定一覧

| 名前 | 環境 | 種別 | 認証 | env（非秘密） | secrets（Secret Manager） | スケール | timeout |
|------|------|------|------|---------------|---------------------------|----------|---------|
| `satehits-api` | production | Service | `--allow-unauthenticated` | `ENVIRONMENT=production`, `CORS_ORIGINS`, `COOKIE_DOMAIN`, `MAIL_FROM_ADDRESS`, `OUTBOX_BATCH_SIZE=20`, `OUTBOX_FLUSH_TIME_BUDGET_SECONDS=120`, **`OUTBOX_FLUSH_ENDPOINT_ENABLED=false`**, `TRUSTED_PROXY_HOPS=1`, `STORAGE_ENDPOINT`, `STORAGE_REGION=auto`, `STORAGE_BUCKET`, `STORAGE_PUBLIC_BASE_URL` | `DATABASE_URL`, `JWT_SECRET`, `TURNSTILE_SECRET_KEY`, `STORAGE_ACCESS_KEY`, `STORAGE_SECRET_KEY` | cpu 1 / 256Mi / min 0 / max 1 / concurrency 80 / CPU はリクエスト中のみ | 60s |
| `satehits-api-internal` | production | Service | `--no-allow-unauthenticated`（Scheduler SA に `roles/run.invoker`） | 同上、ただし **`OUTBOX_FLUSH_ENDPOINT_ENABLED=true`** | 同上 + **`RESEND_API_KEY`** | cpu 1 / 256Mi / min 0 / max 1 / concurrency 80 | 300s（`OUTBOX_FLUSH_TIME_BUDGET_SECONDS` 120 < Scheduler attempt-deadline 180 < 300） |
| `satehits-migrate` | production | Job | ランタイム SA | `MIGRATIONS_DIR=/app/migrations` | `DATABASE_URL` | tasks 1 / max-retries 0 | 10m |
| `satehits-api-stg` | staging | Service | `--allow-unauthenticated` | `satehits-api` と同じ、ただし `ENVIRONMENT=staging`。`CORS_ORIGINS` / `COOKIE_DOMAIN` / `STORAGE_*` は staging Environment の Variables | `DATABASE_URL_STG`, `JWT_SECRET_STG`, `TURNSTILE_SECRET_KEY_STG`, `STORAGE_ACCESS_KEY_STG`, `STORAGE_SECRET_KEY_STG` | cpu 1 / 256Mi / min 0 / **max 1** / concurrency 80 | 60s |
| `satehits-api-internal-stg` | staging | Service | `--no-allow-unauthenticated` | `satehits-api-internal` と同じ、ただし `ENVIRONMENT=staging` | 同上 + **`RESEND_API_KEY_STG`** | cpu 1 / 256Mi / min 0 / max 1 / concurrency 80 | 300s |
| `satehits-migrate-stg` | staging | Job | ランタイム SA | `MIGRATIONS_DIR=/app/migrations` | `DATABASE_URL_STG` | tasks 1 / max-retries 0 | 10m |
| `satehits-seed` / `satehits-seed-stg` | 両方 | Job（手動・実行後削除） | ランタイム SA | `SEED_ADMIN_PASSWORD` | `DATABASE_URL` / `DATABASE_URL_STG` | tasks 1 / max-retries 0 | 5m |

共通: ポート `8080`、ランタイム SA `satehits-run-sa`（両環境共用）、イメージ `asia-northeast1-docker.pkg.dev/PROJECT/satehits/api:<git sha>`。env / secrets は **overwrite（全置換）** で、workflow にない値はデプロイのたびに消える。設定変更はコンソールではなく workflow か GitHub Variables で行う。

ログイン失敗のレートリミット（`LOGIN_RATE_LIMIT_*`）は既定値（5 / 20 / 15 分）を使う。変えたくなったら workflow の `env_vars` に追加する。

## 5. 運用

### ロールバック

Cloud Run はリビジョンを保持するので、直前のリビジョンにトラフィックを戻す（staging は `-stg` 付きの名前で同様）。

```bash
gcloud run revisions list --service=satehits-api --project="$PROJECT_ID" --region="$REGION"
gcloud run services update-traffic satehits-api \
  --project="$PROJECT_ID" --region="$REGION" \
  --to-revisions=<前のリビジョン名>=100
```

private 側も同じように戻す。マイグレーションは前進のみ（down なし）なので、**スキーマ変更を含むデプロイのロールバックはコードが旧スキーマで動くかを先に確認する**。加算的な変更（列追加・テーブル追加）にとどめる方針。

次のデプロイ（その環境のブランチへの push）で自動的に新しいリビジョンへ 100% 戻る。

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
2. SQL を修正して `develop` に push する（失敗したファイルは記録されていないので、次回の Job が再適用する）。**適用済みファイルは編集しない**（記録済みで再実行されない）。直す場合は新しい番号のファイルを追加する
3. サービスは前のリビジョンのまま（migrate 失敗時はサービスのデプロイに進まない）

staging で migrate が通ってから `main` へマージすれば、production での migrate 失敗はほぼ防げる（staging の Supabase が開発用の再利用である間はスキーマの差異に注意）。手動で再実行するだけなら `gcloud run jobs execute satehits-migrate --wait`（staging は `satehits-migrate-stg`）。

### コスト

- **CPU always allocated は使わない**（`--cpu-throttling`、リクエスト処理中のみ課金）
- `--min-instances=0`（アイドル時は 0 台。コールドスタートは数百 ms 〜 1 秒程度）
- staging は 3 リソース（サービス 2 + Job 1）増えるが、min 0 なので使わない間はほぼ無料。staging の Scheduler を作ると毎分 private サービスを起こすので、不要なら作らないか `gcloud scheduler jobs pause outbox-flush-stg` で止める
- Scheduler の 1 分ごとの flush は private サービスを毎分起こす。無料枠内の見込みだが billing account 単位で共有されるため保証はない（`docs/infrastructure.md` §11）
- Artifact Registry はイメージが sha ごとに溜まる（staging / production で共用。同じ sha なら 1 つ）。容量課金（数十 MB / イメージ）。`scripts/setup-gcp.sh` が cleanup policy（直近 5 版を保持し、30 日より古いものを削除）を設定済み

## 6. 既知の制約

- `main` ブランチは現状 initial commit のままなので、**develop → main のマージが production の初回デプロイ**になる。マージ前に §3 (1)〜(4) を済ませ、staging で一度通しておく
- staging の Supabase は開発用プロジェクトの再利用なので、ローカル開発と staging がデータ・スキーマを共有する。production の Supabase（Tokyo）を作ったら `DATABASE_URL` を入れ替えるだけでよい
- サーバーのポートは `:8080` 固定（`backend/cmd/api/main.go`）。Cloud Run の既定 `PORT=8080` と一致しているので問題ないが、`--port` を変えても効かない
- migrate はロックを取らない（`backend/cmd/migrate/main.go`）。workflow の `concurrency` で環境ごとに直列化しているので通常は問題ないが、手動 `execute` を同時に走らせない
- `docker build` はキャッシュなしで毎回フルビルド（数分）。速度が問題になったら `docker/build-push-action` + GHA キャッシュに切り替える
- seed のメールアドレスが `example.com` 固定（§3 (7)）
