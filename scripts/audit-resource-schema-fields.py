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


@dataclass(frozen=True)
class ResourceSchemaCandidate:
    name: str
    package_dir: Path
    schema_name: str


AUDITS = [
    ResourceSchemaAudit(
        name="graviteeam_alert_notifier",
        package_dir=ROOT / "internal" / "resources" / "alertnotifier",
        schema_name="NewAlertNotifier",
        api_to_tf={
            "configuration": "configuration",
            "enabled": "enabled",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_alert_trigger",
        package_dir=ROOT / "internal" / "resources" / "alerttrigger",
        schema_name="PatchAlertTrigger",
        api_to_tf={
            "alertNotifiers": "alert_notifier_ids",
            "enabled": "enabled",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_application",
        package_dir=ROOT / "internal" / "resources" / "application",
        schema_name="PatchApplication",
        api_to_tf={
            "description": "description",
            "factors": "factors",
            "identityProviders": "identity_providers",
            "metadata": "metadata_json",
            "name": "name",
            "settings": "settings_json",
        },
        intentionally_unmanaged={
            "certificate": "application certificate assignment needs a dedicated ownership model before Terraform can safely update it",
            "enabled": "application lifecycle activation is not currently modeled as Terraform-owned state",
            "requiredPermissions": "API permission filtering metadata is not Terraform-owned application configuration",
            "template": "application template inheritance is API-managed for existing applications",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_application_member",
        package_dir=ROOT / "internal" / "resources" / "applicationmember",
        schema_name="NewMembership",
        api_to_tf={
            "memberId": "member_id",
            "memberType": "member_type",
            "role": "role_id",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_application_secret",
        package_dir=ROOT / "internal" / "resources" / "applicationsecret",
        schema_name="NewClientSecret",
        api_to_tf={
            "name": "name",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_auth_device_notifier",
        package_dir=ROOT / "internal" / "resources" / "authdevicenotifier",
        schema_name="UpdateAuthenticationDeviceNotifier",
        api_to_tf={
            "configuration": "configuration",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_authorization_engine",
        package_dir=ROOT / "internal" / "resources" / "authorizationengine",
        schema_name="UpdateAuthorizationEngine",
        api_to_tf={
            "configuration": "configuration",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_bot_detection",
        package_dir=ROOT / "internal" / "resources" / "botdetection",
        schema_name="UpdateBotDetection",
        api_to_tf={
            "configuration": "configuration",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_certificate",
        package_dir=ROOT / "internal" / "resources" / "certificate",
        schema_name="UpdateCertificate",
        api_to_tf={
            "configuration": "configuration",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_device_identifier",
        package_dir=ROOT / "internal" / "resources" / "deviceidentifier",
        schema_name="UpdateDeviceIdentifier",
        api_to_tf={
            "configuration": "configuration",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_domain",
        package_dir=ROOT / "internal" / "resources" / "domain",
        schema_name="PatchDomain",
        api_to_tf={
            "dataPlaneId": "data_plane_id",
            "description": "description",
            "enabled": "enabled",
            "loginSettings": "login_settings",
            "name": "name",
            "oidc": "oidc",
            "tags": "settings_json",
        },
        intentionally_unmanaged={
            "accountSettings": "advanced domain patch section managed through settings_json",
            "alertEnabled": "advanced domain patch field managed through settings_json",
            "certificateSettings": "domain certificate fallback has a dedicated resource",
            "corsSettings": "advanced domain patch section managed through settings_json",
            "master": "API-managed domain role flag",
            "passwordSettings": "advanced domain patch section managed through settings_json",
            "path": "API-managed domain path",
            "requiredPermissions": "API permission filtering metadata is not Terraform-owned domain configuration",
            "saml": "advanced domain patch section managed through settings_json",
            "scim": "advanced domain patch section managed through settings_json",
            "secretSettings": "advanced domain patch section managed through settings_json",
            "selfServiceAccountManagementSettings": "advanced domain patch section managed through settings_json",
            "tokenExchangeSettings": "advanced domain patch section managed through settings_json",
            "uma": "advanced domain patch section managed through settings_json",
            "vhostMode": "advanced domain patch field managed through settings_json",
            "vhosts": "advanced domain patch section managed through settings_json",
            "webAuthnSettings": "advanced domain patch section managed through settings_json",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_domain_member",
        package_dir=ROOT / "internal" / "resources" / "domainmember",
        schema_name="NewMembership",
        api_to_tf={
            "memberId": "member_id",
            "memberType": "member_type",
            "role": "role_id",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_domain_certificate_settings",
        package_dir=ROOT / "internal" / "resources" / "domaincertificatesettings",
        schema_name="CertificateSettings",
        api_to_tf={
            "fallbackCertificate": "fallback_certificate_id",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_factor",
        package_dir=ROOT / "internal" / "resources" / "factor",
        schema_name="UpdateFactor",
        api_to_tf={
            "name": "name",
        },
        intentionally_unmanaged={
            "configuration": "provider currently manages stock factors with empty plugin configuration",
            "type": "derived from immutable factor_type and preserved through plugin mapping",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_form",
        package_dir=ROOT / "internal" / "resources" / "form",
        schema_name="UpdateForm",
        api_to_tf={
            "content": "content",
            "enabled": "enabled",
        },
        intentionally_unmanaged={
            "assets": "file assets need a separate ownership model before Terraform can update them safely",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_application_form",
        package_dir=ROOT / "internal" / "resources" / "applicationform",
        schema_name="UpdateForm",
        api_to_tf={
            "content": "content",
            "enabled": "enabled",
        },
        intentionally_unmanaged={
            "assets": "file assets need a separate ownership model before Terraform can update them safely",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_form",
        package_dir=ROOT / "internal" / "resources" / "orgform",
        schema_name="UpdateForm",
        api_to_tf={
            "content": "content",
            "enabled": "enabled",
        },
        intentionally_unmanaged={
            "assets": "file assets need a separate ownership model before Terraform can update them safely",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_i18n_dictionary",
        package_dir=ROOT / "internal" / "resources" / "i18ndictionary",
        schema_name="UpdateI18nDictionary",
        api_to_tf={
            "entries": "entries",
            "locale": "locale",
            "name": "name",
        },
        intentionally_unmanaged={},
    ),
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
    ResourceSchemaAudit(
        name="graviteeam_group",
        package_dir=ROOT / "internal" / "resources" / "group",
        schema_name="UpdateGroup",
        api_to_tf={
            "description": "description",
            "members": "members",
            "name": "name",
            "roles": "roles",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_group",
        package_dir=ROOT / "internal" / "resources" / "orggroup",
        schema_name="UpdateGroup",
        api_to_tf={
            "description": "description",
            "members": "members",
            "name": "name",
            "roles": "roles",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_settings",
        package_dir=ROOT / "internal" / "resources" / "orgsettings",
        schema_name="PatchOrganization",
        api_to_tf={
            "identities": "identities",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_member",
        package_dir=ROOT / "internal" / "resources" / "orgmember",
        schema_name="NewMembership",
        api_to_tf={
            "memberId": "member_id",
            "memberType": "member_type",
            "role": "role_id",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_role",
        package_dir=ROOT / "internal" / "resources" / "role",
        schema_name="UpdateRole",
        api_to_tf={
            "description": "description",
            "name": "name",
            "oauthScopes": "oauth_scopes",
            "permissions": "permissions",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_role",
        package_dir=ROOT / "internal" / "resources" / "orgrole",
        schema_name="UpdateRole",
        api_to_tf={
            "description": "description",
            "name": "name",
            "permissions": "permissions",
        },
        intentionally_unmanaged={
            "oauthScopes": "organization roles do not attach domain OAuth scopes",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_password_policy",
        package_dir=ROOT / "internal" / "resources" / "passwordpolicy",
        schema_name="UpdatePasswordPolicy",
        api_to_tf={
            "defaultPolicy": "default_policy",
            "excludePasswordsInDictionary": "exclude_passwords_in_dictionary",
            "excludeUserProfileInfoInPassword": "exclude_user_profile_info_in_password",
            "expiryDuration": "expiry_duration",
            "includeNumbers": "include_numbers",
            "includeSpecialCharacters": "include_special_characters",
            "lettersInMixedCase": "letters_in_mixed_case",
            "maxConsecutiveLetters": "max_consecutive_letters",
            "maxLength": "max_length",
            "minLength": "min_length",
            "name": "name",
            "oldPasswords": "old_passwords",
            "passwordHistoryEnabled": "password_history_enabled",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_extension_grant",
        package_dir=ROOT / "internal" / "resources" / "extensiongrant",
        schema_name="UpdateExtensionGrant",
        api_to_tf={
            "configuration": "configuration",
            "createUser": "create_user",
            "grantType": "grant_type",
            "identityProvider": "identity_provider",
            "name": "name",
            "type": "type",
            "userExists": "user_exists",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_protected_resource",
        package_dir=ROOT / "internal" / "resources" / "protectedresource",
        schema_name="UpdateProtectedResource",
        api_to_tf={
            "description": "description",
            "features": "feature",
            "name": "name",
            "resourceIdentifiers": "resource_identifiers",
            "settings": "settings_json",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_protected_resource_member",
        package_dir=ROOT / "internal" / "resources" / "protectedresourcemember",
        schema_name="NewMembership",
        api_to_tf={
            "memberId": "member_id",
            "memberType": "member_type",
            "role": "role_id",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_protected_resource_secret",
        package_dir=ROOT / "internal" / "resources" / "protectedresourcesecret",
        schema_name="NewClientSecret",
        api_to_tf={
            "name": "name",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_service_resource",
        package_dir=ROOT / "internal" / "resources" / "serviceresource",
        schema_name="UpdateServiceResource",
        api_to_tf={
            "configuration": "configuration",
            "name": "name",
            "type": "type",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_tag",
        package_dir=ROOT / "internal" / "resources" / "orgtag",
        schema_name="UpdateTag",
        api_to_tf={
            "description": "description",
            "name": "name",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_user_token",
        package_dir=ROOT / "internal" / "resources" / "orgusertoken",
        schema_name="NewAccountAccessToken",
        api_to_tf={
            "name": "name",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_email_template",
        package_dir=ROOT / "internal" / "resources" / "emailtemplate",
        schema_name="UpdateEmail",
        api_to_tf={
            "content": "content",
            "enabled": "enabled",
            "expiresAfter": "expires_after",
            "from": "from",
            "fromName": "from_name",
            "subject": "subject",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_entrypoint",
        package_dir=ROOT / "internal" / "resources" / "orgentrypoint",
        schema_name="UpdateEntrypoint",
        api_to_tf={
            "description": "description",
            "name": "name",
            "tags": "tags",
            "url": "url",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_user",
        package_dir=ROOT / "internal" / "resources" / "user",
        schema_name="UpdateUser",
        api_to_tf={
            "displayName": "display_name",
            "email": "email",
            "firstName": "first_name",
            "forceResetPassword": "force_reset_password",
            "lastName": "last_name",
            "preRegistration": "pre_registration",
        },
        intentionally_unmanaged={
            "accountNonExpired": "API account lifecycle flag preserved from GET/merge; not a Terraform-owned profile field",
            "accountNonLocked": "managed through the dedicated locked attribute and lock/unlock endpoints",
            "additionalInformation": "arbitrary profile map requires a separate JSON design before ownership is safe",
            "client": "API-managed client context",
            "createdAt": "server-managed timestamp",
            "credentialsNonExpired": "API credential lifecycle flag preserved from GET/merge",
            "enabled": "managed through the dedicated status endpoint",
            "externalId": "local 4.11.4 does not round-trip the planned value; it is API-generated or ignored",
            "loggedAt": "server-managed login timestamp",
            "loginsCount": "server-managed counter",
            "preferredLanguage": "local 4.11.4 accepts but does not round-trip the planned value",
            "registrationCompleted": "API lifecycle flag preserved from GET/merge",
            "source": "API-managed identity source",
            "updatedAt": "server-managed timestamp",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_org_user",
        package_dir=ROOT / "internal" / "resources" / "orguser",
        schema_name="UpdateUser",
        api_to_tf={
            "email": "email",
            "firstName": "first_name",
            "forceResetPassword": "force_reset_password",
            "lastName": "last_name",
            "preRegistration": "pre_registration",
        },
        intentionally_unmanaged={
            "accountNonExpired": "API account lifecycle flag preserved from GET/merge; not a Terraform-owned profile field",
            "accountNonLocked": "organization user lock endpoints are not currently modeled as a Terraform attribute",
            "additionalInformation": "arbitrary profile map requires a separate JSON design before ownership is safe",
            "client": "API-managed client context",
            "createdAt": "server-managed timestamp",
            "credentialsNonExpired": "API credential lifecycle flag preserved from GET/merge",
            "displayName": "local 4.11.4 derives this from firstName and lastName for organization users",
            "enabled": "managed through the dedicated status endpoint",
            "externalId": "local 4.11.4 does not round-trip the planned value",
            "loggedAt": "server-managed login timestamp",
            "loginsCount": "server-managed counter",
            "preferredLanguage": "local 4.11.4 accepts but does not round-trip the planned value",
            "registrationCompleted": "API lifecycle flag preserved from GET/merge",
            "source": "API-managed identity source",
            "updatedAt": "server-managed timestamp",
        },
    ),
    ResourceSchemaAudit(
        name="graviteeam_theme",
        package_dir=ROOT / "internal" / "resources" / "theme",
        schema_name="NewTheme",
        api_to_tf={
            "css": "css",
            "faviconUrl": "favicon_url",
            "logoUrl": "logo_url",
            "logoWidth": "logo_width",
            "primaryButtonColorHex": "primary_button_color_hex",
            "primaryTextColorHex": "primary_text_color_hex",
            "secondaryButtonColorHex": "secondary_button_color_hex",
            "secondaryTextColorHex": "secondary_text_color_hex",
        },
        intentionally_unmanaged={},
    ),
    ResourceSchemaAudit(
        name="graviteeam_user_certificate_credential",
        package_dir=ROOT / "internal" / "resources" / "usercertificatecredential",
        schema_name="NewCertificateCredential",
        api_to_tf={
            "certificatePem": "certificate_pem",
        },
        intentionally_unmanaged={},
    ),
]

AUDIT_CANDIDATES = [
    ResourceSchemaCandidate("graviteeam_alert_notifier", ROOT / "internal" / "resources" / "alertnotifier", "NewAlertNotifier"),
    ResourceSchemaCandidate("graviteeam_alert_trigger", ROOT / "internal" / "resources" / "alerttrigger", "PatchAlertTrigger"),
    ResourceSchemaCandidate("graviteeam_application", ROOT / "internal" / "resources" / "application", "PatchApplication"),
    ResourceSchemaCandidate("graviteeam_application_member", ROOT / "internal" / "resources" / "applicationmember", "NewMembership"),
    ResourceSchemaCandidate("graviteeam_application_secret", ROOT / "internal" / "resources" / "applicationsecret", "NewClientSecret"),
    ResourceSchemaCandidate("graviteeam_auth_device_notifier", ROOT / "internal" / "resources" / "authdevicenotifier", "UpdateAuthenticationDeviceNotifier"),
    ResourceSchemaCandidate("graviteeam_authorization_engine", ROOT / "internal" / "resources" / "authorizationengine", "UpdateAuthorizationEngine"),
    ResourceSchemaCandidate("graviteeam_bot_detection", ROOT / "internal" / "resources" / "botdetection", "UpdateBotDetection"),
    ResourceSchemaCandidate("graviteeam_certificate", ROOT / "internal" / "resources" / "certificate", "UpdateCertificate"),
    ResourceSchemaCandidate("graviteeam_device_identifier", ROOT / "internal" / "resources" / "deviceidentifier", "UpdateDeviceIdentifier"),
    ResourceSchemaCandidate("graviteeam_domain", ROOT / "internal" / "resources" / "domain", "PatchDomain"),
    ResourceSchemaCandidate("graviteeam_domain_certificate_settings", ROOT / "internal" / "resources" / "domaincertificatesettings", "CertificateSettings"),
    ResourceSchemaCandidate("graviteeam_domain_member", ROOT / "internal" / "resources" / "domainmember", "NewMembership"),
    ResourceSchemaCandidate("graviteeam_email_template", ROOT / "internal" / "resources" / "emailtemplate", "UpdateEmail"),
    ResourceSchemaCandidate("graviteeam_extension_grant", ROOT / "internal" / "resources" / "extensiongrant", "UpdateExtensionGrant"),
    ResourceSchemaCandidate("graviteeam_factor", ROOT / "internal" / "resources" / "factor", "UpdateFactor"),
    ResourceSchemaCandidate("graviteeam_form", ROOT / "internal" / "resources" / "form", "UpdateForm"),
    ResourceSchemaCandidate("graviteeam_application_form", ROOT / "internal" / "resources" / "applicationform", "UpdateForm"),
    ResourceSchemaCandidate("graviteeam_group", ROOT / "internal" / "resources" / "group", "UpdateGroup"),
    ResourceSchemaCandidate("graviteeam_i18n_dictionary", ROOT / "internal" / "resources" / "i18ndictionary", "UpdateI18nDictionary"),
    ResourceSchemaCandidate("graviteeam_identity_provider", ROOT / "internal" / "resources" / "identityprovider", "UpdateIdentityProvider"),
    ResourceSchemaCandidate("graviteeam_org_identity_provider", ROOT / "internal" / "resources" / "orgidentityprovider", "UpdateIdentityProvider"),
    ResourceSchemaCandidate("graviteeam_org_entrypoint", ROOT / "internal" / "resources" / "orgentrypoint", "UpdateEntrypoint"),
    ResourceSchemaCandidate("graviteeam_org_form", ROOT / "internal" / "resources" / "orgform", "UpdateForm"),
    ResourceSchemaCandidate("graviteeam_org_group", ROOT / "internal" / "resources" / "orggroup", "UpdateGroup"),
    ResourceSchemaCandidate("graviteeam_org_settings", ROOT / "internal" / "resources" / "orgsettings", "PatchOrganization"),
    ResourceSchemaCandidate("graviteeam_org_reporter", ROOT / "internal" / "resources" / "orgreporter", "UpdateReporter"),
    ResourceSchemaCandidate("graviteeam_org_role", ROOT / "internal" / "resources" / "orgrole", "UpdateRole"),
    ResourceSchemaCandidate("graviteeam_org_tag", ROOT / "internal" / "resources" / "orgtag", "UpdateTag"),
    ResourceSchemaCandidate("graviteeam_org_member", ROOT / "internal" / "resources" / "orgmember", "NewMembership"),
    ResourceSchemaCandidate("graviteeam_org_user_token", ROOT / "internal" / "resources" / "orgusertoken", "NewAccountAccessToken"),
    ResourceSchemaCandidate("graviteeam_org_user", ROOT / "internal" / "resources" / "orguser", "UpdateUser"),
    ResourceSchemaCandidate("graviteeam_password_policy", ROOT / "internal" / "resources" / "passwordpolicy", "UpdatePasswordPolicy"),
    ResourceSchemaCandidate("graviteeam_protected_resource", ROOT / "internal" / "resources" / "protectedresource", "UpdateProtectedResource"),
    ResourceSchemaCandidate("graviteeam_protected_resource_member", ROOT / "internal" / "resources" / "protectedresourcemember", "NewMembership"),
    ResourceSchemaCandidate("graviteeam_protected_resource_secret", ROOT / "internal" / "resources" / "protectedresourcesecret", "NewClientSecret"),
    ResourceSchemaCandidate("graviteeam_reporter", ROOT / "internal" / "resources" / "reporter", "UpdateReporter"),
    ResourceSchemaCandidate("graviteeam_role", ROOT / "internal" / "resources" / "role", "UpdateRole"),
    ResourceSchemaCandidate("graviteeam_scope", ROOT / "internal" / "resources" / "scope", "UpdateScope"),
    ResourceSchemaCandidate("graviteeam_service_resource", ROOT / "internal" / "resources" / "serviceresource", "UpdateServiceResource"),
    ResourceSchemaCandidate("graviteeam_theme", ROOT / "internal" / "resources" / "theme", "NewTheme"),
    ResourceSchemaCandidate("graviteeam_user", ROOT / "internal" / "resources" / "user", "UpdateUser"),
    ResourceSchemaCandidate("graviteeam_user_certificate_credential", ROOT / "internal" / "resources" / "usercertificatecredential", "NewCertificateCredential"),
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
    audited_names = {audit.name for audit in AUDITS}
    unaudited_candidates = [candidate for candidate in AUDIT_CANDIDATES if candidate.name not in audited_names]

    print("# Resource Schema Field Audit")
    print(f"Audited durable resource schemas: {len(AUDITS)}")
    print(f"Unaudited durable resource schemas: {len(unaudited_candidates)}")
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

    print("\n## Durable Resource Schemas Missing Audit")
    if unaudited_candidates:
        for candidate in unaudited_candidates:
            schema_status = "present" if candidate.schema_name in spec["components"]["schemas"] else "missing"
            print(
                f"- {candidate.name}: {candidate.schema_name} ({schema_status}, "
                f"{candidate.package_dir.relative_to(ROOT)})"
            )
    else:
        print("none")

    failures.extend(f"{candidate.name}: missing resource schema audit" for candidate in unaudited_candidates)

    if args.check and failures:
        print("\n## Failures")
        for failure in failures:
            print(f"- {failure}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
