# Relatórios por Persona + Power BI — AIP

## Objetivo

Enviar automaticamente **relatórios semanais e mensais** adaptados a cada perfil
(O&M, Engenharia, Controller, Financeiro, Diretor Técnico, CEO, Conselho),
com Excel em anexo, via **Resend**, e permitir **perguntas interativas**
através do assistente existente (`POST /api/v1/assistant/ask`).

Integração com **Power BI** via endpoints JSON estáveis.

---

## Personas e conteúdo

| Persona | Env recipients | Foco do relatório | Detalhe Excel | Consumo vs norma | Ações prioritárias |
|---------|----------------|-------------------|---------------|------------------|--------------------|
| `om` | `REPORT_RECIPIENTS_OM` | Ação imediata | Sim | Não | Sim |
| `engenharia` | `REPORT_RECIPIENTS_ENGENHARIA` | Causa-raiz | Sim | Não | Sim |
| `controller` | `REPORT_RECIPIENTS_CONTROLLER` | Conformidade | Sim | Sim | Não |
| `financeiro` | `REPORT_RECIPIENTS_FINANCEIRO` | Custo / desvio | Resumo | Sim | Não |
| `diretor_tecnico` | `REPORT_RECIPIENTS_DIRETOR_TECNICO` | Prioridades cross-site | Sim | Sim | Sim |
| `ceo` | `REPORT_RECIPIENTS_CEO` | Risco de negócio | Top-N | Sim | Não |
| `conselho_administracao` | `REPORT_RECIPIENTS_CONSELHO_ADMINISTRACAO` | KPIs agregados | Top-N | Sim | Não |

Só personas com emails definidos recebem o relatório (as outras são omitidas).

---

## Job de envio

```bash
# Semanal (últimos 7 dias) — cron segunda 07:00
REPORT_PERIOD=weekly \
RESEND_API_KEY=re_xxx \
REPORT_RECIPIENTS_OM="om@empresa.com" \
REPORT_RECIPIENTS_CEO="ceo@empresa.com" \
go run ./cmd/report_job

# Mensal (últimos 30 dias) — cron dia 1 08:00
REPORT_PERIOD=monthly \
RESEND_API_KEY=re_xxx \
REPORT_RECIPIENTS_FINANCEIRO="fin@empresa.com" \
REPORT_RECIPIENTS_CONSELHO_ADMINISTRACAO="board@empresa.com" \
go run ./cmd/report_job
```

O antigo `cmd/weekly_report` continua a funcionar (compatibilidade); o novo
job é multi-persona e multi-período.

### Exemplo crontab

```cron
# Relatório semanal — segunda 07:00
0 7 * * 1  cd /opt/aip && REPORT_PERIOD=weekly ./bin/report_job

# Relatório mensal — dia 1 08:00
0 8 1 * *  cd /opt/aip && REPORT_PERIOD=monthly ./bin/report_job
```

---

## Interatividade (perguntas)

Cada email inclui a nota:

> Use `POST /api/v1/assistant/ask` com `"role": "<persona>"`.

Exemplos:

```json
{ "role": "om", "question": "quais sites estão pending_confirmation e o que fazer primeiro?" }
{ "role": "financeiro", "question": "quais torres tiveram maior desvio de consumo esta semana?" }
{ "role": "ceo", "question": "resumo executivo do risco da rede esta semana" }
```

O assistente usa as mesmas tools (incidentes, desvios, previsões) e aplica
grounding — não inventa números.

---

## Power BI

### Endpoints

| Método | Path | Uso |
|--------|------|-----|
| GET | `/api/v1/powerbi/incidents?from=YYYY-MM-DD&to=YYYY-MM-DD` | Tabela de factos de incidentes |
| GET | `/api/v1/powerbi/kpis?from=YYYY-MM-DD&to=YYYY-MM-DD` | Cartões (totais, pendentes, etc.) |

### Ligação no Power BI Desktop

1. **Obter dados → Web**
2. URL (exemplo últimos 30 dias):
   `http://<host>:8090/api/v1/powerbi/incidents?from=2026-07-01&to=2026-08-01`
3. Em produção: usar **On-premises data gateway** se a API for interna.
4. Agendar refresh no Power BI Service.

### Modelo recomendado (fases seguintes)

- Vista SQL `vw_powerbi_incidents` (opcional) para ligação direta Postgres
- Endpoint de desvios de consumo agregado
- Endpoint de previsões mais recentes por torre

---

## Ficheiros novos / alterados

```
cmd/report_job/main.go          # job multi-persona (novo)
internal/reports/personas.go    # catálogo de personas
internal/reports/period_report.go  # HTML + Excel por persona
internal/api/handlers/powerbi_handler.go
internal/api/routes/routers.go  # + rotas powerbi
cmd/api/main.go                 # wire PowerBIHandler
docs/relatorios_personas.md     # este documento
```

---

## Próximos passos naturais

1. Adicionar desvios de consumo agregados aos endpoints Power BI
2. Gerar resumo executivo com Ollama (1 parágrafo) no HTML do CEO/Conselho
3. Dashboard Power BI partilhado + RLS por persona (se necessário)
4. Alertas Teams com link “pedir detalhe ao assistente”
