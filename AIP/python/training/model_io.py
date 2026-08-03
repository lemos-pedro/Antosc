"""
Guardar/carregar modelos + sidecar de versão (JSON).
"""

from __future__ import annotations

import json
from datetime import datetime, timezone
from pathlib import Path

import joblib

MODELS_DIR = Path(__file__).parent.parent / "models"


def save_model(
    model,
    filename: str,
    *,
    model_name: str = "",
    version: str = "",
    feature_names: list[str] | None = None,
    metrics: dict | None = None,
) -> Path:
    MODELS_DIR.mkdir(exist_ok=True)
    path = MODELS_DIR / filename
    joblib.dump(model, path)
    print(f"modelo guardado: {path}")

    meta = {
        "model_name": model_name or Path(filename).stem,
        "version": version,
        "filename": filename,
        "saved_at": datetime.now(timezone.utc).isoformat(),
        "n_features": len(feature_names) if feature_names else None,
        "feature_names": feature_names,
        "metrics": metrics or {},
    }
    meta_path = path.with_suffix(".meta.json")
    meta_path.write_text(json.dumps(meta, indent=2, ensure_ascii=False), encoding="utf-8")
    print(f"metadata: {meta_path}")
    return path


def load_model(filename: str):
    path = MODELS_DIR / filename
    if not path.exists():
        raise FileNotFoundError(f"{path} não encontrado -- treina o modelo primeiro.")
    return joblib.load(path)


def load_meta(filename: str) -> dict | None:
    path = (MODELS_DIR / filename).with_suffix(".meta.json")
    if not path.exists():
        return None
    return json.loads(path.read_text(encoding="utf-8"))
