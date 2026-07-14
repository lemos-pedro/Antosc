"""
Feature engineering de temperatura.

Temperatura elevada acelera degradação
de baterias e equipamentos.
"""

from statistics import mean, variance



def average_temperature(values: list[float]) -> float:

    if not values:
        return 0

    return mean(values)



def maximum_temperature(values: list[float]) -> float:

    if not values:
        return 0

    return max(values)



def temperature_variation(values: list[float]) -> float:

    if len(values) < 2:
        return 0

    return variance(values)



def build_temperature_features(
    values: list[float],
) -> dict:


    return {

        "temperature_avg":
            average_temperature(values),

        "temperature_max":
            maximum_temperature(values),

        "temperature_variance":
            temperature_variation(values),

    }