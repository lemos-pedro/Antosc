"""
Agregador geral de features da torre.
"""


from .battery import build_battery_features
from .temperature import build_temperature_features
from .availability import build_availability_features
from .generator import build_generator_features



def build_features(data:dict)->dict:


    result={}


    if "battery_voltage" in data:

        result.update(
            build_battery_features(
                data["battery_voltage"]
            )
        )


    if "temperature" in data:

        result.update(
            build_temperature_features(
                data["temperature"]
            )
        )


    if "generator_runtime" in data:

        result.update(
            build_generator_features(
                data["generator_runtime"]
            )
        )


    if (
        "total_points" in data
        and
        "failed_points" in data
    ):

        result.update(
            build_availability_features(
                data["total_points"],
                data["failed_points"],
            )
        )


    return result