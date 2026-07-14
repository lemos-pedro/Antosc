"""
Parser do "DataLog.csv" -- export local de um sistema de energia DC
(cabeçalhos em inglês: System Voltage, Rectifier Current, Solar Total
Current, Load Current, Battery Current, Battery Temp, Ambient Temp,
Battery Capacity, Battery Status). Atribuído a Enetek neste projeto,
mas o formato em si não se identifica como fabricante -- confirmar.

Diferente do for_Data.csv em dois aspetos importantes:
  1. Tem TENSÃO DE BATERIA direta (System Voltage) -- em falta no
     histórico Eltek do for_Data.csv.
  2. Tem TEMPERATURA AMBIENTE separada da temperatura de bateria
     (Ambient Temp vs Battery Temp) -- o sinal que faltava do pfSense,
     só que já vem do próprio sistema de energia aqui.

Valores vêm como texto com unidade embutida ("49.2V", "22.6 deg.C",
"0Ah") -- não são floats diretos, por isso não podem ser lidos pelo
training/dataset.py genérico (esse espera "valor" já numérico).

"Battery Status" (Float / Discharge / ...) é um sinal categórico
valioso por si: "Discharge" enquanto a rede devia estar presente é o
mesmo tipo de evento que "MainsLow" no log Eltek -- serve de rótulo.
"""

import csv
import re
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path


@dataclass
class DataLogRow:
    timestamp: datetime
    system_voltage: float | None
    rectifier_current: float | None
    solar_current: float | None
    load_current: float | None
    battery_current: float | None
    battery_temp: float | None
    ambient_temp: float | None
    battery_capacity_ah: float | None
    battery_status: str


_NUMBER = re.compile(r"-?\d+(\.\d+)?")


def _extract_number(raw: str) -> float | None:
    """Extrai o número de um valor tipo "49.2V" ou "22.6 deg.C". Devolve
    None se não conseguir (nunca inventa 0 para um valor ilegível)."""
    if raw is None:
        return None
    match = _NUMBER.search(raw)
    if not match:
        return None
    return float(match.group())


def load_datalog(csv_path: str | Path) -> list[DataLogRow]:
    """
    Lê o DataLog.csv. Os nomes de coluna reais vêm com espaço normal no
    ficheiro mas por vezes chegam como espaço não separável (\\xa0,
    Excel/Windows) -- normalizamos antes de comparar.
    """
    path = Path(csv_path)
    rows: list[DataLogRow] = []

    with path.open("r", encoding="utf-8-sig", newline="") as fh:
        reader = csv.DictReader(fh)

        # normaliza nomes de coluna: \xa0 -> espaço normal, trim
        normalized_fields = {
            (f or "").replace("\xa0", " ").strip(): f
            for f in (reader.fieldnames or [])
        }

        def get(row: dict, name: str) -> str | None:
            original_key = normalized_fields.get(name)
            if original_key is None:
                return None
            return row.get(original_key)

        for row in reader:
            raw_dt = get(row, "Data Time")
            if not raw_dt:
                continue
            try:
                ts = datetime.strptime(raw_dt.strip(), "%Y-%m-%d %H:%M:%S")
            except ValueError:
                continue

            rows.append(DataLogRow(
                timestamp=ts,
                system_voltage=_extract_number(get(row, "System Voltage")),
                rectifier_current=_extract_number(get(row, "Rectifier Current")),
                solar_current=_extract_number(get(row, "Solar Total Current")),
                load_current=_extract_number(get(row, "Load Current")),
                battery_current=_extract_number(get(row, "Battery Current")),
                battery_temp=_extract_number(get(row, "Battery Temp")),
                ambient_temp=_extract_number(get(row, "Ambient Temp")),
                battery_capacity_ah=_extract_number(get(row, "Battery Capacity")),
                battery_status=(get(row, "Battery Status") or "").strip(),
            ))

    return rows


def to_canonical_series(rows: list[DataLogRow]) -> dict[str, list[float]]:
    """
    Converte para o mesmo formato canónico que feature_engineering.
    build_features() espera -- ordenado no tempo, valores só onde
    existem (não preenche com 0).
    """
    ordered = sorted(rows, key=lambda r: r.timestamp)

    series: dict[str, list[float]] = {
        "battery_voltage": [],
        "temperature": [],
        "ambient_temperature": [],
        "rectifier_current": [],
        "solar_current": [],
        "load_current": [],
        "battery_capacity_ah": [],
    }

    for r in ordered:
        if r.system_voltage is not None:
            series["battery_voltage"].append(r.system_voltage)
        if r.battery_temp is not None:
            series["temperature"].append(r.battery_temp)
        if r.ambient_temp is not None:
            series["ambient_temperature"].append(r.ambient_temp)
        if r.rectifier_current is not None:
            series["rectifier_current"].append(r.rectifier_current)
        if r.solar_current is not None:
            series["solar_current"].append(r.solar_current)
        if r.load_current is not None:
            series["load_current"].append(r.load_current)
        if r.battery_capacity_ah is not None:
            series["battery_capacity_ah"].append(r.battery_capacity_ah)

    return {k: v for k, v in series.items() if v}


def discharge_windows(rows: list[DataLogRow]):
    """
    Janelas em que Battery Status == "Discharge" -- a bateria está a
    alimentar a carga em vez de ficar em "Float" (carga de manutenção),
    o que normalmente significa falha de rede/retificador. Mesmo tipo
    de rótulo que MainsLow no log Eltek, só que vindo de outra fonte.
    """
    ordered = sorted(rows, key=lambda r: r.timestamp)

    windows = []
    start = None

    for r in ordered:
        is_discharge = r.battery_status.lower() == "discharge"
        if is_discharge and start is None:
            start = r.timestamp
        elif not is_discharge and start is not None:
            windows.append((start, r.timestamp))
            start = None

    if start is not None:
        windows.append((start, ordered[-1].timestamp))

    return windows
