"""
Feature engineering para combustível do gerador (ComAp: litros).

Uma queda súbita de combustível sem correspondência em horas de
funcionamento do gerador é um padrão típico de furto -- este módulo
não decide isso sozinho, só produz as features (avg/min/max/queda);
a deteção de anomalia fica a cargo do ewma_baseline (desvio da EWMA)
sobre "fuel_liters".
"""

from statistics import mean


def average_fuel(values: list[float]) -> float:
    if not values:
        return 0
    return mean(values)


def minimum_fuel(values: list[float]) -> float:
    if not values:
        return 0
    return min(values)


def maximum_fuel(values: list[float]) -> float:
    if not values:
        return 0
    return max(values)


def fuel_drop(values: list[float]) -> float:
    """Diferença entre primeiro e último valor da série. Positivo =
    consumo/perda de combustível ao longo do período."""
    if len(values) < 2:
        return 0
    return values[0] - values[-1]


def build_fuel_features(values: list[float]) -> dict:
    return {
        "fuel_liters_avg": average_fuel(values),
        "fuel_liters_min": minimum_fuel(values),
        "fuel_liters_max": maximum_fuel(values),
        "fuel_liters_drop": fuel_drop(values),
    }
