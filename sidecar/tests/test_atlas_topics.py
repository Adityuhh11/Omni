import numpy as np

from omni_memory.atlas_topics import topics

WORDS = [
    "invoice payment amount",
    "mountain trail hiking",
    "flour butter recipe",
    "goroutine channel mutex",
]


def test_each_clump_is_named_by_the_words_only_it_uses():
    rng = np.random.default_rng(1)
    centres = np.array([[1, 0, 0], [-1, 0, 0], [0, 1, 0], [0, -1, 0]], dtype=np.float32)
    coords, texts = [], []
    for c, words in enumerate(WORDS):
        for n in range(60):
            coords.append(centres[c] + rng.normal(0, 0.05, 3))
            texts.append(f"{words} with the usual note number {n}")
    found = topics(np.array(coords, dtype=np.float32), texts)

    named = {t.label for t in found}
    for words in WORDS:
        assert any(set(label.split(" · ")) <= set(words.split()) for label in named), (words, named)
    assert all("the" not in t.label.split(" · ") for t in found)
    assert len(named) == len(found)  # a clump split in two is named once
    assert [t.count for t in found] == sorted((t.count for t in found), reverse=True)


def test_a_map_too_small_to_group_has_no_topics():
    assert topics(np.zeros((10, 3), dtype=np.float32), ["x"] * 10) == []
