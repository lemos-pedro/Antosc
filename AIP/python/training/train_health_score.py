"""Serializa metadados do health_score (regras vivem em models/health_score.py)."""
from pathlib import Path
import joblib

MODEL_PATH = Path(__file__).parent.parent / "models" / "health_score.joblib"


def train():
    MODEL_PATH.parent.mkdir(exist_ok=True)
    joblib.dump({"name": "health_score", "version": "1.1.0", "method": "rules"}, MODEL_PATH)
    print(f"metadata gravado em {MODEL_PATH}")


if __name__ == "__main__":
    train()
