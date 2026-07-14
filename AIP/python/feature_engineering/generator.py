"""
Feature engineering do gerador.

Usado para identificar:
- excesso de funcionamento
- muitas partidas
- dependência da rede elétrica
"""


from statistics import mean



def average_runtime(
    hours:list[float],
)->float:


    if not hours:
        return 0


    return mean(hours)



def runtime_growth(
    hours:list[float],
)->float:


    if len(hours)<2:
        return 0


    return hours[-1] - hours[0]



def build_generator_features(
    hours:list[float],
)->dict:


    return {

        "generator_runtime_avg":
            average_runtime(hours),


        "generator_runtime_growth":
            runtime_growth(hours),

    }