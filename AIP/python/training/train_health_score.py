"""
Treino do primeiro modelo de saúde da torre.

Fase inicial:
- modelo baseado em regras
- sem ML
- serve como baseline

Posteriormente será substituído por:
- XGBoost
- Random Forest
- modelos temporais
"""


import joblib
from pathlib import Path


MODEL_PATH = (
    Path(__file__)
    .parent
    .parent
    / "models"
    / "health_score.joblib"
)



class HealthScoreModel:


    def predict(self, features:dict)->dict:


        score = 100


        risks = []


        # bateria

        voltage = features.get(
            "battery_voltage_avg",
            0
        )


        if voltage < 48:

            score -= 30

            risks.append(
                "battery voltage critical"
            )


        elif voltage < 50:

            score -= 15

            risks.append(
                "battery voltage low"
            )

        drop = features.get(
            "battery_voltage_drop",
            0
        )

        if drop > 1:

            score -= 20

            risks.append(
                "battery voltage decreasing"
            )

        # temperatura

        temperature = features.get(
            "temperature_max",
            0
        )

        if temperature > 40:

            score -= 20

            risks.append(
                "high temperature"
            )

        # disponibilidade

        availability = features.get(
            "availability_percent",
            100
        )

        if availability < 95:

            score -= 15

            risks.append(
                "low availability"
            )

        score=max(
            0,
            min(score,100)
        )

        if score >= 80:

            status="healthy"

        elif score >= 50:

            status="warning"

        else:

            status="critical"

        return {

            "score":score,

            "status":status,

            "risks":risks,

        }


def train():

    model=HealthScoreModel()

    MODEL_PATH.parent.mkdir(
        exist_ok=True
    )

    joblib.dump(
        model,
        MODEL_PATH
    )

    print(
        f"model saved: {MODEL_PATH}"
    )

if __name__=="__main__":

    train()