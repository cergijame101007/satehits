#!/usr/bin/env bash
# Cloud Scheduler → private Cloud Run（Outbox flush）の初回セットアップ（冪等）
# Cloud Shell での実行を想定。必要な権限: run.admin / iam.serviceAccountAdmin / cloudscheduler.admin
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?PROJECT_ID is required}"
REGION="${REGION:-asia-northeast1}"
PUBLIC_SERVICE="${PUBLIC_SERVICE:-satehits-api}"
PRIVATE_SERVICE="${PRIVATE_SERVICE:-satehits-api-internal}"
SA_NAME="${SA_NAME:-scheduler-sa}"
SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
JOB_NAME="${JOB_NAME:-outbox-flush}"
# アプリ側の OUTBOX_FLUSH_TIME_BUDGET_SECONDS（既定 120）より長く、Cloud Run request timeout より短くする
ATTEMPT_DEADLINE="${ATTEMPT_DEADLINE:-180s}"

PRIVATE_SERVICE_URL="${PRIVATE_SERVICE_URL:-}"
if [[ -z "${PRIVATE_SERVICE_URL}" ]]; then
  PRIVATE_SERVICE_URL="$(gcloud run services describe "${PRIVATE_SERVICE}" \
    --project="${PROJECT_ID}" \
    --region="${REGION}" \
    --format='value(status.url)')"
fi

echo "Using private service URL: ${PRIVATE_SERVICE_URL}"
echo "OIDC audience must be the service URL (no path/query)."

# 1. サービスアカウント作成（既存ならスキップ）
if ! gcloud iam service-accounts describe "${SA_EMAIL}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
  gcloud iam service-accounts create "${SA_NAME}" \
    --project="${PROJECT_ID}" \
    --display-name="Cloud Scheduler invoker for outbox flush"
fi

# 2. private サービスへの呼び出し権限
gcloud run services add-iam-policy-binding "${PRIVATE_SERVICE}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/run.invoker" \
  --quiet

# 3. Scheduler ジョブ（1 分ごと）
FLUSH_URI="${PRIVATE_SERVICE_URL}/internal/outbox/flush"
if gcloud scheduler jobs describe "${JOB_NAME}" --project="${PROJECT_ID}" --location="${REGION}" >/dev/null 2>&1; then
  gcloud scheduler jobs update http "${JOB_NAME}" \
    --project="${PROJECT_ID}" \
    --location="${REGION}" \
    --schedule="* * * * *" \
    --attempt-deadline="${ATTEMPT_DEADLINE}" \
    --uri="${FLUSH_URI}" \
    --http-method=POST \
    --oidc-service-account-email="${SA_EMAIL}" \
    --oidc-token-audience="${PRIVATE_SERVICE_URL}"
else
  gcloud scheduler jobs create http "${JOB_NAME}" \
    --project="${PROJECT_ID}" \
    --location="${REGION}" \
    --schedule="* * * * *" \
    --attempt-deadline="${ATTEMPT_DEADLINE}" \
    --uri="${FLUSH_URI}" \
    --http-method=POST \
    --oidc-service-account-email="${SA_EMAIL}" \
    --oidc-token-audience="${PRIVATE_SERVICE_URL}"
fi

echo "Done."
echo "Ensure ${PRIVATE_SERVICE} is deployed with OUTBOX_FLUSH_ENDPOINT_ENABLED=true and --no-allow-unauthenticated."
echo "Public service ${PUBLIC_SERVICE} should keep OUTBOX_FLUSH_ENDPOINT_ENABLED=false."
