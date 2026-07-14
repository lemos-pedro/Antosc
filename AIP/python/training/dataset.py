"""
training/dataset.py

Carrega o dataset histórico for_Data.csv (formato longo) e converte para
o mesmo formato que a inferência em produção usa
(inference.service.group_series): séries agrupadas por nome bruto de
métrica, ordenadas no tempo. Isto garante que o histórico usado para
validar/treinar modelos passa exatamente pelo mesmo caminho
(vendor.normalize -> feature_engineering.build_features) que os dados
em produção -- sem lógica duplicada ou divergente.

Formato de entrada (confirmado no projeto):
    ID; datainsercao; dataaquisicao; entidade; fonte; grandeza; valor

    entidade  -> tower_id (ex.: A111, A112, A114, A119, A122)
    fonte     -> vendor (eltek, comap, netsure, minipfsense)
    grandeza  -> nome bruto da métrica (ex.: battery_voltage_v)
    valor     -> valor numérico da leitura
    dataaquisicao -> timestamp da leitura em campo (usa-se este, não
                     datainsercao, que é apenas quando entrou no sistema)

Nota: "netsure" e "minipfsense" ainda não têm adaptador em vendors/ --
ver get_readable_vendors(). Linhas dessas fontes são carregadas (não se
perdem dados), mas ficam de fora da normalização até haver um vendor
dedicado ou confirmação de que mapeiam para um dos existentes.
"""

import csv
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path


@dataclass
class Reading:
    tower_id: str
    vendor: str
    metric: str
    value: float
    collected_at: datetime


# Formatos de data observados em exports deste tipo; tenta cada um por
# ordem. "%d/%m/%Y %H:%M" é o formato confirmado no for_Data.csv real
# (sem segundos, dia/mês/ano) -- fica primeiro por ser o mais comum.
_DATE_FORMATS = (
    "%d/%m/%Y %H:%M",
    "%d/%m/%Y %H:%M:%S",
    "%Y-%m-%d %H:%M:%S",
    "%Y-%m-%dT%H:%M:%S",
    "%Y-%m-%d",
)


def _parse_datetime(raw: str) -> datetime:
    raw = raw.strip()
    for fmt in _DATE_FORMATS:
        try:
            return datetime.strptime(raw, fmt)
        except ValueError:
            continue
    raise ValueError(f"formato de data não reconhecido: {raw!r}")


def load_readings(csv_path: str | Path, delimiter: str = ";") -> list[Reading]:
    """
    Lê o CSV linha a linha (não carrega tudo para memória de mais do que
    o necessário -- o ficheiro tem >1M linhas) e devolve a lista de
    leituras. Linhas com valor não-numérico ou data inválida são
    ignoradas e contadas em skipped, nunca silenciosamente convertidas
    para 0 (regra do projeto: nunca fabricar dados).
    """
    path = Path(csv_path)
    readings: list[Reading] = []
    skipped = 0

    with path.open("r", encoding="utf-8-sig", newline="") as fh:
        reader = csv.DictReader(fh, delimiter=delimiter)

        for row in reader:
            try:
                readings.append(
                    Reading(
                        tower_id=row["entidade"].strip(),
                        vendor=row["fonte"].strip().lower(),
                        metric=row["grandeza"].strip(),
                        value=float(str(row["valor"]).replace(",", ".")),
                        collected_at=_parse_datetime(row["dataaquisicao"]),
                    )
                )
            except (KeyError, ValueError, AttributeError):
                skipped += 1
                continue

    if skipped:
        print(f"load_readings: {skipped} linha(s) ignorada(s) (valor/data inválidos)")

    return readings


def group_by_tower(readings: list[Reading]) -> dict[str, list[Reading]]:
    """Agrupa leituras por torre, mantendo tudo o resto junto (vendor pode
    variar dentro da mesma torre se houver mais do que um equipamento)."""
    grouped: dict[str, list[Reading]] = {}
    for r in readings:
        grouped.setdefault(r.tower_id, []).append(r)
    return grouped


def readings_to_raw_series(readings: list[Reading]) -> dict[str, list[float]]:
    """
    Agrupa por nome bruto de métrica, ordenado no tempo -- exatamente o
    mesmo formato que inference.service.group_series produz a partir do
    towercore em produção.
    """
    ordered = sorted(readings, key=lambda r: r.collected_at)

    grouped: dict[str, list[float]] = {}
    for r in ordered:
        grouped.setdefault(r.metric, []).append(r.value)

    return grouped


def group_by_vendor(readings: list[Reading]) -> dict[str, list[Reading]]:
    """Agrupa leituras por vendor -- usado para normalizar cada subconjunto
    com o adaptador certo antes de juntar, em vez de escolher um único
    "vendor dominante" para a torre inteira (que descartaria em silêncio
    métricas de outras fontes, ex. o gerador ComAp numa torre cuja fonte
    principal é Eltek)."""
    grouped: dict[str, list[Reading]] = {}
    for r in readings:
        grouped.setdefault(r.vendor, []).append(r)
    return grouped


def canonical_series_for_tower(readings: list[Reading]) -> dict[str, list[float]]:
    """
    Normaliza as leituras de uma torre para séries canónicas, agrupando
    por vendor primeiro -- assim uma torre com Eltek (energia) + ComAp
    (gerador) mantém as métricas de ambos, em vez de perder as do vendor
    "não dominante". Fontes sem adaptador registado (ex. netsure,
    minipfsense -- ainda por confirmar em campo) são ignoradas aqui e
    reportadas, nunca silenciosamente inventadas.
    """
    from vendors.registry import get_vendor

    merged: dict[str, list[float]] = {}
    unmapped_vendors: set[str] = set()

    for vendor_name, vendor_readings in group_by_vendor(readings).items():
        try:
            vendor = get_vendor(vendor_name)
        except ValueError:
            unmapped_vendors.add(vendor_name)
            continue

        raw = readings_to_raw_series(vendor_readings)
        canonical = vendor.normalize(raw)

        for key, values in canonical.items():
            if not values:
                continue
            merged.setdefault(key, []).extend(values)

    if unmapped_vendors:
        print(
            "canonical_series_for_tower: fonte(s) sem adaptador, "
            f"ignorada(s): {sorted(unmapped_vendors)}"
        )

    return merged


def dominant_vendor(readings: list[Reading]) -> str | None:
    """
    Devolve o vendor mais frequente num conjunto de leituras de uma
    torre. Usado quando uma torre tem leituras de fontes distintas
    (ex. eltek + comap) e é preciso escolher qual adaptador de
    normalização aplicar às métricas de energia/bateria.
    """
    if not readings:
        return None

    counts: dict[str, int] = {}
    for r in readings:
        counts[r.vendor] = counts.get(r.vendor, 0) + 1

    return max(counts, key=counts.get)
