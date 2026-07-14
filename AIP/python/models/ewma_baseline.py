"""
EWMA Baseline -- primeiro modelo estatístico do AIP (sem ML).

Complementa o health_score (baseado em limiares absolutos fixos): este
modelo sinaliza desvio relativo ao comportamento recente da própria
torre, usando o z-score da EWMA calculado em feature_engineering/ewma.py.

Uma torre "normal" para os seus próprios padrões mas a desviar-se
depressa é apanhada aqui antes de cruzar um limiar absoluto.
"""

from .base import BaseModel


# Limiares do z-score da EWMA. |desvio| > 3 é estatisticamente incomum
# (regra prática ~3 desvios-padrão); 2-3 é "atenção".
DEVIATION_WARNING = 2.0
DEVIATION_CRITICAL = 3.0

# Métricas monitorizadas por este modelo e o respetivo nome de feature.
# fuel_liters entrou por ser um sinal direto de furto de combustível:
# uma queda súbita sem correspondência em horas de funcionamento do
# gerador aparece aqui como desvio negativo grande da EWMA.
MONITORED_METRICS = ["battery_voltage", "temperature", "fuel_liters"]


class EwmaBaselineModel(BaseModel):

    name = "ewma_baseline"
    version = "1.0.0"

    def predict(self, features: dict[str, float]) -> dict:
        score = 100
        risks: list[str] = []
        max_abs_deviation = 0.0

        for metric in MONITORED_METRICS:
            deviation = features.get(f"{metric}_ewma_deviation")
            if deviation is None:
                continue

            abs_dev = abs(deviation)
            max_abs_deviation = max(max_abs_deviation, abs_dev)

            if abs_dev >= DEVIATION_CRITICAL:
                score -= 35
                direction = "acima" if deviation > 0 else "abaixo"
                risks.append(
                    f"{metric} muito {direction} do padrão recente "
                    f"(z={deviation:.2f})"
                )
            elif abs_dev >= DEVIATION_WARNING:
                score -= 15
                direction = "acima" if deviation > 0 else "abaixo"
                risks.append(
                    f"{metric} a desviar-se do padrão recente "
                    f"(z={deviation:.2f}, {direction} da média)"
                )

        score = max(0, min(score, 100))

        if score >= 80:
            status = "healthy"
        elif score >= 50:
            status = "warning"
        else:
            status = "critical"

        if risks:
            explanation = "; ".join(risks)
        elif max_abs_deviation == 0.0:
            explanation = "histórico insuficiente para calcular baseline EWMA"
        else:
            explanation = "sem desvios relevantes face ao padrão recente"

        return {
            "score": score,
            "status": status,
            "explanation": explanation,
        }

    def metadata(self) -> dict:
        return {
            "name": self.name,
            "version": self.version,
        }
