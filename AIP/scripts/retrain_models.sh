#!/usr/bin/env bash
# Re-treino dos modelos Python (anomaly v2 + health metadata).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PYTHONPATH="$ROOT/python${PYTHONPATH:+:$PYTHONPATH}"
CSV="${ANOMALY_TRAIN_CSV:-}"

echo "== health_score metadata =="
python3 "$ROOT/python/training/train_health_score.py"

echo "== anomaly (FEATURE_ORDER v2) =="
if [[ -n "$CSV" && -f "$CSV" ]]; then
  python3 -m training.train_anomaly "$CSV"
else
  echo "(sem ANOMALY_TRAIN_CSV — bootstrap sintético)"
  (cd "$ROOT/python" && python3 -m training.train_anomaly)
fi

echo "Modelos em $ROOT/python/models/"
ls -la "$ROOT/python/models/"*.joblib "$ROOT/python/models/"*.meta.json 2>/dev/null || true
echo "Reinicia o serviço de inferência para carregar o novo anomaly.joblib"
