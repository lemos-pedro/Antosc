"""
AIP Inference Service

Responsabilidades deste ficheiro:

    • Expor a API HTTP (FastAPI)
    • Validar pedidos
    • Agrupar a série recebida por categoria canónica (bateria, temperatura...)
    • Selecionar o adaptador do fabricante
    • Agregar a série em features (média, mínimo, queda, etc via feature_engineering)
    • Selecionar o modelo
    • Executar a inferência
    • Devolver a resposta

Toda a lógica específica de fabricantes e modelos fica fora deste ficheiro.

NOTA IMPORTANTE (corrigido em relação à v1): o pipeline anterior só
convertia o último valor de cada métrica (vendor.normalize), perdendo toda
a informação de tendência -- o que inutilizava os modelos anomaly/forecast,
que dependem de agregados como battery_voltage_drop. Agora a série completa
é agrupada por categoria e passada ao feature_engineering antes do modelo.
"""

from collections import defaultdict

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
    predicted_failure_window_days: int | None = None
    confidence: float | None = None


# ---------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------

def group_by_canonical_category(
    series: list[FeaturePoint],
    raw_to_canonical: dict[str, str],
) -> dict[str, list[float]]:
    """
    Converte a série bruta (lista de pontos com nome de métrica do
    fabricante) em listas agrupadas por categoria canónica, na ordem
    cronológica em que chegaram -- é isto que o feature_engineering espera
    (ex: {"battery_voltage": [51.2, 51.0, 50.8, ...]}).

    Pontos cujo nome não está mapeado em raw_to_canonical são ignorados --
    não é erro, só significa que essa métrica não tem uso conhecido ainda.
    """
    grouped: dict[str, list[float]] = defaultdict(list)

    for point in series:
        canonical = raw_to_canonical.get(point.name)
        if canonical is None:
            continue
        grouped[canonical].append(point.value)

    return dict(grouped)


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

        vendor = get_vendor(req.vendor)

        raw_to_canonical = getattr(vendor, "RAW_TO_CANONICAL", {})
        if not raw_to_canonical:
            raise ValueError(
                f"vendor '{req.vendor}' sem RAW_TO_CANONICAL definido -- "
                "não é possível agregar a série para este fabricante"
            )

        grouped = group_by_canonical_category(req.series, raw_to_canonical)

        canonical_features = build_features(grouped)

        model = get_model(req.model)

        result = model.predict(canonical_features)

        return PredictResponse(
            score=result["score"],
            status=result["status"],
            explanation=result["explanation"],
            predicted_failure_window_days=result.get("predicted_failure_window_days"),
            confidence=result.get("confidence"),
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
