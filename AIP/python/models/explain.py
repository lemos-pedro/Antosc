"""
Atribuição de importância a features (explicabilidade).

Para modelos baseados em regras / física: contribuições aditivas exactas
(o "SHAP" exacto do modelo de regras).

Para IsolationForest / árvores: tenta shap.TreeExplainer se a lib `shap`
estiver instalada; senão, fallback por desvio face a valores de referência
de domínio (sempre há explicação, nunca falha o predict).
"""

from __future__ import annotations

from typing import Any


def top_contributions(
    contributions: dict[str, float],
    k: int = 5,
) -> list[dict[str, Any]]:
    """Ordena por |contrib| descendente e devolve lista estável para JSON."""
    items = [
        {"feature": name, "contribution": round(float(val), 3)}
        for name, val in contributions.items()
        if val is not None
    ]
    items.sort(key=lambda x: abs(x["contribution"]), reverse=True)
    return items[:k]


def format_contributions_pt(contributions: list[dict[str, Any]]) -> str:
    if not contributions:
        return ""
    parts = []
    for c in contributions:
        sign = "+" if c["contribution"] > 0 else ""
        parts.append(f"{c['feature']} ({sign}{c['contribution']})")
    return "Principais factores: " + "; ".join(parts)


def domain_reference_contributions(features: dict[str, float]) -> dict[str, float]:
    """
    Fallback quando não há SHAP de modelo: quanto cada feature se afasta
    do 'normal' operacional (valores de referência do domínio telecom).
    Contribuição positiva = piora o risco; negativa = está saudável.
    """
    refs = {
        "battery_voltage_avg": (51.0, -1.0),
        "battery_voltage_min": (50.0, -1.0),
        "battery_voltage_drop": (0.3, 1.0),
        "battery_voltage_drop_short": (0.3, 1.0),
        "battery_voltage_slope_6": (-0.02, -1.0),  # slope negativo (tensão a cair) piora
        "temperature_max": (35.0, 1.0),
        "temperature_variance": (5.0, 1.0),
        "temperature_slope_6": (0.2, 1.0),
        "availability_percent": (99.0, -1.0),
        "generator_runtime_growth": (8.0, 1.0),
        "generator_runtime_slope_6": (0.5, 1.0),
    }
    out: dict[str, float] = {}
    for feat, (ref, direction) in refs.items():
        if feat not in features:
            continue
        delta = float(features[feat]) - ref
        # normaliza para escala ~pontos de score
        out[feat] = round(delta * direction * 5.0, 3)
    return out


def try_shap_tree(model, feature_vector, feature_names: list[str]) -> dict[str, float] | None:
    """Tenta TreeExplainer; devolve None se shap não estiver disponível."""
    try:
        import shap  # type: ignore
        import numpy as np
    except Exception:
        return None
    try:
        explainer = shap.TreeExplainer(model)
        sv = explainer.shap_values(feature_vector)
        if isinstance(sv, list):
            sv = sv[0]
        values = np.array(sv).reshape(-1)
        return {
            name: float(values[i])
            for i, name in enumerate(feature_names)
            if i < len(values)
        }
    except Exception:
        return None
