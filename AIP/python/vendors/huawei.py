from .base import BaseVendor


class HuaweiVendor(BaseVendor):

    name = "huawei"

    RAW_TO_CANONICAL = {
        "battery_voltage_v": "battery_voltage",
        "battery_temperature_c": "temperature",
    }

    def normalize(
        self,
        features: dict[str, float],
    ) -> dict[str, float]:

        return {

            "battery_voltage":
                features.get("dc_output_voltage", features.get("battery_voltage_v", 0)),

            "battery_temperature":
                features.get("battery_temperature_c", 0),

            "rectifier_current":
                features.get("rectifier_current_a", 0),

            "mains_voltage":
                features.get("mains_voltage_l1_v", 0),

        }
