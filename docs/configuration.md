# Configuracao

Variaveis de ambiente previstas para a primeira versao do backend.

## Aplicacao
- `APP_NAME` (default: `towercore-api`)
- `APP_ENV` (`dev`, `staging`, `prod`)
- `APP_PORT` (default: `8080`)
- `LOG_LEVEL` (`debug`, `info`, `warn`, `error`)

## Base de dados
- `DB_HOST`
- `DB_PORT`
- `DB_NAME`
- `DB_USER`
- `DB_PASSWORD`
- `DB_SSL_MODE` (default sugerido: `disable` em dev)

## Cache (opcional na v1)
- `REDIS_HOST`
- `REDIS_PORT`
- `REDIS_PASSWORD`

## Scheduler
- `POLL_INTERVAL_SECONDS` (ex.: `60`)
- `POLL_BATCH_SIZE` (ex.: `100`)
- `POLL_TIMEOUT_SECONDS` (ex.: `10`)

## Integracoes
- `SNMP_TIMEOUT_SECONDS`
- `SNMP_RETRIES`

## Boas praticas
- Nunca versionar segredos em texto puro.
- Definir defaults seguros para desenvolvimento local.
- Documentar novas variaveis no momento em que forem introduzidas no codigo.
