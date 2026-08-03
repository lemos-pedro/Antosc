# AIP — Relatórios · Power BI · Assistente · Predict Batch (v4)

Pacote: **aip-relatorios-personas.zip**

## Nesta versão

| Novo | Para quê |
|------|----------|
| `cmd/predict_batch` | Preencher `ai_predictions` para todas as torres (cron) |
| `middleware.APIKey` | Proteger Power BI + exports com `AIP_API_KEY` |
| Tools portfolio | Assistente responde a perguntas de rede inteira |

## Porquê o predict_batch

Os endpoints `/powerbi/predictions` e a tool `risk_ranking` só têm dados
se existirem linhas em `ai_predictions`. Ninguém faz POST manual a ~200
sites. O batch fecha esse buraco operacional.

## Integração rápida

1. Extrair ZIP na raiz do repo AIP
2. Interfaces: `interfaces_patch.md` (`ListAll`, `ListLatestPerTower`)
3. Substituir `internal/tools/registry.go`
4. Copiar `internal/api/middleware/apikey.go`
5. Wire Power BI + rotas como em `cmd/api_main.go` / `routers.go`
6. Env:
   ```bash
   AIP_API_KEY=...                 # produção
   PREDICT_MODELS=health_score
   REPORT_RECIPIENTS_CEO=...
   RESEND_API_KEY=...
   ```
7. Cron:
   ```cron
   0 */6 * * *  predict_batch
   0 7 * * 1    REPORT_PERIOD=weekly report_job
   0 8 1 * *    REPORT_PERIOD=monthly report_job
   ```

## Testes smoke

```bash
curl -s localhost:8090/health
curl -s -H "X-API-Key: $AIP_API_KEY" localhost:8090/api/v1/powerbi/catalog
PREDICT_MODELS=health_score go run ./cmd/predict_batch
curl -s -H "X-API-Key: $AIP_API_KEY" localhost:8090/api/v1/powerbi/predictions | head
REPORT_PERIOD=weekly go run ./cmd/report_job
```

## Docs

- `docs/runbook_producao.md` — ops, cron, falhas
- `docs/powerbi.md` — ligação Desktop + modelo
- `docs/relatorios_personas.md` — emails por persona
