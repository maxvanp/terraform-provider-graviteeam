#!/usr/bin/env python3

import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
VERIFIER = ROOT / "scripts" / "verify-release-artifacts.sh"


def run(verifier_dist: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [str(VERIFIER), str(verifier_dist)],
        cwd=ROOT,
        text=True,
        capture_output=True,
        check=False,
    )


def checksum_file(dist: Path) -> Path:
    matches = list(dist.glob("terraform-provider-graviteeam_*_SHA256SUMS"))
    assert len(matches) == 1, f"expected one checksum file, found {matches}"
    return matches[0]


def archive_line(lines: list[str]) -> str:
    return next(line for line in lines if line.rstrip().endswith(".zip"))


def expect_failure(dist: Path, message: str) -> None:
    result = run(dist)
    assert result.returncode != 0, f"{message} was accepted:\n{result.stdout}{result.stderr}"


def main() -> None:
    dist = Path(sys.argv[1]) if len(sys.argv) > 1 else ROOT / "dist"
    dist = dist.resolve()
    baseline = run(dist)
    assert baseline.returncode == 0, f"baseline verification failed:\n{baseline.stdout}{baseline.stderr}"

    with tempfile.TemporaryDirectory(prefix="graviteeam-artifacts-", dir=dist.parent) as tmp:
        copied = Path(tmp) / "dist"
        shutil.copytree(dist, copied, copy_function=os.link)
        sums = checksum_file(copied)
        lines = sums.read_text().splitlines(keepends=True)
        target = archive_line(lines)

        sums.unlink()
        sums.write_text("".join(line for line in lines if line != target))
        expect_failure(copied, "missing archive checksum")

        sums.unlink()
        sums.write_text("".join(
            re.sub(r"^[0-9a-fA-F]{64}", "0" * 64, line, count=1) if line == target else line
            for line in lines
        ))
        expect_failure(copied, "corrupt archive checksum")

    print("release artifact regression checks passed")


if __name__ == "__main__":
    main()
