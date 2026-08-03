#!/usr/bin/env bash
# Exporta CSV dos datasets Power BI (para modelar offline no Desktop).
# Uso: AIP_API_KEY=... ./scripts/export_powerbi_csv.sh [outdir]
set -euo pipefail

BASE="${AIP_BASE_URL:-http://localhost:8090}"
KEY="${AIP_API_KEY:-}"
OUT="${1:-./powerbi_export}"
FROM="${PBI_FROM:-2026-01-01}"
TO="${PBI_TO:-$(date -u +%Y-%m-%d)}"

HDR=()
[[ -n "$KEY" ]] && HDR=(-H "X-API-Key: $KEY")

mkdir -p "$OUT"

export_one() {
  local name="$1" url="$2"
  echo "→ $name"
  curl -fsS "${HDR[@]}" "$url" -o "$OUT/${name}.json"
  # JSON array → CSV simples via python
  python3 - "$OUT/${name}.json" "$OUT/${name}.csv" << 'PY'
import json, csv, sys
src, dst = sys.argv[1], sys.argv[2]
with open(src, encoding="utf-8") as f:
    data = json.load(f)
if isinstance(data, dict):
    data = [data]
if not data:
    open(dst, "w").write("")
    raise SystemExit
keys = list(data[0].keys())
with open(dst, "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=keys)
    w.writeheader()
    for row in data:
        w.writerow({k: row.get(k, "") for k in keys})
PY
}

export_one towers "$BASE/api/v1/powerbi/towers"
export_one incidents "$BASE/api/v1/powerbi/incidents?from=$FROM&to=$TO"
export_one consumption "$BASE/api/v1/powerbi/consumption?from=$FROM&to=$TO"
export_one predictions "$BASE/api/v1/powerbi/predictions"
export_one kpis "$BASE/api/v1/powerbi/kpis?from=$FROM&to=$TO"

echo "Export em $OUT (json + csv)"
ls -la "$OUT"
