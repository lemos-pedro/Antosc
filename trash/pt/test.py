"""
snmp_community_scan.py

Scan de descoberta SNMP v2c para o parque de torres Antosc.
Objetivo: para cada IP, determinar (a) se responde SNMP, (b) qual/quais
communities funcionam, (c) sysDescr, (d) vendor provável — via enterprise
OID confirmado e/ou heurística de texto no sysDescr —, e (e) comparar
com o vendor esperado do registo mestre, sinalizando discrepâncias.

IMPORTANTE (organizacional):
- Executar apenas com autorização documentada (já confirmada pelo Diretor
  Técnico). Manter CONCURRENCY_LIMIT baixo evita gerar padrões de tráfego
  que pareçam um port/IP sweep agressivo e voltem a ser escalados pela GETIC
  como incidente de segurança.
- A ordem de scan é baralhada (shuffle) propositadamente, para não varrer
  sequencialmente um /24 inteiro, o que também ajuda a evitar esse padrão.
- Correr fora de horário de pico se possível.

Entrada:
- CSV com colunas: ip[,vendor_esperado][,site_id][,nome]
  Por omissão lê "rectificadores_master.csv" (gerado a partir do registo
  mestre). Se não existir, usa a pequena lista de exemplo em IPS_FALLBACK.
  IPs duplicados no CSV (mesmo IP associado a vendors esperados diferentes
  em linhas distintas) são preservados como lista de "vendor_esperado".

Saídas (nome-base configurável via --output-prefix, default "snmp_scan"):
- <prefix>_results.csv   -> uma linha por IP, resumo (inclui comparação
  com vendor esperado)
- <prefix>_results.json  -> detalhe completo (todas as communities testadas)
- <prefix>_progress.jsonl -> gravação incremental linha-a-linha à medida que
  cada IP termina, para não perder trabalho se o processo for interrompido
"""

import argparse
import asyncio
import csv
import json
import logging
import random
import sys
import time
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from pathlib import Path

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

# Usada apenas se o CSV de entrada não existir (ver --input).
IPS_FALLBACK = [

    "192.168.112.5",
    "192.168.141.5",
    "192.168.201.165",
    "192.168.202.101",
    "192.168.201.229",
    "192.168.165.5",
    "192.168.207.37",
    "192.168.133.5",
    "192.168.123.5",
    "192.168.208.229",
    "192.168.206.101"
    "192.168.208.5",
    "192.168.142.5",
    "192.168.209.37",
    "192.168.147.5",
    "192.168.138.5",
    "192.168.207.229",
    "192.168.144.5",
    "192.168.151.5",
    "192.168.209.37",
    "192.168.209.69",
    "192.168.202.37",
    "192.168.130.5",
    "192.168.200.69",
    "192.168.121.5",
    "192.168.152.5",
    "192.168.134.5",
    "192.168.143.5",
    "192.168.140.5",
    "192.168.122.5",

]


COMMUNITIES = [
    "public",
    "private",
    "Public",
    "Private",
    "v1community",
    "v2community",
    "comunity1",
    "comunity2",
    "V1community",
    "V2community",
    "Comunity1",
    "Comunity2",

    # adicionar aqui outras communities conhecidas/candidatas
]

# OIDs standard do MIB-2 (grupo "system") -- úteis tanto para identificação
# de vendor como para inventário (sysName/sysUpTime dão contexto extra nos
# outputs sem custo adicional de pedidos, já que vêm do mesmo GET).
SYS_DESCR_OID = "1.3.6.1.2.1.1.1.0"
SYS_OBJECT_ID_OID = "1.3.6.1.2.1.1.2.0"
SYS_NAME_OID = "1.3.6.1.2.1.1.5.0"
SYS_UPTIME_OID = "1.3.6.1.2.1.1.3.0"


@dataclass(frozen=True)
class VendorInfo:
    """Fonte única de verdade por vendor.

    Antes havia dois dicts separados (OID e hints de texto) que era fácil
    desalinhar ao adicionar/corrigir um vendor. Agora é uma entrada só,
    com o grau de confiança do OID registado explicitamente.
    """
    display_name: str
    enterprise_oid: str            # "1.3.6.1.4.1.<nº IANA>" -- ver
                                    # https://www.iana.org/assignments/enterprise-numbers
    text_hints: tuple[str, ...]     # substrings (lowercase) a procurar no sysDescr
    confirmed: bool                 # True = validado num equipamento Antosc real
    note: str = ""


