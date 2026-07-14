"""
Interface base para adaptadores de fabricantes.

Cada fabricante converte os NOMES BRUTOS das métricas (como chegam do
towercore, ex. "battery_voltage_v") para categorias canónicas usadas
pelo feature_engineering (ex. "battery_voltage").

Entrada e saída são séries temporais agrupadas por nome
(dict[str, list[float]]), não valores escalares -- o feature_engineering
precisa da série completa (para calcular média, mínimo, máximo, queda),
não só do último valor.
"""

from abc import ABC, abstractmethod


class BaseVendor(ABC):

    name: str

    @abstractmethod
    def normalize(
        self,
        raw_series: dict[str, list[float]],
    ) -> dict[str, list[float]]:
        """
        Converte séries de métricas específicas do fabricante para
        séries de categorias canónicas.
        """
        raise NotImplementedError
