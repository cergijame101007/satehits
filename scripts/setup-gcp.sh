#!/usr/bin/env bash
# GitHub Actions → Cloud Run 継続デプロイ（.github/workflows/deploy-backend.yml）の GCP 側初回セットアップ（冪等）
# Cloud Shell での実行を想定。必要な権限: プロジェクトの Owner 相当
#   （serviceusage / artifactregistry / iam / secretmanager / resourcemanager.projectIamAdmin）
# 手順書: docs/deploy_cloud_run.md
set -euo pipefail

PROJECT_ID="${PROJECT_ID:?PROJECT_ID is required}"
REGION="${REGION:-asia-northeast1}"
# GitHub リポジトリ（owner/name）。Workload Identity の attribute-condition で、このリポジトリからの OIDC トークンだけを受け付ける
GITHUB_REPO="${GITHUB_REPO:-cergijame101007/satehits}"
ARTIFACT_REPO="${ARTIFACT_REPO:-satehits}"
RUNTIME_SA_NAME="${RUNTIME_SA_NAME:-satehits-run-sa}"
DEPLOY_SA_NAME="${DEPLOY_SA_NAME:-github-deployer}"
POOL_ID="${POOL_ID:-github}"
PROVIDER_ID="${PROVIDER_ID:-github}"
# 値は入れない（枠だけ作る）。値の投入は docs/deploy_cloud_run.md の手順で行う
# production は素の名前、staging は "_STG" サフィックス（GCP プロジェクトは両環境で共用）
SECRET_BASE_NAMES=(DATABASE_URL JWT_SECRET TURNSTILE_SECRET_KEY RESEND_API_KEY STORAGE_ACCESS_KEY STORAGE_SECRET_KEY)
SECRET_SUFFIXES=("" "_STG")

RUNTIME_SA_EMAIL="${RUNTIME_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
DEPLOY_SA_EMAIL="${DEPLOY_SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"

PROJECT_NUMBER="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
echo "Project: ${PROJECT_ID} (${PROJECT_NUMBER}), region: ${REGION}, GitHub repo: ${GITHUB_REPO}"

# 1. API 有効化
gcloud services enable \
  run.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  iamcredentials.googleapis.com \
  cloudscheduler.googleapis.com \
  sts.googleapis.com \
  --project="${PROJECT_ID}"

# 2. Artifact Registry（Docker）リポジトリ（既存ならスキップ）
if ! gcloud artifacts repositories describe "${ARTIFACT_REPO}" \
  --project="${PROJECT_ID}" --location="${REGION}" >/dev/null 2>&1; then
  gcloud artifacts repositories create "${ARTIFACT_REPO}" \
    --project="${PROJECT_ID}" \
    --location="${REGION}" \
    --repository-format=docker \
    --description="satehits backend images"
fi

# 3. ランタイム SA（Cloud Run サービス / Job の実行者）。Secret Manager の値を読む
if ! gcloud iam service-accounts describe "${RUNTIME_SA_EMAIL}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
  gcloud iam service-accounts create "${RUNTIME_SA_NAME}" \
    --project="${PROJECT_ID}" \
    --display-name="Cloud Run runtime for satehits"
fi
gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
  --member="serviceAccount:${RUNTIME_SA_EMAIL}" \
  --role="roles/secretmanager.secretAccessor" \
  --condition=None \
  --quiet >/dev/null

# 4. デプロイ SA（GitHub Actions が WIF で impersonate する）
if ! gcloud iam service-accounts describe "${DEPLOY_SA_EMAIL}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
  gcloud iam service-accounts create "${DEPLOY_SA_NAME}" \
    --project="${PROJECT_ID}" \
    --display-name="GitHub Actions deployer for satehits"
fi
for role in roles/run.admin roles/artifactregistry.writer; do
  gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member="serviceAccount:${DEPLOY_SA_EMAIL}" \
    --role="${role}" \
    --condition=None \
    --quiet >/dev/null
