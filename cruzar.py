#!/usr/bin/env python3
import json, re, csv
import pandas as pd

def normalize(s):
    if not isinstance(s, str):
        return ""
    s = s.strip().upper()
    repl = {"Á":"A","À":"A","Ã":"A","Â":"A","É":"E","Ê":"E","Í":"I","Ó":"O","Õ":"O","Ô":"O","Ú":"U","Ç":"C"}
    for a,b in repl.items():
        s = s.replace(a,b)
    s = re.sub(r"[_\-\s./]", "", s)
    return s

# 1. Towers reais (do JSON colado na conversa)
with open("towers.json") as f:
    towers_body = json.load(f)
towers = towers_body["data"]
print(f"Towers carregadas: {len(towers)}")

# 2. Registo Mestre
df = pd.read_excel("/mnt/user-data/uploads/antosc_registo_mestre_real.xlsx", sheet_name="Registo Mestre")
df = df[df["Site ID ANTOSC"].notna() & df["Nome"].notna()]
print(f"Registo Mestre linhas válidas: {len(df)}")

# 3. Hizima stations (do JSON colado na conversa anterior)
with open("hizima_stations.json") as f:
    hizima_body = json.load(f)
hizima_station_names = {s["stationName"].strip() for s in hizima_body["stations"] if s.get("stationName")}
print(f"Hizima stationName únicos: {len(hizima_station_names)}")

# Índice towers por nome normalizado
tower_by_norm = {}
for t in towers:
    norm = normalize(t.get("name", ""))
    if norm:
        tower_by_norm.setdefault(norm, []).append(t)

rows = []
pairs = []

for _, row in df.iterrows():
    site_id = str(row["Site ID ANTOSC"]).strip()
    nome = str(row["Nome"]).strip()
    norm_nome = normalize(nome)

    has_lock = site_id in hizima_station_names
    matches = tower_by_norm.get(norm_nome, [])

    if not has_lock:
        status = "sem_lock_hizima"
        tower_id = ""
    elif len(matches) == 1:
        status = "ok"
        tower_id = matches[0]["tower_id"]
        pairs.append(f"{tower_id}:{site_id}")
    elif len(matches) == 0:
        status = "torre_nao_encontrada_na_bd"
        tower_id = ""
    else:
        status = "ambiguo"
        tower_id = ";".join(m["tower_id"] for m in matches)

    rows.append({"site_id_antosc": site_id, "nome_excel": nome, "tower_id": tower_id, "status": status})

with open("relatorio_cruzamento.csv", "w", newline="", encoding="utf-8") as f:
    w = csv.DictWriter(f, fieldnames=["site_id_antosc","nome_excel","tower_id","status"])
    w.writeheader()
    w.writerows(rows)

station_map = ",".join(pairs)
with open("hizima_station_map.env", "w", encoding="utf-8") as f:
    f.write(f"HIZIMA_STATION_MAP={station_map}\n")

from collections import Counter
print("\nResumo:")
for status, count in Counter(r["status"] for r in rows).most_common():
    print(f"  {status}: {count}")

print(f"\nTotal pares gerados: {len(pairs)}")


pip install pandas openpyxl --break-system-packages
python3 cruzar.py