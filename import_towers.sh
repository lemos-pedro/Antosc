#!/usr/bin/env bash
# ============================================================
# Antosc System - Importação de torres de teste (lote de 50)
#
# Todas as torres apontam para o noc_simulator local
# (127.0.0.1:161, community Antosc-noc, v2c) — não para os
# IPs reais da lista (esses ficam só registados no log para
# referência futura, quando a GETIC liberar acesso real).
#
# Region_id: o script tenta resolver via API (GET/POST /regions).
# Se POST /regions não existir (404/405), cai automaticamente
# para inserção direta na tabela `regions` via `docker exec psql`,
# de forma idempotente (não duplica em reruns).
#
# Uso:
#   chmod +x import_towers.sh
#   ./import_towers.sh
#
# Variáveis de ambiente:
#   PG_CONTAINER   nome do container Postgres (default: towercore-db)
#   PG_USER        utilizador psql            (default: postgres)
#   PG_DB          base de dados              (default: towercore)
#   PARALLEL_JOBS  torres criadas em paralelo (default: 8)
#   API            base URL da API            (default: http://localhost:8000/api/v1)
#
# Exemplo:
#   PG_CONTAINER=antosc-system-postgres-1 PG_USER=postgres PG_DB=antosc \
#     PARALLEL_JOBS=10 ./import_towers.sh
# ============================================================

set -uo pipefail

# ------------------------------------------------------------
# 0. Configuração
# ------------------------------------------------------------
API="${API:-http://localhost:8000/api/v1}"
TS=$(date +%Y%m%d_%H%M%S)
LOG="towers_import_${TS}.log"
SQL_FALLBACK="regions_insert_fallback_${TS}.sql"
SITES_FILE="${SITES_FILE:-./sites.json}"

SNMP_TARGET="127.0.0.1"
SNMP_PORT=161
SNMP_VERSION="v2c"
SNMP_COMMUNITY="Antosc-noc"

OPERATOR_CODE="ANTOSC"
OPERATOR_NAME="Antosc / Grupo Anglobal"

PG_CONTAINER="${PG_CONTAINER:-towercore-db}"
PG_USER="${PG_USER:-postgres}"
PG_DB="${PG_DB:-towercore}"
PARALLEL_JOBS="${PARALLEL_JOBS:-8}"

CURL_OPTS=(--max-time 15 --retry 2 --retry-delay 1 --retry-connrefused -s)
UUID_RE='^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$'

log() { echo "$1" | tee -a "$LOG"; }
die() { log "[ERRO FATAL] $1"; exit 1; }

# Filtro jq que lida com o shape mismatch conhecido:
# backend devolve array puro OU envelope {data:[...], meta:{...}}
JQ_ITEMS='def items: if type == "array" then . else (.data // []) end;'

# ------------------------------------------------------------
# 1. Pré-requisitos
# ------------------------------------------------------------
for bin in curl jq docker; do
  command -v "$bin" >/dev/null 2>&1 || die "comando '$bin' não encontrado no PATH."
done

[[ -f "$SITES_FILE" ]] || die "ficheiro de sites não encontrado: $SITES_FILE (define SITES_FILE=... ou cria ./sites.json)"
jq empty "$SITES_FILE" 2>/dev/null || die "'$SITES_FILE' não é um JSON válido."

log "=== Início da importação: $(date) ==="
log "[*] API=$API | PG_CONTAINER=$PG_CONTAINER | PG_DB=$PG_DB | PARALLEL_JOBS=$PARALLEL_JOBS"

# ------------------------------------------------------------
# 2. Helper: psql -> stdout só com o(s) UUID(s) válidos
# ------------------------------------------------------------
psql_uuid() {
  # $1 = comando SQL
  docker exec -i "$PG_CONTAINER" psql -U "$PG_USER" -d "$PG_DB" -t -A -q \
    -v "ON_ERROR_STOP=1" -c "$1" 2>>"$LOG" | grep -E "$UUID_RE" | head -n1
}

