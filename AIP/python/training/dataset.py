"""
Carregamento do dataset de treino a partir de um CSV de features
históricas, exportado do AIP.

Formato esperado, uma linha por (tower_id, timestamp), colunas =
FEATURE_ORDER de python/models/anomaly.py:

    tower_id,timestamp,battery_voltage_avg,battery_voltage_min,...,availability_percent

Como gerar este CSV a partir do AIP:
    - query SQL sobre ai_features (pivotada de linhas longas para colunas)
      e exportar com \copy no psql, ou um pequeno script à parte.
    - fica fora do escopo deste ficheiro de propósito: dataset.py só
      carrega e valida o CSV, não decide como ele foi gerado.
"""

from pathlib import Path

import pandas as pd

from models.anomaly import FEATURE_ORDER


def load_features_csv(path: str | Path) -> pd.DataFrame:
    df = pd.read_csv(path)

    missing = [c for c in FEATURE_ORDER if c not in df.columns]
    if missing:
        raise ValueError(
            f"CSV sem as colunas esperadas: {missing}. "
            f"Colunas esperadas: {FEATURE_ORDER}"
        )

    # Preenche em falta com 0, mesma convenção "segura" usada no resto do AIP.
    df[FEATURE_ORDER] = df[FEATURE_ORDER].fillna(0.0)

    return df


def to_training_matrix(df: pd.DataFrame):
    """Devolve só as colunas de features, na ordem certa, prontas para o
    modelo -- sem tower_id/timestamp, que são só metadados."""
    return df[FEATURE_ORDER].values
