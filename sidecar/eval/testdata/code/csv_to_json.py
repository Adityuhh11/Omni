"""Convert a CSV file with a header row into a JSON array of objects."""
import csv
import json
import sys


def convert(src: str, dst: str) -> int:
    with open(src, newline="", encoding="utf-8") as f:
        rows = list(csv.DictReader(f))
    with open(dst, "w", encoding="utf-8") as out:
        json.dump(rows, out, indent=2, ensure_ascii=False)
    return len(rows)


if __name__ == "__main__":
    print(convert(sys.argv[1], sys.argv[2]), "rows written")
