"""
Monitorização de drift de features (PSI + mean shift).

Compara estatísticas do CSV actual (ou amostra) com o baseline guardado
no sidecar anomaly.meta.json (ou ficheiro de baseline dedicado).

Uso:
    python -m training.drift --csv features_wide.csv
    python -m training.drift --csv features_wide.csv --baseline models/anomaly_baseline.json
    python -m training.drift --update-baseline --csv features_wide.csv

Exit codes:
    0 — sem drift relevante
    1 — drift detectado (acima dos limiares)
    2 — erro de configuração / dados
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

import numpy as np

from models.anomaly import FEATURE_ORDER

MODELS_DIR = Path(__file__).parent.parent / "models"
DEFAULT_BASELINE = MODELS_DIR / "feature_baseline.json"


def _stats(X: np.ndarray, names: list[str]) -> dict:
    out = {}
    for i, name in enumerate(names):
        col = X[:, i]
        out[name] = {
            "mean": float(np.mean(col)),
            "std": float(np.std(col)),
            "min": float(np.min(col)),
            "max": float(np.max(col)),
            "p50": float(np.median(col)),
        }
    return out


def population_stability_index(expected: np.ndarray, actual: np.ndarray, bins: int = 10) -> float:
    """PSI clássico entre duas distribuições 1D."""
    eps = 1e-6
    qs = np.linspace(0, 100, bins + 1)
    breaks = np.unique(np.percentile(expected, qs))
    if len(breaks) < 3:
        return 0.0
    exp_counts = np.histogram(expected, bins=breaks)[0].astype(float)
    act_counts = np.histogram(actual, bins=breaks)[0].astype(float)
    exp_perc = exp_counts / (exp_counts.sum() + eps)
    act_perc = act_counts / (act_counts.sum() + eps)
    exp_perc = np.clip(exp_perc, eps, None)
    act_perc = np.clip(act_perc, eps, None)
    return float(np.sum((act_perc - exp_perc) * np.log(act_perc / exp_perc)))


def load_csv_matrix(path: str) -> np.ndarray:
    import pandas as pd
    from training.dataset import load_features_csv, to_training_matrix

    df = load_features_csv(path)
    return to_training_matrix(df)


def compare(
    current: dict,
    baseline: dict,
    *,
    mean_shift_threshold: float = 2.0,
    psi_threshold: float = 0.25,
    current_matrix: np.ndarray | None = None,
    baseline_samples: dict | None = None,
) -> list[dict]:
    """
    mean_shift: |mean_now - mean_base| / (std_base + eps) > threshold
    psi: se baseline_samples e current_matrix disponíveis
    """
    alerts = []
    for name in FEATURE_ORDER:
        if name not in current or name not in baseline:
            continue
        b, c = baseline[name], current[name]
        std_b = b.get("std", 0) or 0.0
        eps = 1e-6
        z = abs(c["mean"] - b["mean"]) / (std_b + eps)
        psi = None
        if current_matrix is not None and baseline_samples and name in baseline_samples:
            i = FEATURE_ORDER.index(name)
            psi = population_stability_index(
                np.array(baseline_samples[name], dtype=float),
                current_matrix[:, i],
            )
        drifted = z >= mean_shift_threshold or (psi is not None and psi >= psi_threshold)
        if drifted:
            alerts.append({
                "feature": name,
                "mean_shift_z": round(z, 3),
                "psi": None if psi is None else round(psi, 3),
                "baseline_mean": b["mean"],
                "current_mean": c["mean"],
            })
    return alerts


def main(argv: list[str] | None = None) -> int:
    ap = argparse.ArgumentParser(description="Feature drift check AIP")
    ap.add_argument("--csv", help="CSV wide actual (FEATURE_ORDER)")
    ap.add_argument("--baseline", type=Path, default=DEFAULT_BASELINE)
    ap.add_argument("--update-baseline", action="store_true", help="grava baseline a partir do CSV")
    ap.add_argument("--mean-z", type=float, default=2.0)
    ap.add_argument("--psi", type=float, default=0.25)
    args = ap.parse_args(argv)

    if not args.csv:
        print("indica --csv features_wide.csv", file=sys.stderr)
        return 2

    X = load_csv_matrix(args.csv)
    stats = _stats(X, FEATURE_ORDER)

    if args.update_baseline:
        # guarda stats + amostra (até 500 pts por feature) para PSI
        sample = {}
        n = min(500, len(X))
        for i, name in enumerate(FEATURE_ORDER):
            sample[name] = X[:n, i].tolist()
        payload = {
            "updated_at": datetime.now(timezone.utc).isoformat(),
            "n_samples": int(len(X)),
            "feature_version": "2.0.0",
            "stats": stats,
            "samples": sample,
        }
        args.baseline.parent.mkdir(exist_ok=True)
        args.baseline.write_text(json.dumps(payload, indent=2), encoding="utf-8")
        print(f"baseline actualizado: {args.baseline} ({len(X)} amostras)")
        return 0

    if not args.baseline.exists():
        print(
            f"baseline em falta ({args.baseline}). Corre com --update-baseline primeiro.",
            file=sys.stderr,
        )
        return 2

    base = json.loads(args.baseline.read_text(encoding="utf-8"))
    alerts = compare(
        stats,
        base.get("stats", {}),
        mean_shift_threshold=args.mean_z,
        psi_threshold=args.psi,
        current_matrix=X,
        baseline_samples=base.get("samples"),
    )

    report = {
        "checked_at": datetime.now(timezone.utc).isoformat(),
        "n_samples": int(len(X)),
        "n_alerts": len(alerts),
        "alerts": alerts,
    }
    print(json.dumps(report, indent=2, ensure_ascii=False))

    if alerts:
        print(
            f"DRIFT: {len(alerts)} feature(s) acima do limiar — considera re-treino (make retrain)",
            file=sys.stderr,
        )
        return 1
    print("sem drift relevante", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
