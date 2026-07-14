#!/usr/bin/env bash

set -euo pipefail

ref="${1:-4.12.1}"
url="https://raw.githubusercontent.com/gravitee-io/gravitee-access-management/${ref}/docs/mapi/openapi.yaml"

curl -fsSL "${url}" -o docs/openapi.yaml
echo "Updated docs/openapi.yaml from ${url}"
