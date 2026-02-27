# Runbook Inicial

Guia operacional minimo para ambientes de desenvolvimento e pre-producao.

## Startup (planeado)
1. Carregar variaveis de ambiente.
2. Validar conexao com banco.
3. Subir API HTTP.
4. Iniciar scheduler de coleta.

## Health checks
- Liveness: processo ativo e loop HTTP responsivo.
- Readiness: dependencia critica (DB) operacional.

## Incidentes comuns (esperados)
- Timeout de coleta SNMP.
- Queda/intermitencia de banco.
- Aumento de latencia em endpoint de consulta historica.

## Resposta inicial a incidente
1. Confirmar escopo (uma torre, um grupo ou todo ambiente).
2. Verificar logs por `request_id` e por `tower_id`.
3. Validar conectividade com dependencias externas.
4. Isolar adapter problematico, se aplicavel.
5. Registrar causa e acao corretiva.

## Observabilidade minima
- Logs estruturados com nivel e contexto.
- Metricas de disponibilidade, falha e latencia.
- Painel de saude por torre e por adapter.
