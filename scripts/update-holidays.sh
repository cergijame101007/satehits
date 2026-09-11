#!/usr/bin/env bash
# 内閣府「国民の祝日」CSV を取得し、backend（Go embed 用 CSV）と frontend（JSON）の祝日データを更新する。
# 年 1 回、翌年分が公開されたら（例年 2 月頃）`make update-holidays` で実行してコミットする（docs/holidays.md）。
#
# 出力:
#   backend/internal/domain/holiday/syukujitsu.csv  … 元 CSV を UTF-8 / LF に変換したもの（名称列も残す）
#   frontend/src/data/holidays.json                  … 日付のみの配列 ["1955-01-01", ...]（昇順）
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_URL="${HOLIDAY_SOURCE_URL:-https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv}"
BACKEND_CSV="${ROOT_DIR}/backend/internal/domain/holiday/syukujitsu.csv"
FRONTEND_JSON="${ROOT_DIR}/frontend/src/data/holidays.json"

for cmd in curl iconv awk sort; do
  if ! command -v "${cmd}" >/dev/null 2>&1; then
    echo "error: ${cmd} is required" >&2
    exit 1
  fi
done

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT

echo "Downloading ${SOURCE_URL}"
curl -sSfL -o "${TMP_DIR}/syukujitsu_sjis.csv" "${SOURCE_URL}"

# Shift_JIS → UTF-8、CRLF → LF、先頭 BOM があれば除去
iconv -f SHIFT_JIS -t UTF-8 "${TMP_DIR}/syukujitsu_sjis.csv" \
  | tr -d '\r' \
  | sed '1s/^\xEF\xBB\xBF//' \
  > "${TMP_DIR}/syukujitsu_utf8.csv"

# 形式の簡易検証（ヘッダと行数）。出典側の形式変更に気付けるようにする
if ! head -n 1 "${TMP_DIR}/syukujitsu_utf8.csv" | grep -q '国民の祝日'; then
  echo "error: unexpected header: $(head -n 1 "${TMP_DIR}/syukujitsu_utf8.csv")" >&2
  exit 1
fi
ROW_COUNT="$(grep -Ec '^[0-9]{4}/[0-9]{1,2}/[0-9]{1,2},' "${TMP_DIR}/syukujitsu_utf8.csv")"
if [[ "${ROW_COUNT}" -lt 100 ]]; then
  echo "error: too few holiday rows (${ROW_COUNT})" >&2
  exit 1
fi

# frontend 用 JSON（YYYY-MM-DD の昇順配列）。日付として解釈できない行は捨てる
awk -F',' '
  $1 ~ /^[0-9]{4}\/[0-9]{1,2}\/[0-9]{1,2}$/ {
    split($1, ymd, "/")
    printf "%04d-%02d-%02d\n", ymd[1], ymd[2], ymd[3]
  }
' "${TMP_DIR}/syukujitsu_utf8.csv" | sort -u > "${TMP_DIR}/dates.txt"

{
  echo '['
  awk '{ printf "%s  \"%s\"", (NR > 1 ? ",\n" : ""), $0 } END { print "" }' "${TMP_DIR}/dates.txt"
  echo ']'
} > "${TMP_DIR}/holidays.json"

mkdir -p "$(dirname "${BACKEND_CSV}")" "$(dirname "${FRONTEND_JSON}")"
cp "${TMP_DIR}/syukujitsu_utf8.csv" "${BACKEND_CSV}"
cp "${TMP_DIR}/holidays.json" "${FRONTEND_JSON}"

LAST_DATE="$(tail -n 1 "${TMP_DIR}/dates.txt")"
echo "Updated ${BACKEND_CSV#"${ROOT_DIR}"/} (${ROW_COUNT} rows)"
echo "Updated ${FRONTEND_JSON#"${ROOT_DIR}"/} ($(wc -l < "${TMP_DIR}/dates.txt") dates)"
echo "Last holiday in data: ${LAST_DATE}"
echo "Review the diff and commit (e.g. 'chore: 祝日データを ${LAST_DATE%%-*} 年分まで更新')."
