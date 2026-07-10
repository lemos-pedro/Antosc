# Arquitetura (Planeamento)

## Objetivo
Construir um backend para monitorizacao operacional de torres, com coleta periodica de dados, avaliacao de estado e exposicao via API.

## Principios
- Dominio no centro (`core`).
- Dependencias apontam para dentro (ports and adapters).
- Integracoes externas isoladas em `adapters`.
- Observabilidade e rastreabilidade desde o inicio.

## Fluxo de dados alvo
1. `scheduler` dispara coleta em torres elegiveis.
2. `adapters/*` consultam equipamentos/protocolos externos.
3. `core/services` aplica regras, calcula estado e metricas.
4. `infrastructure/database` persiste eventos, snapshots e historico.
5. `api/handlers` entrega dados consolidados para clientes.

## Componentes principais
- `core/domain`: Torre, Equipamento, Falha, Alarme, SLA.
- `core/services`: avaliacao de disponibilidade, classificacao de eventos, consolidacao por torre.
- `adapters`: conectores por fabricante/protocolo.
- `api`: endpoints para consulta e operacao.
- `scheduler`: jobs de polling.

## Decisoes pendentes
- Banco principal: PostgreSQL ou outro.
- Estrategia de fila/eventos: sincrono no inicio ou broker desde a primeira versao.
- Modelo de autenticacao da API.
- Nivel de multi-tenant (por cliente/operador).







Power
├── System Status
├── Mains Status
├── Load Status
├── Battery Status

Battery
├── Voltage
├── Current
├── Remaining Ah
├── Temperature

Rectifiers
├── R1 Status
├── R1 Current
├── R1 Voltage
├── R2 Status
├── R2 Current
├── R2 Voltage

AC Input
├── L1 Voltage
├── L2 Voltage
├── L3 Voltage

Temperature
├── Battery
├── Controller
├── Rectifiers

History
├── Graph Battery Voltage
├── Graph Load Current
├── Graph Temperature
├── Graph Mains Voltage