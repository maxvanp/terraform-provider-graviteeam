#!/usr/bin/env python3
"""Audit Terraform provider coverage against the bundled Gravitee AM OpenAPI spec."""

from __future__ import annotations

import argparse
import re
import sys
from collections import defaultdict
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[1]
OPENAPI_PATH = ROOT / "docs" / "openapi.yaml"
API_COVERAGE_PATH = ROOT / "docs" / "api-coverage.md"

WRITE_METHODS = {"post", "put", "patch", "delete"}
HTTP_METHODS = WRITE_METHODS | {"get"}

RESOURCE_DOCS = ROOT / "docs" / "resources"
DATASOURCE_DOCS = ROOT / "docs" / "data-sources"
RESOURCE_EXAMPLES = ROOT / "examples" / "resources"
DATASOURCE_EXAMPLES = ROOT / "examples" / "data-sources"
RESOURCE_SRC = ROOT / "internal" / "resources"
DATASOURCE_SRC = ROOT / "internal" / "datasources"


def load_spec() -> dict:
    with OPENAPI_PATH.open("r", encoding="utf-8") as handle:
        return yaml.safe_load(handle)


def canonical_family(path: str) -> str:
    parts = [part for part in path.strip("/").split("/") if part]

    if parts[:5] == ["organizations", "{organizationId}", "environments", "{environmentId}", "domains"]:
        if len(parts) == 5:
            return "environment:domains"
        if len(parts) >= 6 and parts[5] == "_hrid":
            return "environment:domains/_hrid"
        if len(parts) >= 6 and parts[5] == "{domain}":
            tail = parts[6:]
            return "domain:" + collapse_tail(tail) if tail else "environment:domains"

    if parts[:4] == ["organizations", "{organizationId}", "environments", "{environmentId}"]:
        return "environment:" + collapse_tail(parts[4:])

    if parts[:2] == ["organizations", "{organizationId}"]:
        return "org:" + collapse_tail(parts[2:])

    if parts and parts[0] == "platform":
        return "platform:" + collapse_tail(parts[1:])

    if parts and parts[0] == "user":
        return "self:" + collapse_tail(parts[1:])

    return "unknown:" + collapse_tail(parts)


def collapse_tail(parts: list[str]) -> str:
    kept: list[str] = []
    skip_next_placeholder = False
    for part in parts:
        if part.startswith("{") and part.endswith("}"):
            if skip_next_placeholder:
                skip_next_placeholder = False
            continue
        kept.append(part)
        skip_next_placeholder = True
    return "/".join(kept) or "_root"


def family_methods(spec: dict) -> dict[str, set[str]]:
    families: dict[str, set[str]] = defaultdict(set)
    for path, item in spec["paths"].items():
        family = canonical_family(path)
        for method in item:
            method_lower = method.lower()
            if method_lower in HTTP_METHODS:
                families[family].add(method_lower)
    return families


def covered_families() -> tuple[set[str], set[str]]:
    text = API_COVERAGE_PATH.read_text(encoding="utf-8")
    resources: set[str] = set()
    datasources: set[str] = set()
    section = None
    for line in text.splitlines():
        if line.startswith("## Covered Resources"):
            section = "resources"
            continue
        if line.startswith("## Covered Data Sources"):
            section = "datasources"
            continue
        if line.startswith("## "):
            section = None
            continue
        match = re.match(r"\| `([^`]+)` \|", line)
        if not match:
            continue
        if section == "resources":
            resources.add(match.group(1))
        elif section == "datasources":
            datasources.add(match.group(1))
    return resources, datasources


def terraform_names(source_root: Path, suffix: str) -> set[str]:
    names: set[str] = set()
    pattern = re.compile(r'resp\.TypeName\s*=\s*req\.ProviderTypeName\s*\+\s*"_(.+?)"')
    for path in source_root.glob(f"*/*{suffix}.go"):
        text = path.read_text(encoding="utf-8")
        for match in pattern.finditer(text):
            names.add(match.group(1))
    return names


def source_dirs(source_root: Path) -> dict[str, Path]:
    result: dict[str, Path] = {}
    pattern = re.compile(r'resp\.TypeName\s*=\s*req\.ProviderTypeName\s*\+\s*"_(.+?)"')
    for path in source_root.glob("*/*.go"):
        text = path.read_text(encoding="utf-8")
        match = pattern.search(text)
        if match:
            result[match.group(1)] = path.parent
    return result


def has_test(path: Path) -> bool:
    return any(path.glob("*_test.go"))


def print_section(title: str, rows: list[str]) -> None:
    print(f"\n## {title}")
    if rows:
        for row in rows:
            print(row)
    else:
        print("none")


def doc_summary_counts() -> dict[str, int]:
    text = API_COVERAGE_PATH.read_text(encoding="utf-8")
    counts: dict[str, int] = {}
    for label, value in re.findall(r"\| ([^|`]+?) \| `(\d+)` \|", text):
        counts[label.strip()] = int(value)
    return counts


def documented_known_gap_families() -> set[str]:
    text = API_COVERAGE_PATH.read_text(encoding="utf-8")
    match = re.search(
        r"## Known Gaps\n(?P<body>.*?)(?:\n### Read-Only and Admin Metadata Not Covered|\n## |\Z)",
        text,
        re.DOTALL,
    )
    if not match:
        return set()
    return set(re.findall(r"\| `([^`]+)` \|", match.group("body")))


