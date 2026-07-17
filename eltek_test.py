"""
snmp_community_scan.py

Scan de descoberta SNMP v2c para o parque de torres Antosc.
Objetivo: para cada IP, determinar (a) se responde SNMP, (b) qual/quais
communities funcionam, (c) sysDescr, (d) vendor provável via enterprise OID.

IMPORTANTE (organizacional):
- Executar apenas com autorização documentada (já confirmada pelo Diretor
  Técnico). Manter CONCURRENCY_LIMIT baixo evita gerar padrões de tráfego
  que pareçam um port/IP sweep agressivo e voltem a ser escalados pela GETIC
  como incidente de segurança.
- Correr fora de horário de pico se possível.

Saídas:
- snmp_scan_results.csv   -> uma linha por IP, resumo
- snmp_scan_results.json  -> detalhe completo (todas as communities testadas)
"""

import asyncio
import csv
import json
import logging
import time
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone

from pysnmp.hlapi.v3arch.asyncio import (
    SnmpEngine,
    CommunityData,
    UdpTransportTarget,
    ContextData,
    ObjectType,
    ObjectIdentity,
    get_cmd,
)
from pysnmp.proto import rfc1905

# ----------------------------------------------------------------------------
# Configuração
# ----------------------------------------------------------------------------

# Rótulo do lote atual -- muda isto a cada execução (ex.: "neteco", "eltek", "enetek")
BATCH_LABEL = "full"  # 141 IPs, inventario completo fornecido (deduplicado)

IPS = {
    "192.168.102.5",
    "192.168.113.5",
    "192.168.127.5",
    "192.168.128.5",
    "192.168.129.5",
    "192.168.139.5",
    "192.168.154.5",
    "192.168.155.5",
    "192.168.156.5",
    "192.168.157.5",
    "192.168.158.5",
    "192.168.159.5",
    "192.168.160.5",
    "192.168.161.5",
    "192.168.162.5",
    "192.168.163.5",
    "192.168.164.5",
    "192.168.166.5",
    "192.168.167.5",
    "192.168.168.5",
    "192.168.169.5",
    "192.168.170.5",
    "192.168.171.5",
    "192.168.172.5",
    "192.168.173.5",
    "192.168.174.5",
    "192.168.176.5",
    "192.168.177.5",
    "192.168.178.5",
    "192.168.180.5",
    "192.168.183.5",
    "192.168.184.5",
    "192.168.188.5",
    "192.168.191.5",
    "192.168.192.5",
    "192.168.194.5",
    "192.168.196.5",
    "192.168.198.5",
    "192.168.199.5",
    "192.168.200.5",
    "192.168.201.5",
    "192.168.202.37",
    "192.168.203.101",
    "192.168.204.165",
    "192.168.210.5",
  
}

# Lista de candidatos a testar por IP. Ajusta conforme o que já se sabe
# ter sido usado historicamente (ex.: "public" confirmado num Eltek).
COMMUNITIES = [
    "community1",
    "public",
    "private",
]

# OIDs de enterprise para identificação de vendor (sysObjectID costuma bastar,
# mas manter get direto ao ramo enterprise como confirmação secundária).
SYS_DESCR_OID = "1.3.6.1.2.1.1.1.0"
SYS_OBJECT_ID_OID = "1.3.6.1.2.1.1.2.0"

VENDOR_ENTERPRISE_OIDS = {
    "huawei": "1.3.6.1.4.1.2011",
   # "eltek": "1.3.6.1.4.1.12148",
    #"vertive_emerson": "1.3.6.1.4.1.476",
    #"enetek": "1.3.6.1.4.1.53318",  # confirmado via MIB da Enetek: enterprises 53318
}

TIMEOUT_SECONDS = 2
RETRIES = 1
CONCURRENCY_LIMIT = 5          # nº de IPs em paralelo -- manter baixo
DELAY_BETWEEN_COMMUNITIES = 0.15  # segundos, evita rajada por IP

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("snmp_scan")


# ----------------------------------------------------------------------------
# Modelo de resultado
# ----------------------------------------------------------------------------

@dataclass
class CommunityResult:
    community: str
    ok: bool
    sys_descr: str | None = None
    sys_object_id: str | None = None
    error: str | None = None


@dataclass
class ScanResult:
    ip: str
    batch_label: str = BATCH_LABEL
    responded: bool = False
    working_communities: list[str] = field(default_factory=list)
    all_attempts: list[CommunityResult] = field(default_factory=list)
    sys_descr: str | None = None
    sys_object_id: str | None = None
    vendor_guess: str | None = None
    vendor_confirmed_via: list[str] = field(default_factory=list)
    scanned_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )


# ----------------------------------------------------------------------------
# SNMP helpers
# ----------------------------------------------------------------------------

def _is_real_value(varBinds) -> bool:
    """Um GET a um OID que não existe no agente ainda devolve varBinds
    (não None), mas com um valor especial NoSuchObject/NoSuchInstance/
    EndOfMibView. Sem este filtro, qualquer OID de vendor "confirma"
    presença mesmo quando o ramo não existe -- falso positivo."""
    if not varBinds:
        return False
    for _, value in varBinds:
        if isinstance(
            value,
            (rfc1905.NoSuchObject, rfc1905.NoSuchInstance, rfc1905.EndOfMibView),
        ):
            return False
    return True


