# Configuracao

Variaveis de ambiente previstas para a primeira versao do backend.

## Aplicacao
- `APP_NAME` (default: `towercore-api`)
- `APP_ENV` (`dev`, `staging`, `prod`)
- `APP_PORT` (default: `8000`)
- `LOG_LEVEL` (`debug`, `info`, `warn`, `error`)

## Cache
- `CACHE_TOWER_LIST_TTL_SECONDS` (default: `5`)
- `CACHE_TOWER_DETAIL_TTL_SECONDS` (default: `10`)

## Base de dados
- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_USER`
- `DB_PASSWORD`
- `DB_SSL_MODE` (default sugerido: `disable` em dev)
- `DB_MAX_OPEN_CONNS` (default: `10`)
- `DB_MAX_IDLE_CONNS` (default: `5`)
- `DB_CONN_MAX_LIFETIME_MINUTES` (default: `30`)

## Cache (opcional na v1)
- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_PASSWORD`

## Scheduler
- `SCHEDULER_ENABLED` (default: `true`)
- `POLL_INTERVAL_SECONDS` (ex.: `60`)
- `POLL_BATCH_SIZE` (ex.: `100`)
- `POLL_TIMEOUT_SECONDS` (ex.: `10`)

## Integracoes
- `SNMP_TIMEOUT_SECONDS`
- `SNMP_RETRIES`

## Limites de escrita
- `RATE_LIMIT_WRITE_PER_MINUTE`
  - `0` desativa o rate limiting para ambientes de teste/carga.

## Boas praticas
- Nunca versionar segredos em texto puro.
- Definir defaults seguros para desenvolvimento local.
- Documentar novas variaveis no momento em que forem introduzidas no codigo.
