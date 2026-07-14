"""
Demo: log de eventos Eltek + DataLog Enetek, com dados reais dos
equipamentos (não o for_Data.csv histórico do towercore).

Uso:
    python3 scripts/demo_local_logs.py <elteklogs.csv> <DataLog.csv>
"""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent / "python"))

from training.eltek_event_log import parse_eventlog, pair_outage_windows
from training.enetek_datalog import load_datalog, to_canonical_series, discharge_windows
from feature_engineering.features import build_features
from models.registry import get_model


def main():
    if len(sys.argv) != 3:
        print("uso: python3 demo_local_logs.py <elteklogs.csv> <DataLog.csv>")
        sys.exit(1)

    eltek_path, enetek_path = sys.argv[1], sys.argv[2]

    print("===== Log de eventos Eltek =====")
    events = parse_eventlog(eltek_path)
    print(f"{len(events)} eventos lidos\n")

    for desc in ("MainsLow", "Door open", "BatteryTemp"):
        windows = pair_outage_windows(events, desc)
        print(f"-- {desc}: {len(windows)} janela(s) --")
        for w in windows[:5]:
            mins = w.duration.total_seconds() / 60
            print(f"   {w.start_time} -> {w.end_time}  ({mins:.1f} min)")
    print()

    print("===== DataLog =====")
    rows = load_datalog(enetek_path)
    print(f"{len(rows)} linhas lidas")

    windows = discharge_windows(rows)
    print(f"{len(windows)} janela(s) de Discharge (confirmar se o site tem solar antes de tratar como falha)\n")

    canon = to_canonical_series(rows)
    features = build_features(canon)

    for model_name in ("health_score", "ewma_baseline"):
        result = get_model(model_name).predict(features)
        print(f"[{model_name}] score={result['score']} status={result['status']} -- {result['explanation']}")


if __name__ == "__main__":
    main()