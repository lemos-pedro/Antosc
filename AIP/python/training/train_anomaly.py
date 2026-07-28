"""
Treino do modelo de deteção de anomalias.

Uso:
    python -m training.train_anomaly caminho/para/features_historicas.csv

Ver training/dataset.py para o formato esperado do CSV.
"""

import sys

from .trainer import train_isolation_forest


def main():
    if len(sys.argv) != 2:
        print("uso: python -m training.train_anomaly <caminho_csv>")
        sys.exit(1)

    csv_path = sys.argv[1]
    train_isolation_forest(csv_path, "anomaly.joblib")


if __name__ == "__main__":
    main()
