import asyncio
import csv
from pysnmp.hlapi.v3arch.asyncio import *

IPS = [
    "192.168.150.5",
    "192.168.137.5",
    "192.168.156.5",
    "192.168.206.133",
    "192.168.113.5",
    "192.168.112.5",
    "192.168.149.5",
    "192.168.141.5",
    "192.168.210.5",
    "192.168.201.165",
    "192.168.202.101",
    "192.168.206.37",
    "192.168.129.5",
    "192.168.128.5",
    "192.168.102.5",
    "192.168.201.229",
    "192.168.205.197",
    "192.168.165.5",
    "192.168.200.133",
    "192.168.203.101",
    "192.168.178.5",
    "192.168.207.37",
    "192.168.164.5",
    "192.168.170.5",
    "192.168.204.197",
    "192.168.133.5",
    "192.168.159.5",
    "192.168.203.5",
    "192.168.118.5"
]

COMMUNITIES = [
    "public",
    "private",
    "admin",
    "Admin",
    "snmp"
]

VENDOR_MAP = {
    "2011": "Huawei",
    "12148": "Eltek",
    "53318": "Enetek",
    "476": "Vertiv"
}


async def get_sysinfo(ip, community):
    try:
        iterator = get_cmd(
            SnmpEngine(),
            CommunityData(community),
            await UdpTransportTarget.create((ip, 161), timeout=2, retries=1),
            ContextData(),
            ObjectType(ObjectIdentity("1.3.6.1.2.1.1.2.0")),  # sysObjectID
            ObjectType(ObjectIdentity("1.3.6.1.2.1.1.1.0"))   # sysDescr
        )

        errorIndication, errorStatus, errorIndex, varBinds = await iterator

        if errorIndication or errorStatus:
            return None

        sysObjectID = str(varBinds[0][1])
        sysDescr = str(varBinds[1][1])

        vendor = "Unknown"

        for oid, name in VENDOR_MAP.items():
            if oid in sysObjectID:
                vendor = name
                break

        return {
            "ip": ip,
            "community": community,
            "vendor": vendor,
            "sysObjectID": sysObjectID,
            "sysDescr": sysDescr
        }

    except Exception:
        return None


async def scan_ip(ip):
    for community in COMMUNITIES:
        result = await get_sysinfo(ip, community)
        if result:
            print(f"[OK] {ip} -> {result['vendor']}")
            return result

    print(f"[FAIL] {ip}")
    return {
        "ip": ip,
        "community": "-",
        "vendor": "NO_RESPONSE",
        "sysObjectID": "-",
        "sysDescr": "-"
    }


async def main():
    tasks = [scan_ip(ip) for ip in IPS]
    results = await asyncio.gather(*tasks)

    with open("retificadores.csv", "w", newline="", encoding="utf-8") as f:
        writer = csv.DictWriter(
            f,
            fieldnames=[
                "ip",
                "community",
                "vendor",
                "sysObjectID",
                "sysDescr"
            ]
        )
        writer.writeheader()
        writer.writerows(results)

    print("\nScan terminado -> retificadores.csv")


if __name__ == "__main__":
    asyncio.run(main())