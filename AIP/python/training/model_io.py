"""
Guardar/carregar modelos treinados em disco (.joblib), num único sítio para
não haver Path/joblib.dump espalhados e inconsistentes entre scripts de
treino, tal como já acontecia em train_health_score.py.
"""

from pathlib import Path

import joblib

MODELS_DIR = Path(__file__).parent.parent / "models"


def save_model(model, filename: str) -> Path:
    MODELS_DIR.mkdir(exist_ok=True)
    path = MODELS_DIR / filename
    joblib.dump(model, path)
    print(f"modelo guardado: {path}")
    return path


def load_model(filename: str):
    path = MODELS_DIR / filename
    if not path.exists():
        raise FileNotFoundError(f"{path} não encontrado -- treina o modelo primeiro.")
    return joblib.load(path)
