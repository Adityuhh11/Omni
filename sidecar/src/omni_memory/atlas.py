"""The atlas: every chunk of the index placed in 3D, so the index can be looked at as a map.

UMAP lays the vectors out (near in meaning is near in space). It is an optional extra
(`uv sync --extra map`) the launcher installs the first time someone opens the map, and it is
imported only here. A layout takes a minute or more, so it is built in the background and kept
next to the index (`atlas.npz`); it is shown until it is rebuilt, marked stale once the index
holds a different number of chunks.
"""

from __future__ import annotations

import importlib
import importlib.util
import logging
import os
import threading
import time
from dataclasses import dataclass
from pathlib import Path

import numpy as np

from omni_memory.atlas_topics import Topic, topics
from omni_memory.store import FlatStore
from omni_memory.types import SearchHit

logger = logging.getLogger(__name__)
progress = logging.getLogger("uvicorn.error")  # shows in the output the launcher keeps

MIN_POINTS = 16  # fewer than this is not a map, and UMAP's neighbourhoods need some room
LOCATE_FROM = 12  # the nearest hits a query is placed among


@dataclass(frozen=True, slots=True)
class Layout:
    """One built map: point i is chunk ids[i] of source sources[i], at coords[i] (unit ball)."""

    ids: list[str]
    sources: list[str]
    coords: np.ndarray  # (n, 3) float32
    topics: list[Topic]  # its regions, named


class Atlas:
    """Builds, keeps and answers questions about the map of one store."""

    def __init__(self, store: FlatStore, directory: Path) -> None:
        self._store = store
        self._path = directory / "atlas.npz"
        self._lock = threading.Lock()
        # none | building | ready | failed | missing (the extra is not installed)
        self._state = "none"
        self._message = ""
        self._layout: Layout | None = None
        self._index: dict[str, int] = {}
        self._load()

    def status(self) -> dict[str, object]:
        extra = installed()
        with self._lock:
            count = 0 if self._layout is None else len(self._layout.ids)
            return {
                "state": self._state,
                "message": self._message,
                "count": count,
                "stale": self._layout is not None and count != self._store.live_count(),
                "installed": extra,
            }

    def layout(self) -> Layout | None:
        with self._lock:
            return self._layout

    def build(self) -> None:
        """Start laying the index out again, unless that is already under way."""
        with self._lock:
            if self._state == "building":
                return
            self._state, self._message = "building", ""
        threading.Thread(target=self._build, name="atlas", daemon=True).start()

    def forget(self) -> bool:
        """Drop the map and its file (removing the map); False while one is being built."""
        with self._lock:
            if self._state == "building":
                return False
            self._path.unlink(missing_ok=True)
            self._layout, self._index, self._state, self._message = None, {}, "none", ""
            return True

    def point(self, index: int) -> str | None:
        """The chunk id of point `index`, or None when there is no such point."""
        with self._lock:
            if self._layout is None or not 0 <= index < len(self._layout.ids):
                return None
            return self._layout.ids[index]

    def locate(self, hits: list[SearchHit]) -> tuple[list[float], list[int]] | None:
        """Where a query lands: among the points of its best hits, pulled toward the closest.

        UMAP could place a new vector itself, but only with the fitted model kept around (large,
        and slow to load); the weighted middle of the nearest points lands in the same place.
        """
        with self._lock:
            if self._layout is None:
                return None
            found = [(self._index[h.id], h.score) for h in hits if h.id in self._index]
            coords = self._layout.coords
        found = found[:LOCATE_FROM]
        if not found:
            return None
        points = [i for i, _ in found]
        scores = np.array([s for _, s in found], dtype=np.float32)
        weights = np.exp((scores - scores.max()) * 20)  # a few hundredths of cosine decide a lot
        middle = (coords[points] * weights[:, None]).sum(axis=0) / weights.sum()
        return [float(v) for v in middle], points

    # --- building and the file -----------------------------------------------------------

    def _build(self) -> None:
        try:
            ids, sources, vectors = self._store.every_vector()
            if len(ids) < MIN_POINTS:
                raise _Refused(f"the index holds {len(ids)} chunks; a map needs {MIN_POINTS}")
            started = time.monotonic()
            coords = _lay_out(vectors)
            laid_out = time.monotonic()
            chunks = self._store.get(ids)
            named = topics(coords, [c.text if (c := chunks.get(i)) else "" for i in ids])
            progress.info(
                "map: %d chunks laid out in %.1f s, %d regions named in %.1f s",
                len(ids),
                laid_out - started,
                len(named),
                time.monotonic() - laid_out,
            )
            layout = Layout(ids=ids, sources=sources, coords=coords, topics=named)
            np.savez(
                self._path,
                ids=np.array(ids),
                sources=np.array(sources),
                coords=coords,
                topic_labels=np.array([t.label for t in named], dtype=str),
                topic_at=np.array([t.at for t in named], dtype=np.float32).reshape(-1, 3),
                topic_counts=np.array([t.count for t in named], dtype=np.int64),
            )
            self._keep(layout, "ready", "")
        except _Missing as error:
            self._keep(self._layout, "missing", str(error))
        except _Refused as error:
            self._keep(self._layout, "failed", str(error))
        except Exception as error:
            logger.exception("building the map failed")
            self._keep(self._layout, "failed", f"{type(error).__name__}: {error}")

    def _keep(self, layout: Layout | None, state: str, message: str) -> None:
        with self._lock:
            self._layout, self._state, self._message = layout, state, message
            self._index = {} if layout is None else {id_: i for i, id_ in enumerate(layout.ids)}

    def _load(self) -> None:
        if not self._path.exists():
            return
        try:
            with np.load(self._path) as saved:
                layout = Layout(
                    ids=[str(i) for i in saved["ids"]],
                    sources=[str(s) for s in saved["sources"]],
                    coords=saved["coords"].astype(np.float32),
                    topics=[
                        Topic(str(label), (float(at[0]), float(at[1]), float(at[2])), int(count))
                        for label, at, count in zip(
                            saved["topic_labels"],
                            saved["topic_at"],
                            saved["topic_counts"],
                            strict=True,
                        )
                    ],
                )
        except Exception:
            logger.exception("the saved map cannot be read; it will be built again")
            return
        self._keep(layout, "ready", "")


class _Missing(Exception):
    """The map's extra (UMAP) is not installed."""


class _Refused(Exception):
    """The index cannot be mapped (too small); says why."""


def installed() -> bool:
    """Whether the map's extra is installed (found, not imported: importing it costs seconds)."""
    importlib.invalidate_caches()
    return importlib.util.find_spec("umap") is not None


def _lay_out(vectors: np.ndarray) -> np.ndarray:
    try:
        # The optional extra, imported by name so the type check passes where it is not
        # installed; importing it costs seconds.
        importlib.invalidate_caches()  # the launcher may have installed it a moment ago
        umap = importlib.import_module("umap.umap_")
    except ImportError as error:
        raise _Missing("the map's layout library is not installed") from error
    # Half the cores: the launcher, a search and an index job keep running beside it.
    reducer = umap.UMAP(
        n_components=3,
        metric="cosine",
        n_neighbors=15,
        min_dist=0.1,
        compatibility_layout=True,  # the layout this was tuned on; the 0.6 default changes later
        low_memory=True,
        n_jobs=max(1, (os.cpu_count() or 2) // 2),
    )
    coords = np.asarray(reducer.fit_transform(vectors), dtype=np.float32)
    coords -= coords.mean(axis=0)
    radius = float(np.percentile(np.linalg.norm(coords, axis=1), 99)) or 1.0
    return coords / radius  # inside the unit ball, a few outliers just past it
