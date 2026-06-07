#!/usr/bin/env python3
"""Update the statement-coverage baseline in docs/api-coverage.md."""

from __future__ import annotations

import argparse
import re
import subprocess
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
API_COVERAGE_PATH = ROOT / "docs" / "api-coverage.md"
DEFAULT_COVERPROFILE = Path("/tmp/graviteeam-coverage.out")
COVERAGE_LINE_RE = re.compile(
    r"(\| `go test \./\.\.\. -coverprofile=/tmp/graviteeam-coverage\.out -covermode=atomic` \| `)"
    r"(\d+\.\d+%)"
    r"(` total statement coverage \|)"
)


def total_coverage(go: str, coverprofile: Path) -> str:
    result = subprocess.run(
        [go, "tool", "cover", f"-func={coverprofile}"],
        check=True,
        cwd=ROOT,
        text=True,
        capture_output=True,
    )
    for line in reversed(result.stdout.splitlines()):
        if line.startswith("total:"):
            fields = line.split()
            if fields:
                return fields[-1]
    raise RuntimeError(f"could not find total coverage in {coverprofile}")


def update_doc(coverage: str) -> bool:
    text = API_COVERAGE_PATH.read_text(encoding="utf-8")
    updated, count = COVERAGE_LINE_RE.subn(rf"\g<1>{coverage}\g<3>", text, count=1)
    if count != 1:
        raise RuntimeError(f"could not find coverage baseline row in {API_COVERAGE_PATH}")
    if updated == text:
        return False
    API_COVERAGE_PATH.write_text(updated, encoding="utf-8")
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--coverprofile", type=Path, default=DEFAULT_COVERPROFILE)
    parser.add_argument("--go", default="go")
    args = parser.parse_args()

    coverage = total_coverage(args.go, args.coverprofile)
    changed = update_doc(coverage)
    status = "updated" if changed else "already current"
    print(f"{API_COVERAGE_PATH.relative_to(ROOT)} coverage baseline {status}: {coverage}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
