"""
Interface base para adaptadores de fabricantes.

Cada fabricante converte os nomes das métricas para um
formato canónico utilizado pelos modelos de IA.
"""

from abc import ABC, abstractmethod


class BaseVendor(ABC):

    name: str

    @abstractmethod
    def normalize(
        self,
        features: dict[str, float],
    ) -> dict[str, float]:
        """
        Converte métricas específicas do fabricante para
        features canónicas.
        """
        raise NotImplementedError