async def snmp_get(ip: str, community: str, oid: str):
    """Faz um GET SNMP único. Retorna (varBinds | None, error_msg | None)."""
    try:
        transport = await UdpTransportTarget.create(
            (ip, 161), timeout=TIMEOUT_SECONDS, retries=RETRIES
        )

        errorIndication, errorStatus, errorIndex, varBinds = await get_cmd(
            SnmpEngine(),
            CommunityData(community),
            transport,
            ContextData(),
            ObjectType(ObjectIdentity(oid)),
        )

        if errorIndication:
            return None, str(errorIndication)
        if errorStatus:
            return None, f"{errorStatus.prettyPrint()} at {errorIndex}"
        if not _is_real_value(varBinds):
            return None, "no such object/instance at this OID"

        return varBinds, None

    except Exception as exc:  # noqa: BLE001
        return None, f"exception: {exc}"


def guess_vendor_from_sys_object_id(sys_object_id: str | None) -> str | None:
    if not sys_object_id:
        return None
    for vendor, oid in VENDOR_ENTERPRISE_OIDS.items():
        if oid in sys_object_id:
            return vendor
    return None


async def probe_community(ip: str, community: str) -> CommunityResult:
    varBinds, err = await snmp_get(ip, community, SYS_DESCR_OID)

    if varBinds is None:
        return CommunityResult(community=community, ok=False, error=err)

    sys_descr = str(varBinds[0][1]) if varBinds else None

    obj_id_binds, _ = await snmp_get(ip, community, SYS_OBJECT_ID_OID)
    sys_object_id = str(obj_id_binds[0][1]) if obj_id_binds else None

    return CommunityResult(
        community=community,
        ok=True,
        sys_descr=sys_descr,
        sys_object_id=sys_object_id,
    )


async def confirm_vendor_enterprise_oid(ip: str, community: str) -> list[str]:
    """Testa diretamente os ramos enterprise de cada vendor conhecido.
    Confirma resposta mesmo quando sysObjectID não é conclusivo."""
    confirmed = []
    for vendor, oid in VENDOR_ENTERPRISE_OIDS.items():
        varBinds, _ = await snmp_get(ip, community, oid)
        if varBinds is not None:
            confirmed.append(vendor)
        await asyncio.sleep(DELAY_BETWEEN_COMMUNITIES)
    return confirmed


async def scan_ip(ip: str, semaphore: asyncio.Semaphore) -> ScanResult:
    async with semaphore:
        result = ScanResult(ip=ip)
        log.info(f"[SCAN] {ip}")

        for community in COMMUNITIES:
            attempt = await probe_community(ip, community)
            result.all_attempts.append(attempt)

            if attempt.ok:
                result.responded = True
                result.working_communities.append(community)

                if result.sys_descr is None:
                    result.sys_descr = attempt.sys_descr
                    result.sys_object_id = attempt.sys_object_id
                    result.vendor_guess = guess_vendor_from_sys_object_id(
                        attempt.sys_object_id
                    )

                log.info(f"  [OK] {ip} -> community='{community}'")
                if attempt.sys_descr:
                    log.info(f"        sysDescr: {attempt.sys_descr[:100]}")

                # Confirma vendor via enterprise OID direto (não interrompe
                # nem sai do loop de communities: outra community pode
                # também responder e vale registar).
                confirmed = await confirm_vendor_enterprise_oid(ip, community)
                for v in confirmed:
                    if v not in result.vendor_confirmed_via:
                        result.vendor_confirmed_via.append(v)

            await asyncio.sleep(DELAY_BETWEEN_COMMUNITIES)

        if not result.responded:
            log.warning(f"  [FAIL] {ip} sem resposta SNMP em nenhuma community")

        return result


# ----------------------------------------------------------------------------
# Execução principal + export
# ----------------------------------------------------------------------------

async def main():
    start = time.monotonic()
    semaphore = asyncio.Semaphore(CONCURRENCY_LIMIT)

    tasks = [scan_ip(ip, semaphore) for ip in IPS]
    results: list[ScanResult] = await asyncio.gather(*tasks)

    elapsed = time.monotonic() - start

    # ---- JSON completo (inclui todas as tentativas, communities falhadas, etc.)
    json_path = f"snmp_scan_results_{BATCH_LABEL}.json"
    with open(json_path, "w", encoding="utf-8") as f:
        json.dump([asdict(r) for r in results], f, indent=2, ensure_ascii=False)

    # ---- CSV resumo, uma linha por IP
    csv_path = f"snmp_scan_results_{BATCH_LABEL}.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "batch_label", "ip", "responded", "working_communities", "sys_descr",
            "sys_object_id", "vendor_guess", "vendor_confirmed_via",
        ])
        for r in results:
            writer.writerow([
                r.batch_label,
                r.ip,
                r.responded,
                ";".join(r.working_communities),
                (r.sys_descr or "").replace("\n", " ")[:200],
                r.sys_object_id or "",
                r.vendor_guess or "",
                ";".join(r.vendor_confirmed_via),
            ])

    # ---- Resumo no terminal
    total = len(results)
    ok = sum(1 for r in results if r.responded)
    fail = total - ok

    print("\n" + "=" * 60)
    print(f"Scan concluído em {elapsed:.1f}s | {total} IPs | {ok} OK | {fail} FAIL")
    print("=" * 60)

    by_community: dict[str, list[str]] = {}
    for r in results:
        for c in r.working_communities:
            by_community.setdefault(c, []).append(r.ip)

    for community, ips in by_community.items():
        print(f"\nCommunity '{community}' funcionou em {len(ips)} IP(s):")
        for ip in ips:
            print(f"  - {ip}")

    no_response = [r.ip for r in results if not r.responded]
    if no_response:
        print(f"\nSem resposta em nenhuma community ({len(no_response)}):")
        for ip in no_response:
            print(f"  - {ip}")

    print(f"\nFicheiros gerados: {csv_path}, {json_path}")


if __name__ == "__main__":
    asyncio.run(main())