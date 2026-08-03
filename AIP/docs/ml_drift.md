# Drift monitoring

Compara o CSV wide actual com um **baseline** de features.

```bash
# 1) definir baseline (após re-treino estável)
python -m training.drift --csv features_wide.csv --update-baseline

# 2) verificar periodicamente
make drift   # ou ANOMALY_TRAIN_CSV=features_wide.csv make drift
```

## Métricas

| Métrica | Limiar default | Significado |
|---------|----------------|-------------|
| Mean shift z | ≥ 2.0 | média actual afastou-se ≥2σ da baseline |
| PSI | ≥ 0.25 | mudança de distribuição (Population Stability Index) |

Exit code `1` = drift → considerar `make retrain`.

Cron sugerido (semanal, antes do re-treino):

```cron
0 2 * * 0  cd /opt/aip && ANOMALY_TRAIN_CSV=/data/features_wide.csv ./scripts/check_drift.sh
```
