"""
Ponto de entrada único para (re)treinar todos os modelos que precisam de
treino. health_score e forecast não precisam (baseados em regra/extrapolação);
só anomaly precisa de dados históricos.

Uso:
    python -m training.pipeline caminho/para/features_historicas.csv

Pensado para correr periodicamente (ex: mensal, via cron/Task Scheduler) à
medida que mais histórico se acumula em ai_features -- mais dados tornam o
Isolation Forest mais preciso a distinguir normal de anómalo.
"""

import sys

from .trainer import train_isolation_forest


def run(csv_path: str):
    print("=== pipeline de treino AIP ===")
    train_isolation_forest(csv_path, "anomaly.joblib")
    print("=== concluído ===")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("uso: python -m training.pipeline <caminho_csv>")
        sys.exit(1)
    run(sys.argv[1])
