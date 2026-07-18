# Session Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preflight Session Manager locally, invoke it with the AWS CLI-compatible full argument contract, propagate cancellation through AWS/plugin work, and clean up remotely created sessions after local handoff failures.

**Architecture:** `internal/session_manager` owns executable discovery, the six plugin arguments, and process I/O. `pkg/command` owns typed remote-session creation, refreshes the ECS runtime ID after ExecuteCommand, and performs failure-only termination, while `internal/handler` injects AWS/client/view/plugin dependencies so ordering and cancellation are testable without credentials.

**Tech Stack:** Go 1.25+, AWS SDK for Go v2 ECS/SSM, Cobra contexts, `os/exec`, `context.WithoutCancel`, `errors.Join`, and Go `testing` fakes.

---

This is implementation plan 2 of 4. It assumes `docs/superpowers/plans/2026-07-18-target-resolution-and-input.md` is complete and `internal/view` returns `target.Resolved` with `SSMTarget()`.

## File map

- `internal/session_manager/session.go`: plugin discovery, version preflight, invocation JSON, environment forwarding, process execution.
- `internal/session_manager/session_test.go`: six-argument and preflight contract tests.
- `pkg/command/session.go`: `RemoteSession` lifecycle and failure-only cleanup.
- `pkg/command/exec.go`: context-aware ECS ExecuteCommand session creation.
- `pkg/command/portforward.go`: context-aware SSM StartSession creation.
- `pkg/command/*_test.go`: SDK input, context, response validation, and cleanup tests.
- `internal/handler/dependencies.go`: narrow production factories and test injection.
- `internal/handler/exec.go`, `internal/handler/portforward.go`: preflight-first orchestration.
- `internal/handler/*_test.go`: command-level ordering and lifecycle regression tests.
- `cmd/root.go`, `main.go`: one signal-aware Cobra execution/error path.

### Task 1: Full Session Manager plugin invocation and bounded preflight

**Files:**
- Rewrite: `internal/session_manager/session.go`
- Create: `internal/session_manager/session_test.go`

- [ ] **Step 1: Write failing six-argument tests**

```go
func TestInvocationArgumentsMatchAWSCLIContract(t *testing.T) {
	invocation := Invocation{
		Response: SessionResponse{SessionID: "s-1", StreamURL: "wss://example", TokenValue: "token"},
		Region:   "ap-northeast-1",
		Target:   "ecs:cluster_task_runtime",
	}
	got, err := invocation.arguments("team-profile", "https://ssm.example")
	if err != nil { t.Fatal(err) }
	if len(got) != 6 { t.Fatalf("len(arguments) = %d, want 6: %#v", len(got), got) }
	if got[1] != "ap-northeast-1" || got[2] != "StartSession" || got[3] != "team-profile" || got[5] != "https://ssm.example" {
		t.Fatalf("arguments = %#v", got)
	}
	var request struct{ Target string `json:"Target"` }
	if err := json.Unmarshal([]byte(got[4]), &request); err != nil { t.Fatal(err) }
	if request.Target != invocation.Target { t.Fatalf("request target = %q", request.Target) }
	var response SessionResponse
	if err := json.Unmarshal([]byte(got[0]), &response); err != nil { t.Fatal(err) }
	if response != invocation.Response { t.Fatalf("response = %#v", response) }
}

func TestInvocationArgumentsAllowEmptyProfileAndEndpoint(t *testing.T) {
	invocation := validInvocation()
	got, err := invocation.arguments("", "")
	if err != nil { t.Fatal(err) }
	if got[3] != "" || got[5] != "" { t.Fatalf("arguments = %#v", got) }
}
```

Add table rows rejecting an empty session ID, stream URL, token, region, and target before starting the process.

- [ ] **Step 2: Write failing preflight tests**

Inject OS boundaries instead of mutating package globals:

```go
type processFactory func(context.Context, string, ...string) command

type command interface {
	CombinedOutput() ([]byte, error)
}

type dependencies struct {
	lookPath       func(string) (string, error)
	commandContext processFactory
	preflightLimit time.Duration
}
```

