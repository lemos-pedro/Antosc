"""
Demo: saúde das 5 torres do historico for_Data.csv, calculada pelo AIP.

Uso:
    python3 scripts/demo_health_check.py /caminho/para/for_Data.csv

Corre os dois modelos disponíveis (health_score, ewma_baseline) para
cada torre, usando exatamente o mesmo caminho de código que a API em
produção usa (vendors -> feature_engineering -> models), só que com o
histórico em CSV em vez de pedidos ao towercore.
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent / "python"))

from training.dataset import load_readings, group_by_tower, canonical_series_for_tower
from feature_engineering.features import build_features
from models.registry import get_model


def main():
    if len(sys.argv) != 2:
        print("uso: python3 demo_health_check.py <for_Data.csv>")
        sys.exit(1)

    csv_path = sys.argv[1]

    readings = load_readings(csv_path)
    by_tower = group_by_tower(readings)

    print(f"\n{len(readings)} leituras válidas, {len(by_tower)} torres\n")

    for tower_id in sorted(by_tower.keys()):
        rs = by_tower[tower_id]
        canon = canonical_series_for_tower(rs)
        features = build_features(canon)

        print(f"===== Torre {tower_id} ({len(rs)} leituras) =====")

        for model_name in ("health_score", "ewma_baseline"):
            result = get_model(model_name).predict(features)
            print(
                f"  [{model_name:14s}] score={result['score']:>4} "
                f"status={result['status']:8s} -- {result['explanation']}"
            )
        print()


if __name__ == "__main__":
    main()
