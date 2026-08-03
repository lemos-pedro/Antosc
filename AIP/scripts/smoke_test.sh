#!/usr/bin/env bash
set -euo pipefail

BASE="${AIP_BASE_URL:-http://localhost:8090}"
KEY="${AIP_API_KEY:-}"
TOKEN="${AIP_TEST_TOKEN:-}"

HDR=()
if [[ -n "$TOKEN" ]]; then
  HDR=(-H "Authorization: Bearer $TOKEN")
elif [[ -n "$KEY" ]]; then
  HDR=(-H "X-API-Key: $KEY")
fi

pass=0
fail=0

check() {
  local name="$1" url="$2" expect="${3:-200}"
  code=$(curl -s -o /tmp/aip_smoke_body -w "%{http_code}" "${HDR[@]}" "$url" || echo "000")
  if [[ "$code" == "$expect" ]]; then
    echo "OK  $name ($code)"
    pass=$((pass+1))
  else
    echo "FAIL $name (got $code, expected $expect) body=$(head -c 120 /tmp/aip_smoke_body 2>/dev/null)"
    fail=$((fail+1))
  fi
}

echo "Smoke test contra $BASE"
check "health" "$BASE/health" 200

# Auth endpoints existem
code=$(curl -s -o /tmp/aip_smoke_body -w "%{http_code}" -X POST "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' -d '{}' || echo "000")
if [[ "$code" == "400" || "$code" == "401" ]]; then
  echo "OK  auth login endpoint ($code)"
  pass=$((pass+1))
else
  echo "FAIL auth login endpoint ($code)"
  fail=$((fail+1))
fi

if [[ ${#HDR[@]} -eq 0 ]]; then
  echo "WARN sem TOKEN nem API_KEY — endpoints protegidos não testados em profundidade"
  echo "     export AIP_TEST_TOKEN=\$(login) ou AIP_API_KEY=..."
else
  FROM=$(date -u -d '7 days ago' +%Y-%m-%d 2>/dev/null || echo "2026-07-01")
  TO=$(date -u +%Y-%m-%d)
  check "powerbi catalog" "$BASE/api/v1/powerbi/catalog" 200
  check "powerbi kpis" "$BASE/api/v1/powerbi/kpis?from=$FROM&to=$TO" 200
  check "incidents pending" "$BASE/api/v1/incidents/pending" 200
fi

echo "--- pass=$pass fail=$fail"
[[ "$fail" -eq 0 ]]
