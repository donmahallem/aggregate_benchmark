"""
Validates data.json produced by the action:
  - non-empty array
  - expected entry count (seeded + 1)
  - latest entry has the expected hash
  - latest entry contains measurements for expected languages
  - no duplicate measurements in the latest entry
  - entries are sorted chronologically

Usage: python test_data_json.py <pages_dir> <expected_hash> <expected_total_entries>
"""

import json
import sys
import os


def fail(msg: str) -> None:
    print(f"FAIL: {msg}", file=sys.stderr)
    sys.exit(1)


def measurement_key(m: dict) -> str:
    return json.dumps(
        {
            "l": m.get("language", ""),
            "d": m.get("day"),
            "y": m.get("year"),
            "p": m.get("part"),
            "g": m.get("group_key", ""),
            "desc": m.get("description", ""),
        },
        sort_keys=True,
    )


def main(pages_dir: str, expected_hash: str, expected_count: int) -> None:
    path = os.path.join(pages_dir, "data.json")

    with open(path, encoding="utf-8") as fh:
        data = json.load(fh)

    if not isinstance(data, list) or len(data) == 0:
        fail("data.json is not a non-empty array")

    if len(data) != expected_count:
        fail(f"expected {expected_count} entries, got {len(data)}")

    latest = data[-1]

    if latest.get("hash") != expected_hash:
        fail(f"expected hash {expected_hash!r}, got {latest.get('hash')!r}")

    measurements = latest.get("measurements") or []
    if not measurements:
        fail("latest entry has no measurements")

    langs = sorted({m.get("language") for m in measurements})
    for required in ("go", "python"):
        if required not in langs:
            fail(f"expected language {required!r} in latest entry, got {langs}")

    keys = [measurement_key(m) for m in measurements]
    if len(set(keys)) != len(keys):
        fail(f"duplicate measurements: {len(keys)} total, {len(set(keys))} unique")

    for i in range(1, len(data)):
        if data[i]["timestamp"] < data[i - 1]["timestamp"]:
            fail(f"entries out of chronological order at index {i}")

    print(
        f"OK: {len(data)} entries, latest hash={expected_hash!r}, "
        f"languages={langs}, no duplicates, sorted"
    )


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(
            f"Usage: {sys.argv[0]} <pages_dir> <expected_hash> <expected_total_entries>",
            file=sys.stderr,
        )
        sys.exit(2)
    main(sys.argv[1], sys.argv[2], int(sys.argv[3]))
