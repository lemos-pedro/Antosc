#!/usr/bin/env python3
"""
nagios_to_towercore.py

Le a lista de hosts do Nagios (objectjson.cgi) e o respectivo estado
(statusjson.cgi), e faz upsert na tabela de torres do towercore (PostgreSQL).

Objetivo de hoje: ter torres REAIS na base de dados para o antosc-front
conseguir listar algo a partir da API GET /towers.

Uso:
    python nagios_to_towercore.py

Configura as variaveis em CONFIG antes de correr.

IMPORTANTE - antes de correr, confirma o nome real da tabela e das colunas
no schema do towercore (migrations 00001_init / 00003_discovery). O script
assume a forma descrita em api.md:
    towers(tower_id uuid pk, name text, status text, operator_id uuid null,
           region_id uuid null, updated_at timestamptz)
Ajusta TABLE_NAME / COLUMN_MAP se o schema real for diferente.
"""

import sys
import uuid
import requests
import psycopg2
from datetime import datetime, timezone

# ---------------------------------------------------------------------------
# CONFIG - ajustar antes de correr
# ---------------------------------------------------------------------------
NAGIOS_BASE_URL = "http://172.17.0.31//nagios/cgi-bin"  # <-- preencher
NAGIOS_USER = "nagiosadmin"        # <-- preencher se Nagios pedir auth basica
NAGIOS_PASSWORD = "123"    # <-- preencher

DB_HOST = "localhost"
DB_PORT = 5432
DB_NAME = "towercore"
DB_USER = "A.lemos"
DB_PASSWORD = "A.lemos13:7002"

TABLE_NAME = "towers"

# Mapeamento de status do Nagios (host state) para o vocabulario do towercore
# Nagios host state: 0=UP, 1=DOWN, 2=UNREACHABLE
NAGIOS_STATE_MAP = {
    0: "online",
    1: "offline",
    2: "offline",
}
DEFAULT_STATUS = "degraded"  # quando o estado nao for reconhecido

# ---------------------------------------------------------------------------


def fetch_nagios_hosts():
    """Obtem a lista de hosts configurados no Nagios."""
    url = f"{NAGIOS_BASE_URL}/objectjson.cgi"
    params = {"query": "hostlist"}
    auth = (NAGIOS_USER, NAGIOS_PASSWORD) if NAGIOS_USER else None

    resp = requests.get(url, params=params, auth=auth, timeout=15, verify=False)
    resp.raise_for_status()
    data = resp.json()

    hostlist = data.get("data", {}).get("hostlist", {})
    if not hostlist:
        print("AVISO: objectjson.cgi devolveu hostlist vazia. Verifica a URL/auth.", file=sys.stderr)
    return hostlist  # dict: { "hostname": "ip_or_address" }


def fetch_nagios_status():
    """Obtem o estado atual de todos os hosts."""
    url = f"{NAGIOS_BASE_URL}/statusjson.cgi"
    params = {"query": "hostlist", "details": "true"}
    auth = (NAGIOS_USER, NAGIOS_PASSWORD) if NAGIOS_USER else None

    resp = requests.get(url, params=params, auth=auth, timeout=15, verify=False)
    resp.raise_for_status()
    data = resp.json()

    hosts = data.get("data", {}).get("hostlist", {})
    status_by_name = {}
    for hostname, info in hosts.items():
        state = info.get("status", -1)
        status_by_name[hostname] = NAGIOS_STATE_MAP.get(state, DEFAULT_STATUS)
    return status_by_name


def connect_db():
    return psycopg2.connect(
        host=DB_HOST, port=DB_PORT, dbname=DB_NAME,
        user=DB_USER, password=DB_PASSWORD,
    )


def upsert_tower(cur, name: str, status: str):
    """Insere a torre se nao existir (por nome), ou atualiza o status/updated_at."""
    cur.execute(
        f"SELECT tower_id FROM {TABLE_NAME} WHERE name = %s",
        (name,)
    )
    row = cur.fetchone()
    now = datetime.now(timezone.utc)

    if row:
        tower_id = row[0]
        cur.execute(
            f"""UPDATE {TABLE_NAME}
                SET status = %s, updated_at = %s
                WHERE tower_id = %s""",
            (status, now, tower_id)
        )
        return tower_id, "updated"
    else:
        tower_id = str(uuid.uuid4())
        cur.execute(
            f"""INSERT INTO {TABLE_NAME} (tower_id, name, status, updated_at)
                VALUES (%s, %s, %s, %s)""",
            (tower_id, name, status, now)
        )
        return tower_id, "created"


def main():
    print("A obter lista de hosts do Nagios...")
    hosts = fetch_nagios_hosts()

    print("A obter estado atual dos hosts...")
    statuses = fetch_nagios_status()

    if not hosts:
        print("Nenhum host encontrado. Aborta.", file=sys.stderr)
        sys.exit(1)

    print(f"{len(hosts)} hosts encontrados no Nagios. A ligar a base de dados...")
    conn = connect_db()
    conn.autocommit = False
    cur = conn.cursor()

    created, updated, errors = 0, 0, 0

    try:
        for hostname in hosts:
            status = statuses.get(hostname, DEFAULT_STATUS)
            try:
                tower_id, action = upsert_tower(cur, hostname, status)
                if action == "created":
                    created += 1
                else:
                    updated += 1
                print(f"  [{action}] {hostname} -> status={status} tower_id={tower_id}")
            except Exception as e:
                errors += 1
                print(f"  [ERRO] {hostname}: {e}", file=sys.stderr)

        conn.commit()
    except Exception as e:
        conn.rollback()
        print(f"Erro fatal, rollback efetuado: {e}", file=sys.stderr)
        sys.exit(1)
    finally:
        cur.close()
        conn.close()

    print(f"\nConcluido. Criadas: {created} | Atualizadas: {updated} | Erros: {errors}")
    print("Confirma agora com: SELECT count(*) FROM towers;")
    print("E testa a API: GET /api/v1/towers")


if __name__ == "__main__":
    main()