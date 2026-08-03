#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PYTHONPATH="$ROOT/python${PYTHONPATH:+:$PYTHONPATH}"
CSV="${ANOMALY_TRAIN_CSV:-${1:-}}"
if [[ -z "$CSV" || ! -f "$CSV" ]]; then
  echo "uso: ANOMALY_TRAIN_CSV=features_wide.csv ./scripts/check_drift.sh"
  echo "  ou: ./scripts/check_drift.sh features_wide.csv"
  exit 2
fi
python3 -m training.drift --csv "$CSV"