# ------------------------------------------------------------
# 3. Login
# ------------------------------------------------------------
log "[*] Autenticando..."
LOGIN_RESP=$(curl "${CURL_OPTS[@]}" -X POST "$API/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo "$LOGIN_RESP" | jq -r '.access_token // empty')
[[ -n "$TOKEN" ]] || die "falha no login. Resposta: $LOGIN_RESP"
AUTH=(-H "Authorization: Bearer $TOKEN")
log "[OK] Token obtido."

# ------------------------------------------------------------
# 4. Operador único do lote (idempotente)
# ------------------------------------------------------------
log "[*] A verificar operador '$OPERATOR_CODE'..."
OPS_RESP=$(curl "${CURL_OPTS[@]}" -w "\n%{http_code}" -X GET "$API/operators?limit=200" "${AUTH[@]}")
OPS_STATUS=$(echo "$OPS_RESP" | tail -n1)
OPS_BODY=$(echo "$OPS_RESP" | sed '$d')

OPERATOR_ID=""
if [[ "$OPS_STATUS" == "200" ]]; then
  OPERATOR_ID=$(echo "$OPS_BODY" | jq -r --arg c "$OPERATOR_CODE" "
    $JQ_ITEMS
    items[] | select(.code == \$c) | (.operator_id // .id)
  " 2>/dev/null | head -n1)
fi

if [[ -z "${OPERATOR_ID:-}" || "$OPERATOR_ID" == "null" ]]; then
  log "[*] Operador não existe. A criar..."
  RESP=$(curl "${CURL_OPTS[@]}" -w "\n%{http_code}" -X POST "$API/operators" \
    -H "Content-Type: application/json" "${AUTH[@]}" \
    -d "{\"name\":\"$OPERATOR_NAME\",\"code\":\"$OPERATOR_CODE\"}")
  STATUS=$(echo "$RESP" | tail -n1)
  BODY=$(echo "$RESP" | sed '$d')
  [[ "$STATUS" == "200" || "$STATUS" == "201" ]] || die "POST /operators falhou (HTTP $STATUS): $BODY"
  OPERATOR_ID=$(echo "$BODY" | jq -r '.operator_id // .id')
fi
[[ -n "$OPERATOR_ID" && "$OPERATOR_ID" != "null" ]] || die "não foi possível obter operator_id."
log "[OK] operator_id = $OPERATOR_ID"

# ------------------------------------------------------------
# 5. Resolução de region_id (API com fallback psql idempotente)
# ------------------------------------------------------------
log "[*] A obter regiões existentes via GET /regions..."
REG_RESP=$(curl "${CURL_OPTS[@]}" -w "\n%{http_code}" -X GET "$API/regions?limit=200" "${AUTH[@]}")
REG_STATUS=$(echo "$REG_RESP" | tail -n1)
REG_BODY=$(echo "$REG_RESP" | sed '$d')

declare -A REGION_MAP
REGIONS_API_OK=true

if [[ "$REG_STATUS" == "200" ]]; then
  while IFS=$'\t' read -r rname rid; do
    [[ -n "$rname" ]] && REGION_MAP["$rname"]="$rid"
  done < <(echo "$REG_BODY" | jq -r "
    $JQ_ITEMS
    items[] | \"\(.name)\t\(.region_id // .id)\"
  " 2>/dev/null)
  log "[OK] GET /regions HTTP 200. ${#REGION_MAP[@]} região(ões) pré-existente(s) via API."
else
  log "[AVISO] GET /regions devolveu HTTP $REG_STATUS — sem cobertura de leitura via API."
  REGIONS_API_OK=false
fi

NEEDED_REGIONS=$(jq -r '[.[].region] | unique | .[]' "$SITES_FILE")
MISSING_REGIONS=()

while IFS= read -r rname; do
  [[ -z "$rname" ]] && continue
  [[ -n "${REGION_MAP[$rname]:-}" ]] && continue

  if [[ "$REGIONS_API_OK" == true ]]; then
    RESP=$(curl "${CURL_OPTS[@]}" -w "\n%{http_code}" -X POST "$API/regions" \
      -H "Content-Type: application/json" "${AUTH[@]}" \
      -d "{\"name\":\"$rname\"}")
    STATUS=$(echo "$RESP" | tail -n1)
    BODY=$(echo "$RESP" | sed '$d')

    if [[ "$STATUS" == "200" || "$STATUS" == "201" ]]; then
      RID=$(echo "$BODY" | jq -r '.region_id // .id')
      if [[ "$RID" =~ $UUID_RE ]]; then
        REGION_MAP["$rname"]="$RID"
        log "[OK] Região criada via API: $rname -> $RID"
        continue
      fi
    elif [[ "$STATUS" == "404" || "$STATUS" == "405" ]]; then
      log "[AVISO] POST /regions não existe (HTTP $STATUS). A mudar para fallback via psql para as restantes."
      REGIONS_API_OK=false
    else
      log "[AVISO] POST /regions falhou para '$rname' (HTTP $STATUS): $BODY. A tentar fallback via psql."
    fi
  fi
  MISSING_REGIONS+=("$rname")
done <<< "$NEEDED_REGIONS"

if [[ ${#MISSING_REGIONS[@]} -gt 0 ]]; then
  log "[*] ${#MISSING_REGIONS[@]} região(ões) sem cobertura pela API. A resolver via 'docker exec $PG_CONTAINER psql' (idempotente)..."

  {
    echo "-- Execução manual alternativa, caso o docker exec falhe:"
    for rname in "${MISSING_REGIONS[@]}"; do
      esc=$(printf '%s' "$rname" | sed "s/'/''/g")
      echo "INSERT INTO regions (name) SELECT '$esc' WHERE NOT EXISTS (SELECT 1 FROM regions WHERE name = '$esc') RETURNING region_id, name;"
    done
  } > "$SQL_FALLBACK"
  log "[*] SQL de apoio escrito em: $SQL_FALLBACK"

  for rname in "${MISSING_REGIONS[@]}"; do
    esc=$(printf '%s' "$rname" | sed "s/'/''/g")

    # INSERT idempotente: só insere se ainda não existir.
    RID=$(psql_uuid "INSERT INTO regions (name) SELECT '$esc' WHERE NOT EXISTS (SELECT 1 FROM regions WHERE name = '$esc') RETURNING region_id;")

    if [[ -z "$RID" ]]; then
      # Já existia (ou insert não devolveu linha) -> busca o id existente.
      RID=$(psql_uuid "SELECT region_id FROM regions WHERE name = '$esc' LIMIT 1;")
    fi

    if [[ "$RID" =~ $UUID_RE ]]; then
      REGION_MAP["$rname"]="$RID"
      log "[OK] Região (via psql) pronta: $rname -> $RID"
    else
      log "[ERRO] Não foi possível obter region_id para '$rname' via docker exec (container '$PG_CONTAINER'). Corre manualmente o SQL em $SQL_FALLBACK."
    fi
  done
fi

log "[*] Mapa de regiões final (${#REGION_MAP[@]} entradas)."

# ------------------------------------------------------------
# 6. Criação das torres + ativação SNMP (paralelizado)
# ------------------------------------------------------------
TOTAL=$(jq length "$SITES_FILE")
log ""
log "[*] A criar $TOTAL torres (snmp_target=$SNMP_TARGET:$SNMP_PORT, community=$SNMP_COMMUNITY, version=$SNMP_VERSION, paralelismo=$PARALLEL_JOBS)..."

RESULTS_DIR=$(mktemp -d)
trap 'rm -rf "$RESULTS_DIR"' EXIT

create_one_tower() {
  local site_json="$1"
  local site_id site_ip name vendor region_name region_id
  site_id=$(echo "$site_json" | jq -r '.id')
  site_ip=$(echo "$site_json" | jq -r '.ip')
  vendor=$(echo "$site_json" | jq -r '.vendor')
  region_name=$(echo "$site_json" | jq -r '.region')
  name="TWR-${site_id}"
  region_id="${REGION_MAP[$region_name]:-}"

  local outfile="$RESULTS_DIR/${site_id}.result"

  if [[ -z "$region_id" ]]; then
    echo "FAIL|$site_id|$name|sem region_id para '$region_name'" > "$outfile"
    return
  fi

  # Idempotência: se já existir torre com este nome para o operador, reaproveita.
  local existing
  existing=$(curl "${CURL_OPTS[@]}" -X GET "$API/towers?name=$name&operator_id=$OPERATOR_ID&limit=1" "${AUTH[@]}" \
    | jq -r "$JQ_ITEMS items[0] | (.tower_id // .id // empty)" 2>/dev/null)

  local tower_id
  if [[ -n "$existing" && "$existing" != "null" ]]; then
    tower_id="$existing"
  else
    local create_resp create_status create_body
    create_resp=$(curl "${CURL_OPTS[@]}" -w "\n%{http_code}" -X POST "$API/towers" \
      -H "Content-Type: application/json" "${AUTH[@]}" \
      -d "{\"name\":\"$name\",\"operator_id\":\"$OPERATOR_ID\",\"region_id\":\"$region_id\",\"vendor\":\"$vendor\",\"snmp_version\":\"$SNMP_VERSION\"}")
    create_status=$(echo "$create_resp" | tail -n1)
    create_body=$(echo "$create_resp" | sed '$d')

    if [[ "$create_status" != "200" && "$create_status" != "201" ]]; then
      echo "FAIL|$site_id|$name|POST /towers HTTP $create_status -> $create_body" > "$outfile"
      return
    fi

    tower_id=$(echo "$create_body" | jq -r '.tower_id // .id // empty')
    if [[ -z "$tower_id" || "$tower_id" == "null" ]]; then
      echo "FAIL|$site_id|$name|sem tower_id na resposta -> $create_body" > "$outfile"
      return
    fi
  fi

  local patch_resp patch_status patch_body
  patch_resp=$(curl "${CURL_OPTS[@]}" -w "\n%{http_code}" -X PATCH "$API/towers/$tower_id/snmp" \
    -H "Content-Type: application/json" "${AUTH[@]}" \
    -d "{\"snmp_enabled\":true,\"snmp_version\":\"$SNMP_VERSION\",\"snmp_target\":\"$SNMP_TARGET\",\"snmp_community\":\"$SNMP_COMMUNITY\",\"snmp_port\":$SNMP_PORT}")
  patch_status=$(echo "$patch_resp" | tail -n1)
  patch_body=$(echo "$patch_resp" | sed '$d')

  if [[ "$patch_status" == "200" || "$patch_status" == "204" ]]; then
    echo "OK|$site_id|$name|tower_id=$tower_id|região=$region_name|vendor=$vendor|ip_real(GETIC,futuro)=$site_ip" > "$outfile"
  else
    echo "FAIL|$site_id|$name|tower criada (tower_id=$tower_id) mas PATCH /snmp HTTP $patch_status -> $patch_body" > "$outfile"
  fi
}

# Nota: create_one_tower é chamada via '&' (background job no MESMO shell,
# criado por fork), não via subshell de xargs/parallel — por isso herda
# diretamente REGION_MAP, OPERATOR_ID, etc. sem precisar de 'export'.
# (export -A não é necessário nem totalmente portável entre versões de bash.)
job_count=0
for i in $(seq 0 $((TOTAL - 1))); do
  site=$(jq ".[$i]" "$SITES_FILE")
  create_one_tower "$site" &
  ((job_count++))
  if (( job_count >= PARALLEL_JOBS )); then
    wait -n
    ((job_count--))
  fi
done
wait

# ------------------------------------------------------------
# 7. Resumo (lendo resultados dos workers paralelos)
# ------------------------------------------------------------
OK_COUNT=0
FAIL_COUNT=0
log ""
log "[*] Resultados:"
for f in "$RESULTS_DIR"/*.result; do
  [[ -f "$f" ]] || continue
  line=$(cat "$f")
  status="${line%%|*}"
  if [[ "$status" == "OK" ]]; then
    log "[OK]   ${line#OK|}"
    ((OK_COUNT++)) || true
  else
    log "[FAIL] ${line#FAIL|}"
    ((FAIL_COUNT++)) || true
  fi
done

log ""
log "=== Resumo ==="
log "Sucesso: $OK_COUNT / $TOTAL"
log "Falhas:  $FAIL_COUNT / $TOTAL"
log "Log completo: $LOG"
[[ -f "$SQL_FALLBACK" ]] && log "SQL de apoio (regiões): $SQL_FALLBACK"

[[ "$FAIL_COUNT" -eq 0 ]] || exit 2