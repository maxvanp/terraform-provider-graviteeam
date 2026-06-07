#!/usr/bin/env python3
"""Audit selected Terraform resource fields against OpenAPI update schemas."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parents[1]
OPENAPI_PATH = ROOT / "docs" / "openapi.yaml"


@dataclass(frozen=True)
class ResourceSchemaAudit:
    name: str
    package_dir: Path
    schema_name: str
    api_to_tf: dict[str, str]
    intentionally_unmanaged: dict[str, str]


AUDITS = [
    ResourceSchemaAudit(
        name="graviteeam_identity_provider",
        package_dir=ROOT / "internal" / "resources" / "identityprovider",
        schema_name="UpdateIdentityProvider",
        api_to_tf={
            "configuration": "configuration",
            "domainWhitelist": "domain_whitelist",
            "groupMapper": "group_mapper",
            "mappers": "mappers",
            "name": "name",
            "passwordPolicy": "password_policy_id",
            "roleMapper": "role_mapper",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_identity_provider",
        package_dir=ROOT / "internal" / "resources" / "orgidentityprovider",
        schema_name="UpdateIdentityProvider",
        api_to_tf={
            "configuration": "configuration",
            "domainWhitelist": "domain_whitelist",
            "groupMapper": "group_mapper",
            "mappers": "mappers",
            "name": "name",
            "roleMapper": "role_mapper",
            "type": "type",
        },
        intentionally_unmanaged={
            "passwordPolicy": "organization identity providers have no matching org password policy resource in the provider",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_scope",
        package_dir=ROOT / "internal" / "resources" / "scope",
        schema_name="UpdateScope",
        api_to_tf={
            "description": "description",
            "discovery": "discovery",
            "expiresIn": "expires_in",
            "iconUri": "icon_uri",
            "name": "name",
            "parameterized": "parameterized",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_reporter",
        package_dir=ROOT / "internal" / "resources" / "reporter",
        schema_name="UpdateReporter",
        api_to_tf={
            "configuration": "configuration",
            "enabled": "enabled",
            "inherited": "inherited",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_reporter",
        package_dir=ROOT / "internal" / "resources" / "orgreporter",
        schema_name="UpdateReporter",
        api_to_tf={
            "configuration": "configuration",
            "enabled": "enabled",
            "inherited": "inherited",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
]

TFSDK_RE = re.compile(r'`tfsdk:"([^"]+)"`')


def load_spec() -> dict:
    return yaml.safe_load(OPENAPI_PATH.read_text(encoding="utf-8"))


def model_attrs(package_dir: Path) -> set[str]:
    attrs: set[str] = set()
    for path in package_dir.glob("*_model.go"):
        attrs.update(TFSDK_RE.findall(path.read_text(encoding="utf-8")))
    return attrs


def schema_properties(spec: dict, schema_name: str) -> set[str]:
    schema = spec["components"]["schemas"][schema_name]
    return set(schema.get("properties", {}))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail if selected resource fields are missing")
    args = parser.parse_args()

    spec = load_spec()
    failures: list[str] = []

    print("# Resource Schema Field Audit")
    for audit in AUDITS:
        api_fields = schema_properties(spec, audit.schema_name)
        attrs = model_attrs(audit.package_dir)
        unmanaged = set(audit.intentionally_unmanaged)
        mapped = set(audit.api_to_tf)

        missing_mapping = sorted(api_fields - mapped - unmanaged)
        missing_attrs = sorted(
            f"{api_field}->{tf_attr}"
            for api_field, tf_attr in audit.api_to_tf.items()
            if api_field in api_fields and tf_attr not in attrs
        )
        stale_mapping = sorted(mapped - api_fields)

        print(f"\n## {audit.name}")
        print(f"OpenAPI schema: {audit.schema_name}")
        print(f"OpenAPI fields: {len(api_fields)}")
        print(f"Mapped Terraform fields: {len(mapped & api_fields)}")
        if audit.intentionally_unmanaged:
            print("Intentionally unmanaged:")
            for field, reason in sorted(audit.intentionally_unmanaged.items()):
                print(f"- {field}: {reason}")
        if missing_mapping:
            print("Missing API field mappings:")
            for field in missing_mapping:
                print(f"- {field}")
        else:
            print("Missing API field mappings: none")
        if missing_attrs:
            print("Mapped Terraform attributes missing from model:")
            for item in missing_attrs:
                print(f"- {item}")
        else:
            print("Mapped Terraform attributes missing from model: none")
        if stale_mapping:
            print("Mappings not present in OpenAPI schema:")
            for field in stale_mapping:
                print(f"- {field}")
        else:
            print("Mappings not present in OpenAPI schema: none")

        failures.extend(f"{audit.name}: missing mapping for {field}" for field in missing_mapping)
        failures.extend(f"{audit.name}: missing Terraform model attr {item}" for item in missing_attrs)
        failures.extend(f"{audit.name}: stale mapping for {field}" for field in stale_mapping)

    if args.check and failures:
        print("\n## Failures")
        for failure in failures:
            print(f"- {failure}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
