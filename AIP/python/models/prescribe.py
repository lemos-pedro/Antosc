"""
Camada prescritiva (regras) em cima de health / forecast / anomaly.

Devolve acções concretas com prazo em dias — auditável, sem inventar dados.
"""

from __future__ import annotations

from typing import Any


def prescribe(
    features: dict[str, float],
    *,
    health: dict[str, Any] | None = None,
    forecast: dict[str, Any] | None = None,
    anomaly: dict[str, Any] | None = None,
) -> list[dict[str, Any]]:
    actions: list[dict[str, Any]] = []
    seen: set[str] = set()

    def add(code: str, title: str, due_days: int, priority: str, reason: str):
        if code in seen:
            return
        seen.add(code)
        actions.append({
            "code": code,
            "title": title,
            "due_days": int(due_days),
            "priority": priority,  # critical | high | medium | low
            "reason": reason,
        })

    f = forecast or {}
    h = health or {}
    a = anomaly or {}

    window = f.get("predicted_failure_window_days")
    f_status = str(f.get("status") or "")
    h_status = str(h.get("status") or "")
    a_status = str(a.get("status") or "")

    drop = float(features.get("battery_voltage_drop", 0) or 0)
    drop_s = float(features.get("battery_voltage_drop_short", 0) or 0)
    slope = float(features.get("battery_voltage_slope_6", 0) or 0)
    vmin = float(features.get("battery_voltage_min", features.get("battery_voltage_avg", 99) or 99)
    )
    temp = float(features.get("temperature_max", 0) or 0)
    avail = float(features.get("availability_percent", 100) or 100)
    gen = float(features.get("generator_runtime_growth", 0) or 0)

    # --- bateria ---
    if f_status == "critical_now" or vmin <= 48:
        add(
            "battery_replace_immediate",
            "Substituir bateria imediatamente",
            0,
            "critical",
            "tensão no limiar crítico ou status critical_now",
        )
    elif window is not None and int(window) <= 7:
        add(
            "battery_replace_7d",
            "Substituir bateria",
            int(window),
            "critical",
            f"forecast estima falha em ~{int(window)} dia(s)",
        )
    elif window is not None and int(window) <= 30:
        add(
            "battery_replace_30d",
            "Planear substituição de bateria",
            int(window),
            "high",
            f"janela de falha estimada ~{int(window)} dias",
        )
    elif drop > 1 or drop_s > 0.8 or slope < -0.05:
        add(
            "battery_inspect",
            "Inspecionar banco de baterias e retificador",
            14,
            "high",
            "queda/slope de tensão acima do normal",
        )

    # --- temperatura ---
    if temp > 45:
        add(
            "hvac_urgent",
            "Verificar climatização / ventilação do shelter",
            2,
            "critical",
            f"temperature_max={temp}",
        )
    elif temp > 40:
        add(
            "hvac_check",
            "Inspecionar climatização",
            7,
            "high",
            f"temperature_max={temp}",
        )

    # --- gerador / rede ---
    if gen > 24:
        add(
            "generator_runtime",
            "Analisar dependência de gerador e estabilidade da rede",
            10,
            "medium",
            f"generator_runtime_growth={gen}",
        )

    # --- disponibilidade ---
    if avail < 95:
        add(
            "availability_rootcause",
            "Investigar causa de baixa disponibilidade",
            5,
            "high",
            f"availability_percent={avail}",
        )

    # --- anomaly genérica ---
    if a_status == "anomaly" and not actions:
        add(
            "anomaly_inspect",
            "Inspecionar site (padrão anómalo sem causa óbvia)",
            7,
            "medium",
            str(a.get("explanation") or "anomaly detectada"),
        )

    # --- health warning residual ---
    if h_status == "critical" and not any(x["priority"] == "critical" for x in actions):
        add(
            "health_critical_review",
            "Revisão prioritária do site (health critical)",
            3,
            "high",
            str(h.get("explanation") or "health_score critical"),
        )

    priority_rank = {"critical": 0, "high": 1, "medium": 2, "low": 3}
    actions.sort(key=lambda x: (priority_rank.get(x["priority"], 9), x["due_days"]))
    return actions
