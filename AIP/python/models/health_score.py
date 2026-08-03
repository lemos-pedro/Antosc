"""
Health score baseado em regras com contribuições exactas por feature.
Não depende de joblib para inferência (as regras são código versionado).
O ficheiro .joblib mantém-se por compatibilidade com pipelines antigos.
"""

from __future__ import annotations

from pathlib import Path

from .base import BaseModel
from .explain import format_contributions_pt, top_contributions

MODEL_FILE = Path(__file__).parent / "health_score.joblib"


class HealthScoreModel(BaseModel):

    name = "health_score"
    version = "1.2.0"

    def __init__(self):
        # joblib opcional — inferência usa as regras abaixo
        self._artifact = None
        if MODEL_FILE.exists():
            try:
                import joblib
                self._artifact = joblib.load(MODEL_FILE)
            except Exception:
                self._artifact = None

    def predict(self, features: dict[str, float]) -> dict:
        score = 100.0
        risks: list[str] = []
        contributions: dict[str, float] = {}

        voltage = float(features.get("battery_voltage_avg", 0) or 0)
        if voltage < 48:
            contributions["battery_voltage_avg"] = -30
            score -= 30
            risks.append("battery voltage critical")
        elif voltage < 50:
            contributions["battery_voltage_avg"] = -15
            score -= 15
            risks.append("battery voltage low")

        drop = float(features.get("battery_voltage_drop", 0) or 0)
        drop_short = float(features.get("battery_voltage_drop_short", 0) or 0)
        slope = float(features.get("battery_voltage_slope_6", 0) or 0)
        if drop > 1 or drop_short > 0.8 or slope < -0.05:
            contributions["battery_voltage_drop"] = -20
            score -= 20
            risks.append("battery voltage decreasing")
            if slope < -0.05:
                contributions["battery_voltage_slope_6"] = round(slope * 100, 2)

        temperature = float(features.get("temperature_max", 0) or 0)
        if temperature > 40:
            contributions["temperature_max"] = -20
            score -= 20
            risks.append("high temperature")

        availability = float(features.get("availability_percent", 100) or 100)
        if availability < 95:
            contributions["availability_percent"] = -15
            score -= 15
            risks.append("low availability")

        score = max(0.0, min(100.0, score))
        if score >= 80:
            status = "healthy"
        elif score >= 50:
            status = "warning"
        else:
            status = "critical"

        top = top_contributions(contributions)
        if risks:
            explanation = "riscos: " + "; ".join(risks) + ". " + format_contributions_pt(top)
        else:
            explanation = "sem riscos relevantes nos limiares actuais. " + format_contributions_pt(top)

        return {
            "score": round(score, 1),
            "status": status,
            "risks": risks,
            "explanation": explanation.strip(),
            "feature_contributions": top,
            "confidence": 0.8 if risks else 0.7,
        }

    def metadata(self) -> dict:
        return {"name": self.name, "version": self.version, "method": "rules+additive_contributions"}
