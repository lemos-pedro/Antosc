# Antosc System

Repositorio do projeto Antosc System, atualmente em fase de planeamento.

## Estado atual
- Fase: Planeamento e definicao de arquitetura
- Backend alvo: `towercore` (Go)
- Objetivo imediato: fechar dominio, contratos e base tecnica de arranque

## Estrutura
- `docs/`: documentacao funcional e tecnica
- `towercore/`: backend principal

## Proximos marcos
1. Inicializar `go.mod` e bootstrap da API.
2. Definir contratos do dominio (`internal/core/interfaces`).
3. Criar primeiro fluxo vertical: coleta -> processamento -> persistencia -> exposicao HTTP.
4. Adicionar testes unitarios do dominio e testes de integracao do adapter principal.

## Documentacao
Indice principal em [docs/README.md](/home/lemos/Documentos/Antosc-system/docs/README.md).


"C:\Users\joaquim.pedro\Documents\pgsql\bin\psql.exe" -U A.lemos -d towercore

  curl "http://towercore-api:8080/api/radio-kpi?tower_id=a1b2c3d4-e5f6-7890-g1h2-i3j4k5l6m7n8&sector_id=A&technique=LTE&limit=1"

  # Obter KPIs das últimas 24h para análise de tendência
  curl "http://towercore-api:8080/api/radio-kpi?tower_id=a1b2c3d4-e5f6-7890-g1h2-i3j4k5l6m7n8&from=2026-08-16T00:00:00Z&to=2026-08-17T00:00:00Z"
    🔍 EXEMPLO DE CONSULTAS ÚTEIS:

  # Obter o status atual de todas as interfaces de uma torre (ótimo para dashboard)
  curl "http://towercore-api:8080/api/backhaul/tower/a1b2c3d4-e5f6-7890-g1h2-i3j4k5l6m7n8/status"

  # Obter histórico da interface eth0 das últimas 6 horas
  curl "http://towercore-api:8080/api/backhaul?tower_id=a1b2c3d4-e5f6-7890-g1h2-i3j4k5l6m7n8&interface_name=eth0&from=2026-08-17T04:30:00Z&to=2026-08-17T10:
  30:00Z"

  # Encontrar todas as interfaces com problemas (não up)
  curl "http://towercore-api:8080/api/backhaul?down=true"

  # Encontrar interfaces com utilização alta (>80%)
  curl "http://towercore-api:8080/api/backhaul?high_util=true"

  # Encontrar interfaces com latência alta (>100ms)
  curl "http://towercore-api:8080/api/backhaul?min_latency_ms=100"
