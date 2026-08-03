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
from models.prescribe import prescribe as build_prescriptions
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


class FeatureContribution(BaseModel):
    feature: str
    contribution: float


class Prescription(BaseModel):
    code: str
    title: str
    due_days: int
    priority: str
    reason: str


class PredictResponse(BaseModel):
    score: float
    status: str
    explanation: str
    predicted_failure_window_days: int | None = None
    confidence: float | None = None
    feature_contributions: list[FeatureContribution] | None = None
    prescriptions: list[Prescription] | None = None


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

        if req.model.lower() == "prescribe":
            result = {"score": 0, "status": "prescribe", "explanation": ""}
            model = None
        else:
            model = get_model(req.model)
            result = model.predict(canonical_features)

        # Prescrições: usa o resultado actual como health/forecast/anomaly conforme o modelo pedido
        kw = {}
        key = req.model.lower()
        if key == "forecast":
            kw["forecast"] = result
        elif key == "anomaly":
            kw["anomaly"] = result
        else:
            kw["health"] = result
        # Se o cliente pedir model=prescribe, agrega os três
        prescriptions = None
        if key == "prescribe":
            # corre os três modelos e funde
            bundle = {}
            for name in ("health_score", "forecast", "anomaly"):
                try:
                    bundle[name] = get_model(name).predict(canonical_features)
                except Exception:
                    pass
            prescriptions = build_prescriptions(
                canonical_features,
                health=bundle.get("health_score"),
                forecast=bundle.get("forecast"),
                anomaly=bundle.get("anomaly"),
            )
            # score/status resumo a partir do forecast se existir
            primary = bundle.get("forecast") or bundle.get("health_score") or result
            result = {
                "score": primary.get("score", 0),
                "status": primary.get("status", "unknown"),
                "explanation": "prescrições geradas a partir de health+forecast+anomaly",
                "predicted_failure_window_days": primary.get("predicted_failure_window_days"),
                "confidence": primary.get("confidence"),
                "feature_contributions": primary.get("feature_contributions"),
            }
        else:
            prescriptions = build_prescriptions(canonical_features, **kw)

        contribs = result.get("feature_contributions")
        return PredictResponse(
            score=result["score"],
            status=result["status"],
            explanation=result["explanation"],
            predicted_failure_window_days=result.get("predicted_failure_window_days"),
            confidence=result.get("confidence"),
            feature_contributions=contribs,
            prescriptions=prescriptions,
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
