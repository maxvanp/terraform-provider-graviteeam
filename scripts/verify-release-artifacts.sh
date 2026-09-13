#!/usr/bin/env bash

set -euo pipefail

dist_dir="${1:-dist}"
project="terraform-provider-graviteeam"

checksum_file="$(find "$dist_dir" -maxdepth 1 -type f -name "${project}_*_SHA256SUMS" -print -quit)"
if [[ -z "$checksum_file" ]]; then
  echo "release checksum file not found in $dist_dir" >&2
  exit 1
fi

version="${checksum_file#"$dist_dir/${project}_"}"
version="${version%_SHA256SUMS}"
expected_platforms=(
  darwin_amd64
  darwin_arm64
  linux_amd64
  linux_arm
  linux_arm64
  windows_amd64
)

for platform in "${expected_platforms[@]}"; do
  archive="${dist_dir}/${project}_${version}_${platform}.zip"
  if [[ ! -f "$archive" ]]; then
    echo "missing release archive: $archive" >&2
    exit 1
  fi

  if ! awk -v name="$(basename "$archive")" '$2 == name { found = 1 } END { exit !found }' "$checksum_file"; then
    echo "missing release archive checksum: $archive" >&2
    exit 1
  fi

  binary_suffix=""
  [[ "$platform" == windows_* ]] && binary_suffix=".exe"
  binary="${project}_v${version}${binary_suffix}"
  if ! unzip -Z1 "$archive" | grep -qxF "$binary"; then
    echo "$archive does not contain the expected binary $binary" >&2
    exit 1
  fi
done

archive_count="$(find "$dist_dir" -maxdepth 1 -type f -name "${project}_${version}_*.zip" | wc -l)"
if [[ "$archive_count" -ne "${#expected_platforms[@]}" ]]; then
  echo "expected ${#expected_platforms[@]} release archives, found $archive_count" >&2
  exit 1
fi

(
  cd "$dist_dir"
  sha256sum --ignore-missing --strict -c "$(basename "$checksum_file")"
)

manifest_name="${project}_${version}_manifest.json"
expected_manifest_hash="$(awk -v name="$manifest_name" '$2 == name { print $1 }' "$checksum_file")"
actual_manifest_hash="$(sha256sum terraform-registry-manifest.json | awk '{ print $1 }')"
if [[ -z "$expected_manifest_hash" || "$expected_manifest_hash" != "$actual_manifest_hash" ]]; then
  echo "Terraform Registry manifest checksum is missing or incorrect" >&2
  exit 1
fi

linux_binary="$(find "$dist_dir" -mindepth 2 -maxdepth 2 -type f -path "*/${project}_linux_amd64_v1/${project}_v${version}" -print -quit)"
if [[ -z "$linux_binary" ]] || ! file "$linux_binary" | grep -q "statically linked"; then
  echo "Linux AMD64 provider binary is missing or dynamically linked" >&2
  exit 1
fi

echo "release artifacts verified for version $version"
