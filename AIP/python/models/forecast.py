"""
Modelo de previsão de janela de falha (o "quantas semanas/meses até
falhar", que é o objetivo central do AIP).

v1: extrapolação linear simples a partir da taxa de degradação já calculada
em feature_engineering (battery_voltage_drop). Não precisa de treino --
funciona a partir do primeiro dia com dados. É deliberadamente simples e
auditável: qualquer pessoa consegue verificar a conta à mão.

v2 (evolução natural, ver training/pipeline.py): substituir a extrapolação
linear por um modelo de sobrevivência (ex: Cox proportional hazards) ou
XGBoost treinado em falhas históricas reais, quando houver histórico
suficiente acumulado (ai_events + site_incident_causes do lado Go já estão
a acumular esse histórico).

ASSUNÇÃO A CONFIRMAR: 'battery_voltage_drop' é a diferença entre o primeiro
e o último valor da janela de leitura mais recente enviada ao pipeline.
Este modelo assume que essa janela cobre WINDOW_DAYS dias -- se o AIP Go
agregar a série num período diferente, ajusta a constante abaixo.
"""

from .base import BaseModel

# Dias cobertos pela janela usada para calcular battery_voltage_drop.
# Ajustar se o período de agregação em feature_engineering mudar.
WINDOW_DAYS = 7

# Tensão abaixo da qual se considera falha de bateria (mesmo limiar usado
# em train_health_score.py, para manter os dois modelos consistentes).
CRITICAL_VOLTAGE = 48.0

MAX_FORECAST_DAYS = 180  # não projeta para além de ~6 meses -- confiança cai demais


class ForecastModel(BaseModel):

    name = "forecast"
    version = "1.0.0"

    def predict(self, features: dict[str, float]) -> dict:
        drop = features.get("battery_voltage_drop", 0.0)
        current_voltage = features.get("battery_voltage_min", features.get("battery_voltage_avg", 0.0))

        daily_drop_rate = drop / WINDOW_DAYS if WINDOW_DAYS else 0.0

        if daily_drop_rate <= 0 or current_voltage <= CRITICAL_VOLTAGE:
            # Sem tendência de degradação clara, ou já está em zona crítica --
            # não faz sentido "prever" uma janela futura nestes casos.
            if current_voltage <= CRITICAL_VOLTAGE:
                return {
                    "score": 0.0,
                    "status": "critical_now",
                    "explanation": "tensão da bateria já está no limiar crítico ou abaixo -- ação imediata, não previsão",
                    "predicted_failure_window_days": 0,
                    "confidence": 0.9,
                }
            return {
                "score": 100.0,
                "status": "stable",
                "explanation": "sem tendência de degradação detetada na janela mais recente",
                "predicted_failure_window_days": None,
                "confidence": 0.5,
            }

        days_to_critical = (current_voltage - CRITICAL_VOLTAGE) / daily_drop_rate
        days_to_critical = max(0, min(days_to_critical, MAX_FORECAST_DAYS))

        # Confiança cai quanto mais longe no futuro a previsão está --
        # extrapolação linear é razoável a curto prazo, especulativa a longo prazo.
        confidence = max(0.3, 1 - (days_to_critical / MAX_FORECAST_DAYS))

        status = "at_risk" if days_to_critical <= 30 else "watch"

        return {
            "score": round(max(0.0, 100 - (100 * (1 - days_to_critical / MAX_FORECAST_DAYS))), 1),
            "status": status,
            "explanation": (
                f"ao ritmo atual de degradação (~{daily_drop_rate:.2f}V/dia), "
                f"a bateria atinge o limiar crítico ({CRITICAL_VOLTAGE}V) em cerca de "
                f"{int(days_to_critical)} dias"
            ),
            "predicted_failure_window_days": int(days_to_critical),
            "confidence": round(confidence, 2),
        }

    def metadata(self) -> dict:
        return {"name": self.name, "version": self.version}
