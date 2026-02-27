# Definicoes do Dominio

Este documento define os termos base do sistema para evitar ambiguidades no desenho, implementacao e operacao.

## Torre
Unidade operacional com identidade propria, composta por:
- Infraestrutura fisica
- Energia
- Equipamentos ativos
- Operadores hospedados
- Localizacao
- SLA associado

## Disponibilidade
Percentual do tempo em que o servico permanece operacional dentro de um periodo definido.

Formula:
`Disponibilidade (%) = ((Tempo total - Downtime) / Tempo total) * 100`

Notas:
- Downtime deve ser medido em minutos ou segundos, com a mesma unidade do tempo total.
- Janelas de manutencao planejada devem ser classificadas separadamente.

## Falha
Evento tecnico emitido por equipamento ou sistema indicando:
- Estado anormal
- Limite ultrapassado
- Mudanca critica

`Alarme` e `Falha` nao sao a mesma coisa:
- Alarme: sinalizacao de risco, degradacao ou condicao fora do normal.
- Falha: interrupcao real, comportamento incorreto ou indisponibilidade.

## SLA
Compromisso contratual mensuravel entre prestador e cliente, normalmente expresso por:
- Disponibilidade minima (ex.: 99.9% mensal)
- Tempo maximo de resposta a incidentes
- Tempo maximo de resolucao por severidade

## MTTR e MTBF
- `MTTR (Mean Time To Repair)`: tempo medio para restaurar um servico apos falha.
- `MTBF (Mean Time Between Failures)`: tempo medio entre falhas consecutivas.

Essas metricas ajudam a acompanhar maturidade operacional e qualidade do parque tecnico.
