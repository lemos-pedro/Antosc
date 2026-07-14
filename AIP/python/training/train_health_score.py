"""
Treino do Health Score -- V1.

A V1 é baseada em regras (sem parâmetros aprendidos) e a implementação
vive em models/health_score.py -- não há "treino" nem artefacto joblib
para esta versão (ver o docstring desse ficheiro para o porquê).

Este ficheiro fica como o ponto de entrada previsto pelo roadmap:
"estatístico/EWMA/regressão antes de qualquer modelo ML" e, mais tarde,
"XGBoost sobre for_Data.csv". Quando isso acontecer, é aqui que entra o
treino real (fit + joblib.dump para models/health_score.joblib) e
models/health_score.py passa a fazer joblib.load em vez de reimplementar
as regras.
"""

if __name__ == "__main__":
    print(
        "Health Score V1 é baseado em regras (models/health_score.py); "
        "não há treino nem artefacto a gerar nesta fase. Ver docstring "
        "deste ficheiro para o plano de evolução para EWMA/XGBoost."
    )
