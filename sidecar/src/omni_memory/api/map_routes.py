"""The map's routes: build the atlas, read its points, find where a query lands."""

from __future__ import annotations

import base64
from typing import Annotated

import numpy as np
from fastapi import APIRouter, Depends, HTTPException, Request
from pydantic import BaseModel, Field

from omni_memory.api.deps import Engine, get_engine, get_store
from omni_memory.atlas import Atlas
from omni_memory.store import FlatStore

router = APIRouter(prefix="/map")

TEXT_PREVIEW = 600  # characters of a point's chunk the hover card can show


class MapStatusOut(BaseModel):
    state: str
    message: str
    count: int
    stale: bool


class MapOut(BaseModel):
    """Point i is at coords[3i:3i+3] (float32, little-endian) and belongs to roots[source[i]]."""

    coords: str  # base64
    source: str  # base64 of uint16 per point
    roots: list[str]


class MapPointOut(BaseModel):
    uri: str
    root: str
    kind: str
    locator: str
    text: str


class LocateIn(BaseModel):
    query: str = Field(min_length=1, max_length=2000)


class LocateOut(BaseModel):
    at: list[float]
    points: list[int]  # the hits it was placed among, best first


def get_atlas(request: Request) -> Atlas:
    return request.app.state.atlas


@router.get("/status")
def map_status(atlas: Annotated[Atlas, Depends(get_atlas)]) -> MapStatusOut:
    return MapStatusOut.model_validate(atlas.status())


@router.post("/build", status_code=202)
def build_map(atlas: Annotated[Atlas, Depends(get_atlas)]) -> MapStatusOut:
    atlas.build()
    return MapStatusOut.model_validate(atlas.status())


@router.get("")
def read_map(
    atlas: Annotated[Atlas, Depends(get_atlas)], store: Annotated[FlatStore, Depends(get_store)]
) -> MapOut:
    layout = atlas.layout()
    if layout is None:
        raise HTTPException(status_code=404, detail="there is no map yet")
    roots = {s.id: s.root for s in store.sources()}
    names = sorted(set(layout.sources))
    number = {name: i for i, name in enumerate(names)}
    source = np.array([number[s] for s in layout.sources], dtype="<u2")
    return MapOut(
        coords=base64.b64encode(layout.coords.astype("<f4").tobytes()).decode(),
        source=base64.b64encode(source.tobytes()).decode(),
        roots=[roots.get(name, "") for name in names],  # "" for a folder no longer searched
    )


@router.get("/points/{index}")
def read_point(
    index: int,
    atlas: Annotated[Atlas, Depends(get_atlas)],
    store: Annotated[FlatStore, Depends(get_store)],
) -> MapPointOut:
    id_ = atlas.point(index)
    chunk = None if id_ is None else store.get([id_]).get(id_)
    if chunk is None:
        raise HTTPException(status_code=404, detail="no such point (the index has changed)")
    roots = {s.id: s.root for s in store.sources()}
    return MapPointOut(
        uri=chunk.uri,
        root=roots.get(chunk.source_id, ""),
        kind=chunk.kind.value,
        locator=chunk.locator,
        text=chunk.text[:TEXT_PREVIEW],
    )


@router.post("/locate")
def locate(
    body: LocateIn,
    atlas: Annotated[Atlas, Depends(get_atlas)],
    engine: Annotated[Engine, Depends(get_engine)],
) -> LocateOut:
    found = atlas.locate(engine.searcher.search(body.query, k=40).hits)
    if found is None:
        raise HTTPException(status_code=404, detail="nothing on the map is near that")
    at, points = found
    return LocateOut(at=at, points=points)