Test not-found, non-zero `--version`, empty version output, a command blocked until its context deadline, parent-context cancellation, and success. Assert discovery uses `session-manager-plugin`, version args are exactly `--version`, and the returned runner stores `AWS_PROFILE` before `AWS_DEFAULT_PROFILE`, plus `AWS_ENDPOINT_URL_SSM` before `AWS_ENDPOINT_URL`. For the deadline and cancellation rows, assert `errors.Is(err, context.DeadlineExceeded)` and `errors.Is(err, context.Canceled)` respectively; a bare `signal: killed` process error is not sufficient.

- [ ] **Step 3: Run tests and confirm RED**

Run: `go test ./internal/session_manager -count=1`

Expected: build failure because `Invocation`, `SessionResponse`, `Runner`, and the dependency boundary do not exist.

- [ ] **Step 4: Implement the invocation value and argument builder**

```go
const CommandName = "session-manager-plugin"

type SessionResponse struct {
	SessionID  string `json:"SessionId"`
	StreamURL  string `json:"StreamUrl"`
	TokenValue string `json:"TokenValue"`
}

type Invocation struct {
	Response SessionResponse
	Region   string
	Target   string
}

func (i Invocation) arguments(profile, endpoint string) ([]string, error) {
	if strings.TrimSpace(i.Response.SessionID) == "" || strings.TrimSpace(i.Response.StreamURL) == "" || strings.TrimSpace(i.Response.TokenValue) == "" {
		return nil, errors.New("session response is missing id, stream URL, or token")
	}
	if strings.TrimSpace(i.Region) == "" { return nil, errors.New("AWS region is required for session-manager-plugin") }
	if strings.TrimSpace(i.Target) == "" { return nil, errors.New("session target is required for session-manager-plugin") }
	response, err := json.Marshal(i.Response)
	if err != nil { return nil, fmt.Errorf("encode plugin session response: %w", err) }
	request, err := json.Marshal(struct { Target string `json:"Target"` }{Target: i.Target})
	if err != nil { return nil, fmt.Errorf("encode plugin request: %w", err) }
	return []string{string(response), i.Region, "StartSession", profile, string(request), endpoint}, nil
}
```

- [ ] **Step 5: Implement preflight and the concrete runner**

```go
type Plugin interface {
	Run(context.Context, Invocation) error
}

type Runner struct {
	path     string
	profile  string
	endpoint string
}

func Preflight(ctx context.Context) (Plugin, error) {
	return preflight(ctx, dependencies{
		lookPath: exec.LookPath,
		commandContext: func(ctx context.Context, name string, args ...string) command {
			return exec.CommandContext(ctx, name, args...)
		},
		preflightLimit: 3 * time.Second,
	})
}

func preflight(ctx context.Context, deps dependencies) (*Runner, error) {
	path, err := deps.lookPath(CommandName)
	if err != nil { return nil, fmt.Errorf("%s is required; install it and verify `%s --version`: %w", CommandName, CommandName, err) }
	checkCtx, cancel := context.WithTimeout(ctx, deps.preflightLimit)
	defer cancel()
	output, err := deps.commandContext(checkCtx, path, "--version").CombinedOutput()
	if err != nil {
		processErr := fmt.Errorf("%s --version failed: %s: %w", CommandName, strings.TrimSpace(string(output)), err)
		if contextErr := checkCtx.Err(); contextErr != nil {
			return nil, errors.Join(processErr, contextErr)
		}
		return nil, processErr
	}
	if strings.TrimSpace(string(output)) == "" { return nil, fmt.Errorf("%s --version returned empty output", CommandName) }
	return &Runner{path: path, profile: firstEnvironment("AWS_PROFILE", "AWS_DEFAULT_PROFILE"), endpoint: firstEnvironment("AWS_ENDPOINT_URL_SSM", "AWS_ENDPOINT_URL")}, nil
}
```

`Runner.Run` calls `invocation.arguments`, creates `exec.CommandContext(ctx, r.path, args...)`, attaches `os.Stdin`, `os.Stdout`, and `os.Stderr`, and wraps `Run` errors as `run session-manager-plugin: %w`. If `ctx.Err()` is non-nil, join it so `errors.Is(err, context.Canceled)` or `errors.Is(err, context.DeadlineExceeded)` succeeds. The new runner has no legacy three-argument path.

Keep the existing exported `MakeStartSessionCmd` unchanged as a temporary compile-compatibility wrapper because the old `pkg/command` callers still reference it at this commit boundary. Mark its removal explicitly in Task 4, where every handler has migrated to `Runner`/`RemoteSession`; do not add new callers to it.

