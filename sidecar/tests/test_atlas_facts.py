from omni_memory.atlas_facts import TYPES_SHOWN, Fact, describe


def fact(uri, locator="{}", mtime=5_000_000_000, source="s"):
    return Fact(uri=uri, source_id=source, locator=locator, mtime_ns=mtime)


def test_a_files_pieces_share_a_number_and_are_ranked_in_reading_order():
    facts = {
        "a2": fact("a.md", '{"lines": [40, 60]}'),
        "a1": fact("a.md", '{"lines": [1, 39]}'),
        "b": fact("b.pdf", '{"page": 3}'),
        "a3": fact("a.md", '{"lines": [61, 80]}'),
        "same-name-other-folder": fact("a.md", source="t"),
    }
    ids = list(facts)
    out = describe(ids, facts)
    file = dict(zip(ids, out.file.tolist(), strict=True))
    rank = dict(zip(ids, out.rank.tolist(), strict=True))
    assert file["a1"] == file["a2"] == file["a3"] != file["b"]
    assert file["same-name-other-folder"] != file["a1"]
    assert (rank["a1"], rank["a2"], rank["a3"], rank["b"]) == (0, 1, 2, 0)
    assert out.modified.tolist() == [5] * 5


def test_the_commonest_types_are_named_and_the_rest_are_other():
    extensions = [f"e{i}" for i in range(TYPES_SHOWN + 2)]
    facts = {f"{e}-{n}": fact(f"x{n}.{e}") for i, e in enumerate(extensions) for n in range(20 - i)}
    ids = [*facts, "gone"]  # a chunk the store no longer has
    out = describe(ids, facts)
    assert out.types == [*extensions[:TYPES_SHOWN], "other"]
    assert out.types[out.type[ids.index("e0-0")]] == "e0"
    assert out.types[out.type[ids.index(f"e{TYPES_SHOWN}-0")]] == "other"
    assert out.types[out.type[-1]] == "other"
