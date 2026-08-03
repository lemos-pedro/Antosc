"""
Treino do modelo de deteção de anomalias (vetor temporal v2).

Uso:
    python -m training.train_anomaly
    python -m training.train_anomaly caminho/para/features_historicas.csv

Sem CSV: bootstrap sintético (dev / primeiro deploy).
Com CSV: colunas = FEATURE_ORDER em models/anomaly.py.
"""

import sys

from .trainer import train_isolation_forest


def main():
    csv_path = sys.argv[1] if len(sys.argv) >= 2 else None
    train_isolation_forest(csv_path, "anomaly.joblib")


if __name__ == "__main__":
    main()
