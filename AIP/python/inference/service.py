"""
AIP Inference Service

Responsabilidades deste ficheiro:

    • Expor a API HTTP (FastAPI)
    • Validar pedidos
    • Agrupar a série recebida por métrica (nome -> lista de valores,
      ordenada no tempo)
    • Selecionar o adaptador do fabricante e normalizar para categorias
      canónicas (ainda séries, não escalares)
    • Aplicar feature_engineering sobre as séries canónicas (médias,
      máximos, quedas, variância, etc. -- as features que os modelos
      realmente esperam)
    • Selecionar o modelo
    • Executar a inferência
    • Devolver a resposta

Toda a lógica específica de fabricantes, features e modelos fica fora
deste ficheiro.
"""

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

from vendors.registry import get_vendor
from models.registry import get_model
from feature_engineering.features import build_features


app = FastAPI(
    title="Antosc Intelligence Platform",
    version="1.0.0",
)


# ---------------------------------------------------------------------
# DTOs
# ---------------------------------------------------------------------

class FeaturePoint(BaseModel):
    name: str
    value: float
    timestamp: str


class PredictRequest(BaseModel):
    tower_id: str
    vendor: str
    model: str
    prediction_window_days: int
    series: list[FeaturePoint]


class PredictResponse(BaseModel):
    score: float
    status: str
    explanation: str


# ---------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------

def group_series(series: list[FeaturePoint]) -> dict[str, list[float]]:
    """
    Agrupa a série (múltiplos pontos no tempo, várias métricas
    misturadas) por nome de métrica, ordenada por timestamp:

        [
            {"name":"battery_voltage_v","value":51.2,"timestamp":"t1"},
            {"name":"battery_voltage_v","value":50.8,"timestamp":"t2"},
            {"name":"battery_temperature_c","value":29,"timestamp":"t1"},
        ]

    para

        {
            "battery_voltage_v": [51.2, 50.8],
            "battery_temperature_c": [29],
        }

    O feature_engineering precisa da série completa por métrica (para
    calcular média/mínimo/máximo/queda), não só do último valor.
    """

    ordered = sorted(series, key=lambda p: p.timestamp)

    grouped: dict[str, list[float]] = {}
    for point in ordered:
        grouped.setdefault(point.name, []).append(point.value)

    return grouped


# ---------------------------------------------------------------------
# API
# ---------------------------------------------------------------------

@app.get("/health")
def health():
    return {
        "status": "ok",
        "service": "aip-inference",
    }


@app.post(
    "/predict",
    response_model=PredictResponse,
)
def predict(req: PredictRequest):

    try:

        raw_series = group_series(req.series)

        vendor = get_vendor(req.vendor)

        canonical_series = vendor.normalize(raw_series)

        engineered_features = build_features(canonical_series)

        model = get_model(req.model)

        result = model.predict(engineered_features)

        return PredictResponse(
            score=result["score"],
            status=result["status"],
            explanation=result["explanation"],
        )

    except ValueError as exc:

        raise HTTPException(
            status_code=400,
            detail=str(exc),
        )

    except Exception as exc:

        raise HTTPException(
            status_code=500,
            detail=str(exc),
        )
