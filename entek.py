import asyncio
import csv
from pysnmp.hlapi.v3arch.asyncio import *

IPS = [
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
]

COMMUNITIES = [
    "community1",
    "public",
    "Public"
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