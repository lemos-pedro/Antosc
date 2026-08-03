#!/usr/bin/env bash
# Exporta ai_features (longo) → CSV largo para treino anomaly.
# Requer: psql + PYTHONPATH + pandas
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-$ROOT/data}"
mkdir -p "$OUT_DIR"
LONG="$OUT_DIR/features_long.csv"
WIDE="$OUT_DIR/features_wide.csv"

if [[ -n "${DATABASE_URL:-}" ]]; then
  PSQL=(psql "$DATABASE_URL" -v ON_ERROR_STOP=1)
else
  export PGPASSWORD="${AIP_DB_PASSWORD:-postgres}"
  PSQL=(psql -h "${AIP_DB_HOST:-localhost}" -p "${AIP_DB_PORT:-5432}" \
    -U "${AIP_DB_USER:-postgres}" -d "${AIP_DB_NAME:-aip}" -v ON_ERROR_STOP=1)
fi

echo "Export longo → $LONG"
"${PSQL[@]}" -c "\copy (
  SELECT tower_id, name, value, created_at
  FROM ai_features
  WHERE created_at > NOW() - INTERVAL '90 days'
) TO STDOUT WITH CSV HEADER" > "$LONG"

export PYTHONPATH="$ROOT/python${PYTHONPATH:+:$PYTHONPATH}"
python3 "$ROOT/python/scripts_pivot_features.py" "$LONG" "$WIDE"
echo "Wide CSV: $WIDE"
echo "Treino: ANOMALY_TRAIN_CSV=$WIDE make retrain"