done
# Cloud Run にランタイム SA を割り当てるために、ランタイム SA に対する actAs 権限が必要
gcloud iam service-accounts add-iam-policy-binding "${RUNTIME_SA_EMAIL}" \
  --project="${PROJECT_ID}" \
  --member="serviceAccount:${DEPLOY_SA_EMAIL}" \
  --role="roles/iam.serviceAccountUser" \
  --quiet >/dev/null

# 5. Workload Identity Federation（GitHub OIDC → デプロイ SA、鍵ファイル不要）
if ! gcloud iam workload-identity-pools describe "${POOL_ID}" \
  --project="${PROJECT_ID}" --location=global >/dev/null 2>&1; then
  gcloud iam workload-identity-pools create "${POOL_ID}" \
    --project="${PROJECT_ID}" \
    --location=global \
    --display-name="GitHub Actions"
fi
if ! gcloud iam workload-identity-pools providers describe "${PROVIDER_ID}" \
  --project="${PROJECT_ID}" --location=global --workload-identity-pool="${POOL_ID}" >/dev/null 2>&1; then
  gcloud iam workload-identity-pools providers create-oidc "${PROVIDER_ID}" \
    --project="${PROJECT_ID}" \
    --location=global \
    --workload-identity-pool="${POOL_ID}" \
    --display-name="GitHub" \
    --issuer-uri="https://token.actions.githubusercontent.com" \
    --attribute-mapping="google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref" \
    --attribute-condition="assertion.repository == \"${GITHUB_REPO}\""
fi
gcloud iam service-accounts add-iam-policy-binding "${DEPLOY_SA_EMAIL}" \
  --project="${PROJECT_ID}" \
  --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_ID}/attribute.repository/${GITHUB_REPO}" \
  --role="roles/iam.workloadIdentityUser" \
  --quiet >/dev/null

# 6. Secret Manager の枠（production / staging の両環境分。値は入れない。既存ならスキップ）
SECRET_NAMES=()
for suffix in "${SECRET_SUFFIXES[@]}"; do
  for base in "${SECRET_BASE_NAMES[@]}"; do
    SECRET_NAMES+=("${base}${suffix}")
  done
done
for name in "${SECRET_NAMES[@]}"; do
  if ! gcloud secrets describe "${name}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    gcloud secrets create "${name}" \
      --project="${PROJECT_ID}" \
      --replication-policy=automatic
  fi
done

# 7. GitHub 側に登録する値
PROVIDER_NAME="$(gcloud iam workload-identity-pools providers describe "${PROVIDER_ID}" \
  --project="${PROJECT_ID}" --location=global --workload-identity-pool="${POOL_ID}" \
  --format='value(name)')"

echo
echo "Done. Register the following in GitHub under BOTH environments"
echo "(Settings > Environments > staging AND production), using the same names:"
echo
echo "  [Secrets]  (same value in both environments)"
echo "  GCP_WORKLOAD_IDENTITY_PROVIDER = ${PROVIDER_NAME}"
echo "  GCP_DEPLOY_SERVICE_ACCOUNT     = ${DEPLOY_SA_EMAIL}"
echo
echo "  [Variables]  (same value in both environments)"
echo "  GCP_PROJECT_ID              = ${PROJECT_ID}"
echo "  GCP_REGION                  = ${REGION}"
echo "  GCP_ARTIFACT_REPO           = ${ARTIFACT_REPO}"
echo "  GCP_RUNTIME_SERVICE_ACCOUNT = ${RUNTIME_SA_EMAIL}"
echo
echo "  [Variables]  (different value per environment)"
echo "  CORS_ORIGINS, COOKIE_DOMAIN, STORAGE_ENDPOINT, STORAGE_BUCKET, STORAGE_PUBLIC_BASE_URL"
echo
echo "Next: add secret values (gcloud secrets versions add ...) for all of:"
echo "  production: ${SECRET_BASE_NAMES[*]}"
echo "  staging:    ${SECRET_BASE_NAMES[*]/%/_STG}"
echo "See docs/deploy_cloud_run.md."
