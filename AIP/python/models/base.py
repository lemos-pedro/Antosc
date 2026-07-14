"""
Classe base para todos os modelos do AIP.
"""

from abc import ABC, abstractmethod


class BaseModel(ABC):

    name: str

    version: str

    @abstractmethod
    def predict(
        self,
        features: dict[str, float],
    ) -> dict:
        """
        Executa a inferência.
        """
        raise NotImplementedError

    @abstractmethod
    def metadata(self) -> dict:
        """
        Informação do modelo.
        """
        raise NotImplementedError