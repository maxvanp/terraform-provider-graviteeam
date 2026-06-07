#!/usr/bin/env python3
"""Probe currently documented OpenAPI coverage gaps against a local AM instance.

This script creates disposable fixtures and calls the remaining writable families
that are intentionally not modelled as durable Terraform resources yet. It is an
evidence-gathering tool, not a provider test.
"""

from __future__ import annotations

import argparse
import base64
import json
import socket
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Any


@dataclass
class Result:
    family: str
    method: str
    path: str
    status: int | None
    outcome: str
    detail: str


@dataclass(frozen=True)
class Expectation:
    label: str
    family: str
    method: str
    path_contains: str
    statuses: set[int | None]
    outcomes: set[str]


EXPECTED_RESULTS = [
    Expectation("form preview", "domain:forms/preview", "POST", "/forms/preview", {200}, {"reachable"}),
    Expectation("password policy evaluation", "domain:password-policies/evaluate", "POST", "/evaluate", {200}, {"reachable"}),
    Expectation("domain user bulk create", "domain:users/bulk", "POST", "/users/bulk", {200}, {"reachable"}),
    Expectation("user consents read", "domain:users/consents", "GET", "/consents", {200}, {"reachable"}),
    Expectation("user credentials read", "domain:users/credentials", "GET", "/credentials", {200}, {"reachable"}),
    Expectation("user devices read", "domain:users/devices", "GET", "/devices", {200}, {"reachable"}),
    Expectation("user factors read", "domain:users/factors", "GET", "/factors", {200}, {"reachable"}),
    Expectation("user identities read", "domain:users/identities", "GET", "/identities", {200}, {"reachable"}),
    Expectation("user consents delete by client", "domain:users/consents", "DELETE", "clientId=gap-probe-missing", {204}, {"reachable"}),
    Expectation("user consents revoke by id", "domain:users/consents", "DELETE", "/consents/gap-probe-missing", {404}, {"not-fixtureable"}),
    Expectation("user credentials delete", "domain:users/credentials", "DELETE", "/credentials/gap-probe-missing", {404}, {"not-fixtureable"}),
    Expectation("user devices delete", "domain:users/devices", "DELETE", "/devices/gap-probe-missing", {404}, {"not-fixtureable"}),
    Expectation("user factors delete", "domain:users/factors", "DELETE", "/factors/gap-probe-missing", {204}, {"reachable"}),
    Expectation("user identities delete", "domain:users/identities", "DELETE", "/identities/gap-probe-missing", {204}, {"reachable"}),
    Expectation("org user bulk create", "org:users/bulk", "POST", "/users/bulk", {200}, {"reachable"}),
    Expectation("current user newsletter subscribe", "self:newsletter/_subscribe", "POST", "/newsletter/_subscribe", {200}, {"reachable"}),
    Expectation("current user notification acknowledge", "self:notifications/acknowledge", "POST", "/notifications/gap-probe-missing/acknowledge", {204}, {"reachable"}),
]


