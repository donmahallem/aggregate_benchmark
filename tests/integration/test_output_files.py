"""
Asserts that all expected static + data files were written to the pages dir.
Usage: python test_output_files.py <pages_dir>
"""

import sys
import os

EXPECTED = ["data.json", "index.html", "app.js", "style.css"]


def main(pages_dir: str) -> None:
    missing = [f for f in EXPECTED if not os.path.isfile(os.path.join(pages_dir, f))]
    if missing:
        for f in missing:
            print(f"FAIL: missing {os.path.join(pages_dir, f)}", file=sys.stderr)
        sys.exit(1)
    print(f"OK: all expected files present in {pages_dir}")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <pages_dir>", file=sys.stderr)
        sys.exit(2)
    main(sys.argv[1])
