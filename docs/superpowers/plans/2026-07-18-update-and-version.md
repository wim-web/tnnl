# Verified Update and Version Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Report an honest, consistent version for local, `go install`, and release builds, and replace the running executable only with a checksum-verified binary whose reported version matches the release.

**Architecture:** A small `internal/buildinfo` package resolves linker metadata before module BuildInfo and otherwise returns `dev`. The updater is split into release/download verification, candidate validation, and same-directory atomic installation so every unsafe boundary has focused tests and the orchestration cannot replace the executable before all gates pass.

**Tech Stack:** Go 1.25+, `runtime/debug`, GoReleaser v2.17.0, SHA-256, `net/http`, `archive/tar`, `compress/gzip`, Cobra, and Go `testing`/`httptest`.

---

This is implementation plan 3 of 4. It can run after plans 1 and 2; it assumes the root command already returns errors and accepts a context. The approved design is `docs/superpowers/specs/2026-07-18-product-review-findings-design.md`.

## File map

- `internal/buildinfo/version.go`: linker > module BuildInfo > `dev` resolution.
- `internal/buildinfo/version_test.go`: every build-source precedence case.
- `cmd/root.go`, `main.go`: consume resolved version; remove embedded `.version` runtime path.
- `.version`, `tag.bash`: remove stale source and require an explicit validated tag argument.
- `.goreleaser.yml`: explicit linker flag, stable archive/checksum contract.
- `aqua.yaml`: locally reproducible GoReleaser version.
- `cmd/update/release.go`: release lookup, context-aware downloads, exact checksum parsing and verification.
- `cmd/update/install.go`: archive extraction, candidate version check, unique same-directory replacement.
- `cmd/update/update.go`: dependency-injected orchestration and Cobra `RunE`.
- `cmd/update/*_test.go`: release, checksum, version, atomic install, and end-to-end local fixture tests.

### Task 1: Runtime version resolution and removal of stale embedding