class AMProbe:
    def __init__(self, base_url: str, client_id: str, client_secret: str, org_id: str, env_id: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.client_id = client_id
        self.client_secret = client_secret
        self.org_id = org_id
        self.env_id = env_id
        self.token = self._token()
        self.cleanups: list[tuple[str, str]] = []

    def _token(self) -> str:
        body = urllib.parse.urlencode({"grant_type": "client_credentials"}).encode()
        credentials = base64.b64encode(f"{self.client_id}:{self.client_secret}".encode()).decode()
        req = urllib.request.Request(
            f"{self.base_url}/management/auth/token",
            data=body,
            headers={
                "Authorization": f"Basic {credentials}",
                "Content-Type": "application/x-www-form-urlencoded",
            },
            method="POST",
        )
        with urllib.request.urlopen(req, timeout=20) as resp:
            payload = json.loads(resp.read().decode())
        return payload["access_token"]

    def management(self, path: str) -> str:
        return f"/management/organizations/{self.org_id}/environments/{self.env_id}{path}"

    def org(self, path: str) -> str:
        return f"/management/organizations/{self.org_id}{path}"

    def user(self, path: str) -> str:
        return f"/management/user{path}"

    def request(self, method: str, path: str, body: Any | None = None) -> tuple[int | None, bytes]:
        data = None
        headers = {"Authorization": f"Bearer {self.token}"}
        if body is not None:
            data = json.dumps(body).encode()
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(self.base_url + path, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=20) as resp:
                return resp.status, resp.read()
        except urllib.error.HTTPError as err:
            return err.code, err.read()
        except (TimeoutError, socket.timeout) as err:
            return None, str(err).encode()

    def json_request(self, method: str, path: str, body: Any | None = None) -> tuple[int | None, dict[str, Any]]:
        status, raw = self.request(method, path, body)
        if status is None:
            return None, {"error": raw.decode(errors="replace") or "request timed out"}
        if not raw:
            return status, {}
        try:
            return status, json.loads(raw.decode())
        except json.JSONDecodeError:
            return status, {"raw": raw.decode(errors="replace")}

    def create_domain(self, name: str) -> str | None:
        status, payload = self.json_request(
            "POST",
            self.management("/domains"),
            {"name": name, "dataPlaneId": "default"},
        )
        if status not in {200, 201}:
            return None
        domain_id = payload.get("id")
        if isinstance(domain_id, str) and domain_id:
            self.cleanups.append(("domain", domain_id))
            return domain_id
        return None

    def create_user(self, domain_id: str, username: str) -> str | None:
        status, payload = self.json_request(
            "POST",
            self.management(f"/domains/{domain_id}/users"),
            {
                "username": username,
                "email": f"{username}@example.com",
                "firstName": "Gap",
                "lastName": "Probe",
                "enabled": True,
                "preRegistration": True,
            },
        )
        if status not in {200, 201}:
            return None
        user_id = payload.get("id")
        return user_id if isinstance(user_id, str) and user_id else None

    def create_password_policy(self, domain_id: str, name: str) -> str | None:
        status, payload = self.json_request(
            "POST",
            self.management(f"/domains/{domain_id}/password-policies"),
            {"name": name, "minLength": 8},
        )
        if status not in {200, 201}:
            return None
        policy_id = payload.get("id")
        return policy_id if isinstance(policy_id, str) and policy_id else None

    def create_org_user(self, username: str) -> str | None:
        status, payload = self.json_request(
            "POST",
            self.org("/users"),
            {
                "username": username,
                "password": "SecurePass123!",
                "email": f"{username}@example.com",
                "firstName": "Gap",
                "lastName": "Probe",
                "enabled": False,
                "preRegistration": True,
            },
        )
        if status not in {200, 201}:
            return None
        user_id = payload.get("id")
        if isinstance(user_id, str) and user_id:
            self.cleanups.append(("org_user", user_id))
            return user_id
        return None

    def current_user_email(self) -> str:
        status, payload = self.json_request("GET", self.user(""))
        if status == 200:
            email = payload.get("email")
            if isinstance(email, str) and email:
                return email
        return "admin@example.com"

    def cleanup(self) -> None:
        for kind, obj_id in reversed(self.cleanups):
            if kind == "org_user":
                self.request("DELETE", self.org(f"/users/{obj_id}"))
            elif kind == "domain":
                self.request("DELETE", self.management(f"/domains/{obj_id}"))


def classify(status: int | None, payload: dict[str, Any]) -> tuple[str, str]:
    text = json.dumps(payload, sort_keys=True)[:220]
    if status is None:
        return "timeout", text
    if 200 <= status < 300:
        return "reachable", text
    if status in {400, 403, 404, 405}:
        return "not-fixtureable", text
    return "unexpected", text


def add_result(results: list[Result], family: str, method: str, path: str, status: int | None, payload: dict[str, Any]) -> None:
    outcome, detail = classify(status, payload)
    results.append(Result(family, method, path, status, outcome, detail))


def check_results(results: list[Result]) -> list[str]:
    failures: list[str] = []
    matched_results: set[int] = set()

    for expectation in EXPECTED_RESULTS:
        matches = [
            (index, result)
            for index, result in enumerate(results)
            if result.family == expectation.family
            and result.method == expectation.method
            and expectation.path_contains in result.path
        ]
        if not matches:
            failures.append(f"{expectation.label}: missing result for {expectation.family} {expectation.method}")
            continue
        if len(matches) > 1:
            failures.append(f"{expectation.label}: matched {len(matches)} probe results")
            continue

        index, result = matches[0]
        matched_results.add(index)
        if result.status not in expectation.statuses:
            statuses = ", ".join("timeout" if status is None else str(status) for status in sorted(expectation.statuses, key=lambda item: -1 if item is None else item))
            failures.append(f"{expectation.label}: expected status {statuses}, got {result.status}")
        if result.outcome not in expectation.outcomes:
            failures.append(f"{expectation.label}: expected outcome {', '.join(sorted(expectation.outcomes))}, got {result.outcome}")

    for index, result in enumerate(results):
        if index not in matched_results:
            failures.append(f"unexpected result: {result.family} {result.method} {result.path} status={result.status} outcome={result.outcome}")

    return failures


def run_probe(probe: AMProbe) -> list[Result]:
    results: list[Result] = []
    suffix = str(int(time.time()))
    domain_id = probe.create_domain(f"gap-probe-{suffix}")

    if domain_id:
        user_id = probe.create_user(domain_id, f"gap-probe-user-{suffix}")
        add_result(
            results,
            "domain:forms/preview",
            "POST",
            f"/domains/{domain_id}/forms/preview",
            *probe.json_request(
                "POST",
                probe.management(f"/domains/{domain_id}/forms/preview"),
                {"type": "FORM", "template": "login", "content": "<html>{{content}}</html>"},
            ),
        )
        policy_id = probe.create_password_policy(domain_id, f"gap-probe-policy-{suffix}")
        if policy_id:
            add_result(
                results,
                "domain:password-policies/evaluate",
                "POST",
                f"/domains/{domain_id}/password-policies/{policy_id}/evaluate",
                *probe.json_request(
                    "POST",
                    probe.management(f"/domains/{domain_id}/password-policies/{policy_id}/evaluate"),
                    {"password": "SecurePass123!", "userId": user_id or ""},
                ),
            )
        add_result(
            results,
            "domain:users/bulk",
            "POST",
            f"/domains/{domain_id}/users/bulk",
            *probe.json_request(
                "POST",
                probe.management(f"/domains/{domain_id}/users/bulk"),
                {
                    "action": "CREATE",
                    "items": [
                        {
                            "username": f"gap-probe-bulk-{suffix}",
                            "email": f"gap-probe-bulk-{suffix}@example.com",
                            "enabled": True,
                            "preRegistration": True,
                        }
                    ],
                },
            ),
        )
        if user_id:
            for collection in ["consents", "credentials", "devices", "factors", "identities"]:
                add_result(
                    results,
                    f"domain:users/{collection}",
                    "GET",
                    f"/domains/{domain_id}/users/{user_id}/{collection}",
                    *probe.json_request(
                        "GET",
                        probe.management(f"/domains/{domain_id}/users/{user_id}/{collection}"),
                    ),
                )
            add_result(
                results,
                "domain:users/consents",
                "DELETE",
                f"/domains/{domain_id}/users/{user_id}/consents?clientId=gap-probe-missing",
                *probe.json_request(
                    "DELETE",
                    probe.management(f"/domains/{domain_id}/users/{user_id}/consents?clientId=gap-probe-missing"),
                ),
            )
            for collection in ["consents", "credentials", "devices", "factors", "identities"]:
                add_result(
                    results,
                    f"domain:users/{collection}",
                    "DELETE",
                    f"/domains/{domain_id}/users/{user_id}/{collection}/gap-probe-missing",
                    *probe.json_request(
                        "DELETE",
                        probe.management(f"/domains/{domain_id}/users/{user_id}/{collection}/gap-probe-missing"),
                    ),
                )
    org_user_id = probe.create_org_user(f"gap-probe-org-user-{suffix}")
    add_result(
        results,
        "org:users/bulk",
        "POST",
        "/users/bulk",
        *probe.json_request(
            "POST",
            probe.org("/users/bulk"),
            {
                "action": "CREATE",
                "items": [
                    {
                        "username": f"gap-probe-org-bulk-{suffix}",
                        "password": "SecurePass123!",
                        "email": f"gap-probe-org-bulk-{suffix}@example.com",
                        "enabled": False,
                        "preRegistration": True,
                    }
                ],
            },
        ),
    )
    add_result(
        results,
        "self:newsletter/_subscribe",
        "POST",
        "/user/newsletter/_subscribe",
        *probe.json_request(
            "POST",
            probe.user("/newsletter/_subscribe"),
            {"email": probe.current_user_email()},
        ),
    )
    add_result(
        results,
        "self:notifications/acknowledge",
        "POST",
        "/user/notifications/gap-probe-missing/acknowledge",
        *probe.json_request(
            "POST",
            probe.user("/notifications/gap-probe-missing/acknowledge"),
        ),
    )

    return results


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--api-url", default="http://localhost:8093")
    parser.add_argument("--client-id", default="admin")
    parser.add_argument("--client-secret", default="adminadmin")
    parser.add_argument("--org-id", default="DEFAULT")
    parser.add_argument("--env-id", default="DEFAULT")
    parser.add_argument("--json", action="store_true", help="print machine-readable JSON")
    parser.add_argument("--check", action="store_true", help="fail if local gap behavior differs from documented expectations")
    args = parser.parse_args()

    probe = AMProbe(args.api_url, args.client_id, args.client_secret, args.org_id, args.env_id)
    try:
        results = run_probe(probe)
    finally:
        probe.cleanup()

    if args.json:
        print(json.dumps([result.__dict__ for result in results], indent=2, sort_keys=True))
    else:
        print("# Gravitee AM OpenAPI Gap Probe")
        for result in results:
            print(f"- {result.family} {result.method} {result.path}: {result.outcome} status={result.status} {result.detail}")

    failures = check_results(results) if args.check else []
    if failures:
        print("\n## Failures")
        for failure in failures:
            print(f"- {failure}")
        return 1

    return 0 if all(result.outcome in {"reachable", "not-fixtureable", "timeout"} for result in results) else 1


if __name__ == "__main__":
    sys.exit(main())
