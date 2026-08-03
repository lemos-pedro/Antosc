# Ver também: **[powerbi_modelo.md](powerbi_modelo.md)** (DAX, páginas, M queries, checklist).

# Power BI — Guia de ligação ao AIP

## Endpoints (JSON tabular)

| Dataset | URL | Filtros | Personas que mais usam |
|---------|-----|---------|------------------------|
| Catálogo | `/api/v1/powerbi/catalog` | — | todos |
| Incidentes | `/api/v1/powerbi/incidents?from=&to=` | datas obrigatórias | O&M, Engenharia, Diretor Técnico |
| KPIs | `/api/v1/powerbi/kpis?from=&to=` | datas | CEO, Conselho, Diretor Técnico |
| Consumo | `/api/v1/powerbi/consumption?from=&to=` | datas | Financeiro, Controller |
| Previsões | `/api/v1/powerbi/predictions` | snapshot atual | Engenharia, Diretor Técnico, O&M |
| Torres | `/api/v1/powerbi/towers` | — | dimensão para joins |

Base URL exemplo: `http://localhost:8090`

## Ligação no Power BI Desktop

1. **Obter dados → Web**
2. Colar URL completa, por exemplo:
   ```
   http://localhost:8090/api/v1/powerbi/incidents?from=2026-07-01&to=2026-08-01
   ```
3. Em **Transformar dados**, converter tipos (datas → DateTime, números → Decimal).
4. Repetir para cada dataset (consumption, predictions, towers, kpis).
5. Criar relações:
   - `incidents[tower_id]` → `towers[tower_id]`
   - `consumption[tower_id]` → `towers[tower_id]`
   - `predictions[tower_id]` → `towers[tower_id]`

## Modelo sugerido (star schema simples)

```
towers (dimensão)
  ↑
  ├── incidents (facto)
  ├── consumption (facto)
  └── predictions (facto)

kpis (tabela isolada para cartões — 1 linha por refresh)
```

## Visualizações recomendadas por persona

**CEO / Conselho**
- Cartões: total incidentes, sites afetados, pendentes O&M, fora de norma
- Linha temporal de incidentes (se acumular histórico de refreshes)

**Financeiro / Controller**
- Tabela: tower_id, equipment, desvio %, within_norm
- Barras: top desvios positivos/negativos
- Filtro: within_norm = false

**O&M / Engenharia**
- Tabela incidentes com status e causa ML vs confirmada
- Previsões com score baixo / status critical|at_risk|anomaly
- Mapa ou matriz por vendor/região (via towers)

**Diretor Técnico**
- Ranking de risco (predictions.score ASC)
- Incidentes por vendor (join towers)
- Tendência de pending_confirmation

## Produção

- API interna → **On-premises data gateway** no Power BI Service
- Agendar refresh (ex: de hora a hora ou 1×/dia)
- Não expor `/api/v1/powerbi/*` publicamente sem autenticação

## Autenticação (fase seguinte)

Hoje os endpoints são abertos como o resto da API de leitura.
Quando o middleware de auth do AIP estiver ativo, o gateway usa
credenciais de serviço (Basic ou token) nas ligações Web.
