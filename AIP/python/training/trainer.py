"""
Orquestrador de treino IsolationForest (anomaly).
"""

from __future__ import annotations

from pathlib import Path

import numpy as np
from sklearn.ensemble import IsolationForest

from models.anomaly import FEATURE_ORDER, FEATURE_VERSION
from .dataset import load_features_csv, to_training_matrix
from .model_io import save_model


def _synthetic_matrix(n: int = 400, seed: int = 42) -> np.ndarray:
    """
    Dataset sintético "saudável" + ~5% outliers — permite bootstrap do
    anomaly.joblib sem CSV histórico (dev / primeiro deploy).
    """
    rng = np.random.default_rng(seed)
    n_feat = len(FEATURE_ORDER)
    # baselines por tipo de feature (aproximados)
    centers = np.zeros(n_feat)
    scales = np.ones(n_feat)
    for i, name in enumerate(FEATURE_ORDER):
        if "voltage_avg" in name or "voltage_min" in name or "voltage_max" in name or "ewma" in name and "battery" in name:
            centers[i], scales[i] = 51.0, 0.4
        elif "voltage_drop" in name:
            centers[i], scales[i] = 0.2, 0.15
        elif "temperature" in name and "slope" not in name and "std" not in name:
            centers[i], scales[i] = 32.0, 3.0
        elif "availability" in name:
            centers[i], scales[i] = 99.0, 0.8
        elif "generator" in name and "growth" in name:
            centers[i], scales[i] = 5.0, 3.0
        elif "slope" in name:
            centers[i], scales[i] = 0.0, 0.02
        elif "std" in name:
            centers[i], scales[i] = 0.3, 0.15
        else:
            centers[i], scales[i] = 0.0, 1.0

    X = rng.normal(centers, scales, size=(n, n_feat))
    # 5% outliers
    n_out = max(1, n // 20)
    X[:n_out] = rng.normal(centers + scales * 6, scales * 2, size=(n_out, n_feat))
    return X


def train_isolation_forest(
    csv_path: str | None = None,
    output_filename: str = "anomaly.joblib",
    contamination: float = 0.05,
    random_state: int = 42,
):
    if csv_path:
        df = load_features_csv(csv_path)
        X = to_training_matrix(df)
        source = f"csv:{csv_path}"
    else:
        print("sem CSV — a usar dataset sintético de bootstrap (só para arranque)")
        X = _synthetic_matrix(seed=random_state)
        source = "synthetic_bootstrap"

    print(f"a treinar Isolation Forest com {len(X)} amostras, {X.shape[1]} features ({source})")

    model = IsolationForest(
        contamination=contamination,
        random_state=random_state,
        n_estimators=200,
    )
    model.fit(X)

    n_flagged = int((model.predict(X) == -1).sum())
    print(f"no conjunto de treino, {n_flagged}/{len(X)} amostras sinalizadas como anomalia")

    path = save_model(
        model,
        output_filename,
        model_name="anomaly",
        version=FEATURE_VERSION,
        feature_names=list(FEATURE_ORDER),
        metrics={
            "n_samples": int(len(X)),
            "n_flagged_train": n_flagged,
            "contamination": contamination,
            "source": source,
        },
    )
    # MLflow opcional (se instalado e MLFLOW_TRACKING_URI definido)
    try:
        import mlflow
        import mlflow.sklearn
        if __import__("os").environ.get("MLFLOW_TRACKING_URI"):
            mlflow.set_experiment("aip-anomaly")
            with mlflow.start_run(run_name=f"anomaly-{FEATURE_VERSION}"):
                mlflow.log_params({
                    "contamination": contamination,
                    "n_features": len(FEATURE_ORDER),
                    "source": source,
                    "feature_version": FEATURE_VERSION,
                })
                mlflow.log_metrics({
                    "n_samples": float(len(X)),
                    "n_flagged_train": float(n_flagged),
                })
                mlflow.sklearn.log_model(model, "model")
                print("MLflow: run registado")
    except Exception as exc:
        print(f"MLflow skip: {exc}")
    return path
