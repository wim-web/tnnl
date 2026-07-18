# CI and Product Documentation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the implemented behavior into a documented product contract and prevent pull requests, main pushes, or releases from shipping regressions in formatting, tests, builds, checksums, or reported versions.

**Architecture:** Command help and README content are verified as executable documentation, while one reusable GitHub Actions quality workflow gates both normal changes and releases. A release produces non-published artifacts once, verifies every checksum plus a host-compatible representative binary version, and uploads those exact verified files with the GitHub CLI.

**Tech Stack:** GitHub Actions with SHA-pinned actions, aqua, actionlint v1.7.12, GoReleaser v2.17.0, Bash, Go 1.25+, Cobra help tests, and Markdown.

---

This is implementation plan 4 of 4 and runs last so documentation describes tested behavior rather than intended behavior. Apply the `gha-workflow` skill while changing workflows: actions stay pinned to 40-character SHAs, permissions are job-scoped, every job has a timeout, and release publication depends on the quality gate. Terraform and profile/region flags remain out of scope.

## File map

- `cmd/root_test.go`, command package tests: help text is a user-visible behavior contract.
- `README.md`: installation, AWS context, IAM/readiness, commands, input/wait/update behavior, troubleshooting.
- `script/check-docs.sh`: deterministic presence checks for required product guidance.
- `script/verify-release-artifacts.sh`: checksum and tag-version checks before publication.
- `aqua.yaml`: pinned actionlint alongside GoReleaser from plan 3.
- `.github/workflows/test.yml`: reusable PR/main quality gate.
- `.github/workflows/release.yml`: tag workflow with quality dependency and pre-publish artifact verification.

### Task 1: Actionable root and subcommand help

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`
- Modify: `cmd/exec/exec.go`
- Modify: `cmd/exec/exec_test.go`
- Modify: `cmd/portforward/portforward.go`
- Modify: `cmd/portforward/portforward_test.go`
- Modify: `cmd/remoteportforward/remoteportforward.go`
- Modify: `cmd/remoteportforward/remoteportforward_test.go`
- Modify: `cmd/update/update.go`
- Modify: `cmd/update/update_test.go`

- [ ] **Step 1: Write failing help-contract tests**

Use fresh command factories, set output to a buffer, call `Help`, and assert concrete guidance:

```go
func assertHelpContains(t *testing.T, command *cobra.Command, values ...string) {
	t.Helper()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Help(); err != nil { t.Fatal(err) }
	for _, value := range values {
		if !strings.Contains(output.String(), value) {
			t.Errorf("help does not contain %q:\n%s", value, output.String())
		}
	}
}
```

Tests require:

- root: AWS SDK default configuration, `AWS_PROFILE`, `AWS_REGION`, `aws-vault`, and Session Manager Plugin;
- exec: `--wait 0` means one lookup, positive wait polls readiness, and explicit flags override input JSON;
- portforward: `tnnl portforward make-input-file` and automatic local-port wording;
- remoteportforward: `tnnl remoteportforward make-input-file`, remote host, and local-port wording;
- update: SHA-256 checksum, candidate version, and write permission.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./cmd ./cmd/exec ./cmd/portforward ./cmd/remoteportforward ./cmd/update -run Help -count=1`

Expected: assertion failures because current root text is empty and current port commands name the exec generator.

- [ ] **Step 3: Write concise help that matches implemented behavior**

Root `Long` must state:

```text
tnnl selects a ready ECS task and container for exec or port forwarding.
AWS credentials and Region come from the AWS SDK default chain; set
AWS_PROFILE/AWS_REGION or run through tools such as `aws-vault exec NAME -- tnnl ...`.
session-manager-plugin must be installed and available on PATH.
```

Each flag description names its exact precedence and semantics. Do not add `--profile`, `--region`, root `--cluster`, or root `--service`. Update help says verification happens before replacement, not that it provides a signature or rollback.

- [ ] **Step 4: Run help tests and confirm GREEN**

Run: `go test ./cmd ./cmd/exec ./cmd/portforward ./cmd/remoteportforward ./cmd/update -run Help -count=1`

Expected: PASS.

- [ ] **Step 5: Commit CLI guidance**

```bash
git add cmd
git commit -m "docs(cli): make setup and command behavior discoverable"
```

### Task 2: Quickstart, IAM/readiness, and troubleshooting

**Files:**
- Rewrite: `README.md`
- Create: `script/check-docs.sh`

- [ ] **Step 1: Add a failing documentation contract script**

Create an executable script with exact required tokens:

```bash
#!/usr/bin/env bash
set -euo pipefail

readme="${1:-README.md}"
required=(
  "## Quickstart"
  "session-manager-plugin --version"
  "aws-vault exec"
  "AWS_PROFILE"
  "AWS_REGION"
  "explicit CLI flag > input file > default"
  "ecs:ListClusters"
  "ssm:TerminateSession"
  "ExecuteCommandAgent"
  "runtime ID"
  "tnnl exec --wait"
  "checksum mismatch"
  "candidate version mismatch"
)
for value in "${required[@]}"; do
  grep -Fq "$value" "$readme" || {
    echo "$readme is missing required guidance: $value" >&2
    exit 1
  }
done
```

Run: `bash script/check-docs.sh`

Expected: FAIL on the first absent Quickstart token.

- [ ] **Step 2: Rewrite README in the user's execution order**

Use these sections in order:

1. `Quickstart` with install, plugin check, AWS context, `tnnl version`, and first exec;
2. installation via Go 1.25+ and release archives, including PATH diagnosis;
3. AWS context model with `AWS_PROFILE`, `AWS_REGION`, and `aws-vault exec dev -- env AWS_REGION=ap-northeast-1 tnnl exec`;
4. caller IAM versus ECS task-role IAM;
5. target readiness and selection labels;
6. exec, portforward, remoteportforward, and update examples;
7. strict input JSON and precedence;
8. exact wait behavior;
9. update verification/write-permission behavior;
10. a symptom/cause/action troubleshooting table.

State explicitly that AWS context is caller-owned and tnnl intentionally has no profile/region flags.

- [ ] **Step 3: Document exact permissions without claiming one universal policy**

The caller action inventory contains:

```text
ecs:ListClusters
ecs:ListTasks
ecs:DescribeTasks
ecs:ExecuteCommand       # exec only
ssm:StartSession         # port commands
ssm:TerminateSession     # failure cleanup; denial is joined to the original error
```

The ECS task role inventory contains `ssmmessages:CreateControlChannel`, `CreateDataChannel`, `OpenControlChannel`, and `OpenDataChannel`. For customer-managed ECS Exec KMS encryption, document caller `kms:GenerateDataKey` and task role `kms:Decrypt`. Link to the official AWS ECS Exec and Session Manager prerequisites instead of presenting a copy-paste wildcard production policy.

- [ ] **Step 4: Document exact readiness, input, wait, and update behavior**

Readiness for all three session commands is: task ECS Exec enabled and RUNNING; container name/runtime ID present and RUNNING; `ExecuteCommandAgent` RUNNING. A task label contains service/group plus short task ID, while the hidden value is the full unique ARN.

State `explicit CLI flag > input file > default`, unknown keys/trailing documents are errors, ports are decimal 1–65535, and template files require `--force` to replace. State wait zero performs one lookup and positive wait polls after cluster selection until eligible/timeout/cancellation. State update checks SHA-256 and candidate version before an atomic replacement; it does not claim cryptographic signing.

- [ ] **Step 5: Add a concrete troubleshooting table**

Include credentials/region missing, plugin not found/version check failing, AccessDenied, no eligible tasks, agent/runtime not ready, wait timeout, local port conflict, checksum mismatch, candidate version mismatch, and executable-directory permission failure. Every row gives the command or setting to inspect next.

- [ ] **Step 6: Run docs check and inspect rendering**

Run:

```bash
chmod +x script/check-docs.sh
bash script/check-docs.sh
git diff --check -- README.md script/check-docs.sh
```

Expected: all commands exit 0. Read README once in rendered Markdown and confirm command blocks do not imply root cluster/service/profile/region flags.

- [ ] **Step 7: Commit product documentation**

```bash
git add README.md script/check-docs.sh
git commit -m "docs: add quickstart and session troubleshooting"
```

### Task 3: Pre-publication release artifact verifier

**Files:**
- Create: `script/verify-release-artifacts.sh`
- Create: `script/verify-release-artifacts_test.sh`
- Modify: `aqua.yaml`

- [ ] **Step 1: Write a failing shell fixture test**

The test creates a private directory, a tiny `tnnl` script that prints `1.2.3`, packages `tnnl_test_fixture.tar.gz`, writes `checksums.txt`, and asserts the verifier accepts tag `v1.2.3` with that asset name. Then replace the checksum and expect failure; restore it, change binary output to `9.9.9`, repackage/re-hash, and expect version failure. The fixture script is host-executable on both Darwin and Linux.

Run: `bash script/verify-release-artifacts_test.sh`

Expected: FAIL because the verifier does not exist.

- [ ] **Step 2: Implement the verifier**

