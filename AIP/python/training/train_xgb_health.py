"""
Treino opcional XGBoost de health score a partir de CSV wide + coluna target `label`
(0-100) ou `status` (healthy/warning/critical).

Sem xgboost instalado ou sem CSV: no-op informativo.
"""
from __future__ import annotations

import sys
from pathlib import Path

def main():
    if len(sys.argv) < 2:
        print("uso: python -m training.train_xgb_health features_wide.csv")
        print("CSV precisa de coluna label (0-100) ou status.")
        return 1
    try:
        import xgboost as xgb
        import pandas as pd
        import numpy as np
        import joblib
    except ImportError as e:
        print("dependência em falta:", e)
        return 1

    path = Path(sys.argv[1])
    df = pd.read_csv(path)
    from models.anomaly import FEATURE_ORDER
    cols = [c for c in FEATURE_ORDER if c in df.columns]
    if not cols:
        print("sem colunas FEATURE_ORDER no CSV")
        return 1
    if "label" in df.columns:
        y = df["label"].astype(float).values
    elif "status" in df.columns:
        map_s = {"healthy": 90, "warning": 55, "critical": 20, "normal": 90, "anomaly": 30}
        y = df["status"].map(map_s).fillna(50).values
    else:
        print("CSV precisa de label ou status")
        return 1
    X = df[cols].fillna(0).values
    model = xgb.XGBRegressor(n_estimators=80, max_depth=4, learning_rate=0.08)
    model.fit(X, y)
    out = Path(__file__).parent.parent / "models" / "health_xgb.joblib"
    joblib.dump({"model": model, "features": cols}, out)
    print(f"gravado {out}")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
