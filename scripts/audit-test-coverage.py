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
DISABLED_IMPORT_VERIFY_RE = re.compile(r"\bImportStateVerify\s*:\s*false\b")
IMPORT_CAPABILITY_RE = re.compile(r"ResourceWithImportState|func\s+\(r\s+\*\w+\)\s+ImportState\s*\(")

# These resources are represented and have acceptance tests, but the stock
# docker-compose.test.yml image set does not deploy a usable plugin instance.
LOCAL_COMPOSE_PLUGIN_GAPS = {
    "graviteeam_authorization_engine": "no deployed authorization engine plugin in test environment",
}


@dataclass(frozen=True)
class TerraformType:
    kind: str
    name: str
    package_dir: Path
    factory: str | None
    import_capable: bool
    has_acceptance: bool
    has_import_test: bool
    has_disabled_import_verify: bool
    local_compose_gap: str | None


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
            tf_name = f"graviteeam_{name}"
            types.append(
                TerraformType(
                    kind=kind,
                    name=tf_name,
                    package_dir=package_dir,
                    factory=factories[0] if factories else None,
                    import_capable=kind == "resource" and bool(IMPORT_CAPABILITY_RE.search(source)),
                    has_acceptance=bool(TEST_ACC_RE.search(tests)),
                    has_import_test=bool(IMPORT_RE.search(tests)),
                    has_disabled_import_verify=bool(DISABLED_IMPORT_VERIFY_RE.search(tests)),
                    local_compose_gap=LOCAL_COMPOSE_PLUGIN_GAPS.get(tf_name) if LOCAL_COMPOSE_PLUGIN_GAPS.get(tf_name, "") in tests else None,
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
    disabled_import_verify = [item for item in resources if item.has_disabled_import_verify]
    no_import_capability = [item for item in resources if not item.import_capable]
    local_compose_gaps = [item for item in all_types if item.local_compose_gap]

    print("# Terraform Provider Test Coverage Audit")
    print(f"Resources: {len(resources)}")
    print(f"Data sources: {len(datasources)}")
    print(f"Types with acceptance coverage: {sum(1 for item in all_types if item.has_acceptance)}")
    print(f"Types executable in stock local compose acceptance: {sum(1 for item in all_types if item.has_acceptance and not item.local_compose_gap)}")
    print(f"Import-capable resources: {sum(1 for item in resources if item.import_capable)}")
    print(f"Import-capable resources with import tests: {sum(1 for item in resources if item.import_capable and item.has_import_test)}")
    print(f"Resources with disabled import verification: {len(disabled_import_verify)}")

    print_table(
        "Missing Acceptance Coverage",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in missing_acceptance],
    )
    print_table(
        "Import-Capable Resources Missing Import Tests",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in missing_import],
    )
    print_table(
        "Resources With Disabled Import Verification",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in disabled_import_verify],
    )
    print_table(
        "Resources Without Import Capability",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)})" for item in no_import_capability],
    )
    print_table(
        "Acceptance Tests Not Executable In Stock Local Compose",
        [f"- {item.name} ({item.package_dir.relative_to(ROOT)}): {item.local_compose_gap}" for item in local_compose_gaps],
    )

    if args.check and (missing_acceptance or missing_import or disabled_import_verify or no_import_capability):
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
