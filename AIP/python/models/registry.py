"""
Registo central de modelos.
"""

from .base import BaseModel
from .health_score import HealthScoreModel
from .ewma_baseline import EwmaBaselineModel


MODELS: dict[str, BaseModel] = {

    "health_score": HealthScoreModel(),

    "ewma_baseline": EwmaBaselineModel(),

}


def get_model(
    name: str,
) -> BaseModel:

    model = MODELS.get(
        name.lower()
    )

    if model is None:

        raise ValueError(
            f"Modelo '{name}' não encontrado."
        )

    return model