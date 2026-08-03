# Prescrições · Risk Monte Carlo · Embeddings · MLflow

## Prescrições

Modelo `prescribe` no serviço Python (ou qualquer predict devolve `prescriptions`):

```bash
# via inference
POST /predict  { "model": "prescribe", "tower_id", "vendor", "series": [...] }
```

Exemplos de acções: substituir bateria em X dias, inspecionar HVAC, analisar gerador…

O Go acrescenta "Prescrições: …" à `explanation` em `ai_predictions`.

## Risk portfolio (Monte Carlo)

```bash
GET /api/v1/risk/portfolio?horizon_days=30&sims=5000
Authorization: Bearer <token>
```

Resposta: `expected_failures`, `p50_failures`, `p90_failures`, `prob_at_least_one_failure`, …

Requisito: `make predict` / `predict_batch` com dados.

## Cross-tower embeddings (PCA)

```bash
./scripts/export_ai_features.sh ./data
ANOMALY_TRAIN_CSV=./data/features_wide.csv make embeddings
GET /api/v1/towers/{tower_id}/similar?k=5
```

Migration: `009_embeddings_risk.sql`.

## MLflow (opcional)

```bash
pip install mlflow
mlflow server --backend-store-uri sqlite:///mlflow.db --host 0.0.0.0 --port 5000
export MLFLOW_TRACKING_URI=http://localhost:5000
make retrain
```

Sem `MLFLOW_TRACKING_URI`, o treino ignora MLflow e mantém `.joblib` + `.meta.json`.
