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