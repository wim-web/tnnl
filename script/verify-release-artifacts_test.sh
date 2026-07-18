#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
verifier="$repo_root/script/verify-release-artifacts.sh"
bash_path="$(command -v bash)"
host_tar="$(command -v tar || true)"

if [[ -z "$host_tar" ]]; then
  echo "tar is required to run release artifact verifier tests" >&2
  exit 1
fi

umask 077
fixture_root="$(mktemp -d "${TMPDIR:-/tmp}/tnnl-release-verifier-test.XXXXXX")"
trap 'rm -rf "$fixture_root"' EXIT

dist="$fixture_root/dist"
source_dir="$fixture_root/source"
tool_dir="$fixture_root/tools"
asset="tnnl_test_fixture.tar.gz"
sidecar="release-notes.txt"
tar_called="$fixture_root/tar-called"
mkdir -p "$dist" "$source_dir" "$tool_dir"

export VERIFY_RELEASE_TEST_HOST_TAR="$host_tar"
export VERIFY_RELEASE_TEST_TAR_CALLED="$tar_called"
cat >"$tool_dir/tar" <<'EOF'
#!/bin/sh
: >"$VERIFY_RELEASE_TEST_TAR_CALLED"
exec "$VERIFY_RELEASE_TEST_HOST_TAR" "$@"
EOF
chmod 0755 "$tool_dir/tar"
export PATH="$tool_dir:$PATH"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

write_binary() {
  local version="$1"

  {
    printf '%s\n' '#!/bin/sh'
    printf '%s\n' 'if [ "${1:-}" != "version" ]; then exit 64; fi'
    printf "printf '%%s\\n' '%s'\n" "$version"
  } >"$source_dir/tnnl"
  chmod 0755 "$source_dir/tnnl"
}

package_asset() {
  "$host_tar" -czf "$dist/$asset" -C "$source_dir" tnnl
}

write_checksums() {
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$dist" && sha256sum "$asset" "$sidecar" >checksums.txt)
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$dist" && shasum -a 256 "$asset" "$sidecar" >checksums.txt)
  else
    fail "sha256sum or shasum is required to build the test fixture"
  fi
}

write_sidecar_checksum_only() {
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$dist" && sha256sum "$sidecar" >checksums.txt)
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$dist" && shasum -a 256 "$sidecar" >checksums.txt)
  else
    fail "sha256sum or shasum is required to build the test fixture"
  fi
}

write_corrupt_sidecar_checksum() {
  if command -v sha256sum >/dev/null 2>&1; then
    (cd "$dist" && sha256sum "$asset" >checksums.txt)
  elif command -v shasum >/dev/null 2>&1; then
    (cd "$dist" && shasum -a 256 "$asset" >checksums.txt)
  else
    fail "sha256sum or shasum is required to build the test fixture"
  fi
  printf '%064d  %s\n' 0 "$sidecar" >>"$dist/checksums.txt"
}

expect_success() {
  local description="$1"
  shift
  local output

  if ! output="$("$bash_path" "$verifier" "$@" 2>&1)"; then
    fail "$description: verifier failed: $output"
  fi
}

expect_failure() {
  local description="$1"
  local expected="$2"
  shift 2
  local output status

  set +e
  output="$("$bash_path" "$verifier" "$@" 2>&1)"
  status=$?
  set -e

  if ((status == 0)); then
    fail "$description: verifier unexpectedly succeeded"
  fi
  if [[ "$output" != *"$expected"* ]]; then
    fail "$description: output $output does not contain $expected"
  fi
}

write_binary "1.2.3"
package_asset
printf '%s\n' "release notes" >"$dist/$sidecar"
write_checksums

expect_success "valid release" "v1.2.3" "$dist" "$asset"

unlisted_asset="tnnl_unlisted.tar.gz"
cp "$dist/$asset" "$dist/$unlisted_asset"
expect_failure "unlisted publishable asset" "publishable release asset is not listed in checksum manifest" "v1.2.3" "$dist" "$asset"
rm -f "$dist/$unlisted_asset"

write_sidecar_checksum_only
rm -f "$tar_called"
expect_failure "asset missing from manifest" "release asset is not listed in checksum manifest" "v1.2.3" "$dist" "$asset"
if [[ -e "$tar_called" ]]; then
  fail "asset missing from manifest: archive extraction ran without an asset checksum"
fi
write_checksums

write_corrupt_sidecar_checksum
rm -f "$tar_called"
expect_failure "corrupt checksum" "$sidecar" "v1.2.3" "$dist" "$asset"
if [[ -e "$tar_called" ]]; then
  fail "corrupt checksum: archive extraction ran before the complete manifest passed"
fi
write_checksums

write_binary "9.9.9"
package_asset
write_checksums
expect_failure "wrong binary version" "release version mismatch: got 9.9.9, want 1.2.3" "v1.2.3" "$dist" "$asset"

expect_failure "missing asset" "release asset not found" "v1.2.3" "$dist" "missing.tar.gz"
expect_failure "invalid tag" "invalid release tag" "1.2.3" "$dist" "$asset"

empty_path="$fixture_root/empty-path"
mkdir -p "$empty_path"
set +e
no_checksum_output="$(PATH="$empty_path" "$bash_path" "$verifier" "v1.2.3" "$dist" "$asset" 2>&1)"
no_checksum_status=$?
set -e
if ((no_checksum_status == 0)); then
  fail "missing checksum tool: verifier unexpectedly succeeded"
fi
if [[ "$no_checksum_output" != *"sha256sum or shasum is required"* ]]; then
  fail "missing checksum tool: output $no_checksum_output does not explain the requirement"
fi

echo "release artifact verifier fixture tests passed"
