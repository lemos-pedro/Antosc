"""
Agregador geral de features da torre.

Inclui agregados de domínio (bateria, temperatura, …) e features
temporais (lags, rolling mean/std, slope, EWMA) quando a série tem
pontos suficientes.
"""

from __future__ import annotations

from .availability import build_availability_features
from .battery import build_battery_features
from .generator import build_generator_features
from .temperature import build_temperature_features
from .temporal import build_series_temporal


def build_features(data: dict) -> dict:
    result: dict = {}

    if "battery_voltage" in data:
        series = data["battery_voltage"]
        result.update(build_battery_features(series))
        result.update(build_series_temporal(series, "battery_voltage"))

    if "temperature" in data:
        series = data["temperature"]
        result.update(build_temperature_features(series))
        result.update(build_series_temporal(series, "temperature"))

    if "generator_runtime" in data:
        series = data["generator_runtime"]
        result.update(build_generator_features(series))
        result.update(build_series_temporal(series, "generator_runtime", short=6, long=12))

    if "total_points" in data and "failed_points" in data:
        result.update(
            build_availability_features(
                data["total_points"],
                data["failed_points"],
            )
        )

    return result
