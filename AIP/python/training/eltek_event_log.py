"""
Parser do export de log de eventos Eltek (formato "elteklogs.csv" real,
diferente do for_Data.csv).

Este ficheiro NÃO é uma série de métricas -- é um log de alarmes/eventos
com timestamp, exatamente o tipo de rótulo (label) que faltava para
treino supervisionado: "esta condição aconteceu, nesta hora, e durou X".

Estrutura real observada (múltiplas secções no mesmo ficheiro,
delimitadas por ';'):

    Eventlog
    #;Date/Time;Description;Event
    00001;2026-07-14 08:57:53 ;OutDoorTemp82.1 ;Event :On
    ...

    #;Date;Time;BatteryVoltage;BatteryCurrent;LoadCurrent;Rect.current;
    MainsVolt 1;MainsVolt 2;MainsVolt 3;Dc load power;BatteryTemp;
    BatteryRemCap
    ... (série de métricas real, com tensão de bateria -- que o
    for_Data.csv não tinha)

Tipos de evento reais confirmados: MainsLow (falha de rede elétrica),
MainsVolt 1.1 (tensão de rede fora de gama), BatteryTemp/BatteryTemp1.1
(temperatura de bateria), Door open (abertura de porta do abrigo --
segurança física), Genstart (arranque do gerador), SystemRst (reset do
sistema), OutDoorTemp82.1 (temperatura exterior), BatteryUsedCap
(capacidade de bateria usada, %).
"""

import re
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path


@dataclass
class EltekEvent:
    timestamp: datetime
    description: str
    event_kind: str   # ex.: "Major Alarm", "Minor Low", "Event"
    state: str         # "On" ou "Off" (ou "" para eventos sem estado, ex. BatteryUsedCap)


@dataclass
class OutageWindow:
    """Uma janela de falha real: description ficou "On" desde start_time
    até ser reportada "Off" em end_time. É isto que serve de rótulo
    (label) para treino supervisionado -- "havia uma falha real aqui"."""
    description: str
    start_time: datetime
    end_time: datetime

    @property
    def duration(self) -> timedelta:
        return self.end_time - self.start_time


def parse_eventlog(csv_path: str | Path, encoding: str = "latin-1") -> list[EltekEvent]:
    """
    Lê só a secção "Eventlog" (a primeira do ficheiro -- pára na secção
    seguinte, "EventlogBackup", para não duplicar). Linhas mal formadas
    são ignoradas silenciosamente (log de equipamento, formato pode
    variar entre firmwares).
    """
    path = Path(csv_path)
    events: list[EltekEvent] = []

    with path.open("r", encoding=encoding, errors="replace") as fh:
        in_eventlog = False
        for line in fh:
            stripped = line.strip()

            if stripped == "Eventlog":
                in_eventlog = True
                continue

            if stripped == "EventlogBackup":
                break

            if not in_eventlog:
                continue

            if stripped.startswith("#;"):
                continue

            parts = stripped.split(";")
            if len(parts) < 4:
                continue

            _, raw_dt, raw_desc, raw_event = parts[0], parts[1], parts[2], parts[3]

            try:
                ts = datetime.strptime(raw_dt.strip(), "%Y-%m-%d %H:%M:%S")
            except ValueError:
                continue

            description = raw_desc.strip()

            match = re.match(r"(?P<kind>.*?):(?P<state>\S*)", raw_event.strip())
            if match:
                kind = match.group("kind").strip()
                state = match.group("state").strip()
            else:
                kind, state = raw_event.strip(), ""

            events.append(EltekEvent(
                timestamp=ts,
                description=description,
                event_kind=kind,
                state=state,
            ))

    return events


def pair_outage_windows(events: list[EltekEvent], description: str) -> list[OutageWindow]:
    """
    Emparelha eventos "On"/"Off" da mesma descrição (ex. "MainsLow") em
    janelas de falha com duração real.
    """
    relevant = sorted(
        (e for e in events if e.description == description and e.state in ("On", "Off")),
        key=lambda e: e.timestamp,
    )

    windows: list[OutageWindow] = []
    pending_start: datetime | None = None

    for e in relevant:
        if e.state == "On":
            pending_start = e.timestamp
        elif e.state == "Off" and pending_start is not None:
            windows.append(OutageWindow(
                description=description,
                start_time=pending_start,
                end_time=e.timestamp,
            ))
            pending_start = None

    return windows