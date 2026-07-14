# AIP — Antosc Intelligence Platform

Módulo de IA do Antosc System. Consome dados do `towercore` (só leitura),
gera features/eventos/previsões, e serve-os via API própria.

## O que mudou nesta refactorização

**Corrigido (impedia compilar):**
- `go.mod` não existia — criado (`github.com/antosc/aip`, Go 1.22).
- `internal/confi.go` estava em package `config` mas na pasta errada
  (`internal/` em vez de `internal/config/`) — fundido em `internal/config/config.go`.
- Pacote `postgres` vivia em `/postgres` mas era importado como
  `internal/repository/postgres` — movido para o path correto.
- `cmd/main.go` e `cmd/scheduler/main.go` eram duplicados a fazer o mesmo —
  ficou só `cmd/scheduler/main.go`.
- `ingestion.NewService(...)` tinha assinatura diferente da chamada em
  `main.go` (1 arg vs 3) — corrigido e alinhado, agora recebe logger também.
- `python/trainig` e `python/expriments` (erros ortográficos, duplicados de
  `train`/`experiments`) — removidos.

**Adicionado (não existia):**
- `internal/prediction/` — o contrato Go↔Python que faltava por completo.
  Go chama `POST {AIP_PREDICTION_SERVICE_URL}/predict`, Python
  (`python/inference/service.py`, FastAPI) responde com score/status/explicação.
- API HTTP real: `GET /health`, `GET /api/v1/predictions/{tower_id}`,
  `POST /api/v1/predictions/{tower_id}` — os handlers/routes/dto que
  estavam vazios no zip original.
- `internal/logger` — logging estruturado (JSON, `log/slog`) em vez de
  `log.Println` disperso.
- Batch inserts (`SaveBatch`) em features/eventos — um ciclo de ingestão
  grava centenas de linhas numa query, não uma a uma.
- Regra explícita "sem previsão com histórico insuficiente"
  (`ErrInsufficientHistory`, mínimo 30 pontos) — consistente com a
  restrição já definida para o módulo de IA.
- `towercore.ErrMetricsEndpointNotAvailable`: **correção (13/07):** este endpoint
  afinal existe — `GET /api/v1/metrics` (não estava documentado em `api.md`,
  só descoberto ao ver o handler real). Não segue o envelope
  `{"data":...,"meta":...}` do resto da API — devolve o array direto, total
  no header `X-Total-Count`. Cada métrica é um snapshot por torre com várias
  grandezas (`map[string]float64`), não uma métrica isolada. `GetMetrics`
  agora pagina de verdade e filtra por `from` (o scheduler só pede o que é
  novo desde o último ciclo, guardado em memória em `lastMetricsSince`).

## Validado neste ambiente

```
go mod tidy   # OK
go build ./... # OK, zero erros
go vet ./...  # limpo
gofmt -l .    # limpo (após gofmt -w)
```

## Estrutura

```
cmd/
  api/         # servidor HTTP (predictions, health)
  scheduler/   # loop de ingestão periódica (5 min) do towercore
internal/
  config/      # env vars, DSN
  logger/      # slog estruturado
  repository/
    postgres/  # ai_features, ai_events, ai_predictions
    towercore/ # client HTTP só-leitura para o towercore
  analytics/ingestion/  # orquestra a coleta e persistência
  prediction/           # contrato com o serviço Python de inferência
  api/
    handlers/ routes/ dto/
python/
  feature_engineering/  # Postgres -> datasets
  train/                # EWMA/regressão primeiro, XGBoost depois
  inference/service.py  # FastAPI, serve o modelo treinado
  evaluation/ models/ notebooks/ utils/
migrations/001_init.sql
```

## Próximo passo mais importante

`api.md` do towercore está desatualizado — não lista `GET/POST /api/v1/metrics`,
que já existe de facto no `MetricHandler`. Vale a pena atualizar o contrato
documentado para refletir o comportamento real (envelope diferente do resto
da API, paginação via `X-Total-Count`), para o próximo integrador não passar
pelo mesmo processo de descoberta.
