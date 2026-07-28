"""
Orquestrador genérico de treino: carrega dados, treina, avalia
superficialmente, guarda. Os scripts train_*.py (um por modelo) chamam isto
em vez de duplicarem o mesmo esqueleto cada um.
"""

from sklearn.ensemble import IsolationForest

from .dataset import load_features_csv, to_training_matrix
from .model_io import save_model


def train_isolation_forest(
    csv_path: str,
    output_filename: str,
    contamination: float = 0.05,
    random_state: int = 42,
):
    """
    contamination: proporção esperada de leituras anómalas no histórico
    (0.05 = assume-se que ~5% das leituras históricas já eram anómalas).
    Ajusta conforme fores validando com o O&M quantos alertas fazem sentido
    -- é o parâmetro mais importante para controlar falsos positivos.
    """
    df = load_features_csv(csv_path)
    X = to_training_matrix(df)

    print(f"a treinar Isolation Forest com {len(X)} amostras, {X.shape[1]} features")

    model = IsolationForest(
        contamination=contamination,
        random_state=random_state,
        n_estimators=200,
    )
    model.fit(X)

    n_flagged = int((model.predict(X) == -1).sum())
    print(f"no próprio conjunto de treino, {n_flagged}/{len(X)} amostras seriam sinalizadas como anomalia")

    return save_model(model, output_filename)
