#!/usr/bin/env bash
set -euo pipefail

usage="usage: verify-release-artifacts.sh vX.Y.Z DIST_DIR ASSET_NAME"
if (($# != 3)); then
  echo "$usage" >&2
  exit 2
fi

tag="$1"
dist="$2"
asset="$3"

if [[ ! "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "invalid release tag: $tag (want vX.Y.Z)" >&2
  exit 1
fi
if [[ ! -d "$dist" ]]; then
  echo "release dist directory not found: $dist" >&2
  exit 1
fi
if [[ ! -f "$dist/checksums.txt" ]]; then
  echo "release checksum manifest not found: $dist/checksums.txt" >&2
  exit 1
fi
if [[ ! -f "$dist/$asset" ]]; then
  echo "release asset not found: $dist/$asset" >&2
  exit 1
fi

asset_listed=0
while IFS= read -r manifest_line || [[ -n "$manifest_line" ]]; do
  if [[ "$manifest_line" =~ ^[[:xdigit:]]{64}[[:space:]]+[*]?(.+)$ ]] &&
    [[ "${BASH_REMATCH[1]}" == "$asset" ]]; then
    asset_listed=1
    break
  fi
done <"$dist/checksums.txt"
if ((asset_listed == 0)); then
  echo "release asset is not listed in checksum manifest: $asset" >&2
  exit 1
fi

for publishable_path in "$dist"/tnnl_*.tar.gz; do
  [[ -e "$publishable_path" ]] || continue
  publishable_asset="${publishable_path##*/}"
  publishable_listed=0
  while IFS= read -r manifest_line || [[ -n "$manifest_line" ]]; do
    if [[ "$manifest_line" =~ ^[[:xdigit:]]{64}[[:space:]]+[*]?(.+)$ ]] &&
      [[ "${BASH_REMATCH[1]}" == "$publishable_asset" ]]; then
      publishable_listed=1
      break
    fi
  done <"$dist/checksums.txt"
  if ((publishable_listed == 0)); then
    echo "publishable release asset is not listed in checksum manifest: $publishable_asset" >&2
    exit 1
  fi
done

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$dist" && sha256sum -c checksums.txt)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$dist" && shasum -a 256 -c checksums.txt)
else
  echo "sha256sum or shasum is required to verify release artifacts" >&2
  exit 1
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

tar -xzf "$dist/$asset" -C "$tmp"
if [[ ! -x "$tmp/tnnl" ]]; then
  echo "release archive does not contain an executable tnnl" >&2
  exit 1
fi

expected="${tag#v}"
actual="$("$tmp/tnnl" version)"
if [[ "$actual" != "$expected" ]]; then
  echo "release version mismatch: got $actual, want $expected" >&2
  exit 1
fi
