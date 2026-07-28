from pathlib import Path

import joblib

from .base import BaseModel


MODEL_FILE = (
    Path(__file__).parent /
    "health_score.joblib"
)


class HealthScoreModel(BaseModel):

    name = "health_score"

    version = "1.0.0"

    def __init__(self):

        self.model = joblib.load(
            MODEL_FILE
        )

    def predict(
        self,
        features: dict[str, float],
    ) -> dict:

        return self.model.predict(
            features
        )

    def metadata(self):

        return {
            "name": self.name,
            "version": self.version,
        }