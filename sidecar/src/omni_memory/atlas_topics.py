"""Topic labels for the map: its points grouped into regions, each named by its telling words.

The regions are k-means clusters of the laid-out points (what the eye sees as a clump), not of the
vectors. A region's words are those most of its chunks use and few chunks elsewhere do (a
class-based TF-IDF): "invoice · payment" for a clump of bills, not "the · and".
"""

from __future__ import annotations

import math
import re
from collections import Counter
from collections.abc import Sequence
from dataclasses import dataclass

import numpy as np

WORD = re.compile(r"[a-z][a-z]{2,19}")  # letters only: no numbers, hashes or ids
WORDS_PER_LABEL = 3
MIN_SHARE = 0.04  # a word needs to be in this share of a region's chunks to name it
# Words common in any writing or code; the TF-IDF would mostly push them down anyway, but a
# small region could still be named "with · this".
STOP = frozenset(
    [
        "the",
        "and",
        "for",
        "are",
        "but",
        "not",
        "you",
        "all",
        "any",
        "can",
        "had",
        "her",
        "was",
        "one",
        "our",
        "out",
        "has",
        "have",
        "him",
        "his",
        "how",
        "its",
        "may",
        "new",
        "now",
        "old",
        "see",
        "two",
        "way",
        "who",
        "did",
        "get",
        "let",
        "put",
        "say",
        "she",
        "too",
        "use",
        "that",
        "with",
        "this",
        "from",
        "they",
        "will",
        "would",
        "there",
        "their",
        "what",
        "about",
        "which",
        "when",
        "make",
        "like",
        "time",
        "just",
        "know",
        "take",
        "into",
        "year",
        "your",
        "some",
        "could",
        "them",
        "than",
        "then",
        "look",
        "only",
        "come",
        "over",
        "think",
        "also",
        "back",
        "after",
        "work",
        "first",
        "well",
        "even",
        "want",
        "because",
        "these",
        "give",
        "most",
        "very",
        "been",
        "were",
        "said",
        "each",
        "more",
        "other",
        "such",
        "here",
        "where",
        "much",
        "should",
        "those",
        "while",
        "being",
        "both",
        "same",
        "between",
        "under",
        "again",
        "does",
        "done",
        "none",
        "true",
        "false",
        "null",
        "self",
        "return",
        "import",
        "from",
        "def",
        "class",
        "func",
        "var",
        "const",
        "let",
        "int",
        "str",
        "string",
        "type",
        "else",
        "elif",
        "try",
        "except",
        "finally",
        "pass",
        "break",
        "continue",
        "lambda",
        "yield",
        "async",
        "await",
        "public",
        "private",
        "static",
        "void",
        "new",
        "http",
        "https",
        "www",
        "com",
        "org",
        "html",
        "png",
        "jpg",
        "pdf",
        "file",
        "files",
        "page",
        "line",
        "lines",
        "text",
    ]
)


@dataclass(frozen=True, slots=True)
class Topic:
    label: str
    at: tuple[float, float, float]
    count: int


def topics(coords: np.ndarray, texts: Sequence[str]) -> list[Topic]:
    """Regions of the map with their names, largest first; none for a map too small to group."""
    k = min(24, max(6, len(coords) // 1500))
    if len(coords) < k * 4:
        return []
    groups = _kmeans(coords, k)
    words = _region_words(groups, texts, k)
    found: list[Topic] = []
    for region in range(k):
        members = np.flatnonzero(groups == region)
        if len(members) == 0 or not words[region]:
            continue
        at = coords[members].mean(axis=0)
        label = " · ".join(words[region])
        found.append(Topic(label, (float(at[0]), float(at[1]), float(at[2])), len(members)))
    # A large clump split into two clusters would carry the same name twice: keep the bigger.
    kept: dict[str, Topic] = {}
    for topic in sorted(found, key=lambda t: -t.count):
        kept.setdefault(topic.label, topic)
    return list(kept.values())


def _kmeans(points: np.ndarray, k: int, rounds: int = 30) -> np.ndarray:
    """Which of k clusters each point is in: k-means++ seeds, then Lloyd's rounds (fixed seed: a
    rebuild of the same index names the same regions)."""
    rng = np.random.default_rng(0)
    centres = [points[rng.integers(len(points))]]
    for _ in range(1, k):
        far = np.min(((points[:, None, :] - np.array(centres)[None]) ** 2).sum(-1), axis=1)
        centres.append(points[rng.choice(len(points), p=far / far.sum())])
    centre = np.array(centres)
    groups = np.zeros(len(points), dtype=np.intp)
    for _ in range(rounds):
        groups = ((points[:, None, :] - centre[None]) ** 2).sum(-1).argmin(axis=1)
        moved = centre.copy()
        for c in range(k):
            members = points[groups == c]
            if len(members):
                moved[c] = members.mean(axis=0)
        if np.allclose(moved, centre):
            break
        centre = moved
    return groups


def _region_words(groups: np.ndarray, texts: Sequence[str], k: int) -> list[list[str]]:
    in_region = [Counter[str]() for _ in range(k)]
    everywhere = Counter[str]()
    for region, text in zip(groups, texts, strict=True):
        seen = {w for w in WORD.findall(text.lower()) if w not in STOP}
        in_region[region].update(seen)
        everywhere.update(seen)
    total = len(texts)
    sizes = np.bincount(groups, minlength=k)
    named: list[list[str]] = []
    for region in range(k):
        size = int(sizes[region])
        least = max(3, math.ceil(size * MIN_SHARE))
        scored = sorted(
            (
                (n / size) * math.log(total / everywhere[w]),
                w,
            )
            for w, n in in_region[region].items()
            if n >= least
        )
        chosen: list[str] = []
        for _, word in reversed(scored):
            if any(word.startswith(c) or c.startswith(word) for c in chosen):
                continue  # "invoice" and "invoices" are one word to a reader
            chosen.append(word)
            if len(chosen) == WORDS_PER_LABEL:
                break
        named.append(chosen)
    return named
