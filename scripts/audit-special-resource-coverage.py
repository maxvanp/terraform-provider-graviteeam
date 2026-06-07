#!/usr/bin/env python3
"""Audit Terraform resources that are intentionally outside field-schema audit."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RESOURCE_ROOT = ROOT / "internal" / "resources"
FIELD_AUDIT_PATH = ROOT / "scripts" / "audit-resource-schema-fields.py"

TYPE_NAME_RE = re.compile(r'resp\.TypeName\s*=\s*req\.ProviderTypeName\s*\+\s*"_(.+?)"')
AUDIT_NAME_RE = re.compile(r'name="(graviteeam_[^"]+)"')
TFSDK_RE = re.compile(r'`tfsdk:"([^"]+)"`')


@dataclass(frozen=True)
class SpecialResourceAudit:
    name: str
    package_dir: Path
    expected_attrs: set[str]
    reason: str


SPECIAL_RESOURCES = [
    SpecialResourceAudit(
        name="graviteeam_application_email",
        package_dir=RESOURCE_ROOT / "applicationemail",
        expected_attrs={"id", "domain_id", "application_id", "template", "enabled", "from", "from_name", "subject", "content", "expires_after"},
        reason="application-scoped email template uses the shared UpdateEmail schema already covered by graviteeam_email_template",
    ),
    SpecialResourceAudit(
        name="graviteeam_application_flow",
        package_dir=RESOURCE_ROOT / "applicationflow",
        expected_attrs={"domain_id", "application_id", "flows"},
        reason="manages a complete list payload of Flow objects rather than a single resource update schema",
    ),
    SpecialResourceAudit(
        name="graviteeam_domain_flow",
        package_dir=RESOURCE_ROOT / "domainflow",
        expected_attrs={"domain_id", "flows"},
        reason="manages a complete list payload of Flow objects rather than a single resource update schema",
    ),
    SpecialResourceAudit(
        name="graviteeam_generated_certificate",
        package_dir=RESOURCE_ROOT / "generatedcertificate",
        expected_attrs={"id", "domain_id", "rotation_trigger", "name", "type"},
        reason="wraps the certificate rotation action and then reads the generated certificate entity",
    ),
    SpecialResourceAudit(
        name="graviteeam_group_members",
        package_dir=RESOURCE_ROOT / "groupmembers",
        expected_attrs={"domain_id", "group_id", "members"},
        reason="reconciles group membership through add/remove endpoints instead of a single body schema",
    ),
    SpecialResourceAudit(
        name="graviteeam_group_roles",
        package_dir=RESOURCE_ROOT / "grouproles",
        expected_attrs={"domain_id", "group_id", "roles"},
        reason="reconciles group roles through add/remove endpoints instead of a single body schema",
    ),
    SpecialResourceAudit(
        name="graviteeam_identity_provider_password_policy",
        package_dir=RESOURCE_ROOT / "identityproviderpasswordpolicy",
        expected_attrs={"id", "domain_id", "identity_provider_id", "password_policy_id"},
        reason="manages a relationship through a dedicated password-policy assignment endpoint",
    ),
    SpecialResourceAudit(
        name="graviteeam_org_group_members",
        package_dir=RESOURCE_ROOT / "orggroupmembers",
        expected_attrs={"group_id", "members"},
        reason="reconciles organization group membership through add/remove endpoints instead of a single body schema",
    ),
    SpecialResourceAudit(
        name="graviteeam_user_role",
        package_dir=RESOURCE_ROOT / "userrole",
        expected_attrs={"domain_id", "user_id", "roles"},
        reason="manages a relationship through domain user role endpoints",
    ),
]


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def model_attrs(package_dir: Path) -> set[str]:
    attrs: set[str] = set()
    for path in package_dir.glob("*_model.go"):
        attrs.update(TFSDK_RE.findall(read(path)))
    return attrs


def resource_type_names() -> dict[str, Path]:
    resources: dict[str, Path] = {}
    for package_dir in sorted(path for path in RESOURCE_ROOT.iterdir() if path.is_dir()):
        source = "\n".join(read(path) for path in package_dir.glob("*.go") if not path.name.endswith("_test.go"))
        for suffix in TYPE_NAME_RE.findall(source):
            resources[f"graviteeam_{suffix}"] = package_dir
    return resources


def field_audited_names() -> set[str]:
    return set(AUDIT_NAME_RE.findall(read(FIELD_AUDIT_PATH)))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail if any registered resource is unclassified or malformed")
    args = parser.parse_args()

    resources = resource_type_names()
    field_audited = field_audited_names()
    special = {audit.name: audit for audit in SPECIAL_RESOURCES}
    failures: list[str] = []

    unclassified = sorted(set(resources) - field_audited - set(special))
    stale_special = sorted(set(special) - set(resources))
    overlap = sorted(field_audited & set(special))

    print("# Special Resource Coverage Audit")
    print(f"Registered resources: {len(resources)}")
    print(f"Field-schema audited resources: {len(field_audited & set(resources))}")
    print(f"Special classified resources: {len(set(special) & set(resources))}")

    print("\n## Special Resources")
    for audit in SPECIAL_RESOURCES:
        attrs = model_attrs(audit.package_dir)
        missing_attrs = sorted(audit.expected_attrs - attrs)
        extra_status = ""
        if missing_attrs:
            extra_status = f" (missing attrs: {', '.join(missing_attrs)})"
            failures.extend(f"{audit.name}: missing Terraform model attr {attr}" for attr in missing_attrs)
        print(f"- {audit.name}: {audit.reason}{extra_status}")

    print("\n## Unclassified Registered Resources")
    if unclassified:
        for name in unclassified:
            print(f"- {name} ({resources[name].relative_to(ROOT)})")
            failures.append(f"{name}: missing field audit or special classification")
    else:
        print("none")

    print("\n## Stale Special Classifications")
    if stale_special:
        for name in stale_special:
            print(f"- {name}")
            failures.append(f"{name}: special classification does not match a registered resource")
    else:
        print("none")

    print("\n## Resources In Both Audits")
    if overlap:
        for name in overlap:
            print(f"- {name}")
            failures.append(f"{name}: classified as both field-schema and special resource")
    else:
        print("none")

    if args.check and failures:
        print("\n## Failures")
        for failure in failures:
            print(f"- {failure}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
