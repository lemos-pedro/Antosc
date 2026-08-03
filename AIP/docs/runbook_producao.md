# Runbook de produção — Relatórios AIP + Power BI

## Componentes

| Processo | Binário / comando | Frequência | Depende de |
|----------|-------------------|------------|------------|
| API HTTP | `cmd/api` | sempre ligado | Postgres, Ollama (assistente), prediction service |
| Scheduler ingestão | `cmd/scheduler` | contínuo (ticker 5 min) | TowerCore, Postgres, Teams opcional |
| Relatório email | `cmd/report_job` | cron semanal + mensal | Postgres, Resend, Ollama (resumo CEO) |
| Inference Python | `uvicorn` em `python/inference` | sempre ligado | modelos .joblib |

## Variáveis críticas

```bash
# Base
AIP_DB_*  TOWERCORE_URL  AIP_PREDICTION_SERVICE_URL  AIP_BASE_URL

# Email
RESEND_API_KEY  RESEND_FROM
REPORT_RECIPIENTS_OM=
REPORT_RECIPIENTS_ENGENHARIA=
REPORT_RECIPIENTS_CONTROLLER=
REPORT_RECIPIENTS_FINANCEIRO=
REPORT_RECIPIENTS_DIRETOR_TECNICO=
REPORT_RECIPIENTS_CEO=
REPORT_RECIPIENTS_CONSELHO_ADMINISTRACAO=

# Resumo executivo
OLLAMA_URL=http://127.0.0.1:11434
OLLAMA_MODEL=qwen2.5:3b

# Alertas tempo real (opcional)
TEAMS_WEBHOOK_URL=
```

Só personas com `REPORT_RECIPIENTS_*` preenchido recebem email.

## Cron recomendado

```cron
# Relatório semanal — segunda 07:00 (hora local do servidor)
0 7 * * 1  cd /opt/aip && set -a && . /opt/aip/.env && set +a && REPORT_PERIOD=weekly /opt/aip/bin/report_job >> /var/log/aip/report_weekly.log 2>&1

# Relatório mensal — dia 1 às 08:00
0 8 1 * *  cd /opt/aip && set -a && . /opt/aip/.env && set +a && REPORT_PERIOD=monthly /opt/aip/bin/report_job >> /var/log/aip/report_monthly.log 2>&1
```

## Checklist pós-deploy

1. `curl -s http://localhost:8090/health` → `"status":"ok"`
2. `curl -s http://localhost:8090/api/v1/powerbi/catalog` → 5 datasets
3. `REPORT_PERIOD=weekly go run ./cmd/report_job` (com recipients de teste) → email chega
4. Power BI Desktop liga a `/powerbi/incidents?from=...&to=...`
5. Assistente:
   ```bash
   curl -s -X POST http://localhost:8090/api/v1/assistant/ask \
     -H 'Content-Type: application/json' \
     -d '{"role":"ceo","question":"quais os KPIs da última semana? usa from=YYYY-MM-DD"}'
   ```

## Falhas comuns

| Sintoma | Causa provável | Ação |
|---------|----------------|------|
| Job sai com "nenhum relatório enviado" | nenhum `REPORT_RECIPIENTS_*` | definir pelo menos uma persona |
| "RESEND_API_KEY não definido" | env em falta no cron | carregar `.env` no crontab |
| Resumo CEO genérico / fallback | Ollama offline | `ollama serve` + modelo puxado |
| Power BI consumption vazio | sem normas ativas | criar normas via `POST /api/v1/norms` |
| predictions vazio | nunca correu `POST /predictions/{id}` | disparar previsões ou pipeline batch |
| Assistente "ferramenta desconhecida" | registry antigo | substituir `internal/tools/registry.go` |

## Ordem de arranque

1. Postgres (migrations 001–005 aplicadas)
2. TowerCore
3. Prediction service (Python)
4. Ollama
5. `cmd/api`
6. `cmd/scheduler`
7. Cron do `report_job`

## Segurança (fase seguinte)

- `/api/v1/powerbi/*` e exports devem ficar atrás do mesmo middleware de auth da API
- Gateway Power BI com conta de serviço, não URL pública aberta
- Rotacionar `RESEND_API_KEY` e credenciais DB periodicamente

## Job de previsões em lote (`predict_batch`)

Sem este job, `/api/v1/powerbi/predictions` e a tool `risk_ranking` ficam
vazios — as previsões só existem se alguém fizer POST manual por torre.

```cron
# A cada 6 horas
0 */6 * * *  cd /opt/aip && set -a && . /opt/aip/.env && set +a && PREDICT_MODELS=health_score,anomaly /opt/aip/bin/predict_batch >> /var/log/aip/predict_batch.log 2>&1
```

```bash
# Manual
PREDICT_MODELS=health_score,anomaly,forecast PREDICT_WINDOW_DAYS=7 go run ./cmd/predict_batch
```

Requisitos: cache `towers` preenchida (scheduler), features suficientes
(≥30 pontos), serviço Python de inferência no ar.

## API key (Power BI e exports)

```bash
export AIP_API_KEY="gera-um-segredo-longo"
```

No Power BI Desktop → fonte Web → opções avançadas → cabeçalho HTTP:
`X-API-Key` = o mesmo valor.

Se `AIP_API_KEY` estiver vazio, as rotas ficam abertas (dev local).
