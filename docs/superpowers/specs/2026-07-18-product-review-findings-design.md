# Product Review Findings Implementation Design

**Date:** 2026-07-18
**Status:** Approved

## Purpose

Implement the thirteen findings retained after the product review so that `tnnl`
selects the requested ECS task correctly, handles unavailable targets safely,
starts and cleans up Session Manager sessions reliably, and ships reproducible
and verifiable binaries.

The implementation preserves the product's existing operating model: callers
select the AWS account and region before invoking `tnnl`, for example through
environment variables or `aws-vault`. `tnnl` will not add profile or region
flags. The Terraform fixture is not part of this work.

## Goals

1. Select an ECS task by its unique ARN rather than by its non-unique group.
2. Release automatically selected local ports before the plugin binds them.
3. Return actionable errors for empty or partially failed ECS results instead
   of panicking.
4. Invoke `session-manager-plugin` with the full AWS CLI-compatible argument
   shape so KMS-enabled sessions are supported.
5. Make `exec --wait N` wait up to `N` seconds for an eligible task before
   presenting task and container choices.
6. Validate task, container, Execute Command agent, runtime ID, and ARN state
   before starting a session.
7. Preflight the plugin before creating a remote session, propagate context,
   and terminate sessions when local handoff fails.
8. Apply configuration precedence as explicit CLI flag, then input file, then
   default; reject unknown JSON fields and invalid values.
9. Add automated tests for the core selection, command, input, session, port,
   update, and version flows.
10. Retrieve every page from ListClusters and ListTasks.
11. Verify update checksums and versions and replace the executable atomically.
12. Report consistent versions for Go installs and GoReleaser artifacts, with
    an honest `dev` value for unversioned local builds.
13. Document setup, IAM requirements, plugin diagnostics, input precedence,
    waiting behavior, and common errors.

## Non-goals

- Adding `--profile` or `--region`; AWS configuration remains caller-owned.
- Adding root-level cluster or service flags.
- Changing the Terraform fixture.
- Persisting AWS data, recent targets, or user configuration.
- Redesigning the terminal UI beyond what is needed for correct unique choices
  and useful empty-state errors.
- Supporting Windows releases; the current GoReleaser targets remain Darwin
  and Linux.

## Architecture

The current package layout remains recognizable. Small interfaces and typed
values are introduced at external boundaries instead of creating a framework
or rewriting the CLI.

### Command and context boundary

`main` creates a signal-aware context and passes it to the root Cobra command.
Subcommands use `RunE`, return errors rather than calling `log.Fatal`, and pass
`cmd.Context()` through handlers, AWS calls, wait loops, plugin processes, and
session cleanup.

This provides one cancellation path and makes exit behavior testable. User
cancellation remains distinct from an operational error and does not produce a
panic or stack trace.

### Input resolution

Each subcommand resolves its input in this order:

1. Start with command defaults.
2. Decode an optional input file using a strict JSON decoder that rejects
   unknown fields and trailing JSON documents.
3. Apply only flags explicitly supplied by the caller, using
   `Flags().Changed`.
4. Validate the fully resolved value before loading AWS configuration.

Ports must be decimal values from 1 through 65535. `wait` must be non-negative.
Required hosts and port values are trimmed and must not be empty. Input helper
functions return errors; command handlers decide how Cobra presents them.

Input-file generation refuses to replace an existing file unless the command
is explicitly given a force option. Help text names the correct parent command.

### ECS API boundary

The list and describe operations depend on a narrow interface containing only
the ECS calls used by the resolver. Production uses `*ecs.Client`; tests use a
deterministic fake.

ListClusters and ListTasks loop until `NextToken` is empty. Task ARNs are
deduplicated while retaining AWS response order. DescribeTasks requests are
chunked to the service limit. Any `DescribeTasksOutput.Failures` values are
converted into errors containing the failed ARN and AWS reason.

AWS API errors remain wrapped with the operation name and relevant cluster or
service, while preserving the original error for `errors.Is` and `errors.As`.

### Eligible target definition