def check_doc_consistency(
    expected_counts: dict[str, int],
    uncovered_writable: set[str],
    covered_resources: set[str],
    covered_any: set[str],
) -> list[str]:
    errors: list[str] = []
    actual_counts = doc_summary_counts()
    for label, expected in expected_counts.items():
        actual = actual_counts.get(label)
        if actual is None:
            errors.append(f"missing summary row: {label}")
        elif actual != expected:
            errors.append(f"summary mismatch for {label}: doc has {actual}, audit has {expected}")

    documented_gaps = documented_known_gap_families()
    stale_gaps = sorted(documented_gaps & covered_resources)
    for family in stale_gaps:
        errors.append(f"covered family still listed in Known Gaps: {family}")
    missing_gaps = sorted(uncovered_writable - documented_gaps)
    for family in missing_gaps:
        errors.append(f"uncovered writable family missing from Known Gaps: {family}")
    extra_gaps = sorted(documented_gaps - uncovered_writable - covered_any)
    for family in extra_gaps:
        errors.append(f"unknown or non-writable family listed in Known Gaps: {family}")
    return errors


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check-doc", action="store_true", help="fail if docs/api-coverage.md summary or known gaps are stale")
    args = parser.parse_args()

    spec = load_spec()
    methods_by_family = family_methods(spec)
    covered_resources, covered_datasources = covered_families()
    covered_any = covered_resources | covered_datasources
    write_path_entries = sum(
        1
        for operations in spec["paths"].values()
        if {method.lower() for method in operations} & WRITE_METHODS
    )

    writable = {
        family: methods
        for family, methods in methods_by_family.items()
        if methods & WRITE_METHODS
    }
    read_only = {
        family: methods
        for family, methods in methods_by_family.items()
        if not methods & WRITE_METHODS
    }
    uncovered_writable = sorted(set(writable) - covered_resources)
    writable_datasource_only = sorted((set(writable) & covered_datasources) - covered_resources)
    uncovered_read_only = sorted(set(read_only) - covered_any)

    resource_dirs = source_dirs(RESOURCE_SRC)
    datasource_dirs = source_dirs(DATASOURCE_SRC)
    resource_names = set(resource_dirs)
    datasource_names = set(datasource_dirs)

    missing_resource_artifacts = []
    for name in sorted(resource_names):
        missing = []
        if not has_test(resource_dirs[name]):
            missing.append("test")
        if not (RESOURCE_DOCS / f"{name}.md").exists():
            missing.append("doc")
        if not (RESOURCE_EXAMPLES / f"graviteeam_{name}" / "resource.tf").exists():
            missing.append("example")
        if missing:
            missing_resource_artifacts.append(f"- graviteeam_{name}: missing {', '.join(missing)}")

    missing_datasource_artifacts = []
    for name in sorted(datasource_names):
        missing = []
        if not has_test(datasource_dirs[name]):
            missing.append("test")
        if not (DATASOURCE_DOCS / f"{name}.md").exists():
            missing.append("doc")
        if not (DATASOURCE_EXAMPLES / f"graviteeam_{name}" / "data-source.tf").exists():
            missing.append("example")
        if missing:
            missing_datasource_artifacts.append(f"- graviteeam_{name}: missing {', '.join(missing)}")

    print("# Gravitee AM OpenAPI Coverage Audit")
    print(f"OpenAPI paths: {len(spec['paths'])}")
    print(f"OpenAPI path entries with at least one write verb: {write_path_entries}")
    print(f"OpenAPI families: {len(methods_by_family)}")
    print(f"Writable families: {len(writable)}")
    print(f"Read-only families: {len(read_only)}")
    print(f"Covered resource families: {len(covered_resources)}")
    print(f"Covered data source families: {len(covered_datasources)}")
    print(f"Registered resources: {len(resource_names)}")
    print(f"Registered data sources: {len(datasource_names)}")
    print(f"Uncovered writable families: {len(uncovered_writable)}")
    print(f"Writable families covered only by data source: {len(writable_datasource_only)}")
    print(f"Uncovered read-only families: {len(uncovered_read_only)}")

    print_section(
        "Uncovered Writable Families",
        [f"- {family}: {', '.join(sorted(writable[family]))}" for family in uncovered_writable],
    )
    print_section(
        "Writable Families Covered Only By Data Source",
        [f"- {family}: {', '.join(sorted(writable[family]))}" for family in writable_datasource_only],
    )
    print_section(
        "Uncovered Read-Only Families",
        [f"- {family}: {', '.join(sorted(read_only[family]))}" for family in uncovered_read_only],
    )
    print_section("Missing Resource Artifacts", missing_resource_artifacts)
    print_section("Missing Data Source Artifacts", missing_datasource_artifacts)

    if args.check_doc:
        expected_counts = {
            "OpenAPI families": len(methods_by_family),
            "Writable families": len(writable),
            "Read-only families": len(read_only),
            "Uncovered writable families without Terraform resource": len(uncovered_writable),
            "Writable families covered only by data source": len(writable_datasource_only),
            "Uncovered read-only families": len(uncovered_read_only),
            "Registered resources missing test/doc/example artifact": len(missing_resource_artifacts),
            "Registered data sources missing test/doc/example artifact": len(missing_datasource_artifacts),
        }
        errors = check_doc_consistency(expected_counts, set(uncovered_writable), covered_resources, covered_any)
        print_section("Documentation Consistency", [f"- {error}" for error in errors])
        if errors:
            sys.exit(1)


if __name__ == "__main__":
    main()
