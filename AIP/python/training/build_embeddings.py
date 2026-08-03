"""
Gera embeddings cross-tower (PCA sobre FEATURE_ORDER) a partir de CSV wide
e grava em Postgres (tower_embeddings).

Uso:
  PYTHONPATH=python python -m training.build_embeddings data/features_wide.csv
  EMBEDDING_DIMS=16 DATABASE_URL=... python -m training.build_embeddings data/features_wide.csv
"""
from __future__ import annotations

import os
import sys
from pathlib import Path

import numpy as np


def main():
    if len(sys.argv) < 2:
        print("uso: python -m training.build_embeddings features_wide.csv")
        return 1
    csv_path = Path(sys.argv[1])
    dims = int(os.getenv("EMBEDDING_DIMS", "16"))

    import pandas as pd
    from models.anomaly import FEATURE_ORDER
    from sklearn.decomposition import PCA
    from sklearn.preprocessing import StandardScaler

    df = pd.read_csv(csv_path)
    if "tower_id" not in df.columns:
        print("CSV precisa de tower_id")
        return 1
    cols = [c for c in FEATURE_ORDER if c in df.columns]
    # última observação por torre
    df = df.sort_values("tower_id").groupby("tower_id", as_index=False).tail(1)
    X = df[cols].fillna(0).values.astype(float)
    ids = df["tower_id"].astype(str).tolist()

    scaler = StandardScaler()
    Xs = scaler.fit_transform(X)
    n_comp = min(dims, Xs.shape[0], Xs.shape[1])
    pca = PCA(n_components=n_comp)
    emb = pca.fit_transform(Xs)
    # pad se n_comp < dims
    if emb.shape[1] < dims:
        pad = np.zeros((emb.shape[0], dims - emb.shape[1]))
        emb = np.hstack([emb, pad])

    dsn = os.getenv("DATABASE_URL")
    if not dsn:
        host = os.getenv("AIP_DB_HOST", "localhost")
        port = os.getenv("AIP_DB_PORT", "5432")
        user = os.getenv("AIP_DB_USER", "postgres")
        password = os.getenv("AIP_DB_PASSWORD", "postgres")
        name = os.getenv("AIP_DB_NAME", "aip")
        dsn = f"postgresql://{user}:{password}@{host}:{port}/{name}"

    try:
        import psycopg2
        from psycopg2.extras import execute_values
    except ImportError:
        print("instala psycopg2-binary para gravar na DB")
        out = Path("embeddings_preview.csv")
        import csv
        with out.open("w", newline="") as f:
            w = csv.writer(f)
            w.writerow(["tower_id"] + [f"d{i}" for i in range(dims)])
            for tid, vec in zip(ids, emb):
                w.writerow([tid] + list(map(float, vec)))
        print(f"preview em {out}")
        return 0

    conn = psycopg2.connect(dsn)
    cur = conn.cursor()
    rows = [(tid, dims, list(map(float, vec)), "pca-v1") for tid, vec in zip(ids, emb)]
    execute_values(
        cur,
        """
        INSERT INTO tower_embeddings (tower_id, dims, vector, model_version, updated_at)
        VALUES %s
        ON CONFLICT (tower_id) DO UPDATE SET
          dims=EXCLUDED.dims, vector=EXCLUDED.vector,
          model_version=EXCLUDED.model_version, updated_at=NOW()
        """,
        rows,
        template="(%s, %s, %s::double precision[], %s, NOW())",
    )
    conn.commit()
    cur.close()
    conn.close()
    print(f"embeddings gravados: {len(rows)} torres, dims={dims}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