A task is eligible when all of the following are true:

- `EnableExecuteCommand` is true.
- `LastStatus` is `RUNNING`.
- It has at least one container eligible for the requested operation.

An eligible container has a non-empty name, `LastStatus == RUNNING`, a managed
agent named `ExecuteCommandAgent` whose `LastStatus` is `RUNNING`, and a
non-empty runtime ID. Runtime ID is required for all three commands because the
full Session Manager plugin request identifies the target as
`ecs:<cluster>_<task>_<runtime>`; the AWS CLI also re-describes the selected ECS
task to obtain this value for `execute-command`.

Pointer fields are validated before dereference. Task and cluster identifiers
are parsed through helpers that accept both long and short ECS ARN forms and
return descriptive errors for malformed identifiers.

### Wait semantics

`tnnl exec --wait N` means: after the cluster is selected, poll ListTasks and
DescribeTasks until at least one exec-eligible task exists or `N` seconds have
elapsed.

- `N == 0` performs one lookup and returns immediately if no eligible task is
  available.
- `N > 0` polls on a bounded interval until success, timeout, or context
  cancellation.
- A service from the input file restricts every poll to that service.
- Without a service, the poll waits for any eligible task in the selected
  cluster.
- The final choices are built from the successful DescribeTasks response, so
  task and container data are not stale relative to the wait result.

The timeout error names the cluster, optional service, wait duration, and the
eligibility condition.

### Typed terminal choices

The list UI accepts options with separate display labels and internal values.
For tasks, the value is the full Task ARN and the label contains the group and
short Task ID. Selecting a row returns the ARN, which is used to retrieve the
exact task. The implementation never resolves a task from `Task.Group`.

Clusters and containers use the same option type where practical. Rendering an
empty option list returns a typed no-items error before Bubble Tea starts.
Callers add resource-specific context to that error.

### Automatic local ports

Automatic selection opens `127.0.0.1:0`, reads the assigned port, closes the
listener, and only then returns the number. Close errors are returned.

Closing before plugin handoff necessarily leaves a small time-of-check to
time-of-use window. If the plugin reports a bind conflict for an automatically
selected port, the command may retry selection and session creation once. An
explicitly requested local port is never silently changed.

### Session Manager plugin invocation

Plugin preflight happens before ExecuteCommand or StartSession. It resolves the
executable with `exec.LookPath`, runs `--version` with a short timeout, and
returns an installation-oriented error if either step fails.

The plugin command receives the six arguments used after the executable by the
AWS CLI contract:

1. JSON session response
2. AWS region
3. `StartSession`
4. profile name, possibly empty
5. JSON request parameters containing `Target`
6. SSM endpoint override, possibly empty

The profile is read from the standard AWS environment only for forwarding to
the plugin; no new CLI setting is introduced. An endpoint override is forwarded
from the standard AWS endpoint environment variables when present. Empty
values allow the plugin and SDK to use their normal defaults.

After ExecuteCommand creates an ECS Exec session, tnnl follows the AWS CLI
sequence and re-describes the task identified by the ExecuteCommand response.
It finds the returned container name and builds the plugin Target from that
latest runtime ID. A DescribeTasks error, partial failure, missing container, or
missing runtime ID is treated as a post-creation handoff failure and triggers
the same remote-session cleanup described below.

### Session lifecycle

ExecuteCommand and StartSession return a typed local session containing the
session ID, plugin request, and a cleanup function. The handler then runs the
plugin.

If response validation, the post-ExecuteCommand runtime refresh, plugin
creation, or plugin execution fails after the remote session was created, the
handler performs a best-effort SSM TerminateSession call with a short,
independent cleanup timeout. The original error remains primary; a cleanup
error is joined with it. Cancellation follows the same cleanup path.

The AWS API call uses the caller context. Plugin execution uses that context as
well, so process termination and remote cleanup follow a single lifecycle.

### Self-update

The updater downloads the archive and the release checksum manifest into its
private temporary directory. It parses the checksum for the exact asset name
and verifies the archive using SHA-256 before extraction.