```bash
#!/usr/bin/env bash
set -euo pipefail

tag="${1:?usage: verify-release-artifacts.sh vX.Y.Z DIST_DIR ASSET_NAME}"
dist="${2:?usage: verify-release-artifacts.sh vX.Y.Z DIST_DIR ASSET_NAME}"
asset="${3:?usage: verify-release-artifacts.sh vX.Y.Z DIST_DIR ASSET_NAME}"
expected="${tag#v}"
test -f "$dist/checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$dist" && sha256sum -c checksums.txt)
else
  (cd "$dist" && shasum -a 256 -c checksums.txt)
fi
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
tar -xzf "$dist/$asset" -C "$tmp"
actual="$("$tmp/tnnl" version)"
if [[ "$actual" != "$expected" ]]; then
  echo "release version mismatch: got $actual, want $expected" >&2
  exit 1
fi
```

The script only deletes its own `mktemp -d` path. Add `rhysd/actionlint@v1.7.12` to `aqua.yaml`; keep GoReleaser v2.17.0 from plan 3.

- [ ] **Step 3: Run shell tests and syntax checks**

Run:

```bash
chmod +x script/verify-release-artifacts.sh script/verify-release-artifacts_test.sh
bash -n script/verify-release-artifacts.sh
bash -n script/verify-release-artifacts_test.sh
bash script/verify-release-artifacts_test.sh
```

Expected: PASS for valid fixture and internally asserted failures for corrupt checksum/wrong version.

- [ ] **Step 4: Commit the verifier**

```bash
git add script/verify-release-artifacts.sh script/verify-release-artifacts_test.sh aqua.yaml
git commit -m "build: verify release artifacts before publication"
```

### Task 4: Reusable PR/main quality workflow

**Files:**
- Rewrite: `.github/workflows/test.yml`

- [ ] **Step 1: Confirm current workflow misses main/reusable/static/build gates**

Run:

```bash
rg -n 'push:|workflow_call:|gofmt|go vet|go build|actionlint|goreleaser check' .github/workflows/test.yml
```

Expected before implementation: only the existing pull-request race test is represented; the required gate set is incomplete.

- [ ] **Step 2: Implement the reusable quality workflow**

Use this structure, retaining the repository's already SHA-pinned checkout and aqua installer actions and their tag comments:

```yaml
name: Quality

on:
  pull_request:
  push:
    branches: [main]
  workflow_call:

permissions:
  contents: read

concurrency:
  group: quality-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  quality:
    runs-on: ubuntu-latest
    timeout-minutes: 15
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7.0.0
      - name: Install pinned tools
        uses: aquaproj/aqua-installer@96a9bc20066c5bf5e275b41019cfc165b25f4e2e # v4.0.5
        with:
          aqua_version: v2.60.1
          enable_aqua_install: true
          aqua_opts: ""
      - name: Download Go modules
        run: go mod download
      - name: Validate workflows and release config
        run: |
          actionlint
          goreleaser check
      - name: Check formatting
        run: test -z "$(gofmt -l .)"
      - name: Vet
        run: go vet ./...
      - name: Test with race detector
        run: go test -race ./...
      - name: Build development binary
        run: |
          go build -trimpath -o "$RUNNER_TEMP/tnnl" .
          test "$("$RUNNER_TEMP/tnnl" version)" = dev
      - name: Check product documentation
        run: bash script/check-docs.sh
```

Do not add caches until workflow timings demonstrate a need. No secret is used in the quality workflow.

- [ ] **Step 3: Validate locally**

Run:

```bash
actionlint
goreleaser check
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
go build -trimpath -o /tmp/tnnl .
test "$(/tmp/tnnl version)" = dev
bash script/check-docs.sh
```

Expected: all commands exit 0.

- [ ] **Step 4: Commit quality workflow**

```bash
git add .github/workflows/test.yml
git commit -m "ci: gate pull requests and main pushes"
```

### Task 5: Quality-gated release workflow

**Files:**
- Rewrite: `.github/workflows/release.yml`

- [ ] **Step 1: Confirm current release publishes without quality dependency**

Run: `rg -n 'needs: quality|skip=publish|verify-release-artifacts|timeout-minutes|concurrency' .github/workflows/release.yml`

Expected before implementation: no matches for the pre-publication dependency/gates.

- [ ] **Step 2: Implement a least-privilege two-job release**