VENDORS: dict[str, VendorInfo] = {
    "huawei": VendorInfo(
        display_name="Huawei",
        enterprise_oid="1.3.6.1.4.1.2011", 
        text_hints=("huawei",),
        confirmed=True,
    ),
    "eltek": VendorInfo(
        display_name="Eltek",
        enterprise_oid="1.3.6.1.4.1.12148",
        text_hints=("eltek",),
        confirmed=True,
    ),
    "vertive": VendorInfo(
        display_name="Vertiv(e)",
        enterprise_oid="1.3.6.1.4.1.476",
        text_hints=("vertive", "vertiv", "emerson", "netsure"),
        confirmed=True,
        note="Ex-Emerson Network Power; algumas MIBs antigas ainda"
             " reportam sysDescr como 'Netsure'.",
    ),
    "enetek": VendorInfo(
        display_name="Enetek",
        enterprise_oid="1.3.6.1.4.1.53318",
        text_hints=("enetek",),
        confirmed=True,
    ),
}

# Vistas derivadas só para o resto do código não ter de mudar -- editar
# sempre VENDORS acima, nunca estes dois diretamente.
VENDOR_ENTERPRISE_OIDS = {k: v.enterprise_oid for k, v in VENDORS.items()}
VENDOR_TEXT_HINTS = {k: list(v.text_hints) for k, v in VENDORS.items()}


def _oid_tuple(oid: str) -> tuple[int, ...]:
    """Converte '1.3.6.1.4.1.476' em (1, 3, 6, 1, 4, 1, 476)."""
    return tuple(int(p) for p in oid.strip(".").split("."))

TIMEOUT_SECONDS = 2
RETRIES = 1
CONCURRENCY_LIMIT = 5              # nº de IPs em paralelo -- manter baixo
DELAY_BETWEEN_COMMUNITIES = 0.15   # segundos, evita rajada por IP

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
    site_id: str | None = None
    nome: str | None = None
    vendor_esperado: list[str] = field(default_factory=list)
    responded: bool = False
    working_communities: list[str] = field(default_factory=list)
    all_attempts: list[CommunityResult] = field(default_factory=list)
    sys_descr: str | None = None
    sys_object_id: str | None = None
    vendor_guess_oid: str | None = None
    vendor_guess_text: str | None = None
    vendor_confirmed_via: list[str] = field(default_factory=list)
    vendor_match: str | None = None  # "sim" / "nao" / "sem_dados"
    scanned_at: str = field(
        default_factory=lambda: datetime.now(timezone.utc).isoformat()
    )


# ----------------------------------------------------------------------------
# Carregamento de IPs a partir do CSV do registo mestre
# ----------------------------------------------------------------------------

@dataclass
class IpTarget:
    ip: str
    site_id: str | None = None
    nome: str | None = None
    vendor_esperado: list[str] = field(default_factory=list)


