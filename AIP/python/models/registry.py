"""
Registo central de modelos.

Carregamento preguiçoso (lazy) -- um modelo só é instanciado (e o .joblib
só é lido do disco) na primeira vez que é pedido, não no arranque do
serviço. Isto permite o AIP arrancar e responder por "health_score" mesmo
que "anomaly" ainda não tenha sido treinado (ver
python/training/train_anomaly.py), em vez de o serviço inteiro falhar a
arrancar por causa de um modelo em falta.
"""

from .base import BaseModel
from .health_score import HealthScoreModel
from .anomaly import AnomalyModel
from .forecast import ForecastModel


_FACTORIES = {
    "health_score": HealthScoreModel,
    "anomaly": AnomalyModel,
    "forecast": ForecastModel,
}

_INSTANCES: dict[str, BaseModel] = {}


def get_model(name: str) -> BaseModel:

    key = name.lower()

    if key in _INSTANCES:
        return _INSTANCES[key]

    factory = _FACTORIES.get(key)

    if factory is None:
        raise ValueError(f"Modelo '{name}' não encontrado.")

    # Instancia aqui, não no import do módulo -- é o que torna isto lazy.
    # Se o .joblib estiver em falta (ex: anomaly antes do primeiro treino),
    # o erro só acontece quando alguém pedir esse modelo específico.
    instance = factory()
    _INSTANCES[key] = instance
    return instance
