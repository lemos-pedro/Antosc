"""
Features temporais genéricas: lags, rolling stats, slope.

Recebe uma série ordenada cronologicamente (mais antigo → mais recente)
como list[float]. Se a série for curta, devolve o que for possível
sem inventar valores (0 só quando não há pontos).
"""

from __future__ import annotations

from statistics import mean, pstdev


def _tail(values: list[float], n: int) -> list[float]:
    if n <= 0 or not values:
        return []
    return values[-n:] if len(values) >= n else list(values)


def lag(values: list[float], k: int) -> float:
    """Valor há k pontos (k=1 → penúltimo). 0 se série insuficiente."""
    if k <= 0 or len(values) <= k:
        return 0.0
    return float(values[-(k + 1)])


def rolling_mean(values: list[float], window: int) -> float:
    w = _tail(values, window)
    return float(mean(w)) if w else 0.0


def rolling_std(values: list[float], window: int) -> float:
    w = _tail(values, window)
    if len(w) < 2:
        return 0.0
    return float(pstdev(w))


def slope(values: list[float], window: int) -> float:
    """
    Declive linear OLS simples nos últimos `window` pontos.
    Positivo = tendência a subir; negativo = a descer.
    """
    w = _tail(values, window)
    n = len(w)
    if n < 2:
        return 0.0
    # x = 0..n-1
    x_mean = (n - 1) / 2.0
    y_mean = mean(w)
    num = 0.0
    den = 0.0
    for i, y in enumerate(w):
        dx = i - x_mean
        num += dx * (y - y_mean)
        den += dx * dx
    if den == 0:
        return 0.0
    return float(num / den)


def ewma_last(values: list[float], alpha: float = 0.3) -> float:
    """Último valor do EWMA (mais peso no recente)."""
    if not values:
        return 0.0
    s = float(values[0])
    for v in values[1:]:
        s = alpha * float(v) + (1 - alpha) * s
    return s


def fourier_hour(hour: float) -> dict[str, float]:
    """Componentes Fourier diários (hora 0-23)."""
    import math
    ang = 2 * math.pi * (hour % 24) / 24.0
    return {
        "tod_sin": math.sin(ang),
        "tod_cos": math.cos(ang),
    }


def build_series_temporal(
    values: list[float],
    prefix: str,
    short: int = 6,
    long: int = 24,
) -> dict[str, float]:
    """
    Gera um bloco de features temporais para uma série canónica.

    Exemplos de chaves:
      battery_voltage_lag1, battery_voltage_roll_mean_6,
      battery_voltage_roll_std_24, battery_voltage_slope_6,
      battery_voltage_ewma
    """
    if not values:
        return {}

    out = {
        f"{prefix}_lag1": lag(values, 1),
        f"{prefix}_lag3": lag(values, 3),
        f"{prefix}_roll_mean_{short}": rolling_mean(values, short),
        f"{prefix}_roll_mean_{long}": rolling_mean(values, long),
        f"{prefix}_roll_std_{short}": rolling_std(values, short),
        f"{prefix}_roll_std_{long}": rolling_std(values, long),
        f"{prefix}_slope_{short}": slope(values, short),
        f"{prefix}_slope_{long}": slope(values, long),
        f"{prefix}_ewma": ewma_last(values),
        f"{prefix}_n_points": float(len(values)),
    }
    # fase na série como proxy de sazonalidade diária (quando não há timestamp)
    phase_hour = float(len(values) % 24)
    for k, v in fourier_hour(phase_hour).items():
        out[f"{prefix}_{k}"] = v
    return out
