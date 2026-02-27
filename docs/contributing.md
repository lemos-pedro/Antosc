# Contribuindo

## Objetivo
Definir um padrao minimo para colaboracao tecnica desde a fase inicial.

## Fluxo de trabalho
1. Criar branch a partir da principal (`feature/*`, `fix/*`, `docs/*`).
2. Implementar mudanca pequena e focada.
3. Atualizar documentacao relacionada.
4. Abrir PR com contexto, impacto e como validar.

## Padroes de codigo (Go)
- Priorizar simplicidade e clareza.
- Regra de negocio deve ficar em `internal/core`.
- Handlers HTTP sem logica de negocio.
- Testes unitarios para servicos e validadores do dominio.

## Padrao de commit (sugestao)
- `feat:`
- `fix:`
- `docs:`
- `refactor:`
- `test:`
- `chore:`

## Definicao de pronto (DoD)
- Codigo implementado e revisado.
- Testes locais relevantes executados.
- Documentacao atualizada.
- Sem segredos ou dados sensiveis versionados.
