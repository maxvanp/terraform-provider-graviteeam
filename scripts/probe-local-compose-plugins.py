#!/usr/bin/env python3
"""Probe plugin availability in the stock local AM docker-compose stack."""

from __future__ import annotations

import argparse
import base64
import json
import socket
import sys
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from typing import Any


EXPECTED_LOCAL_GAPS = {
    "authorization-engines/openfga": "OpenFGA is loaded by the image but not deployed without the Enterprise/preview feature",
}


@dataclass(frozen=True)
class PluginProbe:
    category: str
    plugin_id: str
    deployed: bool
    feature: str | None
    schema_status: int | None
    detail: str


def token(base_url: str, client_id: str, client_secret: str) -> str:
    body = urllib.parse.urlencode({"grant_type": "client_credentials"}).encode()
    credentials = base64.b64encode(f"{client_id}:{client_secret}".encode()).decode()
    req = urllib.request.Request(
        f"{base_url}/management/auth/token",
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


class LocalComposeUnavailable(RuntimeError):
    """Raised when the stock local compose API is not reachable."""


def token_or_unavailable(base_url: str, client_id: str, client_secret: str) -> str:
    try:
        return token(base_url, client_id, client_secret)
    except urllib.error.URLError as err:
        reason = getattr(err, "reason", err)
        raise LocalComposeUnavailable(f"{base_url}: {reason}") from err
    except TimeoutError as err:
        raise LocalComposeUnavailable(f"{base_url}: {err}") from err


def request(base_url: str, access_token: str, path: str) -> tuple[int | None, Any]:
    req = urllib.request.Request(
        base_url + path,
        headers={"Authorization": f"Bearer {access_token}"},
        method="GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            raw = resp.read()
            if not raw:
                return resp.status, None
            return resp.status, json.loads(raw.decode())
    except urllib.error.HTTPError as err:
        raw = err.read()
        try:
            return err.code, json.loads(raw.decode()) if raw else None
        except json.JSONDecodeError:
            return err.code, raw.decode(errors="replace")
    except (TimeoutError, socket.timeout) as err:
        return None, str(err)


def probe_category(base_url: str, access_token: str, category: str) -> list[PluginProbe]:
    status, payload = request(base_url, access_token, f"/management/platform/plugins/{category}")
    if status != 200:
        raise RuntimeError(f"{category}: plugin list returned status {status}: {payload}")
    if not isinstance(payload, list):
        raise RuntimeError(f"{category}: plugin list returned non-list payload: {payload!r}")

    probes: list[PluginProbe] = []
    for plugin in payload:
        plugin_id = plugin.get("id")
        if not plugin_id:
            continue
        schema_status, schema_payload = request(
            base_url,
            access_token,
            f"/management/platform/plugins/{category}/{plugin_id}/schema",
        )
        deployed = bool(plugin.get("deployed"))
        feature = plugin.get("feature")
        if deployed and schema_status == 200:
            detail = "deployed with schema"
        elif deployed:
            detail = f"deployed but schema returned {schema_status}"
        elif feature:
            detail = f"not deployed; feature={feature}; schema returned {schema_status}"
        else:
            detail = f"not deployed; schema returned {schema_status}"
        if schema_status not in (200, 204, 404, None):
            detail += f"; schema payload={schema_payload!r}"
        probes.append(
            PluginProbe(
                category=category,
                plugin_id=plugin_id,
                deployed=deployed,
                feature=feature,
                schema_status=schema_status,
                detail=detail,
            )
        )
    return probes


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default="http://localhost:8093")
    parser.add_argument("--client-id", default="admin")
    parser.add_argument("--client-secret", default="adminadmin")
    parser.add_argument(
        "--category",
        action="append",
        default=["authorization-engines"],
        help="plugin category to probe; can be passed multiple times",
    )
    parser.add_argument("--check", action="store_true", help="fail if documented local plugin gaps change")
    args = parser.parse_args()

    base_url = args.base_url.rstrip("/")
    try:
        access_token = token_or_unavailable(base_url, args.client_id, args.client_secret)
    except LocalComposeUnavailable as err:
        print("# Local Compose Plugin Probe")
        print(f"- skipped: local Gravitee AM Management API is unavailable ({err})")
        return 0

    all_probes: list[PluginProbe] = []
    for category in args.category:
        all_probes.extend(probe_category(base_url, access_token, category))

    print("# Local Compose Plugin Probe")
    for probe in all_probes:
        print(f"- {probe.category}/{probe.plugin_id}: {probe.detail}")

    if not args.check:
        return 0

    probe_by_key = {f"{probe.category}/{probe.plugin_id}": probe for probe in all_probes}
    failures: list[str] = []
    for key, expectation in EXPECTED_LOCAL_GAPS.items():
        probe = probe_by_key.get(key)
        if probe is None:
            failures.append(f"{key}: expected documented gap but plugin was not listed ({expectation})")
            continue
        if probe.deployed:
            failures.append(f"{key}: expected documented local gap but plugin is now deployed")
        if probe.schema_status == 200:
            failures.append(f"{key}: expected no usable schema for non-deployed plugin, got status 200")

    if failures:
        print("\n## Unexpected Plugin Probe Results")
        for failure in failures:
            print(f"- {failure}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
