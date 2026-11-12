"""What the map shows of each point besides where it is: its file, place in it, type and age.

Read from the store each time the map is read (one query, no text), so a renamed or edited file
shows as it is now even before the map is laid out again.
"""

from __future__ import annotations

import json
from collections import Counter
from collections.abc import Mapping, Sequence
from dataclasses import dataclass
from pathlib import PurePosixPath

import numpy as np

TYPES_SHOWN = 8  # the map has eight distinct hues; the rest of the types share "other"
OTHER = "other"


@dataclass(frozen=True, slots=True)
class Fact:
    """One chunk as the map needs it."""

    uri: str
    source_id: str
    locator: str
    mtime_ns: int


@dataclass(frozen=True, slots=True)
class PointFacts:
    file: np.ndarray  # uint32 per point: points of one file share a number
    rank: np.ndarray  # uint32 per point: its place in reading order within its file
    type: np.ndarray  # uint16 per point: an index into types
    types: list[str]  # the commonest file types first, "other" last when there are more
    modified: np.ndarray  # uint32 per point: the file's last change, in seconds since 1970


def describe(ids: Sequence[str], facts: Mapping[str, Fact]) -> PointFacts:
    """The facts of the map's points, in its order; a chunk gone from the store is its own file."""
    n = len(ids)
    rows = [facts.get(id_) for id_ in ids]
    files: dict[tuple[str, str], int] = {}
    file = np.zeros(n, dtype="<u4")
    modified = np.zeros(n, dtype="<u4")
    kinds: list[str] = []
    for i, row in enumerate(rows):
        key = (row.source_id, row.uri) if row else ("", ids[i])
        file[i] = files.setdefault(key, len(files))
        modified[i] = row.mtime_ns // 1_000_000_000 if row else 0
        kinds.append(_type(row.uri) if row else OTHER)

    order = sorted(range(n), key=lambda i: (file[i], _position(rows[i]), ids[i]))
    rank = np.zeros(n, dtype="<u4")
    previous, place = -1, 0
    for i in order:
        place = place + 1 if file[i] == previous else 0
        previous = int(file[i])
        rank[i] = place

    common = [t for t, _ in Counter(k for k in kinds if k != OTHER).most_common()]
    shown = common[:TYPES_SHOWN]
    types = shown + ([OTHER] if len(common) > TYPES_SHOWN or OTHER in kinds else [])
    number = {t: i for i, t in enumerate(types)}
    other = number.get(OTHER, 0)
    type_ = np.array([number.get(k, other) for k in kinds], dtype="<u2")
    return PointFacts(file=file, rank=rank, type=type_, types=types, modified=modified)


def _type(uri: str) -> str:
    return PurePosixPath(uri).suffix.lower().lstrip(".") or "no extension"


def _position(row: Fact | None) -> float:
    """Where a chunk starts in its file: its first line, page or second (what chunkers write)."""
    if row is None:
        return 0
    try:
        at = json.loads(row.locator)
    except ValueError:
        return 0
    if not isinstance(at, dict):
        return 0
    lines = at.get("lines")
    if isinstance(lines, list) and lines:
        return float(lines[0])
    for key in ("page", "t"):
        if isinstance(at.get(key), int | float):
            return float(at[key])
    return 0