**Files:**
- Create: `internal/buildinfo/version.go`
- Create: `internal/buildinfo/version_test.go`
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`
- Modify: `main.go`
- Modify: `tag.bash`
- Delete: `.version`

- [ ] **Step 1: Write failing source-precedence tests**

```go
func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name   string
		linker string
		module string
		ok     bool
		want   string
	}{
		{name: "linker wins", linker: "v1.2.3", module: "v9.9.9", ok: true, want: "1.2.3"},
		{name: "module backs go install", linker: "dev", module: "v2.3.4", ok: true, want: "2.3.4"},
		{name: "empty linker uses module", linker: "", module: "v2.3.4", ok: true, want: "2.3.4"},
		{name: "devel module is honest", linker: "", module: "(devel)", ok: true, want: "dev"},
		{name: "missing build info", linker: "", ok: false, want: "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{Main: debug.Module{Version: tt.module}}
			if got := resolveVersion(tt.linker, info, tt.ok); got != tt.want {
				t.Fatalf("resolveVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
```

Add rows for whitespace and a single optional leading `v`. Version output remains compatible with current release artifacts: `0.6.17`, not `v0.6.17`.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./internal/buildinfo -count=1`

Expected: package-not-found failure.

- [ ] **Step 3: Implement the resolver**

```go
package buildinfo

import (
	"runtime/debug"
	"strings"
)

var linkerVersion = "dev"

func Current() string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(linkerVersion, info, ok)
}

func resolveVersion(linker string, info *debug.BuildInfo, ok bool) string {
	if value := canonical(linker); value != "" && value != "dev" {
		return value
	}
	if ok && info != nil {
		if value := canonical(info.Main.Version); value != "" && value != "dev" && value != "(devel)" {
			return value
		}
	}
	return "dev"
}

func canonical(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}
```

- [ ] **Step 4: Wire root output to BuildInfo**

Set `var Version = buildinfo.Current()` in `cmd/root.go`; both `tnnl version` and `tnnl --version` write `Version` through `cmd.OutOrStdout()` and return write errors. Remove `embed` and `.version` assignment from `main.go` while preserving the signal-aware `cmd.ExecuteContext` flow from plan 2.

Extend `cmd/root_test.go` to set/restore `Version`, run both version paths, and assert identical output.

- [ ] **Step 5: Make tagging require the intended release version**

Replace `.version` lookup with:

```bash
set -euo pipefail
version="${1:-}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "usage: ./tag.bash vX.Y.Z" >&2
  exit 2
fi
```

Keep the existing confirmation before `git tag` and `git push`, quote every expansion, and correct the success message so it does not add a second `v`.

- [ ] **Step 6: Verify versions and shell syntax**

Run:

```bash
go test ./internal/buildinfo ./cmd -run 'Test(ResolveVersion|Version)' -count=1
bash -n tag.bash
./tag.bash
go build -o /tmp/tnnl-dev .
test "$(/tmp/tnnl-dev version)" = dev
go build -ldflags '-X github.com/wim-web/tnnl/internal/buildinfo.linkerVersion=1.2.3' -o /tmp/tnnl-release .
test "$(/tmp/tnnl-release version)" = 1.2.3
```

Expected: tests and syntax pass; no-argument tag script exits 2 before Git; local build reports `dev`; linker build reports `1.2.3`.

- [ ] **Step 7: Commit version resolution**

```bash
git add internal/buildinfo cmd/root.go cmd/root_test.go main.go tag.bash .version
git commit -m "feat: resolve version from build metadata"
```

### Task 2: Explicit GoReleaser artifact contract

**Files:**
- Rewrite: `.goreleaser.yml`
- Modify: `aqua.yaml`

- [ ] **Step 1: Pin and install the test tool, then record the failing artifact contract**

Add `goreleaser/goreleaser@v2.17.0` to `aqua.yaml` first; this is the test runner prerequisite, not the production contract under test. Run `aqua install`, then execute these commands individually so an invalid old config cannot hide which assertion failed:

```bash
goreleaser check
goreleaser release --snapshot --clean --skip=publish
test -f dist/checksums.txt
test -f dist/tnnl_linux_amd64.tar.gz
red_dir="$(mktemp -d)"
red_asset="tnnl_$(go env GOOS)_$(go env GOARCH).tar.gz"
tar -xzf "dist/$red_asset" -C "$red_dir"
test "$("$red_dir/tnnl" version)" != dev
rm -rf "$red_dir"
```

Expected before implementation: at least `goreleaser check` rejects the v1-shaped file or the extracted binary reports `dev` because no linker contract exists. Record that non-zero result as RED. Do not use `goreleaser build` for this assertion; it does not produce the archive/checksum contract.

- [ ] **Step 2: Pin GoReleaser and define every artifact name**

Keep the `goreleaser/goreleaser@v2.17.0` pin added in Step 1. Use this complete configuration shape:

```yaml
version: 2
project_name: tnnl
builds:
  - id: tnnl
    main: .
    binary: tnnl
    env:
      - CGO_ENABLED=0
    ldflags:
      - >-
        -s -w
        -X github.com/wim-web/tnnl/internal/buildinfo.linkerVersion={{ .Version }}
    goos: [darwin, linux]
    goarch: [amd64, arm64]
archives:
  - ids: [tnnl]
    formats: [tar.gz]
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"
checksum:
  name_template: checksums.txt
  algorithm: sha256
```

Remove the release-time `go mod tidy` hook because release builds must not mutate dependency files.

- [ ] **Step 3: Verify the dry-run contract**

Run:

```bash
goreleaser check
goreleaser release --snapshot --clean --skip=publish
test -f dist/checksums.txt
test -f dist/tnnl_linux_amd64.tar.gz
```

Expected: all commands exit 0. Extract one archive and assert its `version` output is non-empty and not `dev`; snapshot text itself is allowed in this local dry run.

- [ ] **Step 4: Commit the release contract**

```bash
git add .goreleaser.yml aqua.yaml
git commit -m "build: define verifiable release artifacts"
```

### Task 3: Exact release checksum parsing and verification

**Files:**
- Create: `cmd/update/release.go`
- Create: `cmd/update/release_test.go`
- Modify: `cmd/update/update.go`

- [ ] **Step 1: Write failing manifest tests**

```go
func TestChecksumForAssetSelectsExactFilename(t *testing.T) {
	manifest := []byte(strings.Join([]string{
		strings.Repeat("a", 64) + "  tnnl_linux_arm64.tar.gz",
		strings.Repeat("b", 64) + "  tnnl_linux_amd64.tar.gz",
	}, "\n"))
	got, err := checksumForAsset(manifest, "tnnl_linux_amd64.tar.gz")
	if err != nil { t.Fatal(err) }
	if hex.EncodeToString(got[:]) != strings.Repeat("b", 64) { t.Fatalf("checksum = %x", got) }
}
```

Add cases for missing filename, duplicate exact filename, malformed nonblank line, non-64-character digest, non-hex digest, and a filename with the requested name only as a suffix.

- [ ] **Step 2: Write failing file hash tests**

Write known bytes into `t.TempDir`, calculate their expected digest in the test, and assert match succeeds and a one-byte-different digest fails with the asset path in the error.

- [ ] **Step 3: Run tests and confirm RED**

Run: `go test ./cmd/update -run 'Test(ChecksumForAsset|VerifyFileSHA256)' -count=1`

Expected: build failure because the functions do not exist.

- [ ] **Step 4: Implement strict exact-name parsing**

```go
func checksumForAsset(manifest []byte, assetName string) ([sha256.Size]byte, error) {
	var found [sha256.Size]byte
	matches := 0
	scanner := bufio.NewScanner(bytes.NewReader(manifest))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" { continue }
		fields := strings.Fields(line)
		if len(fields) != 2 { return found, fmt.Errorf("malformed checksum line %q", line) }
		name := strings.TrimPrefix(fields[1], "*")
		if name != assetName { continue }
		digest, err := hex.DecodeString(fields[0])
		if err != nil || len(digest) != sha256.Size { return found, fmt.Errorf("invalid SHA-256 for %s", assetName) }
		matches++
		copy(found[:], digest)
	}
	if err := scanner.Err(); err != nil { return found, fmt.Errorf("read checksum manifest: %w", err) }
	if matches != 1 { return found, fmt.Errorf("checksum manifest has %d entries for %s", matches, assetName) }
	return found, nil
}
```

`verifyFileSHA256` opens the archive, streams it through `sha256.New`, compares with `subtle.ConstantTimeCompare`, and returns `checksum mismatch for <path>` without extracting anything.

- [ ] **Step 5: Run tests and confirm GREEN**

Run: `go test ./cmd/update -run 'Test(ChecksumForAsset|VerifyFileSHA256)' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit checksum primitives**

```bash
git add cmd/update/release.go cmd/update/release_test.go cmd/update/update.go
git commit -m "feat(update): verify release archive checksum"
```

### Task 4: Candidate version gate

**Files:**
- Create: `cmd/update/install.go`
- Create: `cmd/update/install_test.go`
- Modify: `cmd/update/update_test.go`

- [ ] **Step 1: Write failing candidate tests**

Use executable shell fixtures under `t.TempDir()` on Darwin/Linux:

```go
func TestVerifyCandidateVersionRejectsMismatch(t *testing.T) {
	path := writeVersionScript(t, "9.9.9")
	err := verifyCandidateVersion(context.Background(), path, "v1.2.3")
	if err == nil || !strings.Contains(err.Error(), "candidate version 9.9.9 does not match release 1.2.3") {
		t.Fatalf("verifyCandidateVersion() error = %v", err)
	}
}
```

Add matching `v`/non-`v` normalization, empty output, non-zero exit, and timeout cases.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./cmd/update -run TestVerifyCandidateVersion -count=1`

Expected: build failure because the function does not exist.

- [ ] **Step 3: Implement context-derived verification**

```go
func verifyCandidateVersion(ctx context.Context, candidatePath, releaseTag string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(checkCtx, candidatePath, "version").Output()
	if err != nil { return fmt.Errorf("run candidate version: %w", err) }
	got := normalizeVersion(string(output))
	want := normalizeVersion(releaseTag)
	if got == "" { return errors.New("candidate version output is empty") }
	if got != want { return fmt.Errorf("candidate version %s does not match release %s", got, want) }
	return nil
}
```

If `checkCtx.Err()` is non-nil, wrap that deadline/cancellation so `errors.Is` works.

- [ ] **Step 4: Run tests and confirm GREEN**

Run: `go test ./cmd/update -run TestVerifyCandidateVersion -count=1`

Expected: PASS.

- [ ] **Step 5: Commit candidate verification**

```bash
git add cmd/update/install.go cmd/update/install_test.go cmd/update/update_test.go
git commit -m "feat(update): validate candidate binary version"
```

### Task 5: Unique same-directory atomic replacement

**Files:**
- Modify: `cmd/update/install.go`
- Modify: `cmd/update/install_test.go`
- Remove moved functions from: `cmd/update/update.go`

- [ ] **Step 1: Write failing fixed-temp regression test**

```go
func TestReplaceExecutableDoesNotTouchFixedTempName(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "tnnl")
	candidate := filepath.Join(dir, "candidate")
	fixed := filepath.Join(dir, ".tnnl.new")
	os.WriteFile(current, []byte("old"), 0o755)
	os.WriteFile(candidate, []byte("new"), 0o755)
	os.WriteFile(fixed, []byte("sentinel"), 0o600)
	if err := replaceExecutable(current, candidate); err != nil { t.Fatal(err) }
	got, _ := os.ReadFile(fixed)
	if string(got) != "sentinel" { t.Fatalf("fixed temp changed to %q", got) }
}
```

Add successful content/mode, no leftover `.tnnl.new-*`, and write/rename failure cleanup tests. A failure must remove only the unique file created by that call.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./cmd/update -run TestReplaceExecutable -count=1`

Expected: the fixed-name test fails because current code truncates/renames `.tnnl.new`.

- [ ] **Step 3: Implement unique synced replacement**

```go
func replaceExecutable(currentPath, candidatePath string) (err error) {
	src, err := os.Open(candidatePath)
	if err != nil { return fmt.Errorf("open candidate: %w", err) }
	defer src.Close()
	tmp, err := os.CreateTemp(filepath.Dir(currentPath), ".tnnl.new-*")
	if err != nil { return fmt.Errorf("create update file: %w", err) }
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed { _ = os.Remove(tmpPath) }
	}()
	if _, err := io.Copy(tmp, src); err != nil { _ = tmp.Close(); return fmt.Errorf("copy candidate: %w", err) }
	if err := tmp.Chmod(0o755); err != nil { _ = tmp.Close(); return fmt.Errorf("chmod update file: %w", err) }
	if err := tmp.Sync(); err != nil { _ = tmp.Close(); return fmt.Errorf("sync update file: %w", err) }
	if err := tmp.Close(); err != nil { return fmt.Errorf("close update file: %w", err) }
	if err := os.Rename(tmpPath, currentPath); err != nil { return fmt.Errorf("replace executable %q; check write permission: %w", currentPath, err) }
	committed = true
	return nil
}
```

- [ ] **Step 4: Run tests and confirm GREEN**

Run: `go test -race ./cmd/update -run TestReplaceExecutable -count=1`

Expected: PASS; pre-existing `.tnnl.new` is untouched.

- [ ] **Step 5: Commit atomic replacement**

```bash
git add cmd/update/install.go cmd/update/install_test.go cmd/update/update.go
git commit -m "fix(update): replace executable atomically"
```

### Task 6: Verified updater orchestration

**Files:**
- Rewrite: `cmd/update/update.go`
- Rewrite: `cmd/update/update_test.go`
- Modify: `cmd/update/release.go`
- Modify: `cmd/update/release_test.go`
- Modify: `cmd/update/install.go`
- Modify: `cmd/update/install_test.go`

- [ ] **Step 1: Define the injected updater and write the happy-path test**

```go
type updater struct {
	client         *http.Client
	latestURL      string
	goos           string
	goarch         string
	executablePath func() (string, error)
}

func (u updater) run(ctx context.Context, out io.Writer) error
```

Use `httptest.Server` for the latest redirect, archive, and `checksums.txt`; generate the tar.gz and an executable version-script fixture in `t.TempDir`. Assert the old executable is replaced only after both downloads, hash verification, extraction, and candidate version execution.

The fixture's `/releases/latest` response sets an absolute `Location` on the same test server, `/releases/tag/v1.2.3`. `fetchLatestRelease` derives the asset base from that redirect origin/path, so subsequent requests are `/releases/download/v1.2.3/<asset>` on the same server. This same derivation maps GitHub's production redirect to `https://github.com/wim-web/tnnl/releases/download/<tag>` without a test-only branch.

- [ ] **Step 2: Write failing safety-order tests**

Add:

- `TestUpdaterLeavesExecutableUntouchedOnChecksumMismatch`;
- `TestUpdaterDoesNotExtractBeforeChecksumVerification` using an invalid gzip plus mismatched checksum and asserting the checksum error wins;
- `TestUpdaterLeavesExecutableUntouchedOnCandidateVersionMismatch`;
- `TestUpdaterPropagatesCallerCancellation`;
- `TestUpdaterCancellationStopsCurrentVersionProbe`: make the installed fixture block in its `version` command, cancel the caller context, assert a prompt `context.Canceled` result and no asset request;
- `TestUpdaterAlreadyLatestMakesNoAssetRequests`;
- `TestFetchLatestReleaseUsesInjectedClientWithoutAuthorization`.

- [ ] **Step 3: Run orchestration tests and confirm RED**

Run: `go test ./cmd/update -run 'TestUpdater|TestFetchLatestReleaseUsesInjected' -count=1`

Expected: build/signature failures because current code uses global clients/background contexts and downloads no checksum manifest.

- [ ] **Step 4: Implement the immutable verification order**

Inside `run`:

```go
func (u updater) run(ctx context.Context, out io.Writer) error {
	executable, err := u.executablePath()
	if err != nil { return fmt.Errorf("resolve executable: %w", err) }
	latest, err := fetchLatestRelease(ctx, u.client, u.latestURL)
	if err != nil { return err }
	current, err := currentVersion(ctx, executable)
	if err != nil { return err }
	if current == normalizeVersion(latest.TagName) {
		_, err := fmt.Fprintf(out, "already latest version: %s\n", latest.TagName)
		return err
	}
	assetName := fmt.Sprintf("tnnl_%s_%s.tar.gz", u.goos, u.goarch)
	archiveURL, err := latest.assetURL(assetName)
	if err != nil { return err }
	checksumURL, err := latest.assetURL("checksums.txt")
	if err != nil { return err }
	tempDir, err := os.MkdirTemp("", "tnnl-update-*")
	if err != nil { return fmt.Errorf("create update directory: %w", err) }
	defer os.RemoveAll(tempDir)
	archivePath := filepath.Join(tempDir, assetName)
	manifestPath := filepath.Join(tempDir, "checksums.txt")
	if err := downloadFile(ctx, u.client, archiveURL, archivePath); err != nil { return err }
	if err := downloadFile(ctx, u.client, checksumURL, manifestPath); err != nil { return err }
	manifest, err := os.ReadFile(manifestPath)
	if err != nil { return fmt.Errorf("read checksum manifest: %w", err) }
	want, err := checksumForAsset(manifest, assetName)
	if err != nil { return err }
	if err := verifyFileSHA256(archivePath, want); err != nil { return err }
	candidate := filepath.Join(tempDir, binaryName)
	if err := extractBinaryFromArchive(archivePath, candidate); err != nil { return err }
	if err := verifyCandidateVersion(ctx, candidate, latest.TagName); err != nil { return err }
	if err := replaceExecutable(executable, candidate); err != nil { return err }
	_, err = fmt.Fprintf(out, "updated: v%s -> %s\n", current, latest.TagName)
	return err
}
```

All network helpers accept caller `ctx` and `u.client`; they derive bounded child timeouts but never start from `context.Background()`. All output uses `out`.

Define `fetchLatestRelease(ctx, client, latestURL)` to shallow-copy the injected `http.Client`, set `CheckRedirect` on that copy to return `http.ErrUseLastResponse`, and issue the request with the caller-derived context. Parse the absolute `Location` by requiring an HTTP(S) URL and a `/releases/tag/<tag>` path; preserve its origin and any path prefix before `/releases` when constructing `DownloadBaseURL` with `/releases/download/<escaped-tag>`. This makes the single-server fixture real and does not mutate the injected client.

Change the version probe boundary to `currentVersion(ctx, executable) (string, error)` and `readBinaryVersion(ctx, executable)`. The latter uses `context.WithTimeout(ctx, 5*time.Second)` and joins the child context error when `exec.CommandContext` returns only a process error. `currentVersion` returns a wrapped caller cancellation instead of falling back; for ordinary non-cancellation execution failures it may still fall back to `buildinfo.Current()`.

- [ ] **Step 5: Convert update to Cobra `RunE`**

```go
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Install the latest checksum-verified tnnl release",
	RunE: func(cmd *cobra.Command, args []string) error {
		return productionUpdater().run(cmd.Context(), cmd.OutOrStdout())
	},
}
```

Remove `log.Fatal`, global `http.DefaultClient` mutation in tests, and fixed background contexts.

- [ ] **Step 6: Run complete updater verification**

Run:

```bash
gofmt -w internal/buildinfo cmd/update cmd/root.go main.go
go test -race ./internal/buildinfo ./cmd/update ./cmd
go vet ./internal/buildinfo/... ./cmd/update/... ./cmd/...
go test -race ./...
go build ./...
goreleaser check
```

Expected: all commands exit 0.

- [ ] **Step 7: Commit verified orchestration**

```bash
git add cmd/update
git commit -m "feat(update): install only verified releases"
```

## Plan completion check

- GoReleaser linker metadata wins, `go install @version` uses module BuildInfo, and local builds say `dev`.
- `.version` can no longer silently lie about normal source builds.
- Archive/checksum names and SHA-256 algorithm are repository contracts.
- Checksum verification precedes extraction; candidate version verification precedes replacement.
- Replacement uses a unique same-directory file, sync, close, chmod, and rename.
- Update uses caller context, injected HTTP client, Cobra output, and one returned error path.
- Existing executable remains untouched on checksum, extraction, candidate, or download failures.
