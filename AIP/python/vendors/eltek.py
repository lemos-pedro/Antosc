"""
Adaptador Eltek -- recalibrado contra for_Data.csv real (5 torres,
A111/A112/A114/A119/A122). Os nomes brutos assumidos inicialmente
(battery_voltage_v, battery_temperature_c) NÃO existem neste histórico
-- os reais são os abaixo. Confirmado por inspeção direta do CSV, não
assumido.

Métricas reais observadas (fonte="eltek"):
    I1DC, I2DC, I3DC, I4DC  -- corrente de saída por módulo retificador
    BatTemperatura          -- temperatura da bateria (°C)
    Wh1..Wh4                -- contadores de energia acumulada (Wh),
                               monotonicamente crescentes -- úteis para
                               faturação/reconciliação de energia, não
                               para saúde da torre diretamente
                               (ver ANTOSC_PRE-FATURAÇÃO_ENERGIA)

NOTA: não existe leitura de tensão de bateria neste histórico. O
health_score deve tratar "battery_voltage_avg" em falta como "sem dados"
e não como "bateria crítica" -- ver models/health_score.py.
"""

from .base import BaseVendor


def _sum_channels(*channels: list[float]) -> list[float]:
    """
    Soma elemento-a-elemento várias séries de canais (ex. corrente por
    módulo retificador), assumindo que estão alinhadas no tempo (mesma
    cardinalidade e ordem -- válido aqui porque vêm do mesmo ciclo de
    leitura do towercore/CSV, jÃ¡ ordenadas por timestamp).

    Se os comprimentos não coincidirem, devolve [] em vez de somar
    dados desalinhados às cegas (nunca fabricar dados).
    """
    non_empty = [c for c in channels if c]
    if not non_empty:
        return []

    length = len(non_empty[0])
    if any(len(c) != length for c in non_empty):
        return []

    return [sum(values) for values in zip(*non_empty)]


class EltekVendor(BaseVendor):

    name = "eltek"

    def normalize(
        self,
        raw_series: dict[str, list[float]],
    ) -> dict[str, list[float]]:

        return {
            "temperature":
                raw_series.get("BatTemperatura", []),

            "rectifier_current":
                _sum_channels(
                    raw_series.get("I1DC", []),
                    raw_series.get("I2DC", []),
                    raw_series.get("I3DC", []),
                    raw_series.get("I4DC", []),
                ),

            # Contadores de energia acumulada -- não consumidos pelos
            # modelos de saúde ainda; ficam disponíveis para uma futura
            # feature de reconciliação de faturação.
            "energy_wh_total":
                _sum_channels(
                    raw_series.get("Wh1", []),
                    raw_series.get("Wh2", []),
                    raw_series.get("Wh3", []),
                    raw_series.get("Wh4", []),
                ),
        }
