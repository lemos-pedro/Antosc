"""
Health Score -- V1, baseado em regras (sem ML).

Ver histórico desta decisão no docstring anterior deste ficheiro: a V1
não usa joblib/pickle porque é puramente baseada em regras.

REGRA IMPORTANTE (corrigida depois de validar contra for_Data.csv
real): nem todas as torres/vendors têm todas as métricas. Ex.: as
torres Eltek deste histórico não têm leitura de tensão de bateria (só
corrente por módulo e temperatura). Se uma feature está simplesmente
AUSENTE, isso é "sem dados" -- nunca "crítico". Um default de 0 para
"battery_voltage_avg" faria o modelo gritar "bateria crítica" em toda
torre sem essa leitura, o que é falso e mina a confiança no sistema.
Por isso cada verificação só corre quando a feature existe mesmo.
"""

from .base import BaseModel


class HealthScoreModel(BaseModel):

    name = "health_score"
    version = "1.0.0"

    def predict(self, features: dict[str, float]) -> dict:
        score = 100
        risks: list[str] = []
        checks_run = 0

        # bateria -- nível (só corre se houver leitura de tensão)
        voltage = features.get("battery_voltage_avg")
        if voltage is not None:
            checks_run += 1
            if voltage < 48:
                score -= 30
                risks.append("battery voltage critical")
            elif voltage < 50:
                score -= 15
                risks.append("battery voltage low")

        # bateria -- tendência (queda ao longo da série)
        drop = features.get("battery_voltage_drop")
        if drop is not None and drop > 1:
            score -= 20
            risks.append("battery voltage decreasing")

        # temperatura
        temperature = features.get("temperature_max")
        if temperature is not None:
            checks_run += 1
            if temperature > 40:
                score -= 20
                risks.append("high temperature")

        # disponibilidade
        availability = features.get("availability_percent")
        if availability is not None:
            checks_run += 1
            if availability < 95:
                score -= 15
                risks.append("low availability")

        score = max(0, min(score, 100))

        if checks_run == 0:
            # Nenhuma das métricas que este modelo sabe avaliar está
            # presente -- não inventamos uma saúde "boa" nem "crítica"
            # por omissão. score=0 aqui NÃO significa "crítico": olhar
            # sempre para "status" e "explanation", nunca só para o
            # score, quando status="unknown".
            return {
                "score": 0,
                "status": "unknown",
                "explanation": "dados insuficientes para calcular saúde da torre",
            }

        if score >= 80:
            status = "healthy"
        elif score >= 50:
            status = "warning"
        else:
            status = "critical"

        explanation = (
            "; ".join(risks) if risks else "sem riscos identificados"
        )

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
