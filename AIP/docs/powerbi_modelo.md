# Porque não há .pbix binário no repo

O formato `.pbix` é proprietário (ZIP com modelo tabular interno da Microsoft).
Versionar um binário gerado noutro PC quebra com versões do Desktop e credenciais embutidas.
O procedimento abaixo produz o **mesmo resultado** no teu Power BI Desktop, com a tua API key.

# Modelo Power BI AIP — guia completo para construir o .pbix

Os endpoints REST já existem. Este documento é o **modelo dimensional + medidas DAX + páginas** para montares o ficheiro `.pbix` no Power BI Desktop (15–20 min).

> Não enviamos um `.pbix` binário daqui: o formato é proprietário e depende da tua versão do Desktop. Com estes passos o resultado é idêntico e controlado por ti.

## 0. Pré-requisitos

- API AIP a correr (`make run-api`)
- `AIP_API_KEY` ou JWT de um user
- Power BI Desktop

Header em todas as fontes Web (opções avançadas):

| Nome do cabeçalho | Valor |
|-------------------|--------|
| `X-API-Key` | o valor de `AIP_API_KEY` |

(ou `Authorization` = `Bearer <token>`)

Base: `http://<host>:8090` (produção: URL interna da API).

---

## 1. Importar as 5 tabelas

**Obter dados → Web** (uma a uma). Preferir **Transformar** e tipar colunas.

### 1.1 `towers` (dimensão)

```
http://HOST:8090/api/v1/powerbi/towers
```

| Coluna | Tipo |
|--------|------|
| tower_id | Texto |
| name | Texto |
| vendor | Texto |
| operator_id | Texto |
| region_id | Texto |
| availability_7d | Decimal |
| availability_30d | Decimal |

### 1.2 `incidents` (facto)

```
http://HOST:8090/api/v1/powerbi/incidents?from=2026-01-01&to=2026-12-31
```

Ajusta `from`/`to` ao período de análise (ou usa parâmetros PBI).

Tipos importantes: `incident_started_at`, `confirmed_at`, `created_at` → Data/Hora; `ml_confidence` → Decimal.

### 1.3 `consumption` (facto)

```
http://HOST:8090/api/v1/powerbi/consumption?from=2026-01-01&to=2026-12-31
```

### 1.4 `predictions` (facto / snapshot)

```
http://HOST:8090/api/v1/powerbi/predictions
```

### 1.5 `kpis` (cartões)

```
http://HOST:8090/api/v1/powerbi/kpis?from=2026-01-01&to=2026-12-31
```

Tabela com **1 linha** por refresh — ideal para cartões KPI.

---

## 2. Relações (vista Modelo)

```
towers[tower_id] 1 ── * incidents[tower_id]
towers[tower_id] 1 ── * consumption[tower_id]
towers[tower_id] 1 ── * predictions[tower_id]
```

- Cardinalidade: um-para-muitos  
- Direção do filtro: torre → factos (single)  
- `kpis` **sem** relação (tabela isolada)

---

## 3. Colunas calculadas úteis

Na tabela `incidents`:

```dax
Incident Month = DATE(YEAR(incidents[incident_started_at]), MONTH(incidents[incident_started_at]), 1)

Is Pending = incidents[status] = "pending_confirmation"

Is Corrected = incidents[status] = "corrected"

ML vs O&M Match =
IF(
    OR(ISBLANK(incidents[ml_predicted_cause]), ISBLANK(incidents[confirmed_cause])),
    BLANK(),
    incidents[ml_predicted_cause] = incidents[confirmed_cause]
)
```

Na tabela `predictions`:

```dax
Is At Risk =
predictions[status] IN { "critical", "critical_now", "at_risk", "anomaly", "warning" }

Risk Rank =
RANKX(ALL(predictions), predictions[score], , ASC, DENSE)
```

Na tabela `consumption`:

