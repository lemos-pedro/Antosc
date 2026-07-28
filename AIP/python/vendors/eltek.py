from .base import BaseVendor


class EltekVendor(BaseVendor):

    name = "eltek"

    # Mapa nome bruto (como vem do towercore) -> categoria canónica usada
    # pelo feature_engineering (ver feature_engineering/features.py).
    # Chave é o nome bruto, valor é a categoria para onde a lista de
    # leituras vai ser agrupada antes de calcular médias/quedas/etc.
    RAW_TO_CANONICAL = {
        "battery_voltage_v": "battery_voltage",
        "battery_temperature_c": "temperature",
    }

    def normalize(
        self,
        features: dict[str, float],
    ) -> dict[str, float]:
        """Mantido para compatibilidade -- devolve um snapshot pontual.
        Para o pipeline real de previsão, ver RAW_TO_CANONICAL, usado por
        inference/service.py para agregar a série completa antes de chamar
        o modelo (normalize() sozinho perde informação de tendência)."""

        return {

            "battery_voltage":
                features.get("battery_voltage_v", 0),

            "battery_temperature":
                features.get("battery_temperature_c", 0),

            "rectifier_current":
                features.get("rectifier_1_output_a", 0),

            "mains_voltage":
                features.get("mains_voltage_l1_v", 0),

        }
