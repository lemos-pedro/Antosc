# Explicabilidade (SHAP-style) + Forecast v2

## Feature contributions

Todos os modelos devolvem `feature_contributions` no `/predict`:

```json
{
  "score": 45.0,
  "status": "warning",
  "explanation": "riscos: ... Principais factores: battery_voltage_drop (-20); temperature_max (-20)",
  "feature_contributions": [
    {"feature": "battery_voltage_drop", "contribution": -20.0},
    {"feature": "temperature_max", "contribution": -20.0}
  ],
  "confidence": 0.8
}
```

| Modelo | Método de atribuição |
|--------|----------------------|
| `health_score` | Contribuições **exactas** das regras (aditivas) |
| `forecast` v2 | Multi-sinal + aceleradores (temp, gerador, disponibilidade) |
| `anomaly` | TreeSHAP se `shap` instalado; senão desvio vs referência de domínio |

## Forecast v2

- Taxa de degradação de bateria × aceleradores (temperatura, gerador, availability)
- Estados: `critical_now`, `critical`, `at_risk`, `watch`, `stable`
- `predicted_failure_window_days` + confidence

## Go

O serviço Go acrescenta "Principais factores: ..." à `explanation` gravada em `ai_predictions` (visível em relatórios, Power BI e assistente).

## Opcional

```bash
pip install shap   # TreeSHAP no IsolationForest
```
