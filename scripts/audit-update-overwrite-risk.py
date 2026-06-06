#!/usr/bin/env python3
"""Flag resource Update methods that call broad update client methods without a preceding read."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
RESOURCE_ROOT = ROOT / "internal" / "resources"

DEDICATED_UPDATE_METHODS = {
    "UpdateApplicationType",
    "UpdateOrgUserStatus",
    "UpdateOrgUsername",
    "UpdateUserStatus",
    "UpdateUsername",
}

# These update methods use set-like replacement semantics, a reviewed state-based
# merge/clear pattern, dedicated subresource APIs, or intentionally reject updates.
REVIEWED_WITHOUT_GET = {
    "application/application_resource.go:ApplicationResource",
    "applicationflow/application_flow_resource.go:ApplicationFlowResource",
    "applicationmember/application_member_resource.go:ApplicationMemberResource",
    "applicationsecret/application_secret_resource.go:ApplicationSecretResource",
    "domain/domain_resource.go:DomainResource",
    "domainflow/domain_flow_resource.go:DomainFlowResource",
    "domainmember/domain_member_resource.go:DomainMemberResource",
    "domaincertificatesettings/domain_certificate_settings_resource.go:DomainCertificateSettingsResource",
    "grouproles/group_roles_resource.go:GroupRolesResource",
    "groupmembers/group_members_resource.go:GroupMembersResource",
    "identityproviderpasswordpolicy/identity_provider_password_policy_resource.go:IdentityProviderPasswordPolicyResource",
    "identityprovider/identity_provider_resource.go:IdentityProviderResource",
    "i18ndictionary/i18n_dictionary_resource.go:I18nDictionaryResource",
    "orggroupmembers/org_group_members_resource.go:OrgGroupMembersResource",
    "orgidentityprovider/org_identity_provider_resource.go:OrgIdentityProviderResource",
    "orgmember/org_member_resource.go:OrgMemberResource",
    "orgsettings/org_settings_resource.go:OrgSettingsResource",
    "orgusertoken/org_user_token_resource.go:OrgUserTokenResource",
    "protectedresource/protected_resource_resource.go:ProtectedResourceResource",
    "protectedresourcemember/protected_resource_member_resource.go:ProtectedResourceMemberResource",
    "protectedresourcesecret/protected_resource_secret_resource.go:ProtectedResourceSecretResource",
    "usercertificatecredential/user_certificate_credential_resource.go:UserCertificateCredentialResource",
    "user/user_resource.go:UserResource",
    "userrole/user_role_resource.go:UserRoleResource",
    "orguser/org_user_resource.go:OrgUserResource",
}

UPDATE_FUNC_RE = re.compile(
    r"func\s+\(r\s+\*(?P<type>\w+)\)\s+Update\s*\([^)]*\)\s*\{",
    re.MULTILINE,
)
CLIENT_UPDATE_RE = re.compile(r"r\.client\.(?P<method>Update[A-Za-z0-9_]+|AddOrUpdate[A-Za-z0-9_]+)\(")
CLIENT_GET_RE = re.compile(r"r\.client\.(?P<method>Get[A-Za-z0-9_]+)\(")


def extract_block(text: str, start: int) -> str:
    depth = 0
    for index in range(start, len(text)):
        char = text[index]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return text[start : index + 1]
    raise ValueError("unterminated Go block")


def audit_file(path: Path, include_reviewed: bool) -> list[tuple[str, bool]]:
    text = path.read_text(encoding="utf-8")
    findings: list[tuple[str, bool]] = []
    rel = path.relative_to(RESOURCE_ROOT).as_posix()
    for match in UPDATE_FUNC_RE.finditer(text):
        resource_type = match.group("type")
        key = f"{rel}:{resource_type}"
        reviewed = key in REVIEWED_WITHOUT_GET
        if reviewed and not include_reviewed:
            continue
        body = extract_block(text, match.end() - 1)
        updates = list(CLIENT_UPDATE_RE.finditer(body))
        if not updates:
            continue
        gets = list(CLIENT_GET_RE.finditer(body))
        for update in updates:
            method = update.group("method")
            if method in DEDICATED_UPDATE_METHODS:
                continue
            has_prior_get = any(get.start() < update.start() for get in gets)
            if not has_prior_get:
                findings.append((f"- {key}: {method} without preceding Get* in Update", reviewed))
    return findings


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--all", action="store_true", help="include reviewed broad update methods in the report")
    parser.add_argument("--check", action="store_true", help="fail when unreviewed broad update methods are found")
    args = parser.parse_args()

    findings: list[tuple[str, bool]] = []
    for path in sorted(RESOURCE_ROOT.glob("*/*.go")):
        if path.name.endswith("_test.go") or path.name.endswith("_model.go"):
            continue
        findings.extend(audit_file(path, args.all))

    print("# Update Overwrite Risk Audit")
    if findings:
        for finding, reviewed in findings:
            if reviewed:
                finding += " (reviewed)"
            print(finding)
        if args.check and any(not reviewed for _, reviewed in findings):
            sys.exit(1)
        return
    print("none")


if __name__ == "__main__":
    main()
