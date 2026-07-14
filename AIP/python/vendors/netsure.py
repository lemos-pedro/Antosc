"""
Adaptador Netsure -- fonte de dados presente no for_Data.csv histórico
(4.863 leituras, bem menos que eltek/comap -- provavelmente cobre um
período ou site mais curto/específico).

Métricas reais observadas (fonte="netsure"):
    voltagem  -- valores tipo 53499, 53505... consistentes com tensão
                 de barramento DC (~53.5V, típico de float charge de
                 um sistema -48V) escalada por 1000 (millivolt).
                 ASSUNÇÃO A CONFIRMAR: não há documentação/validação de
                 campo deste fator de escala -- é inferido pela gama de
                 valores, ao mesmo nível de confiança que outros
                 fatores "a confirmar" no towercore (ex. fuel_percent
                 reg.54 a 85%). Não usar em decisão crítica sem
                 confirmar.
    Itotal    -- mesma lógica, valores tipo 19144 -> possivelmente mA
                 (÷1000 = ~19A). Mesma ressalva de confirmação.
    erro      -- texto (ex. "Rectifier Lost, its owner: Rect Group"),
                 não numérico -- fica de fora da normalização (é
                 descartado já no parsing do CSV, training/dataset.py,
                 por não converter para float). É informação valiosa
                 para eventos/alarmes, mas isso é um caso de uso
                 diferente (eventos, não features numéricas) e não é
                 tratado aqui.
"""

from .base import BaseVendor

_MILLI_SCALE = 1000.0


def _scale(values: list[float], factor: float) -> list[float]:
    return [v / factor for v in values]


class NetsureVendor(BaseVendor):

    name = "netsure"

    def normalize(
        self,
        raw_series: dict[str, list[float]],
    ) -> dict[str, list[float]]:

        return {
            "battery_voltage":
                _scale(raw_series.get("voltagem", []), _MILLI_SCALE),

            "battery_current":
                _scale(raw_series.get("Itotal", []), _MILLI_SCALE),
        }
