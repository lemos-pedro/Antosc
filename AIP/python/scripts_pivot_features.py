#!/usr/bin/env python3
"""Pivot ai_features long → wide CSV alinhado com FEATURE_ORDER."""
from __future__ import annotations

import sys
from pathlib import Path

import pandas as pd

# allow running from repo root
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))
from models.anomaly import FEATURE_ORDER  # noqa: E402


def main():
    if len(sys.argv) != 3:
        print("uso: python scripts_pivot_features.py features_long.csv features_wide.csv")
        sys.exit(1)
    src, dst = sys.argv[1], sys.argv[2]
    df = pd.read_csv(src)
    # expected columns: tower_id, name, value, created_at
    for col in ("tower_id", "name", "value"):
        if col not in df.columns:
            raise SystemExit(f"coluna em falta: {col}")
    if "created_at" not in df.columns:
        df["created_at"] = 0

    # bucket by hour for snapshots
    df["timestamp"] = pd.to_datetime(df["created_at"], errors="coerce").dt.floor("h")
    piv = df.pivot_table(
        index=["tower_id", "timestamp"],
        columns="name",
        values="value",
        aggfunc="mean",
    ).reset_index()
    piv.columns = [str(c) for c in piv.columns]
    for c in FEATURE_ORDER:
        if c not in piv.columns:
            piv[c] = 0.0
    out = piv[["tower_id", "timestamp"] + FEATURE_ORDER]
    out.to_csv(dst, index=False)
    print(f"escrito {dst} ({len(out)} linhas, {len(FEATURE_ORDER)} features)")


if __name__ == "__main__":
    main()
