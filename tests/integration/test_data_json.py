"""
Validates data.json produced by the action:
  - object with "history" key containing a non-empty array
  - expected entry count (seeded + 1)
  - latest entry has the expected hash
  - latest entry contains measurements for expected series_keys
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
            "g": m.get("group_key", ""),
            "s": m.get("series_key", ""),
        },
        sort_keys=True,
    )


def main(pages_dir: str, expected_hash: str, expected_count: int) -> None:
    path = os.path.join(pages_dir, "data.json")

    with open(path, encoding="utf-8") as fh:
        data = json.load(fh)

    if not isinstance(data, dict) or "history" not in data:
        fail("data.json is not an object with a 'history' key")

    history = data["history"]
    if not isinstance(history, list) or len(history) == 0:
        fail("data.json history is not a non-empty array")

    if len(history) != expected_count:
        fail(f"expected {expected_count} entries, got {len(history)}")

    latest = history[-1]

    if latest.get("hash") != expected_hash:
        fail(f"expected hash {expected_hash!r}, got {latest.get('hash')!r}")

    measurements = latest.get("measurements") or []
    if not measurements:
        fail("latest entry has no measurements")

    series_keys = sorted({m.get("series_key") for m in measurements})
    for required in ("go", "python"):
        if required not in series_keys:
            fail(f"expected series_key {required!r} in latest entry, got {series_keys}")

    keys = [measurement_key(m) for m in measurements]
    if len(set(keys)) != len(keys):
        fail(f"duplicate measurements: {len(keys)} total, {len(set(keys))} unique")

    for i in range(1, len(history)):
        if history[i]["timestamp"] < history[i - 1]["timestamp"]:
            fail(f"entries out of chronological order at index {i}")

    print(
        f"OK: {len(history)} entries, latest hash={expected_hash!r}, "
        f"series_keys={series_keys}, no duplicates, sorted"
    )


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(
            f"Usage: {sys.argv[0]} <pages_dir> <expected_hash> <expected_total_entries>",
            file=sys.stderr,
        )
        sys.exit(2)
    main(sys.argv[1], sys.argv[2], int(sys.argv[3]))
