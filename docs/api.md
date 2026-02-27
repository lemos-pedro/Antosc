# API HTTP (Contrato v0)

Documento de contrato inicial para alinhar backend e consumidores durante a fase de planeamento.

## Convencoes
- Base path: `/api/v1`
- Formato: `application/json`
- Datas: ISO 8601 em UTC (`YYYY-MM-DDTHH:mm:ssZ`)
- Correlation id:
  - Request header: `X-Request-Id` (opcional)
  - Response header: `X-Request-Id` (ecoado ou gerado)
- Identificadores em path: `snake_case` no nome do parametro, UUID no valor.

## Regras gerais
- Paginação por `limit` e `offset`.
- `limit` default: `50`, maximo: `200`.
- Ordenacao padrao: `created_at desc` quando aplicavel.
- Campos desconhecidos em payload de escrita devem retornar `400`.

## Endpoints v0

### `GET /health`
Health check da aplicacao.

`200 OK`
```json
{
  "status": "ok",
  "service": "towercore-api",
  "time": "2026-02-26T10:00:00Z"
}
```

### `GET /towers`
Lista torres.

Query params:
- `status` (opcional): `online | degraded | offline`
- `operator_id` (opcional)
- `region_id` (opcional)
- `limit` (opcional)
- `offset` (opcional)

`200 OK`
```json
{
  "data": [
    {
      "tower_id": "1be55f39-6b6b-43f5-b2f1-fca90d5f7e14",
      "name": "TWR-LUANDA-001",
      "status": "degraded",
      "operator_id": "7e7f5ca9-3d53-4f60-a63c-e30e474ed8f9",
      "region_id": "d8c13b69-9167-4866-b2a5-7fc56c66dd17",
      "updated_at": "2026-02-26T09:50:00Z"
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "total": 1
  }
}
```

### `GET /towers/{tower_id}`
Detalhe da torre com estado atual e resumo de disponibilidade.

`200 OK`
```json
{
  "tower_id": "1be55f39-6b6b-43f5-b2f1-fca90d5f7e14",
  "name": "TWR-LUANDA-001",
  "status": "degraded",
  "availability_30d": 99.7,
  "operator_id": "7e7f5ca9-3d53-4f60-a63c-e30e474ed8f9",
  "region_id": "d8c13b69-9167-4866-b2a5-7fc56c66dd17",
  "updated_at": "2026-02-26T09:50:00Z"
}
```

`404 Not Found` quando a torre nao existir.

### `GET /towers/{tower_id}/events`
Historico de alarmes e falhas da torre.

Query params:
- `from` (opcional, ISO 8601)
- `to` (opcional, ISO 8601)
- `severity` (opcional): `info | warning | critical`
- `limit` (opcional)
- `offset` (opcional)

`200 OK`
```json
{
  "data": [
    {
      "event_id": "4e1d2ad6-9d69-4e5c-bf7d-5d2eb74b1663",
      "type": "failure",
      "severity": "critical",
      "message": "Power module unreachable",
      "occurred_at": "2026-02-26T08:40:00Z"
    }
  ],
  "meta": {
    "limit": 50,
    "offset": 0,
    "total": 1
  }
}
```

### `GET /sla/global`
SLA consolidado de todas as torres nos ultimos 30 dias.

`200 OK`
```json
{
  "window_days": 30,
  "availability_percent": 99.85,
  "affected_towers": 12
}
```

### `GET /operators`
Lista operadores.

Query params:
- `limit` (opcional)
- `offset` (opcional)

### `GET /operators/{operator_id}`
Detalhe de um operador.

### `POST /operators`
Cria operador.

Request:
```json
{
  "name": "Operator A",
  "code": "OP-A"
}
```

`201 Created`

### `PUT /operators/{operator_id}`
Atualiza operador.

Request:
```json
{
  "name": "Operator A Updated",
  "code": "OP-A"
}
```

`200 OK`

### `DELETE /operators/{operator_id}`
Remove operador.

`204 No Content`

### `GET /regions`
Lista regioes.

### `GET /regions/{region_id}`
Detalhe de regiao, incluindo metricas agregadas.

### `GET /tickets`
Lista tickets ordenados por data de criacao (desc).

Query params:
- `status` (opcional): `open | acknowledged | closed`
- `tower_id` (opcional)
- `limit` (opcional)
- `offset` (opcional)

### `POST /tickets/{ticket_id}/ack`
Confirma recebimento do ticket.

`200 OK`

### `POST /tickets/{ticket_id}/close`
Fecha ticket manualmente.

`200 OK`

### `POST /metrics`
Ingestao de metricas de equipamentos/towers.

Request:
```json
{
  "tower_id": "1be55f39-6b6b-43f5-b2f1-fca90d5f7e14",
  "collected_at": "2026-02-26T09:59:00Z",
  "metrics": {
    "signal_strength": -72,
    "voltage": 53.2
  }
}
```

`202 Accepted`

## Modelo de erro (padrao)

`4xx/5xx`
```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Tower not found",
    "request_id": "req-123"
  }
}
```

Codigos sugeridos:
- `VALIDATION_ERROR`
- `UNAUTHORIZED`
- `FORBIDDEN`
- `RESOURCE_NOT_FOUND`
- `CONFLICT`
- `INTERNAL_ERROR`

## Fora do escopo v0
- Webhooks de notificacao.
- Bulk operations em operadores/regioes.
- Versionamento por header.

## Decisoes pendentes
- Modelo de autenticacao/autorizacao (API key, JWT, OAuth2).
- Politica de rate limiting por cliente.
- Idempotencia em endpoints de escrita critica (`/metrics`, `/tickets/*`).
