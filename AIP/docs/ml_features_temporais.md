# Features temporais

`feature_engineering/temporal.py` acrescenta a cada série canónica:

| Feature | Significado |
|---------|-------------|
| `{prefix}_lag1` / `_lag3` | valor há 1 / 3 pontos |
| `{prefix}_roll_mean_{n}` | média móvel |
| `{prefix}_roll_std_{n}` | desvio móvel |
| `{prefix}_slope_{n}` | declive linear recente |
| `{prefix}_ewma` | média exponencial |
| `{prefix}_n_points` | tamanho da série |

Prefixos: `battery_voltage`, `temperature`, `generator_runtime`.

Bateria ganha também `battery_voltage_drop_short` (queda só nos últimos 6 pontos).

## Quem usa

- **forecast v2.1** — drop curto + slope se a série completa não mostrar queda  
- **health_score 1.2** — slope negativo ou drop curto conta como risco  
- **anomaly** — mantém `FEATURE_ORDER` antigo (compatível com `.joblib` actual); as features novas ficam disponíveis para o próximo re-treino alargado  

## Re-treino

```bash
make retrain
# cron sugerido (domingo 03:00)
# 0 3 * * 0  cd /opt/aip && ./scripts/retrain_models.sh
```
