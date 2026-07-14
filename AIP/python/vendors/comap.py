"""
Adaptador ComAp -- recalibrado contra for_Data.csv real.

Métricas reais observadas (fonte="comap"):
    litros            -- combustível no depósito (L)
    U1G, U2G, U3G      -- tensão de saída do gerador, 3 fases
    I1, I2, I3         -- corrente de saída do gerador, 3 fases
    U1R, U2R, U3R      -- tensão da rede elétrica (mains), 3 fases
    horas              -- horas de funcionamento acumuladas (maioria
                          das linhas vazia neste histórico -- 609 em
                          ~550k leituras -- pouco fiável por agora)
    kwh                -- energia (igualmente escasso/vazio)

Isto é diferente do profile.go do towercore (fuel_liters/fuel_percent/
battery_voltage/run_hours via Modbus, dados em produção) -- este CSV é
um export histórico mais antigo, com nomenclatura própria. Os dois não
se cruzam diretamente; mantemos os nomes tal como vêm no CSV.
"""

from .base import BaseVendor


def _avg_phases(*phases: list[float]) -> list[float]:
    """Média elemento-a-elemento entre fases (ex. tensão trifásica),
    assumindo séries alinhadas no tempo. Devolve [] se os comprimentos
    não coincidirem (nunca fabricar dados a partir de séries
    desalinhadas)."""
    non_empty = [p for p in phases if p]
    if not non_empty:
        return []

    length = len(non_empty[0])
    if any(len(p) != length for p in non_empty):
        return []

    return [sum(values) / len(values) for values in zip(*non_empty)]


class ComApVendor(BaseVendor):

    name = "comap"

    def normalize(
        self,
        raw_series: dict[str, list[float]],
    ) -> dict[str, list[float]]:

        return {
            "fuel_liters":
                raw_series.get("litros", []),

            "generator_runtime":
                raw_series.get("horas", []),

            "generator_voltage":
                _avg_phases(
                    raw_series.get("U1G", []),
                    raw_series.get("U2G", []),
                    raw_series.get("U3G", []),
                ),

            "generator_current":
                _avg_phases(
                    raw_series.get("I1", []),
                    raw_series.get("I2", []),
                    raw_series.get("I3", []),
                ),

            "mains_voltage":
                _avg_phases(
                    raw_series.get("U1R", []),
                    raw_series.get("U2R", []),
                    raw_series.get("U3R", []),
                ),
        }