- [ ] **Step 6: Run tests and confirm GREEN**

Run: `go test ./internal/session_manager -count=1`

Expected: PASS; KMS-capable full invocation has six arguments after the executable and preflight is bounded.

- [ ] **Step 7: Commit plugin contract**

```bash
git add internal/session_manager
git commit -m "fix: use full Session Manager plugin contract"
```

### Task 2: Failure-only remote session cleanup

**Files:**
- Create: `pkg/command/session.go`
- Create: `pkg/command/session_test.go`

- [ ] **Step 1: Write failing lifecycle tests**

```go
func TestRemoteSessionTerminatesWhenPluginFails(t *testing.T) {
	pluginErr := errors.New("plugin failed")
	var terminatedID string
	session := RemoteSession{
		ID: "s-1",
		Invocation: validInvocation(),
		terminate: func(ctx context.Context, id string) error {
			if ctx.Err() != nil { t.Fatalf("cleanup context already canceled: %v", ctx.Err()) }
			terminatedID = id
			return nil
		},
		cleanupTimeout: time.Second,
	}
	err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error { return pluginErr }))
	if !errors.Is(err, pluginErr) || terminatedID != "s-1" { t.Fatalf("Run() = %v, terminated=%q", err, terminatedID) }
}

func TestRemoteSessionDoesNotTerminateAfterSuccessfulPlugin(t *testing.T) {
	terminated := false
	session := validRemoteSession(func(context.Context, string) error { terminated = true; return nil })
	if err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error { return nil })); err != nil { t.Fatal(err) }
	if terminated { t.Fatal("successful plugin run terminated the remote session") }
}

func TestRemoteSessionJoinsCleanupError(t *testing.T) {
	pluginErr := errors.New("plugin failed")
	cleanupErr := errors.New("terminate failed")
	session := validRemoteSession(func(context.Context, string) error { return cleanupErr })
	err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error { return pluginErr }))
	if !errors.Is(err, pluginErr) || !errors.Is(err, cleanupErr) { t.Fatalf("Run() = %v", err) }
}
```

Add a canceled-parent test; cleanup must receive a live, independently timed context.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./pkg/command -run TestRemoteSession -count=1`

Expected: build failure because `RemoteSession` does not exist.

- [ ] **Step 3: Implement the lifecycle**

```go
type terminateFunc func(context.Context, string) error

type RemoteSession struct {
	ID             string
	Invocation     session_manager.Invocation
	terminate      terminateFunc
	cleanupTimeout time.Duration
}

func cleanupCreatedSession(ctx context.Context, sessionID string, timeout time.Duration, terminate terminateFunc, primary error) error {
	if sessionID == "" { return primary }
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	if cleanupErr := terminate(cleanupCtx, sessionID); cleanupErr != nil {
		return errors.Join(primary, fmt.Errorf("terminate remote session %s: %w", sessionID, cleanupErr))
	}
	return primary
}

