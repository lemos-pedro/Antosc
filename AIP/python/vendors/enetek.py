from .base import BaseVendor


class EnetekVendor(BaseVendor):

    name = "enetek"

    def normalize(
        self,
        raw_series: dict[str, list[float]],
    ) -> dict[str, list[float]]:

        return {
            "battery_voltage":
                raw_series.get("battery_voltage_v", []),

            # Ver nota equivalente em vendors/eltek.py: proxy de
            # temperatura até existir fonte ambiente/equipamento própria.
            "temperature":
                raw_series.get("battery_temperature_c", []),

            "rectifier_current":
                raw_series.get("rectifier_current_a", []),

            "mains_voltage":
                raw_series.get("mains_voltage_l1_v", []),
        }
