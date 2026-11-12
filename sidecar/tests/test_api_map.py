import base64

import numpy as np
import pytest

from api_helpers import index, running_app
from helpers import wait_for

pytest.importorskip("umap", reason="the map's extra is not installed (uv sync --extra map)")

TOPICS = ["invoice payment due", "hiking trail mountain", "python function return", "recipe flour"]


@pytest.fixture
def many(tmp_path):
    root = tmp_path / "many"
    root.mkdir()
    for n in range(24):
        (root / f"note{n}.md").write_text(f"{TOPICS[n % 4]} number {n}")
    return root


def built(api):
    assert api.post("/map/build").status_code == 202
    return wait_for(lambda: (s := api.get("/map/status").json())["state"] != "building" and s, 120)


def test_the_map_lays_every_chunk_out_and_answers_hover_and_locate(api, many):
    assert api.get("/map/status").json()["state"] == "none"
    assert api.get("/map").status_code == 404
    index(api, many)
    status = built(api)
    assert (status["state"], status["count"], status["stale"]) == ("ready", 24, False), status

    body = api.get("/map").json()
    coords = np.frombuffer(base64.b64decode(body["coords"]), dtype="<f4").reshape(-1, 3)
    source = np.frombuffer(base64.b64decode(body["source"]), dtype="<u2")
    assert coords.shape == (24, 3) and np.isfinite(coords).all()
    assert set(source) == {0} and body["roots"] == [str(many)]
    file = np.frombuffer(base64.b64decode(body["file"]), dtype="<u4")
    assert len(set(file)) == 24 and body["types"] == ["md"]
    assert all(t["label"] and len(t["at"]) == 3 for t in body["topics"]), body["topics"]

    point = api.get("/map/points/0").json()
    assert point["root"] == str(many) and point["text"]
    assert api.get("/map/points/24").status_code == 404

    found = api.post("/map/locate", json={"query": "mountain trail"}).json()
    assert len(found["at"]) == 3 and found["points"]


def test_a_saved_map_comes_back_and_is_stale_once_the_index_changes(tmp_path, many):
    with running_app(tmp_path / "idx") as api:
        index(api, many)
        built(api)
    (many / "extra.md").write_text("something new entirely")
    with running_app(tmp_path / "idx") as api:
        assert api.get("/map/status").json() == {
            "state": "ready",
            "message": "",
            "count": 24,
            "stale": False,
            "installed": True,
        }
        index(api, many)
        assert api.get("/map/status").json()["stale"] is True


def test_too_small_an_index_is_refused_with_a_reason(api, folder):
    index(api, folder)
    status = built(api)
    assert status["state"] == "failed" and "needs" in status["message"]


def test_removing_the_map_forgets_it_and_its_file(tmp_path, many):
    with running_app(tmp_path / "idx") as api:
        index(api, many)
        built(api)
        assert api.delete("/map").status_code == 204
        assert api.get("/map/status").json()["state"] == "none"
        assert api.get("/map").status_code == 404
    assert not (tmp_path / "idx" / "atlas.npz").exists()