```dax
Is Breach = NOT consumption[within_norm]
```

---

## 4. Medidas DAX (tabela `_Medidas`)

Cria uma tabela vazia **Enter data** chamada `_Medidas` e cola:

```dax
Total Incidentes = COUNTROWS(incidents)

Pendentes Confirmacao = CALCULATE(COUNTROWS(incidents), incidents[status] = "pending_confirmation")

Confirmados = CALCULATE(COUNTROWS(incidents), incidents[status] = "confirmed")

Corrigidos = CALCULATE(COUNTROWS(incidents), incidents[status] = "corrected")

Sites Com Incidente = DISTINCTCOUNT(incidents[tower_id])

Taxa Confirmacao % =
DIVIDE([Confirmados] + [Corrigidos], [Total Incidentes])

Precisao ML % =
DIVIDE(
    CALCULATE(COUNTROWS(incidents), incidents[ML vs O&M Match] = TRUE()),
    CALCULATE(COUNTROWS(incidents), NOT ISBLANK(incidents[confirmed_cause]))
)

Sites Em Risco =
CALCULATE(DISTINCTCOUNT(predictions[tower_id]), predictions[Is At Risk] = TRUE())

Score Medio Saude = AVERAGE(predictions[score])

Breaches Consumo = CALCULATE(COUNTROWS(consumption), consumption[within_norm] = FALSE())

Desvio Medio % = AVERAGE(consumption[deviation_percent])

Disponibilidade 30d Media = AVERAGE(towers[availability_30d])
```

Cartões executivos podem usar também campos diretos de `kpis` (total_incidents, pending_confirmation, etc.).

---

## 5. Páginas sugeridas por persona

### Página «Executivo» (CEO / Conselho)
- Cartões: Total Incidentes, Pendentes, Sites Em Risco, Precisão ML %, Disponibilidade 30d
- Gráfico de barras: incidentes por mês (`Incident Month`)
- Tabela top 10 sites em risco (predictions ordenado por score ASC, filtro Is At Risk)

### Página «O&M»
- Tabela: pending_confirmation com tower_id, ml_predicted_cause, confidence
- Mapa ou tabela por region_id (via towers)
- Filtro: status = pending_confirmation

### Página «Engenharia»
- Dispersão: ml_confidence vs status
- Precisão ML por vendor (towers[vendor])
- Predictions com explanation

### Página «Controller / Financeiro»
- Breaches de consumo por equipment_type
- Desvio % médio por tower
- Filtro within_norm = false

### Página «Diretor Técnico»
- Ranking risco + incidentes 30d por site
- Disponibilidade 7d vs 30d

---

## 6. Parâmetros de data (recomendado)

1. **Gerir parâmetros** → `pFrom`, `pTo` (texto, yyyy-mm-dd)
2. Em cada query M de incidents/kpis/consumption, concatenar:

```powerquery
Base = "http://HOST:8090/api/v1/powerbi/incidents",
Url = Base & "?from=" & pFrom & "&to=" & pTo,
Fonte = Json.Document(Web.Contents(Url, [Headers=[#"X-API-Key"=apiKey]]))
```

Assim o refresh usa sempre o intervalo certo.

---

## 7. Refresh agendado (Power BI Service)

1. Publicar o `.pbix` no workspace  
2. Gateway on-premises se a API for só rede interna  
3. Credenciais da fonte: chave de API no cabeçalho  
4. Agendar refresh 2–4×/dia (alinhar com `predict_batch`)

---

## 8. Checklist de validação

- [ ] 5 tabelas carregam sem erro 401  
- [ ] Relações tower_id ativas  
- [ ] Cartões KPI mostram números (não blank)  
- [ ] Filtro por vendor/region funciona  
- [ ] predictions não vazia (senão correr `make predict`)  

Se predictions estiver vazia → `make predict` na API.  
Se incidents vazia → período `from`/`to` ou ainda sem causas geradas.
