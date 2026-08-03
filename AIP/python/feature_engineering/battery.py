"""
Feature engineering para bateria.

Recebe uma série temporal:

[
 {
   timestamp: "...",
   value: 51.2
 }
]

e gera features usadas pelos modelos IA.
"""

from statistics import mean


def average_voltage(values: list[float]) -> float:
    """
    Média da tensão da bateria.
    """
    if not values:
        return 0

    return mean(values)



def minimum_voltage(values: list[float]) -> float:
    """
    Menor tensão observada.
    """
    if not values:
        return 0

    return min(values)



def maximum_voltage(values: list[float]) -> float:
    """
    Maior tensão observada.
    """
    if not values:
        return 0

    return max(values)



def voltage_drop(values: list[float]) -> float:
    """
    Diferença entre primeiro e último valor.

    Positivo = queda de tensão.
    """

    if len(values) < 2:
        return 0


    return values[0] - values[-1]



def short_window_drop(values: list[float], window: int = 6) -> float:
    """Queda só na janela recente (mais sensível a degradação actual)."""
    if len(values) < 2:
        return 0.0
    w = values[-window:] if len(values) >= window else values
    if len(w) < 2:
        return 0.0
    return float(w[0] - w[-1])


def build_battery_features(values: list[float]) -> dict:

    return {
        "battery_voltage_avg": average_voltage(values),
        "battery_voltage_min": minimum_voltage(values),
        "battery_voltage_max": maximum_voltage(values),
        "battery_voltage_drop": voltage_drop(values),
        "battery_voltage_drop_short": short_window_drop(values, 6),
    }