func (s RemoteSession) Run(ctx context.Context, plugin session_manager.Plugin) error {
	if err := plugin.Run(ctx, s.Invocation); err != nil {
		pluginErr := fmt.Errorf("session-manager-plugin handoff failed: %w", err)
		return cleanupCreatedSession(ctx, s.ID, s.cleanupTimeout, s.terminate, pluginErr)
	}
	return nil
}
```

Use `cleanupCreatedSession` from both AWS constructors when an API returned a session ID but later response validation or runtime refresh fails. Normal plugin exit does not call TerminateSession. This avoids adding an unconditional IAM requirement and avoids turning an already-closed successful session into an `InvalidSessionId` error.

- [ ] **Step 4: Run tests and confirm GREEN**

Run: `go test ./pkg/command -run TestRemoteSession -count=1`

Expected: PASS, including independent cleanup after parent cancellation.

- [ ] **Step 5: Commit lifecycle**

```bash
git add pkg/command/session.go pkg/command/session_test.go
git commit -m "fix: clean up failed remote session handoffs"
```

### Task 3: Context-aware ECS/SSM session creation

**Files:**
- Rewrite: `pkg/command/exec.go`
- Rewrite: `pkg/command/portforward.go`
- Create: `pkg/command/exec_test.go`
- Create: `pkg/command/portforward_test.go`

- [ ] **Step 1: Define narrow SDK interfaces in tests and production**

```go
type ExecSessionAPI interface {
	ExecuteCommand(context.Context, *ecs.ExecuteCommandInput, ...func(*ecs.Options)) (*ecs.ExecuteCommandOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput, ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}

type SessionAPI interface {
	StartSession(context.Context, *ssm.StartSessionInput, ...func(*ssm.Options)) (*ssm.StartSessionOutput, error)
	TerminateSession(context.Context, *ssm.TerminateSessionInput, ...func(*ssm.Options)) (*ssm.TerminateSessionOutput, error)
}

type ExecTarget struct {
	Cluster       string
	TaskARN       string
	ContainerName string
}

type PortTarget struct {
	SSMTarget string
}
```

- [ ] **Step 2: Write failing ExecuteCommand tests**

The fake captures context and input and returns an ECS `types.Session`. Assert `Cluster`, exact Task ARN, container, command, `Interactive=true`, and region. After ExecuteCommand, assert a DescribeTasks call uses the returned cluster/task identifiers, finds the returned container name, and builds the plugin target from the refreshed runtime ID. Assert API errors wrap the operation. Table-test nil/missing `Session`, ID, stream URL, token, cluster ARN, task ARN, container name, DescribeTasks `Failures`, missing task/container, and missing runtime ID; whenever a session ID exists, later validation/refresh failure attempts best-effort termination and joins a cleanup error.

- [ ] **Step 3: Write failing StartSession tests**

Assert the exact `ecs:<cluster>_<task>_<runtime>` target, document name, parameters, context marker, and conversion of SSM output into `SessionResponse`. Test StartSession errors and TerminateSession input after plugin failure. Lock down cleanup ownership with these validation cases:

- `TestStartPortForwardSessionTerminatesWhenStartedResponseIsInvalid`: when StartSession returned a non-empty Session ID but StreamUrl or TokenValue is missing, terminate that created ID before returning the validation error.
- `TestStartPortForwardSessionJoinsValidationAndTerminateErrors`: preserve both the validation error and a cleanup sentinel through `errors.Is`.
- A missing Session ID cannot be terminated; assert no TerminateSession call in that row.

Apply the same post-creation cleanup rule to the ExecuteCommand validation/refresh table from Step 2.

- [ ] **Step 4: Run tests and confirm RED**

Run: `go test ./pkg/command -run 'Test(StartExecSession|StartPortForwardSession)' -count=1`

Expected: build failure because both current functions accept concrete clients and return `*exec.Cmd`.

- [ ] **Step 5: Implement typed session constructors**

`StartExecSession` calls `ExecuteCommand(ctx, ...)`, captures the session ID, validates returned cluster/task/container fields, then calls `DescribeTasks(ctx, ...)` and finds that exact container's current runtime ID. It derives `ClusterName` and `TaskID` with `internal/target` helpers and only then returns:

```go
return RemoteSession{
	ID: sessionID,
	Invocation: session_manager.Invocation{
		Response: session_manager.SessionResponse{SessionID: sessionID, StreamURL: streamURL, TokenValue: token},
		Region: region,
		Target: fmt.Sprintf("ecs:%s_%s_%s", clusterName, taskID, runtimeID),
	},
	terminate: func(cleanupCtx context.Context, id string) error {
		_, err := ssmClient.TerminateSession(cleanupCtx, &ssm.TerminateSessionInput{SessionId: aws.String(id)})
		return err
	},
	cleanupTimeout: 5 * time.Second,
}, nil
```

`StartPortForwardSession` uses the same conversion/lifecycle with `StartSession(ctx, ...)`. Its `DocumentName` is still the typed `DocumentName` constant. Both constructors validate target/region/response fields and wrap errors with operation and target context. If validation or the exec refresh fails after a non-empty session ID is returned, call `cleanupCreatedSession` before returning.

At this commit boundary, retain the old exported `ExecCommand` and `PortForwardCommand` functions beside the new constructors so existing handlers compile. They remain unchanged and are deleted in Task 4 immediately after handler migration. This task's new tests call only the typed constructors.

- [ ] **Step 6: Run command tests and confirm GREEN**

Run: `go test ./pkg/command -count=1`

Expected: PASS; every SDK call sees caller context and every returned session has a cleanup function.

- [ ] **Step 7: Commit typed AWS sessions**

```bash
git add pkg/command
git commit -m "refactor: model remote sessions explicitly"
```

### Task 4: Preflight-first handlers with injectable dependencies

**Files:**
- Create: `internal/handler/dependencies.go`
- Rewrite: `internal/handler/exec.go`
- Rewrite: `internal/handler/portforward.go`
- Create: `internal/handler/exec_test.go`
- Create: `internal/handler/portforward_test.go`
- Delete after handler migration: `internal/view/ecs.go`
- Delete after handler migration: `internal/listview/cluster.go`
- Delete after handler migration: `internal/listview/task.go`
- Delete after handler migration: `internal/listview/container.go`
- Remove after handler migration: `internal/listview/simple.go` `RenderList` compatibility wrapper
- Delete legacy function: `internal/session_manager/session.go` `MakeStartSessionCmd`
- Delete legacy functions: `pkg/command/exec.go` `ExecCommand`, `pkg/command/portforward.go` `PortForwardCommand`

- [ ] **Step 1: Write failing handler ordering tests**

Create fake composite ECS and SSM clients implementing the target resolver plus command interfaces. Record an ordered event slice. Cover:

1. plugin preflight failure records only `preflight` and makes no config, ECS, ExecuteCommand, StartSession, or TerminateSession call;
2. success records `preflight`, `load-config`, target List/Describe calls, remote session creation, then `plugin-run`;
3. plugin failure records `terminate-session` after `plugin-run`;
4. explicit context cancellation reaches config, List/Describe, Execute/Start, plugin, and cleanup;
5. view cancellation makes no remote API call;
6. AWS config/API errors preserve their sentinel through `errors.Is`;
7. two tasks with the same service/group label remain distinct: make the chooser return the second option value and assert ExecuteCommand/StartSession receives that task's full ARN-derived target, never the first replica.

- [ ] **Step 2: Write failing port auto-allocation tests**

Inject `availablePort func() (int, error)`. When `LocalPortNumber` is empty, assert it is called once and StartSession parameters contain the returned decimal port. When explicit, assert it is never called. When allocation fails, assert StartSession is not called. The listener-close behavior itself remains covered by `pkg/port` tests from plan 1.

- [ ] **Step 3: Run handler tests and confirm RED**

Run: `go test ./internal/handler -count=1`

Expected: build failure because handlers instantiate concrete clients and preflight is not injected.

- [ ] **Step 4: Add one dependency struct and production constructor**

```go
type dependencies struct {
	loadConfig    func(context.Context) (aws.Config, error)
	newECS        func(aws.Config) ecsAPI
	newSSM        func(aws.Config) ssmAPI
	preflight     func(context.Context) (session_manager.Plugin, error)
	choose        view.Choose
	availablePort func() (int, error)
}

func productionDependencies() dependencies {
	return dependencies{
		loadConfig: func(ctx context.Context) (aws.Config, error) { return config.LoadDefaultConfig(ctx) },
		newECS: func(cfg aws.Config) ecsAPI { return ecs.NewFromConfig(cfg) },
		newSSM: func(cfg aws.Config) ssmAPI { return ssm.NewFromConfig(cfg) },
		preflight: session_manager.Preflight,
		choose: listview.RenderOptions,
		availablePort: port.AvailablePort,
	}
}
```

`ecsAPI` embeds `target.ECSAPI` and `command.ExecSessionAPI`; `ssmAPI` embeds `command.SessionAPI`.

- [ ] **Step 5: Implement preflight-first exec flow**

```go
func ExecHandler(ctx context.Context, in input.ExecInput) error {
	return execHandler(ctx, in, productionDependencies())
}

func execHandler(ctx context.Context, in input.ExecInput, deps dependencies) error {
	plugin, err := deps.preflight(ctx)
	if err != nil { return err }
	cfg, err := deps.loadConfig(ctx)
	if err != nil { return fmt.Errorf("load AWS configuration: %w", err) }
	ecsClient := deps.newECS(cfg)
	resolved, quit, err := view.ResolveTarget(ctx, target.NewResolver(ecsClient), deps.choose, in.Cluster, in.Service, time.Duration(in.Wait)*time.Second)
	if err != nil { return err }
	if quit { return nil }
	ssmClient := deps.newSSM(cfg)
	remote, err := command.StartExecSession(ctx, ecsClient, ssmClient, command.ExecTarget{Cluster: resolved.ECSCluster, TaskARN: resolved.TaskARN, ContainerName: resolved.ContainerName}, in.Cmd, cfg.Region)
	if err != nil { return err }
	return remote.Run(ctx, plugin)
}
```

The port flow is the same ordering. It allocates an empty local port after target resolution and immediately before `StartPortForwardSession`, supplies `command.PortTarget{SSMTarget: resolved.SSMTarget()}`, builds parameters from validated input, and runs the returned `RemoteSession`. Exec deliberately does not use the selection-time runtime ID as its final plugin target; `StartExecSession` refreshes it after ExecuteCommand. Do not infer or add profile/region flags.

This is also the activation point for plan 1's exact-ARN resolver: remove `TasksRunningWaiter`, the dot goroutine, every handler pointer dereference/string split, and the legacy `Cluster2Task2Container` path. After both handlers compile against `ResolveTarget`, delete the four obsolete view/listview AWS files and the temporary `RenderList` wrapper listed above. The handler tests from Steps 1-2 must demonstrate that duplicate service labels still select the exact returned Task ARN and that view cancellation performs no ExecuteCommand/StartSession call before these legacy paths are removed.

- [ ] **Step 6: Run handler tests and confirm GREEN**

Run: `go test ./internal/handler -count=1`

Expected: PASS with preflight preceding every remote session API and failure-only termination proven.

- [ ] **Step 7: Commit handler orchestration**

```bash
git add internal/handler internal/session_manager/session.go \
  internal/view/ecs.go internal/listview \
  pkg/command/exec.go pkg/command/portforward.go
git commit -m "fix: preflight and clean up Session Manager sessions"
```

### Task 5: Signal-aware root execution and consistent errors

**Files:**
- Modify: `cmd/root.go`
- Modify: `main.go`
- Create: `cmd/root_test.go`

- [ ] **Step 1: Write failing root execution tests**

Test `ExecuteContext` with a child command that blocks on `cmd.Context().Done()`; cancel the supplied context and assert `context.Canceled`. Capture stderr for a sentinel command error and assert Cobra returns it rather than exiting the test process. Set `SilenceUsage=true` and assert operational errors do not print usage.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./cmd -count=1`

Expected: build failure because root execution has no context-returning API and calls `os.Exit` internally.

- [ ] **Step 3: Return errors from the command package**

```go
func ExecuteContext(ctx context.Context) error {
	RootCmd.SetContext(ctx)
	return RootCmd.ExecuteContext(ctx)
}
```

Set `RootCmd.SilenceUsage = true` and `RootCmd.SilenceErrors = true`, give root useful `Short`/`Long` text, and make both version paths `RunE` so output errors can be returned. Remove `os.Exit` from `cmd`; `SilenceErrors` prevents Cobra and `main` from printing the same error twice.

- [ ] **Step 4: Own signal and exit handling only in main**

```go
func main() {
	cmd.Version = strings.TrimSpace(version)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Keep the current embedded `.version` declaration only long enough to compile this plan. The next version plan removes the embed and assignment atomically when it introduces `internal/buildinfo`.

- [ ] **Step 5: Run focused and complete verification**

Run: `gofmt -w cmd internal pkg main.go && go test -race ./... && go vet ./... && go build ./...`

Expected: all commands exit 0. Then run `rg -n 'context\.Background\(\)|log\.Fatal' cmd internal pkg` and inspect every match: production session paths must have none; only test setup or intentionally independent cleanup roots may remain.

- [ ] **Step 6: Commit root lifecycle**

```bash
git add cmd/root.go cmd/root_test.go main.go
git commit -m "fix: propagate cancellation through CLI sessions"
```

## Plan completion check

- Plugin discovery and bounded version check happen before ExecuteCommand/StartSession.
- Plugin receives response, region, StartSession, profile, Target request, and endpoint arguments.
- AWS standard environment supplies optional forwarded profile/endpoint; no new AWS flags exist.
- All SDK calls use the Cobra/root context; ECS Exec re-describes the returned task after ExecuteCommand and uses the refreshed runtime ID.
- Any response-validation or exec-refresh failure after a session ID is returned follows the same detached cleanup path as plugin failure.
- Plugin failure or cancellation attempts TerminateSession with an independent timeout and joins cleanup errors.
- Successful plugin completion does not unconditionally terminate or require additional successful IAM calls.
- Missing response fields, canceled operations, and local port allocation failures return errors rather than panic or exit inside a package.