```yaml
name: Release

on:
  push:
    tags: ["v[0-9]+.[0-9]+.[0-9]+"]

permissions: {}

concurrency:
  group: release-${{ github.ref }}
  cancel-in-progress: false

jobs:
  quality:
    permissions:
      contents: read
    uses: ./.github/workflows/test.yml

  release:
    needs: quality
    runs-on: ubuntu-latest
    timeout-minutes: 30
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0 # v7.0.0
        with:
          fetch-depth: 0
      - name: Install pinned tools
        uses: aquaproj/aqua-installer@96a9bc20066c5bf5e275b41019cfc165b25f4e2e # v4.0.5
        with:
          aqua_version: v2.60.1
          enable_aqua_install: true
          aqua_opts: ""
      - name: Build without publishing
        run: goreleaser release --clean --skip=publish
      - name: Verify checksums and version
        run: >-
          bash script/verify-release-artifacts.sh
          "$GITHUB_REF_NAME" dist tnnl_linux_amd64.tar.gz
      - name: Publish the verified files
        run: >-
          gh release create "$GITHUB_REF_NAME"
          dist/tnnl_*.tar.gz dist/checksums.txt
          --verify-tag --generate-notes --title "$GITHUB_REF_NAME"
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The verify step invokes `bash script/verify-release-artifacts.sh "$GITHUB_REF_NAME" dist tnnl_linux_amd64.tar.gz`. The publish step cannot run if reusable quality, the one artifact build, checksum verification, or binary version verification fails. It uploads the already verified `dist` files and never runs GoReleaser a second time. All external actions are fixed to 40-character SHAs with tag comments.

- [ ] **Step 3: Validate workflow and release dry run**

Run:

```bash
actionlint
goreleaser check
goreleaser release --snapshot --clean --skip=publish
test -f dist/checksums.txt
snapshot_dir="$(mktemp -d)"
snapshot_os="$(go env GOOS)"
snapshot_arch="$(go env GOARCH)"
snapshot_asset="tnnl_${snapshot_os}_${snapshot_arch}.tar.gz"
tar -xzf "dist/${snapshot_asset}" -C "$snapshot_dir"
snapshot_version="$("$snapshot_dir/tnnl" version)"
bash script/verify-release-artifacts.sh --allow-snapshot "v${snapshot_version}" dist "$snapshot_asset"
rm -rf "$snapshot_dir"
```

Expected: actionlint/config/dry build/checksum checks pass. The local snapshot check derives its expected value from the dry-run artifact; the tag workflow uses the independently supplied strict Git tag and is covered by the shell fixture test.

- [ ] **Step 4: Commit release workflow**

```bash
git add .github/workflows/release.yml
git commit -m "ci: verify artifacts before release publication"
```

### Task 6: Requirement audit and final product gate

**Files:**
- Inspect: all changed files
- Modify only if the audit finds an uncovered retained finding.

- [ ] **Step 1: Map each retained finding to evidence**

Create a temporary audit checklist outside Git or in the task notes with these 13 rows and point each to at least one test/build/docs check:

1. unique replica task selection;
2. released automatic port;
3. empty result safety;
4. full plugin arguments/KMS compatibility;
5. meaningful wait;
6. readiness/pointer/ARN safety;
7. preflight/context/failure cleanup;
8. input precedence/strict validation;
9. core-flow automated tests;
10. pagination/chunking;
11. checksum/version/atomic update;
12. consistent build versions;
13. onboarding/help/troubleshooting.

Explicitly confirm no profile/region/root target flags and no Terraform changes were introduced.

- [ ] **Step 2: Run the full local gate from a clean state**

Run:

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test -race -count=1 ./...
go build -trimpath -o /tmp/tnnl .
test "$(/tmp/tnnl version)" = dev
actionlint
goreleaser check
bash script/check-docs.sh
bash script/verify-release-artifacts_test.sh
git diff --check
git status --short
```

Expected: every validation exits 0; status contains only intentional implementation/documentation changes or is clean after commits.

- [ ] **Step 3: Search for the original failure patterns**

Run:

```bash
rg -n 'log\.Fatal|MakeStartSessionCmd|context\.Background\(\)|strings\.Split\(.*TaskArn|\.tnnl\.new' cmd internal pkg main.go
rg -n 'ListClusters\(.*context\.Background|ListTasks\(.*context\.Background|DescribeTasks\(.*context\.Background' internal
```

Expected: no production matches. `context.WithoutCancel` is allowed only for bounded TerminateSession cleanup, and the unique temp prefix `.tnnl.new-*` is allowed in `os.CreateTemp`/tests.

- [ ] **Step 4: Commit any audit-only corrections and record completion evidence**

If the audit required changes, stage only those exact files and commit with `fix: close product review audit gaps`. If it required none, do not create an empty commit. Record the final command outputs for the user handoff.

## Plan completion check

- A first-time user can install, set AWS context externally, verify the plugin, understand IAM/readiness, and complete each primary command from README/help.
- PR and main workflows enforce format, vet, race tests, build, version, release config, workflow syntax, and docs.
- Release publication has a successful reusable quality dependency, verifies dry-run checksums plus tag version, and uploads those exact artifacts without rebuilding them.
- Actions use 40-character SHAs, permissions are explicit/minimal, jobs are bounded, and release concurrency is not canceled halfway.
- The final audit maps all 13 retained findings and confirms both user-declared non-goals stayed unchanged.
