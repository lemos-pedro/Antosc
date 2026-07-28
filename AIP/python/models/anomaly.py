"""
Modelo de deteção de anomalias.

Ao contrário do health_score (baseado em regras fixas), este modelo aprende
o "normal" a partir de dados históricos e sinaliza desvios -- incluindo
combinações de valores que nenhuma regra manual previu. É o primeiro passo
da camada preditiva descrita no objetivo do AIP: detetar problemas antes de
se tornarem falha.

Features esperadas (mesmo vetor produzido por feature_engineering/features.py):
    battery_voltage_avg, battery_voltage_min, battery_voltage_max,
    battery_voltage_drop, temperature_avg, temperature_max,
    temperature_variance, generator_runtime_avg, generator_runtime_growth,
    availability_percent

Treinado em python/training/train_anomaly.py, guardado em anomaly.joblib.
"""

from pathlib import Path

import joblib
import numpy as np

from .base import BaseModel


MODEL_FILE = Path(__file__).parent / "anomaly.joblib"

# Mesma ordem usada no treino -- tem de bater certo com training/dataset.py.
# Centralizado aqui em vez de duplicado, para as duas pontas nunca divergirem.
FEATURE_ORDER = [
    "battery_voltage_avg",
    "battery_voltage_min",
    "battery_voltage_max",
    "battery_voltage_drop",
    "temperature_avg",
    "temperature_max",
    "temperature_variance",
    "generator_runtime_avg",
    "generator_runtime_growth",
    "availability_percent",
]


def vectorize(features: dict[str, float]) -> np.ndarray:
    """Converte o dict de features no vetor ordenado que o modelo espera.
    Features em falta entram como 0 -- é o mesmo comportamento "seguro" que
    o resto do AIP já usa (ex: train_health_score.py com features.get(x, 0))."""
    return np.array([[features.get(name, 0.0) for name in FEATURE_ORDER]])


class AnomalyModel(BaseModel):

    name = "anomaly"
    version = "1.0.0"

    def __init__(self):
        if not MODEL_FILE.exists():
            raise FileNotFoundError(
                f"{MODEL_FILE} não encontrado -- corre "
                "python/training/train_anomaly.py primeiro."
            )
        self.model = joblib.load(MODEL_FILE)

    def predict(self, features: dict[str, float]) -> dict:
        x = vectorize(features)

        # decision_function: quanto mais negativo, mais anómalo.
        # predict: -1 = anomalia, 1 = normal (convenção do scikit-learn).
        raw_score = float(self.model.decision_function(x)[0])
        is_anomaly = int(self.model.predict(x)[0]) == -1

        # Normaliza para 0-100 (100 = totalmente normal), mais fácil de
        # interpretar pelo Ollama e pelos perfis do que o raw_score do sklearn.
        normalized_score = max(0.0, min(100.0, (raw_score + 0.5) * 100))

        status = "anomaly" if is_anomaly else "normal"

        explanation = self._explain(features) if is_anomaly else "sem desvio significativo face ao padrão histórico"

        return {
            "score": round(normalized_score, 1),
            "status": status,
            "explanation": explanation,
        }

    def metadata(self) -> dict:
        return {"name": self.name, "version": self.version}

    @staticmethod
    def _explain(features: dict[str, float]) -> str:
        """Explicação simples baseada nos limiares conhecidos do domínio --
        não vem do modelo em si (Isolation Forest não dá causa, só deteção).
        Serve de ponto de partida para o Ollama elaborar a resposta final,
        não como substituto de uma análise de causa-raiz."""
        reasons = []

        if features.get("battery_voltage_drop", 0) > 1:
            reasons.append("queda de tensão da bateria acima do normal")
        if features.get("battery_voltage_min", 999) < 48:
            reasons.append("tensão mínima da bateria abaixo do limiar crítico")
        if features.get("temperature_max", 0) > 40:
            reasons.append("temperatura máxima elevada")
        if features.get("temperature_variance", 0) > 10:
            reasons.append("variação de temperatura instável")
        if features.get("availability_percent", 100) < 95:
            reasons.append("disponibilidade abaixo do esperado")
        if features.get("generator_runtime_growth", 0) > 24:
            reasons.append("crescimento anormal de horas de gerador (possível dependência maior da rede)")

        if not reasons:
            return "padrão fora do normal detetado pelo modelo, sem causa óbvia nos limiares conhecidos -- requer inspeção"

        return "; ".join(reasons)
