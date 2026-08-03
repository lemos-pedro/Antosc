# AIP — Antosc Intelligence Platform (completo)

Núcleo original + relatórios por persona, Power BI, assistente portfolio,
predict_batch e API key. **Compila e tem toolchain de ops.**

## Requisitos

- Go 1.22+
- PostgreSQL 14+
- (opcional) Ollama, TowerCore, serviço Python de inferência, Resend

## Setup

```bash
# 1) DB
make migrate

# 2) Auth — definir secret
# AIP_JWT_SECRET=pelo-menos-32-caracteres-aleatorios

# 3) Arrancar API e criar o primeiro admin
make run-api
curl -X POST localhost:8090/api/v1/auth/bootstrap \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@empresa.com","full_name":"Admin","password":"senha-forte-123"}'
```

Ver `docs/auth.md`.

## Setup (resto)

```bash
cp .env.example .env   # editar credenciais
make migrate           # aplica migrations 001–006
make seed              # admin + users por persona
make build             # binários em ./bin
make run-api           # terminal 1
make run-scheduler     # terminal 2
make predict           # preenche ai_predictions
make smoke             # valida endpoints (API a correr)
```

## Make targets

| Target | Função |
|--------|--------|
| `make build` | compila api, scheduler, report_job, predict_batch, weekly_report |
| `make migrate` | SQL em ordem (precisa `psql` + `AIP_DB_*`) |
| `make run-api` | API :8090 |
| `make run-scheduler` | ingestão TowerCore |
| `make predict` | batch de previsões |
| `make report-weekly` | emails semanais por persona |
| `make report-monthly` | emails mensais |
| `make smoke` | smoke test HTTP |
| `make seed` | cria admin + users por persona |
| `docs/STATUS_ROADMAP.md` | o que está feito vs. falta |
| `make vet` | go vet |

## Cron produção

```cron
0 */6 * * *  cd /opt/aip && . .env && ./bin/predict_batch
0 7 * * 1    cd /opt/aip && . .env && REPORT_PERIOD=weekly ./bin/report_job
0 8 1 * *    cd /opt/aip && . .env && REPORT_PERIOD=monthly ./bin/report_job
```

Ver `docs/runbook_producao.md`.

## Módulos

- **API** — predictions, norms, incidents, consumption, assistant, Power BI, exports
- **Scheduler** — ingestão 5 min + alertas Teams
- **report_job** — relatórios por persona (HTML + Excel + ranking risco + resumo Ollama)
- **predict_batch** — previsões para todas as torres
- **Python** — feature engineering, health_score, anomaly, forecast

## Power BI

`/api/v1/powerbi/{catalog,incidents,kpis,consumption,predictions,towers}`  
Header: `X-API-Key` se `AIP_API_KEY` estiver definido.

## Personas (email)

`REPORT_RECIPIENTS_OM`, `_ENGENHARIA`, `_CONTROLLER`, `_FINANCEIRO`,
`_DIRETOR_TECNICO`, `_CEO`, `_CONSELHO_ADMINISTRACAO`

## Validação desta build

```
go build ./...   # OK
make build       # binários em ./bin
```




curl -s -X POST localhost:8090/api/v1/auth/login -H "Content-Type: application/json" -d '{\"email\":\"admin@antosc.local\",\"password\":\"Trocar123!\"}'