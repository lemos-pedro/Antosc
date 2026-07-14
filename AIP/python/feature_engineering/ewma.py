"""
Baseline estatístico -- EWMA (Exponentially Weighted Moving Average).

Primeiro passo do roadmap de IA, antes de qualquer modelo ML:
"estatístico (EWMA/regressão) antes de qualquer modelo XGBoost".

Ideia: em vez de comparar o último valor com limiares fixos (o que o
health_score baseado em regras já faz), calcula-se uma média móvel
exponencial da série e mede-se o desvio do último valor face a essa
média, normalizado pelo desvio-padrão da série -- um "z-score" simples.
Isto apanha comportamento anómalo relativo ao histórico da própria
torre, mesmo quando o valor absoluto ainda não cruzou um limiar rígido
(ex.: uma torre que opera normalmente a 46V e cai para 44V é um desvio
relevante, mesmo que 44V não dispare um limiar absoluto pensado para
outras torres).
"""

from statistics import mean, pstdev


def ewma(values: list[float], alpha: float = 0.3) -> list[float]:
    """
    Média móvel exponencial ponto a ponto.

    alpha mais alto -> reage mais depressa a mudanças recentes.
    alpha mais baixo -> suaviza mais, mais lento a reagir.
    """
    if not values:
        return []

    result = [values[0]]
    for v in values[1:]:
        result.append(alpha * v + (1 - alpha) * result[-1])

    return result


def ewma_deviation(values: list[float], alpha: float = 0.3) -> float:
    """
    Desvio do último valor face à EWMA da série, normalizado pelo
    desvio-padrão populacional da série (z-score simples).

    Retorna 0 quando não há série suficiente para ter variância
    (não inventamos um desvio sem base estatística).
    """
    if len(values) < 3:
        return 0.0

    curve = ewma(values, alpha)
    baseline = curve[-2]  # EWMA até ao penúltimo ponto, exclui o próprio último valor
    latest = values[-1]

    spread = pstdev(values)
    if spread == 0:
        return 0.0

    return (latest - baseline) / spread


def build_ewma_features(name: str, values: list[float], alpha: float = 0.3) -> dict:
    """
    Produz as features EWMA para uma série canónica (ex. name="battery_voltage"):

        battery_voltage_ewma            -- valor suavizado mais recente
        battery_voltage_ewma_deviation  -- desvio (z-score) do último ponto
    """
    if not values:
        return {}

    curve = ewma(values, alpha)

    return {
        f"{name}_ewma": curve[-1],
        f"{name}_ewma_deviation": ewma_deviation(values, alpha),
    }
