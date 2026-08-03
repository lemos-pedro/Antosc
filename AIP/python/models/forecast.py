"""
Modelo de previsão de janela de falha.

v2.0.0 — multi-sinal + EWMA da taxa de degradação + contribuições.

Sinais:
  - battery_voltage_drop / battery_voltage_min (principal)
  - temperature_max (acelera degradação estimada)
  - generator_runtime_growth (stress de backup)
  - availability_percent (já em falha intermitente)

Continua sem dependência de treino pesado: auditável e funciona desde o
primeiro histórico suficiente. Quando houver falhas rotuladas em massa,
substituir a física por XGBoost/survival (ver training/).
"""

from __future__ import annotations

from .base import BaseModel
from .explain import format_contributions_pt, top_contributions

WINDOW_DAYS = 7
CRITICAL_VOLTAGE = 48.0
MAX_FORECAST_DAYS = 180


class ForecastModel(BaseModel):

    name = "forecast"
    version = "2.1.0"

    def predict(self, features: dict[str, float]) -> dict:
        drop_full = float(features.get("battery_voltage_drop", 0.0) or 0.0)
        drop_short = float(features.get("battery_voltage_drop_short", 0.0) or 0.0)
        # Preferir janela curta se indicar degradação; senão a completa
        drop = drop_short if drop_short > drop_full else drop_full
        slope = float(features.get("battery_voltage_slope_6", 0.0) or 0.0)
        # slope negativo em tensão = degradação (V a descer)
        if slope < 0 and drop <= 0:
            drop = max(drop, abs(slope) * 6)  # aproxima queda na janela short

        current = float(
            features.get(
                "battery_voltage_min",
                features.get("battery_voltage_avg", 0.0),
            )
            or 0.0
        )
        temp_max = float(features.get("temperature_max", 0.0) or 0.0)
        gen_growth = float(features.get("generator_runtime_growth", 0.0) or 0.0)
        availability = float(features.get("availability_percent", 100.0) or 100.0)

        contributions: dict[str, float] = {}

        if current <= CRITICAL_VOLTAGE:
            contributions["battery_voltage_min"] = 50.0
            top = top_contributions(contributions)
            return {
                "score": 0.0,
                "status": "critical_now",
                "explanation": (
                    "tensão da bateria já no limiar crítico ou abaixo — "
                    "ação imediata, não previsão. "
                    + format_contributions_pt(top)
                ),
                "predicted_failure_window_days": 0,
                "confidence": 0.9,
                "feature_contributions": top,
            }

        # Taxa diária base (EWMA simples: peso maior ao drop observado)
        base_daily = (drop / WINDOW_DAYS) if WINDOW_DAYS else 0.0

        # Aceleradores: temperatura alta e uso intensivo de gerador
        accel = 1.0
        if temp_max > 40:
            factor = 1.0 + min(0.5, (temp_max - 40) / 20.0)
            accel *= factor
            contributions["temperature_max"] = round((factor - 1.0) * 20, 2)
        if gen_growth > 24:
            factor = 1.0 + min(0.3, (gen_growth - 24) / 48.0)
            accel *= factor
            contributions["generator_runtime_growth"] = round((factor - 1.0) * 15, 2)
        if availability < 95:
            factor = 1.0 + min(0.4, (95 - availability) / 20.0)
            accel *= factor
            contributions["availability_percent"] = round((factor - 1.0) * 15, 2)

        daily_rate = base_daily * accel
        contributions["battery_voltage_drop"] = round(base_daily * 10, 2)
        if slope < 0:
            contributions["battery_voltage_slope_6"] = round(slope * 10, 2)

        if daily_rate <= 0:
            top = top_contributions(contributions)
            return {
                "score": 85.0,
                "status": "stable",
                "explanation": (
                    "sem tendência de degradação de bateria na janela observada — "
                    "janela de falha indeterminada. "
                    + format_contributions_pt(top)
                ),
                "predicted_failure_window_days": None,
                "confidence": 0.55,
                "feature_contributions": top,
            }

        headroom = max(0.0, current - CRITICAL_VOLTAGE)
        days = headroom / daily_rate
        days = min(MAX_FORECAST_DAYS, max(0.0, days))
        days_int = int(round(days))

        # Score: quanto mais longe a falha, melhor (0-100)
        score = max(0.0, min(100.0, (days / MAX_FORECAST_DAYS) * 100.0))

        if days_int <= 7:
            status = "critical"
        elif days_int <= 30:
            status = "at_risk"
        elif days_int <= 90:
            status = "watch"
        else:
            status = "stable"

        # Confiança: maior se drop for claro e tensão bem medida
        confidence = 0.5
        if drop > 0.5:
            confidence += 0.2
        if current > 0:
            confidence += 0.1
        if temp_max > 0:
            confidence += 0.05
        confidence = min(0.92, confidence)

        contributions["predicted_days"] = float(days_int)
        top = top_contributions(
            {k: v for k, v in contributions.items() if k != "predicted_days"}
        )

        explanation = (
            f"falha estimada em ~{days_int} dia(s) "
            f"(taxa diária efectiva {daily_rate:.4f} V/dia, "
            f"headroom {headroom:.2f} V). "
            + format_contributions_pt(top)
        )

        return {
            "score": round(score, 1),
            "status": status,
            "explanation": explanation,
            "predicted_failure_window_days": days_int,
            "confidence": round(confidence, 2),
            "feature_contributions": top,
        }

    def metadata(self) -> dict:
        return {
            "name": self.name,
            "version": self.version,
            "method": "multi-signal EWMA degradation + domain accelerators",
        }
