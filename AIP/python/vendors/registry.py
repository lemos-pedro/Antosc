"""
Registo central de adaptadores de fabricantes.
"""

from .base import BaseVendor
from .eltek import EltekVendor
from .enetek import EnetekVendor
from .huawei import HuaweiVendor


VENDORS: dict[str, BaseVendor] = {
    "eltek": EltekVendor(),
    "enetek": EnetekVendor(),
    "huawei": HuaweiVendor(),
}


def get_vendor(name: str) -> BaseVendor:
    """
    Obtém o adaptador correspondente ao fabricante.
    """

    vendor = VENDORS.get(name.lower())

    if vendor is None:
        raise ValueError(
            f"Fabricante não suportado: {name}"
        )

    return vendor