After extraction it runs the candidate binary's `version` command with a
timeout and requires it to match the release tag. Only a verified candidate may
be installed.

Installation creates a uniquely named temporary file in the executable's own
directory, writes and syncs it, applies executable permissions, closes it, and
renames it over the current executable. Failures remove only the temporary file
created by this update. No fixed `.tnnl.new` filename is used.

### Version resolution

The runtime version is resolved in this order:

1. A non-empty, non-`dev` linker value supplied by GoReleaser.
2. `debug.ReadBuildInfo().Main.Version` when it is a real module version, as in
   `go install github.com/wim-web/tnnl@vX.Y.Z`.
3. `dev` for local unversioned builds.

The stale embedded `.version` file is no longer a runtime version source.
GoReleaser declares its version linker flag explicitly, and CI verifies the
release build path and the module-version resolution helper.

## Error handling and user-visible behavior

- No command path uses `log.Fatal`; errors return through Cobra with one
  consistent exit path.
- Empty clusters, tasks, and containers name the resource and required state.
- Invalid input is reported before AWS calls or terminal UI startup.
- Plugin-not-found errors include the prerequisite command from the README.
- AWS errors include the failed operation and target context without hiding
  the underlying SDK error.
- Waiting prints bounded progress without emitting an unending line of dots.
- Cancellation terminates the UI or plugin, attempts remote cleanup, and exits
  without a panic.

## Testing strategy

Every production behavior is introduced through a failing test before its
implementation.

### Unit tests

- Port selection proves the returned port can immediately be rebound.
- Input resolution covers defaults, file values, explicit flag overrides,
  unknown keys, trailing JSON, invalid ports, negative waits, and required
  fields.
- Identifier parsing covers short and long task ARNs and malformed values.
- Eligibility covers pending tasks, disabled Execute Command, stopped
  containers, missing or stopped agents, and missing runtime IDs for every
  session command.
- Typed list options prove the second duplicate-group task resolves to its own
  ARN and an empty list never enters Bubble Tea.
- Wait tests use a fake clock or injected polling function and cover success,
  timeout, API failure, and cancellation.
- Pagination tests use multiple ListClusters and ListTasks pages and chunked
  DescribeTasks results.
- Plugin tests assert preflight ordering and all six arguments, including empty
  profile and endpoint values.
- Session lifecycle tests assert TerminateSession on plugin failure and no
  remote API call when preflight fails.
- ECS Exec lifecycle tests assert that ExecuteCommand is followed by
  DescribeTasks, the refreshed runtime ID is used in Target, and refresh
  failures terminate the newly created session.
- Updater tests cover checksum match, checksum mismatch, candidate version
  mismatch, unique temporary files, and successful atomic replacement.
- Version tests cover linker, module, and development fallbacks.

### Command-level tests

Commands run against fake ECS and SSM clients and a fake plugin runner. They
cover exec, portforward, and remoteportforward happy paths plus the known
regressions from the review. No real AWS credentials are required.

### CI gates

CI runs formatting verification, `go vet`, `go test -race ./...`, and a normal
build on pull requests and pushes to the default branch. Release jobs run the
same test gate before GoReleaser and verify the produced binary reports the tag
version.

## Documentation

The README will include:

- Installation through Go and release artifacts.
- The supported AWS context model, including environment variables and an
  `aws-vault` example, without adding profile or region flags.
- Session Manager Plugin installation and `--version` diagnosis.
- Required ECS, SSM, and session-termination IAM actions.
- ECS task and ExecuteCommandAgent readiness requirements.
- Input precedence and strict-schema behavior.
- Correct `--wait` semantics.
- Examples and common errors for all three session commands.
- Safe update behavior and write-permission requirements.

## Completion criteria

The work is complete only when all thirteen retained findings have an automated
regression test or an explicit build/documentation verification, the complete
race-enabled test suite and vet pass, all binaries build, the release-version
check passes, and a requirement-by-requirement audit finds no retained finding
unimplemented.
