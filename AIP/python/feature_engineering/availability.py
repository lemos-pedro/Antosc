"""
Feature engineering de disponibilidade.

Mede qualidade operacional da torre.
"""


def availability_score(
    total_points:int,
    failed_points:int,
) -> float:


    if total_points == 0:
        return 0


    return (
        (total_points - failed_points)
        /
        total_points
    ) * 100



def build_availability_features(
    total_points:int,
    failed_points:int,
)->dict:


    return {

        "availability_percent":
            availability_score(
                total_points,
                failed_points,
            )

    }