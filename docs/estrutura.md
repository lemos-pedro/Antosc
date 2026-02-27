# Estrutura do Backend (`towercore`)

Este documento descreve a estrutura alvo do backend e o papel de cada camada.

```text
towercore/
  cmd/
    api/                # entrypoint da aplicacao HTTP
  internal/
    core/               # dominio e regras de negocio
      domain/        # entidades e value objects 
        |_ Tower, 
        |_ Event, 
        |_ Metric, 
        |_ Operator, 
        |_ Region, 
        |_ Ticket, 
        |_ SLA
      services/         # casos de uso
      interfaces/       # contratos (ports)
    adapters/           # implementacoes dos contratos (drivers externos)
      snmp/
      huawei/
      eltek/
      entek/
      mocks/
    infrastructure/     # recursos tecnicos transversais
      database/
      cache/
      config/
      logger/
    api/                # camada HTTP (entrada externa)
      handlers/
      middleware/
      routes/
    scheduler/          # jobs periodicos de polling/coleta
  pkg/                  # bibliotecas reutilizaveis (uso externo ao internal)
  configs/              # arquivos de configuracao por ambiente
  migrations/           # migracoes de banco
  scripts/              # automacoes de desenvolvimento/operacao
```

## Regras de organizacao
- `internal/core` nunca depende de `internal/api` ou de detalhes de infraestrutura.
- `adapters` implementam interfaces definidas em `core/interfaces`.
- `api/handlers` apenas orquestram request/response, sem regra de negocio.
- `scheduler` reutiliza casos de uso do `core/services`.

## Estado atual
- Estrutura de pastas criada.
- Implementacao ainda nao iniciada (somente `cmd/api/main.go` existe).
