# Anomaly model v2 (features temporais + versionamento)

## Vetor de features

`FEATURE_ORDER` em `models/anomaly.py` (versão **2.0.0**) inclui:

- agregados clássicos (tensão, temperatura, gerador, disponibilidade)
- `battery_voltage_drop_short`
- lags / rolling / slope / EWMA de bateria, temperatura e gerador

**21 features** no total. O `.joblib` antigo (10 features) **não é compatível** — é obrigatório re-treinar.

## Treino

```bash
# Bootstrap (sintético) — primeiro deploy / dev
make retrain

# Com histórico real
export ANOMALY_TRAIN_CSV=/data/features_wide.csv
make retrain
```

CSV largo: colunas = `tower_id,timestamp` + todas as de `FEATURE_ORDER`.

A partir de `ai_features` (longo):

```bash
# 1) export SQL → features_long.csv
# 2) pivot
cd python && PYTHONPATH=. python3 scripts_pivot_features.py features_long.csv features_wide.csv
```

## Versionamento

Cada `save_model` grava sidecar:

```
python/models/anomaly.joblib
python/models/anomaly.meta.json   # version, feature_names, metrics, saved_at
```

## Inferência

Reiniciar uvicorn/inference após re-treino para libertar o singleton em `registry.py`.
