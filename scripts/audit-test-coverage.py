#!/usr/bin/env python3
"""Audit provider resource and data source acceptance-test coverage.

The OpenAPI coverage audit answers "is a family represented?". This script
answers the next question: "is each Terraform type exercised through the local
acceptance-test harness?".
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RESOURCE_ROOT = ROOT / "internal" / "resources"
DATASOURCE_ROOT = ROOT / "internal" / "datasources"

TYPE_NAME_RE = re.compile(r'resp\.TypeName\s*=\s*req\.ProviderTypeName\s*\+\s*"_(.+?)"')
FACTORY_RE = re.compile(r"func\s+(New\w+)\s*\(")
TEST_ACC_RE = re.compile(r"\bfunc\s+TestAcc\w+|resource\.Test\s*\(")
IMPORT_RE = re.compile(r"\bImportState\s*:\s*true\b")
IMPORT_CAPABILITY_RE = re.compile(r"ResourceWithImportState|func\s+\(r\s+\*\w+\)\s+ImportState\s*\(")


@dataclass(frozen=True)
class TerraformType:
    kind: str
    name: str
    package_dir: Path
    factory: str | None
    import_capable: bool
    has_acceptance: bool
    has_import_test: bool


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def package_go_files(package_dir: Path) -> list[Path]:
    return sorted(
        path
        for path in package_dir.glob("*.go")
        if not path.name.endswith("_test.go") and not path.name.endswith("_model.go")
    )


def test_text(package_dir: Path) -> str:
    return "\n".join(read(path) for path in sorted(package_dir.glob("*_test.go")))


def discover(root: Path, kind: str) -> list[TerraformType]:
    types: list[TerraformType] = []
    for package_dir in sorted(path for path in root.iterdir() if path.is_dir()):
        source = "\n".join(read(path) for path in package_go_files(package_dir))
        tests = test_text(package_dir)
        factories = FACTORY_RE.findall(source)
        type_names = sorted(set(TYPE_NAME_RE.findall(source)))
        for name in type_names:
            types.append(
                TerraformType(
                    kind=kind,
                    name=f"graviteeam_{name}",
                    package_dir=package_dir,
                    factory=factories[0] if factories else None,
                    import_capable=kind == "resource" and bool(IMPORT_CAPABILITY_RE.search(source)),
                    has_acceptance=bool(TEST_ACC_RE.search(tests)),
                    has_import_test=bool(IMPORT_RE.search(tests)),
                )
            )
    return types


def print_table(title: str, rows: list[str]) -> None:
    print(f"\n## {title}")
    if not rows:
        print("none")
        return
    for row in rows:
        print(row)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail if required coverage is missing")
    args = parser.parse_args()

    all_types = discover(RESOURCE_ROOT, "resource") + discover(DATASOURCE_ROOT, "data source")
    resources = [item for item in all_types if item.kind == "resource"]
    datasources = [item for item in all_types if item.kind == "data source"]

    missing_acceptance = [item for item in all_types if not item.has_acceptance]
    missing_import = [item for item in resources if item.import_capable and not item.has_import_test]
    no_import_capability = [item for item in resources if not item.import_capable]

    print("# Terraform Provider Test Coverage Audit")
    print(f"Resources: {len(resources)}")
    print(f"Data sources: {len(datasources)}")
    print(f"Types with acceptance coverage: {sum(1 for item in all_types if item.has_acceptance)}")
    print(f"Import-capable resources: {sum(1 for item in resources if item.import_capable)}")
    print(f"Import-capable resources with import tests: {sum(1 for item in resources if item.import_capable and item.has_import_test)}")

    print_table(
        "Missing Acceptance Coverage",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in missing_acceptance],
    )
    print_table(
        "Import-Capable Resources Missing Import Tests",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in missing_import],
    )
    print_table(
        "Resources Without Import Capability",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in no_import_capability],
    )

    if args.check and (missing_acceptance or missing_import or no_import_capability):
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