def load_targets(input_path: Path) -> list[IpTarget]:
    if not input_path.exists():
        log.warning(
            f"'{input_path}' não encontrado -- a usar IPS_FALLBACK ({len(IPS_FALLBACK)} IP(s))."
        )
        return [IpTarget(ip=ip) for ip in IPS_FALLBACK]

    by_ip: dict[str, IpTarget] = {}
    with open(input_path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            ip = (row.get("ip") or "").strip()
            if not ip:
                continue
            vendor = (row.get("vendor_esperado") or "").strip()
            if ip not in by_ip:
                by_ip[ip] = IpTarget(
                    ip=ip,
                    site_id=row.get("site_id"),
                    nome=row.get("nome"),
                )
            if vendor and vendor not in by_ip[ip].vendor_esperado:
                by_ip[ip].vendor_esperado.append(vendor)

    targets = list(by_ip.values())
    log.info(f"Carregados {len(targets)} IP(s) únicos de '{input_path}'.")

    conflitos = [t for t in targets if len(t.vendor_esperado) > 1]
    if conflitos:
        log.warning(
            f"{len(conflitos)} IP(s) aparecem no registo mestre com vendor "
            f"esperado diferente consoante a linha (ver coluna vendor_esperado "
            f"no CSV de saída):"
        )
        for t in conflitos:
            log.warning(f"  - {t.ip}: {t.vendor_esperado}")

    return targets


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


async def snmp_get(ip: str, community: str, oid: str, timeout: int, retries: int):
    """Faz um GET SNMP único. Retorna (varBinds | None, error_msg | None)."""
    try:
        transport = await UdpTransportTarget.create(
            (ip, 161), timeout=timeout, retries=retries
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
    """Identifica o vendor comparando sysObjectID com o enterprise OID de
    cada um, por prefixo de segmentos (não por substring de texto).

    Importante: um match por substring ("476" está contido em "4760...")
    dava falsos positivos -- ex. um sysObjectID "1.3.6.1.4.1.4760.1.1"
    (vendor qualquer com nº IANA 4760) seria erradamente atribuído à
    Vertive (nº IANA 476). Comparar tuplos de inteiros evita isto.
    """
    if not sys_object_id:
        return None
    try:
        got = _oid_tuple(sys_object_id)
    except ValueError:
        return None
    for vendor, info in VENDORS.items():
        prefix = _oid_tuple(info.enterprise_oid)
        if got[: len(prefix)] == prefix:
            return vendor
    return None


def guess_vendor_from_text(sys_descr: str | None) -> str | None:
    if not sys_descr:
        return None
    lowered = sys_descr.lower()
    for vendor, hints in VENDOR_TEXT_HINTS.items():
        if any(hint in lowered for hint in hints):
            return vendor
    return None


def compute_vendor_match(result: ScanResult) -> str:
    detected = {
        v for v in (result.vendor_guess_oid, result.vendor_guess_text) if v
    }
    detected.update(result.vendor_confirmed_via)
    if not result.vendor_esperado:
        return "sem_vendor_esperado"
    if not detected:
        return "sem_dados" if result.responded else "sem_resposta"
    esperado = {v.lower() for v in result.vendor_esperado}
    return "sim" if detected & esperado else "nao"


async def probe_community(
    ip: str, community: str, timeout: int, retries: int
) -> CommunityResult:
    varBinds, err = await snmp_get(ip, community, SYS_DESCR_OID, timeout, retries)

    if varBinds is None:
        return CommunityResult(community=community, ok=False, error=err)

    sys_descr = str(varBinds[0][1]) if varBinds else None

    obj_id_binds, _ = await snmp_get(
        ip, community, SYS_OBJECT_ID_OID, timeout, retries
    )
    sys_object_id = str(obj_id_binds[0][1]) if obj_id_binds else None

    return CommunityResult(
        community=community,
        ok=True,
        sys_descr=sys_descr,
        sys_object_id=sys_object_id,
    )


async def confirm_vendor_enterprise_oid(
    ip: str, community: str, timeout: int, retries: int
) -> list[str]:
    """Testa diretamente os ramos enterprise de cada vendor conhecido.
    Confirma resposta mesmo quando sysObjectID não é conclusivo."""
    confirmed = []
    for vendor, oid in VENDOR_ENTERPRISE_OIDS.items():
        varBinds, _ = await snmp_get(ip, community, oid, timeout, retries)
        if varBinds is not None:
            confirmed.append(vendor)
        await asyncio.sleep(DELAY_BETWEEN_COMMUNITIES)
    return confirmed


async def scan_ip(
    target: IpTarget,
    semaphore: asyncio.Semaphore,
    communities: list[str],
    timeout: int,
    retries: int,
    progress_fh,
    progress_lock: asyncio.Lock,
) -> ScanResult:
    async with semaphore:
        result = ScanResult(
            ip=target.ip,
            site_id=target.site_id,
            nome=target.nome,
            vendor_esperado=target.vendor_esperado,
        )
        log.info(f"[SCAN] {target.ip} (esperado: {target.vendor_esperado or '?'})")

        for community in communities:
            attempt = await probe_community(target.ip, community, timeout, retries)
            result.all_attempts.append(attempt)

            if attempt.ok:
                result.responded = True
                result.working_communities.append(community)

                if result.sys_descr is None:
                    result.sys_descr = attempt.sys_descr
                    result.sys_object_id = attempt.sys_object_id
                    result.vendor_guess_oid = guess_vendor_from_sys_object_id(
                        attempt.sys_object_id
                    )
                    result.vendor_guess_text = guess_vendor_from_text(
                        attempt.sys_descr
                    )

                log.info(f"  [OK] {target.ip} -> community='{community}'")
                if attempt.sys_descr:
                    log.info(f"        sysDescr: {attempt.sys_descr[:100]}")

                # Confirma vendor via enterprise OID direto (não interrompe
                # nem sai do loop de communities: outra community pode
                # também responder e vale registar).
                confirmed = await confirm_vendor_enterprise_oid(
                    target.ip, community, timeout, retries
                )
                for v in confirmed:
                    if v not in result.vendor_confirmed_via:
                        result.vendor_confirmed_via.append(v)

            await asyncio.sleep(DELAY_BETWEEN_COMMUNITIES)

        result.vendor_match = compute_vendor_match(result)

        if not result.responded:
            log.warning(f"  [FAIL] {target.ip} sem resposta SNMP em nenhuma community")
        elif result.vendor_match == "nao":
            log.warning(
                f"  [DISCREPANCIA] {target.ip}: esperado={result.vendor_esperado} "
                f"detetado_oid={result.vendor_guess_oid} "
                f"detetado_texto={result.vendor_guess_text} "
                f"confirmado={result.vendor_confirmed_via}"
            )

        # Grava incrementalmente para não perder o trabalho se o processo for
        # interrompido a meio de um scan grande.
        async with progress_lock:
            progress_fh.write(json.dumps(asdict(result), ensure_ascii=False) + "\n")
            progress_fh.flush()

        return result


# ----------------------------------------------------------------------------
# Execução principal + export
# ----------------------------------------------------------------------------

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--input",
        type=Path,
        default=Path("rectificadores_master.csv"),
        help="CSV com colunas ip[,vendor_esperado][,site_id][,nome]",
    )
    parser.add_argument(
        "--output-prefix",
        type=str,
        default="snmp_scan",
        help="Prefixo para os ficheiros de saída (csv/json/jsonl)",
    )
    parser.add_argument(
        "--concurrency",
        type=int,
        default=CONCURRENCY_LIMIT,
        help="Nº de IPs testados em paralelo (mantém baixo)",
    )
    parser.add_argument(
        "--timeout",
        type=int,
        default=TIMEOUT_SECONDS,
        help="Timeout (segundos) por pedido SNMP",
    )
    parser.add_argument(
        "--retries",
        type=int,
        default=RETRIES,
        help="Nº de retries por pedido SNMP",
    )
    parser.add_argument(
        "--community",
        action="append",
        default=[],
        help="Community extra a testar (pode repetir a flag)",
    )
    parser.add_argument(
        "--no-shuffle",
        action="store_true",
        help="Não baralhar a ordem de scan (por omissão é baralhada)",
    )
    return parser.parse_args()


async def main():
    args = parse_args()
    start = time.monotonic()

    targets = load_targets(args.input)
    if not targets:
        log.error("Nenhum IP para testar. A terminar.")
        sys.exit(1)

    if not args.no_shuffle:
        random.shuffle(targets)

    # dedup preservando ordem
    seen = set()
    communities = []
    for c in COMMUNITIES + args.community:
        if c not in seen:
            seen.add(c)
            communities.append(c)

    semaphore = asyncio.Semaphore(args.concurrency)
    progress_lock = asyncio.Lock()

    progress_path = f"{args.output_prefix}_progress.jsonl"
    with open(progress_path, "w", encoding="utf-8") as progress_fh:
        tasks = [
            scan_ip(
                t, semaphore, communities, args.timeout, args.retries,
                progress_fh, progress_lock,
            )
            for t in targets
        ]
        results: list[ScanResult] = await asyncio.gather(*tasks)

    elapsed = time.monotonic() - start

    # ---- JSON completo (inclui todas as tentativas, communities falhadas, etc.)
    json_path = f"{args.output_prefix}_results.json"
    with open(json_path, "w", encoding="utf-8") as f:
        json.dump([asdict(r) for r in results], f, indent=2, ensure_ascii=False)

    # ---- CSV resumo, uma linha por IP
    csv_path = f"{args.output_prefix}_results.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow([
            "ip", "site_id", "nome", "vendor_esperado", "responded",
            "working_communities", "sys_descr", "sys_object_id",
            "vendor_guess_oid", "vendor_guess_text", "vendor_confirmed_via",
            "vendor_match",
        ])
        for r in sorted(results, key=lambda r: r.ip):
            writer.writerow([
                r.ip,
                r.site_id or "",
                r.nome or "",
                ";".join(r.vendor_esperado),
                r.responded,
                ";".join(r.working_communities),
                (r.sys_descr or "").replace("\n", " ")[:200],
                r.sys_object_id or "",
                r.vendor_guess_oid or "",
                r.vendor_guess_text or "",
                ";".join(r.vendor_confirmed_via),
                r.vendor_match,
            ])

    # ---- Resumo no terminal
    total = len(results)
    ok = sum(1 for r in results if r.responded)
    fail = total - ok
    discrepancias = [r for r in results if r.vendor_match == "nao"]

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
        for ip in sorted(no_response):
            print(f"  - {ip}")

    if discrepancias:
        print(f"\nDiscrepâncias vendor esperado vs. detetado ({len(discrepancias)}):")
        for r in discrepancias:
            print(
                f"  - {r.ip} ({r.site_id or '?'}): esperado={r.vendor_esperado} "
                f"detetado_oid={r.vendor_guess_oid} texto={r.vendor_guess_text} "
                f"confirmado={r.vendor_confirmed_via}"
            )

    print(f"\nFicheiros gerados: {csv_path}, {json_path}, {progress_path}")


if __name__ == "__main__":
    asyncio.run(main())