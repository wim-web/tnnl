# Target Resolution and Input Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every command resolve validated input deterministically and select the exact ready ECS task/container across all API pages without panics or stale identifiers.

**Architecture:** Keep Cobra responsible only for flag collection, move strict decoding/default/override/validation into `internal/input`, keep Bubble Tea responsible only for typed label/value choices, and put AWS pagination/readiness/waiting in a testable `internal/target` resolver. `internal/view` composes the resolver with the interactive chooser and returns a fully validated `target.Resolved` value that plan 2 activates in dependency-injected handlers.

**Tech Stack:** Go 1.25+, Cobra/pflag, AWS SDK for Go v2 ECS types, Bubble Tea/Bubbles, standard `context`, `encoding/json`, `net`, and Go `testing`.

---

This is implementation plan 1 of 4. Execute it before the session-lifecycle plan because that plan consumes `target.Resolved` and the context-aware handler signatures introduced here. The approved design is `docs/superpowers/specs/2026-07-18-product-review-findings-design.md`.

## File map

- `internal/input/input.go`: input value and explicit-override types.
- `internal/input/func.go`: strict JSON read and non-destructive template write.
- `internal/input/resolve.go`: defaults, file values, explicit overrides, normalization.
- `internal/input/validate.go`: command-specific pre-AWS validation.
- `cmd/exec/exec.go`, `cmd/portforward/portforward.go`, `cmd/remoteportforward/remoteportforward.go`: `RunE` command factories that return errors.
- `cmd/inputfile/make.go`: shared safe `make-input-file` Cobra command.
- `internal/listview/simple.go`: typed label/value chooser and empty-list error.
- `internal/target/identifier.go`: safe cluster/task identifier parsing.
- `internal/target/eligibility.go`: task/container readiness rules.
- `internal/target/resolver.go`: pagination, de-duplication, DescribeTasks chunking, failure conversion.
- `internal/target/wait.go`: bounded eligible-task polling with an injected clock.
- `internal/view/resolve.go`: cluster/task/container choice orchestration.
- `internal/handler/exec.go`, `internal/handler/portforward.go`: accept and pass command context; plan 2 atomically moves them to `target.Resolved` with tested session orchestration.
- `pkg/port/available.go`: loopback auto-port selection with explicit listener close.

### Task 1: Strict, non-destructive input-file I/O

**Files:**
- Modify: `internal/input/func.go`
- Replace tests: `internal/input/func_test.go`
- Modify: `cmd/exec/exec.go`
- Modify: `cmd/portforward/portforward.go`
- Modify: `cmd/remoteportforward/remoteportforward.go`
- Modify: `cmd/exec/make_input_file.go`
- Modify: `cmd/portforward/make_input_file.go`
- Modify: `cmd/remoteportforward/make_input_file.go`

- [ ] **Step 1: Write failing strict-decoder and overwrite tests**

Replace the broad round-trip tests with focused behavior tests. Keep one Unicode round trip, then add these cases:

```go
func TestReadInputFileStrict(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{name: "unknown field", content: `{"command":"sh","commnad":"bash"}`, wantErr: `unknown field "commnad"`},
		{name: "trailing document", content: `{"command":"sh"} {"command":"bash"}`, wantErr: "exactly one JSON document"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}
			var got ExecInput
			err := ReadInputFile(&got, path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ReadInputFile() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestMakeInputFileRefusesExistingFileUnlessForced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exec-input.json")
	if err := os.WriteFile(path, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := MakeInputFile(ExecInput{Cmd: "sh"}, path, false); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("MakeInputFile(force=false) error = %v, want fs.ErrExist", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "keep me" {
		t.Fatalf("existing file changed to %q", got)
	}
	if err := MakeInputFile(ExecInput{Cmd: "bash"}, path, true); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 2: Run the focused tests and confirm RED**

Run: `go test ./internal/input -run 'Test(ReadInputFileStrict|MakeInputFileRefusesExistingFileUnlessForced)' -count=1`

Expected: build failure because `ReadInputFile` and `MakeInputFile` currently return no error and `MakeInputFile` has no `force` argument.

- [ ] **Step 3: Implement strict decoding and explicit overwrite**

Use this API and error contract in `internal/input/func.go`:

```go
func ReadInputFile(v any, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open input file %q: %w", path, err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("decode input file %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode input file %q: expected exactly one JSON document", path)
		}
		return fmt.Errorf("decode input file %q: %w", path, err)
	}
	return nil
}

func MakeInputFile(skeleton any, path string, force bool) error {
	data, err := json.MarshalIndent(skeleton, "", "  ")
	if err != nil {
		return fmt.Errorf("encode input template: %w", err)
	}
	flags := os.O_CREATE | os.O_WRONLY
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return fmt.Errorf("create input file %q: %w", path, err)
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		return fmt.Errorf("write input file %q: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync input file %q: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close input file %q: %w", path, err)
	}
	return nil
}
```

Update the existing command call sites in the same commit so the repository keeps compiling: current input reads use `if err := input.ReadInputFile(...); err != nil { log.Fatalln(err) }`, and each current generator calls `input.MakeInputFile(value, path, false)` and handles the returned error through its existing `log.Fatalln` path. Task 3 replaces those temporary process exits with `RunE`.

- [ ] **Step 4: Run the package tests and confirm GREEN**

Run: `go test ./internal/input -count=1`

Expected: PASS, including strict unknown-field/trailing-document rejection and unchanged existing-file content.

- [ ] **Step 5: Commit the codec boundary**

```bash
git add internal/input/func.go internal/input/func_test.go cmd/exec cmd/portforward cmd/remoteportforward
git commit -m "fix: make input files strict and non-destructive"
```

### Task 2: Defaults, explicit overrides, normalization, and validation

**Files:**
- Modify: `internal/input/input.go`
- Create: `internal/input/resolve.go`
- Create: `internal/input/resolve_test.go`
- Create: `internal/input/validate.go`
- Create: `internal/input/validate_test.go`

- [ ] **Step 1: Write failing precedence and validation tests**

Use pointer fields to mean “the caller explicitly supplied this flag”:

```go
func TestResolveExecPrecedence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exec.json")
	os.WriteFile(path, []byte(`{"cluster":" c ","service":"s","command":"bash","wait":20}`), 0o600)
	command := "zsh"
	wait := 0
	got, err := ResolveExec(path, ExecOverrides{Command: &command, Wait: &wait})
	if err != nil {
		t.Fatal(err)
	}
	want := ExecInput{EcsParameter: EcsParameter{Cluster: "c", Service: "s"}, Cmd: "zsh", Wait: 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExec() = %#v, want %#v", got, want)
	}
}

func TestResolveExecUsesDefaultWithoutFileOrOverride(t *testing.T) {
	got, err := ResolveExec("", ExecOverrides{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Cmd != "sh" || got.Wait != 0 {
		t.Fatalf("ResolveExec() = %#v, want command sh and wait 0", got)
	}
}

func TestValidatePortsBeforeAWS(t *testing.T) {
	tests := []struct {
		name string
		in   PortForwardInput
		want string
	}{
		{name: "missing target", in: PortForwardInput{}, want: "target port is required"},
		{name: "non-decimal", in: PortForwardInput{TargetPortNumber: "abc"}, want: "decimal integer"},
		{name: "too large", in: PortForwardInput{TargetPortNumber: "65536"}, want: "between 1 and 65535"},
		{name: "bad optional local", in: PortForwardInput{TargetPortNumber: "80", LocalPortNumber: "0"}, want: "between 1 and 65535"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePortForward(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidatePortForward() error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateExecRejectsNegativeWaitAndBlankCommand(t *testing.T) {
	err := ValidateExec(ExecInput{Cmd: "  ", Wait: -1})
	if err == nil || !strings.Contains(err.Error(), "command is required") || !strings.Contains(err.Error(), "wait must be non-negative") {
		t.Fatalf("ValidateExec() error = %v", err)
	}
}
```

Import `reflect`; do not add a third-party assertion library only for these tests.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./internal/input -run 'Test(Resolve|Validate)' -count=1`

Expected: build failure because override, resolver, and validator types do not exist.

- [ ] **Step 3: Define explicit override types and resolver functions**

Add these types to `internal/input/input.go`:

```go
type ExecOverrides struct {
	Command *string
	Wait    *int
}

type PortForwardOverrides struct {
	TargetPort *string
	LocalPort  *string
}

type RemotePortForwardOverrides struct {
	RemotePort *string
	LocalPort  *string
	Host       *string
}
```

Implement `ResolveExec`, `ResolvePortForward`, and `ResolveRemotePortForward` in `internal/input/resolve.go` with this exact order:

```go
func ResolveExec(path string, overrides ExecOverrides) (ExecInput, error) {
	resolved := ExecInput{Cmd: "sh", Wait: 0}
	if path != "" {
		if err := ReadInputFile(&resolved, path); err != nil {
			return ExecInput{}, err
		}
	}
	if overrides.Command != nil {
		resolved.Cmd = *overrides.Command
	}
	if overrides.Wait != nil {
		resolved.Wait = *overrides.Wait
	}
	normalizeExec(&resolved)
	if err := ValidateExec(resolved); err != nil {
		return ExecInput{}, err
	}
	return resolved, nil
}

func normalizeECS(value *EcsParameter) {
	value.Cluster = strings.TrimSpace(value.Cluster)
	value.Service = strings.TrimSpace(value.Service)
}

func normalizeExec(value *ExecInput) {
	normalizeECS(&value.EcsParameter)
	value.Cmd = strings.TrimSpace(value.Cmd)
}
```

Implement the port resolvers with the same explicit order and these complete field assignments:

```go
func ResolvePortForward(path string, overrides PortForwardOverrides) (PortForwardInput, error) {
	var resolved PortForwardInput
	if path != "" {
		if err := ReadInputFile(&resolved, path); err != nil { return PortForwardInput{}, err }
	}
	if overrides.TargetPort != nil { resolved.TargetPortNumber = *overrides.TargetPort }
	if overrides.LocalPort != nil { resolved.LocalPortNumber = *overrides.LocalPort }
	normalizeECS(&resolved.EcsParameter)
	resolved.TargetPortNumber = strings.TrimSpace(resolved.TargetPortNumber)
	resolved.LocalPortNumber = strings.TrimSpace(resolved.LocalPortNumber)
	if err := ValidatePortForward(resolved); err != nil { return PortForwardInput{}, err }
	return resolved, nil
}

func ResolveRemotePortForward(path string, overrides RemotePortForwardOverrides) (RemotePortForwardInput, error) {
	var resolved RemotePortForwardInput
	if path != "" {
		if err := ReadInputFile(&resolved, path); err != nil { return RemotePortForwardInput{}, err }
	}
	if overrides.RemotePort != nil { resolved.RemotePortNumber = *overrides.RemotePort }
	if overrides.LocalPort != nil { resolved.LocalPortNumber = *overrides.LocalPort }
	if overrides.Host != nil { resolved.Host = *overrides.Host }
	normalizeECS(&resolved.EcsParameter)
	resolved.RemotePortNumber = strings.TrimSpace(resolved.RemotePortNumber)
	resolved.LocalPortNumber = strings.TrimSpace(resolved.LocalPortNumber)
	resolved.Host = strings.TrimSpace(resolved.Host)
	if err := ValidateRemotePortForward(resolved); err != nil { return RemotePortForwardInput{}, err }
	return resolved, nil
}
```

An empty local port remains empty here; the handler allocates it immediately before session creation.

- [ ] **Step 4: Implement command-specific validation**

Use an error slice and `errors.Join` so all pre-AWS input problems are visible in one response:

```go
func validatePort(name, value string, required bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s must be a decimal integer: %q", name, value)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535: %d", name, n)
	}
	return nil
}

func ValidateExec(v ExecInput) error {
	var errs []error
	if strings.TrimSpace(v.Cmd) == "" {
		errs = append(errs, errors.New("command is required"))
	}
	if v.Wait < 0 {
		errs = append(errs, errors.New("wait must be non-negative"))
	}
	return errors.Join(errs...)
}

func ValidatePortForward(v PortForwardInput) error {
	return errors.Join(
		validatePort("target port", v.TargetPortNumber, true),
		validatePort("local port", v.LocalPortNumber, false),
	)
}

func ValidateRemotePortForward(v RemotePortForwardInput) error {
	var hostErr error
	if strings.TrimSpace(v.Host) == "" {
		hostErr = errors.New("host is required")
	}
	return errors.Join(
		validatePort("remote port", v.RemotePortNumber, true),
		validatePort("local port", v.LocalPortNumber, false),
		hostErr,
	)
}
```

- [ ] **Step 5: Run the input tests and confirm GREEN**

Run: `go test ./internal/input -count=1`

Expected: PASS for default < file < explicit flag precedence, normalization, unknown fields, and every invalid value.

- [ ] **Step 6: Commit input resolution**

```bash
git add internal/input
git commit -m "feat: resolve and validate command input"
```

### Task 3: Cobra `RunE` factories and safe input-template commands

**Files:**
- Modify: `cmd/exec/exec.go`
- Modify: `cmd/portforward/portforward.go`
- Modify: `cmd/remoteportforward/remoteportforward.go`
- Modify: `internal/handler/exec.go`
- Modify: `internal/handler/portforward.go`
- Create: `cmd/exec/exec_test.go`
- Create: `cmd/portforward/portforward_test.go`
- Create: `cmd/remoteportforward/remoteportforward_test.go`
- Create: `cmd/inputfile/make.go`
- Create: `cmd/inputfile/make_test.go`
- Modify: `cmd/exec/make_input_file.go`
- Modify: `cmd/portforward/make_input_file.go`
- Modify: `cmd/remoteportforward/make_input_file.go`

- [ ] **Step 1: Write failing command-level precedence tests**

Construct fresh commands in tests and capture the resolved value without AWS:

```go
func TestExecCommandExplicitFlagsOverrideFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exec.json")
	os.WriteFile(path, []byte(`{"command":"bash","wait":10}`), 0o600)
	var got input.ExecInput
	cmd := newExecCommand(func(ctx context.Context, in input.ExecInput) error {
		got = in
		return nil
	})
	cmd.SetArgs([]string{"--input-file", path, "--command", "zsh", "--wait", "0"})
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got.Cmd != "zsh" || got.Wait != 0 {
		t.Fatalf("handler input = %#v", got)
	}
}
```

Add corresponding tests for port target/local override, remote port/host override, invalid input not invoking the handler, and propagation of a context value from `ExecuteContext` to the fake handler. In `cmd/inputfile/make_test.go`, also write the generator tests before implementation: cover `--output`, existing-file refusal, `--force`, command output, and each parent-specific description.

- [ ] **Step 2: Run command tests and confirm RED**

Run: `go test ./cmd/exec ./cmd/portforward ./cmd/remoteportforward ./cmd/inputfile -count=1`

Expected: build failure because the command factories, context-aware runner signatures, and `cmd/inputfile.New` do not exist.

- [ ] **Step 3: Replace global `Run` bodies with injectable `RunE` factories**

Use this complete factory body for exec:

```go
type execRunner func(context.Context, input.ExecInput) error

func newExecCommand(run execRunner) *cobra.Command {
	c := &cobra.Command{
		Use:   "exec",
		Short: "Run an interactive command in an ECS container",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := cmd.Flags().GetString(inputFileName)
			if err != nil {
				return err
			}
			overrides := input.ExecOverrides{}
			if cmd.Flags().Changed(cmdName) {
				value, err := cmd.Flags().GetString(cmdName)
				if err != nil { return err }
				overrides.Command = &value
			}
			if cmd.Flags().Changed(waitName) {
				value, err := cmd.Flags().GetInt(waitName)
				if err != nil { return err }
				overrides.Wait = &value
			}
			resolved, err := input.ResolveExec(path, overrides)
			if err != nil {
				return err
			}
			return run(cmd.Context(), resolved)
		},
	}
	c.Flags().String(cmdName, "sh", "command to run (default: sh)")
	c.Flags().Int(waitName, 0, "seconds to wait for an eligible task")
	c.Flags().String(inputFileName, "", "input JSON; generate with `tnnl exec make-input-file`")
	return c
}

var ExecCmd = newExecCommand(handler.ExecHandler)
```

Replace each existing `init` body so it only attaches the newly constructed command to `cmd.RootCmd`; remove every old `Flags().String*`/`Flags().Int*` registration from `init`, because the factory already owns those flags. Keeping both registrations would panic during package initialization. Apply the same rule to `PortforwardCmd` and `RemoteportforwardCmd`. Their separate `make_input_file.go` initializers continue to attach the generated child command once.

For the two port factories, collect every changed flag explicitly before calling the matching resolver:

```go
overrides := input.PortForwardOverrides{}
if cmd.Flags().Changed(targetPortName) {
	value, err := cmd.Flags().GetString(targetPortName)
	if err != nil { return err }
	overrides.TargetPort = &value
}
if cmd.Flags().Changed(localPortName) {
	value, err := cmd.Flags().GetString(localPortName)
	if err != nil { return err }
	overrides.LocalPort = &value
}
resolved, err := input.ResolvePortForward(path, overrides)
if err != nil { return err }
return run(cmd.Context(), resolved)
```

```go
overrides := input.RemotePortForwardOverrides{}
if cmd.Flags().Changed(remotePortName) {
	value, err := cmd.Flags().GetString(remotePortName)
	if err != nil { return err }
	overrides.RemotePort = &value
}
if cmd.Flags().Changed(localPortName) {
	value, err := cmd.Flags().GetString(localPortName)
	if err != nil { return err }
	overrides.LocalPort = &value
}
if cmd.Flags().Changed(hostName) {
	value, err := cmd.Flags().GetString(hostName)
	if err != nil { return err }
	overrides.Host = &value
}
resolved, err := input.ResolveRemotePortForward(path, overrides)
if err != nil { return err }
return run(cmd.Context(), resolved)
```

At the same commit boundary, change the exported handler signatures to `ExecHandler(context.Context, input.ExecInput) error`, `PortforwardHandler(context.Context, input.PortForwardInput) error`, and `RemotePortforwardHandler(context.Context, input.RemotePortForwardInput) error`. Thread that context through `config.LoadDefaultConfig`, the existing command constructors, and the temporary exec waiter (`errgroup.WithContext(ctx)`); change the private port helper to accept it too. The legacy interactive view still owns background contexts until plan 2 replaces it with `ResolveTarget`, but do not introduce an unused/ignored handler context. Delete `log` imports and the old `validateInput` functions. Do not allocate an automatic local port inside Cobra; an empty local port is the explicit signal for the handler/session step.

- [ ] **Step 4: Implement the shared safe generator command**

Implement the API already exercised by the failing `cmd/inputfile` tests:

```go
func New(parent, defaultPath string, skeleton any) *cobra.Command {
	var output string
	var force bool
	c := &cobra.Command{
		Use:   "make-input-file",
		Short: fmt.Sprintf("Create an input file template for %s", parent),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := input.MakeInputFile(skeleton, output, force); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "made %s\n", output)
			return err
		},
	}
	c.Flags().StringVarP(&output, "output", "o", defaultPath, "output path")
	c.Flags().BoolVarP(&force, "force", "f", false, "replace an existing file")
	return c
}
```

Each existing `make_input_file.go` becomes a one-line constructor plus `AddCommand`, using its correct parent name, default filename, and skeleton type.

- [ ] **Step 5: Run command tests and confirm GREEN**

Run: `go test ./cmd/exec ./cmd/portforward ./cmd/remoteportforward ./cmd/inputfile -count=1`

Expected: PASS; invalid input returns before the fake handler, explicit flags win, and generator overwrite requires `--force`.

- [ ] **Step 6: Commit command input behavior**

```bash
git add cmd/exec cmd/portforward cmd/remoteportforward cmd/inputfile internal/handler/exec.go internal/handler/portforward.go
git commit -m "feat: make command input precedence explicit"
```

### Task 4: Typed terminal options and empty-state errors

**Files:**
- Modify: `internal/listview/simple.go`
- Create: `internal/listview/simple_test.go`
- Delete after Task 7 migration: `internal/listview/cluster.go`
- Delete after Task 7 migration: `internal/listview/task.go`
- Delete after Task 7 migration: `internal/listview/container.go`

- [ ] **Step 1: Write failing option identity and empty-list tests**

```go
func TestOptionKeepsLabelSeparateFromValue(t *testing.T) {
	got := item{Option: Option{Label: "service:web abc123", Value: "arn:aws:ecs:ap-northeast-1:1:task/c/abc123"}}
	if got.FilterValue() != "service:web abc123" || got.Value != "arn:aws:ecs:ap-northeast-1:1:task/c/abc123" {
		t.Fatalf("item = %#v", got)
	}
}

func TestRenderOptionsRejectsEmptyOptionsBeforeStartingTea(t *testing.T) {
	_, _, err := RenderOptions("Select a task", nil)
	var noItems *NoItemsError
	if !errors.As(err, &noItems) || noItems.Title != "Select a task" {
		t.Fatalf("RenderList() error = %v", err)
	}
}
```

Add a model update test whose selected item has duplicate display text but a distinct value; Enter must store the selected `Value`.

- [ ] **Step 2: Run tests and confirm RED**

Run: `go test ./internal/listview -count=1`

Expected: build failure because `Option` and `NoItemsError` do not exist.

- [ ] **Step 3: Implement typed options**

Use:

```go
type Option struct {
	Label string
	Value string
}

type NoItemsError struct{ Title string }

func (e *NoItemsError) Error() string { return fmt.Sprintf("%s: no selectable items", e.Title) }

type item struct{ Option }

func (i item) FilterValue() string { return i.Label }

func RenderOptions(title string, options []Option) (string, bool, error) {
	if len(options) == 0 {
		return "", false, &NoItemsError{Title: title}
	}
	items := make([]list.Item, 0, len(options))
	for _, option := range options {
		items = append(items, item{Option: option})
	}
	listModel := list.New(items, itemDelegate{}, listWidth, listHeight)
	listModel.Title = title
	program := tea.NewProgram(model{list: listModel})
	result, err := program.Run()
	if err != nil { return "", false, err }
	final, ok := result.(model)
	if !ok { return "", false, fmt.Errorf("unexpected list model %T", result) }
	return final.choice, final.quitting, nil
}

func RenderList(title string, labels []string) (string, bool, error) {
	options := make([]Option, 0, len(labels))
	for _, label := range labels { options = append(options, Option{Label: label, Value: label}) }
	return RenderOptions(title, options)
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok { return }
	text := fmt.Sprintf("%d. %s", index+1, i.Label)
	render := itemStyle.Render
	if index == m.Index() {
		render = func(values ...string) string { return selectedItemStyle.Render("> " + values[0]) }
	}
	fmt.Fprint(w, render(text))
}
```

In `model.Update`, the Enter branch is exactly:

```go
case "enter":
	i, ok := m.list.SelectedItem().(item)
	if !ok { return m, nil }
	m.choice = i.Value
	return m, tea.Quit
```

The existing `q`/Ctrl-C branch continues to set `quitting=true`. `RenderList` remains a temporary label=value compatibility wrapper so the existing cluster/task/container callers compile; Task 8 removes it with those callers. New code uses only `RenderOptions`. Never derive an internal selection key from the rendered label.

- [ ] **Step 4: Run tests and confirm GREEN**

Run: `go test ./internal/listview -count=1`

Expected: PASS without opening Bubble Tea for an empty list.

- [ ] **Step 5: Commit typed choices**

```bash
git add internal/listview/simple.go internal/listview/simple_test.go
git commit -m "fix: preserve unique values in terminal choices"
```

### Task 5: Safe ECS identifiers and readiness eligibility

**Files:**
- Create: `internal/target/identifier.go`
- Create: `internal/target/identifier_test.go`
- Create: `internal/target/eligibility.go`
- Create: `internal/target/eligibility_test.go`

- [ ] **Step 1: Write failing identifier tests**

Cover these exact accepted and rejected forms:

```go
func TestClusterName(t *testing.T) {
	tests := map[string]string{
		"cluster-name": "cluster-name",
		"arn:aws:ecs:ap-northeast-1:123456789012:cluster/cluster-name": "cluster-name",
	}
	for in, want := range tests {
		got, err := ClusterName(in)
		if err != nil || got != want { t.Fatalf("ClusterName(%q) = %q, %v", in, got, err) }
	}
}

func TestTaskIDAcceptsLongAndShortForms(t *testing.T) {
	tests := map[string]string{
		"task/abc": "abc",
		"task/cluster/abc": "abc",
		"arn:aws:ecs:ap-northeast-1:123456789012:task/abc": "abc",
		"arn:aws:ecs:ap-northeast-1:123456789012:task/cluster/abc": "abc",
	}
	for in, want := range tests {
		got, err := TaskID(in)
		if err != nil || got != want { t.Fatalf("TaskID(%q) = %q, %v", in, got, err) }
	}
}
```

Reject empty values, non-ECS ARNs, wrong resource types, and paths with an empty final segment.

- [ ] **Step 2: Write failing eligibility table tests**

Build a ready fixture and mutate one field per row: task exec disabled, task PENDING, missing task ARN, container name missing, container STOPPED, no ExecuteCommandAgent, agent PENDING, and runtime ID missing. Assert only the untouched task is eligible and only its ready containers are returned.

- [ ] **Step 3: Run tests and confirm RED**

Run: `go test ./internal/target -run 'Test(ClusterName|TaskID|Eligible)' -count=1`

Expected: build failure because the target package does not exist.

- [ ] **Step 4: Implement validated final-segment parsing**

Parse identifiers with these functions:

```go
func ClusterName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" { return "", errors.New("cluster identifier is empty") }
	if !strings.HasPrefix(value, "arn:") {
		if strings.Contains(value, "/") { return "", fmt.Errorf("invalid cluster identifier %q", value) }
		return value, nil
	}
	parsed, err := arn.Parse(value)
	if err != nil || parsed.Service != "ecs" || !strings.HasPrefix(parsed.Resource, "cluster/") {
		return "", fmt.Errorf("invalid ECS cluster ARN %q", value)
	}
	parts := strings.Split(parsed.Resource, "/")
	if len(parts) != 2 { return "", fmt.Errorf("invalid ECS cluster ARN %q", value) }
	return finalSegment(value, parts)
}

func TaskID(value string) (string, error) {
	original := value
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "arn:") {
		parsed, err := arn.Parse(value)
		if err != nil || parsed.Service != "ecs" { return "", fmt.Errorf("invalid ECS task ARN %q", original) }
		value = parsed.Resource
	}
	if !strings.HasPrefix(value, "task/") { return "", fmt.Errorf("invalid ECS task identifier %q", original) }
	parts := strings.Split(value, "/")
	if len(parts) != 2 && len(parts) != 3 { return "", fmt.Errorf("invalid ECS task identifier %q", original) }
	return finalSegment(original, parts)
}

func finalSegment(original string, parts []string) (string, error) {
	value := strings.TrimSpace(parts[len(parts)-1])
	if value == "" { return "", fmt.Errorf("identifier %q has an empty final segment", original) }
	return value, nil
}
```

Every malformed form returns an error containing the original value.

- [ ] **Step 5: Implement readiness without pointer dereferences**

```go
func EligibleContainers(task types.Task) []types.Container {
	if !task.EnableExecuteCommand || strings.TrimSpace(aws.ToString(task.LastStatus)) != "RUNNING" || aws.ToString(task.TaskArn) == "" {
		return nil
	}
	var eligible []types.Container
	for _, container := range task.Containers {
		if strings.TrimSpace(aws.ToString(container.Name)) == "" || aws.ToString(container.LastStatus) != "RUNNING" || strings.TrimSpace(aws.ToString(container.RuntimeId)) == "" {
			continue
		}
		agentReady := false
		for _, agent := range container.ManagedAgents {
			if agent.Name == types.ManagedAgentNameExecuteCommandAgent && aws.ToString(agent.LastStatus) == "RUNNING" {
				agentReady = true
				break
			}
		}
		if agentReady { eligible = append(eligible, container) }
	}
	return eligible
}

func IsEligibleTask(task types.Task) bool { return len(EligibleContainers(task)) > 0 }
```

Runtime ID is required for exec too: the full plugin request target is `ecs:<cluster>_<task>_<runtime>`.

- [ ] **Step 6: Run tests and confirm GREEN**

Run: `go test ./internal/target -count=1`

Expected: PASS with no panic from nil pointer fields.

- [ ] **Step 7: Commit identifier/readiness rules**

```bash
git add internal/target/identifier.go internal/target/identifier_test.go internal/target/eligibility.go internal/target/eligibility_test.go
git commit -m "feat: validate ready ECS targets"
```

### Task 6: Complete ECS pagination, DescribeTasks chunking, and failures

**Files:**
- Create: `internal/target/resolver.go`
- Create: `internal/target/resolver_test.go`

- [ ] **Step 1: Write a deterministic fake ECS client**

The fake implements these exact SDK-compatible methods and records every input:

```go
type ECSAPI interface {
	ListClusters(context.Context, *ecs.ListClustersInput, ...func(*ecs.Options)) (*ecs.ListClustersOutput, error)
	ListTasks(context.Context, *ecs.ListTasksInput, ...func(*ecs.Options)) (*ecs.ListTasksOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput, ...func(*ecs.Options)) (*ecs.DescribeTasksOutput, error)
}
```

Its `ListClusters` and `ListTasks` responses are keyed by input `NextToken`; `DescribeTasks` looks up requested ARNs and can inject `types.Failure` values.

- [ ] **Step 2: Write failing pagination/chunk/failure tests**

Add tests that assert:

- two ListClusters pages are concatenated in response order;
- two ListTasks pages are concatenated and duplicate ARNs are removed without reordering;
- 201 task ARNs produce DescribeTasks calls of 100, 100, and 1;
- ListTasks includes `DesiredStatus: types.DesiredStatusRunning` and optional `ServiceName` on every page;
- a failure with ARN, reason, and detail produces an error containing all three;
- a context marker reaches all fake methods;
- empty task pages do not call DescribeTasks.

- [ ] **Step 3: Run resolver tests and confirm RED**

Run: `go test ./internal/target -run 'TestResolver' -count=1`

Expected: build failure because `Resolver`, `Clusters`, and `EligibleTasks` do not exist.

- [ ] **Step 4: Implement pagination and chunking**

```go
type Resolver struct{ client ECSAPI }

func NewResolver(client ECSAPI) *Resolver { return &Resolver{client: client} }

func (r *Resolver) Clusters(ctx context.Context) ([]string, error) {
	var arns []string
	var token *string
	for {
		out, err := r.client.ListClusters(ctx, &ecs.ListClustersInput{NextToken: token})
		if err != nil { return nil, fmt.Errorf("list ECS clusters: %w", err) }
		arns = append(arns, out.ClusterArns...)
		token = out.NextToken
		if token == nil || *token == "" { return arns, nil }
	}
}
```

Implement task paging and ordered de-duplication as:

```go
func (r *Resolver) taskARNs(ctx context.Context, cluster, service string) ([]string, error) {
	var result []string
	seen := map[string]struct{}{}
	var token *string
	for {
		input := &ecs.ListTasksInput{Cluster: aws.String(cluster), DesiredStatus: types.DesiredStatusRunning, NextToken: token}
		if service != "" { input.ServiceName = aws.String(service) }
		out, err := r.client.ListTasks(ctx, input)
		if err != nil { return nil, fmt.Errorf("list ECS tasks in cluster %q: %w", cluster, err) }
		for _, taskARN := range out.TaskArns {
			if _, exists := seen[taskARN]; exists { continue }
			seen[taskARN] = struct{}{}
			result = append(result, taskARN)
		}
		token = out.NextToken
		if aws.ToString(token) == "" { return result, nil }
	}
}
```

Describe in service-limit chunks:

```go
const describeTasksLimit = 100

func (r *Resolver) describeTasks(ctx context.Context, cluster string, arns []string) ([]types.Task, error) {
	var result []types.Task
	for start := 0; start < len(arns); start += describeTasksLimit {
		end := min(start+describeTasksLimit, len(arns))
		out, err := r.client.DescribeTasks(ctx, &ecs.DescribeTasksInput{Cluster: aws.String(cluster), Tasks: arns[start:end]})
		if err != nil { return nil, fmt.Errorf("describe ECS tasks in cluster %q: %w", cluster, err) }
		if err := describeFailuresError(out.Failures); err != nil { return nil, err }
		result = append(result, out.Tasks...)
	}
	return result, nil
}
```

`EligibleTasks` calls `taskARNs`, returns an empty slice without DescribeTasks when no ARN exists, calls `describeTasks`, and filters the result with `IsEligibleTask`. Preserve original API errors with `%w`.

- [ ] **Step 5: Convert partial DescribeTasks failures into actionable errors**

Format each failure as `describe ECS task <arn>: <reason>: <detail>`, join multiple failures with `errors.Join`, and return the error even when the response also contains successful tasks. Filter successful tasks with `IsEligibleTask` only after every chunk succeeds without `Failures`.

- [ ] **Step 6: Run resolver tests and confirm GREEN**

Run: `go test ./internal/target -count=1`

Expected: PASS; 201 tasks are described in three calls and partial AWS failures cannot become an empty UI.

- [ ] **Step 7: Commit complete retrieval**

```bash
git add internal/target/resolver.go internal/target/resolver_test.go
git commit -m "fix: retrieve every eligible ECS task"
```

### Task 7: Correct wait semantics and exact interactive selection

**Files:**
- Create: `internal/target/wait.go`
- Create: `internal/target/wait_test.go`
- Create: `internal/view/resolve.go`
- Create: `internal/view/resolve_test.go`
- Keep for plan 2 migration: `internal/view/ecs.go`
- Keep for plan 2 migration: `internal/listview/cluster.go`
- Keep for plan 2 migration: `internal/listview/task.go`
- Keep for plan 2 migration: `internal/listview/container.go`

- [ ] **Step 1: Write failing fake-clock wait tests**

Define a clock boundary:

```go
type Clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}
```

Tests use a fake whose `Sleep` advances `Now`. Cover: `wait=0` makes one lookup; pending then ready returns the final ready response; deadline returns an error naming cluster/service/duration/readiness; API error returns immediately; canceled context returns `context.Canceled` through `errors.Is`. Add a deadline-boundary row where `Sleep` advances exactly to the deadline and assert no additional `EligibleTasks` call occurs. Have the fake API capture its context and assert a positive wait installs a deadline no later than the requested maximum.

- [ ] **Step 2: Run wait tests and confirm RED**

Run: `go test ./internal/target -run TestWaitForEligibleTasks -count=1`

Expected: build failure because `Clock` and `WaitForEligibleTasks` do not exist.

- [ ] **Step 3: Implement bounded polling**

```go
func (r *Resolver) WaitForEligibleTasks(ctx context.Context, cluster, service string, maxWait time.Duration, clock Clock) ([]types.Task, error) {
	deadline := clock.Now().Add(maxWait)
	waitCtx := ctx
	cancel := func() {}
	if maxWait > 0 {
		waitCtx, cancel = context.WithDeadline(ctx, deadline)
	}
	defer cancel()
	firstLookup := true
	for {
		if !firstLookup && !clock.Now().Before(deadline) {
			return nil, noEligibleTasksError(cluster, service, maxWait)
		}
		tasks, err := r.EligibleTasks(waitCtx, cluster, service)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("wait for eligible ECS task: %w", ctx.Err())
			}
			if maxWait > 0 && errors.Is(err, context.DeadlineExceeded) {
				return nil, noEligibleTasksError(cluster, service, maxWait)
			}
			return nil, err
		}
		if len(tasks) > 0 { return tasks, nil }
		if maxWait == 0 || !clock.Now().Before(deadline) {
			return nil, noEligibleTasksError(cluster, service, maxWait)
		}
		delay := min(2*time.Second, deadline.Sub(clock.Now()))
		if err := clock.Sleep(waitCtx, delay); err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("wait for eligible ECS task: %w", ctx.Err())
			}
			if maxWait > 0 && errors.Is(err, context.DeadlineExceeded) {
				return nil, noEligibleTasksError(cluster, service, maxWait)
			}
			return nil, fmt.Errorf("wait for eligible ECS task: %w", err)
		}
		firstLookup = false
	}
}
```

Production `realClock.Sleep` uses a timer and selects on `ctx.Done()`. A positive wait therefore has two bounds: the loop refuses a lookup once the clock reaches the deadline, and every AWS call receives a context with that same deadline. `wait=0` remains one immediate lookup with the caller context.

- [ ] **Step 4: Write failing exact-selection orchestration tests**

Define `type Choose func(string, []listview.Option) (string, bool, error)`. Use two ready tasks with the same `Group` but different ARNs. Have the fake chooser return `options[1].Value`; assert the returned task is the second ARN. Also cover no clusters, no eligible tasks, no eligible containers, one-task auto-selection, one-container auto-selection, user cancellation, full cluster ARN input, and malformed identifiers.

- [ ] **Step 5: Run selection tests and confirm RED**

Run: `go test ./internal/view -run 'Test(ResolveTarget|TaskOptions|TaskByARN)' -count=1`

Expected: build failure because `ResolveTarget`, `target.Resolved`, and the identity-preserving option helpers do not exist.

- [ ] **Step 6: Implement `target.Resolved` and the view orchestration**

`target.Resolved` contains validated strings so later code never dereferences AWS pointers:

```go
type Resolved struct {
	ECSCluster    string
	ClusterName   string
	Task          types.Task
	TaskARN       string
	TaskID        string
	Container     types.Container
	ContainerName string
	RuntimeID     string
}

func (r Resolved) SSMTarget() string {
	return fmt.Sprintf("ecs:%s_%s_%s", r.ClusterName, r.TaskID, r.RuntimeID)
}
```

The task option label is `<group-or-task> <short-task-id>` and its value is the full Task ARN. The selected value is matched only against `TaskArn`. Container options use name for both label and value because ECS container names are unique inside a task. Return resource-specific empty errors before calling `Choose`.

Use this exact identity-preserving selection helper:

```go
func chooseOption(title string, options []listview.Option, auto bool, choose Choose) (string, bool, error) {
	if len(options) == 0 { return "", false, fmt.Errorf("%s: no eligible items", title) }
	if auto && len(options) == 1 { return options[0].Value, false, nil }
	return choose(title, options)
}

func taskOptions(tasks []types.Task) ([]listview.Option, error) {
	options := make([]listview.Option, 0, len(tasks))
	for _, task := range tasks {
		arnValue := strings.TrimSpace(aws.ToString(task.TaskArn))
		id, err := target.TaskID(arnValue)
		if err != nil { return nil, err }
		group := strings.TrimSpace(aws.ToString(task.Group))
		if group == "" { group = "task" }
		options = append(options, listview.Option{Label: fmt.Sprintf("%s %s", group, id), Value: arnValue})
	}
	return options, nil
}

func taskByARN(tasks []types.Task, selected string) (types.Task, error) {
	for _, task := range tasks {
		if aws.ToString(task.TaskArn) == selected { return task, nil }
	}
	return types.Task{}, fmt.Errorf("selected ECS task %q is no longer available", selected)
}
```

`ResolveTarget` resolves or selects the cluster, calls `WaitForEligibleTasks`, calls `taskOptions` and `chooseOption(..., true, choose)`, resolves the selected task only with `taskByARN`, builds name/value options from `EligibleContainers`, and returns `Resolved` only after `ClusterName`, `TaskID`, task ARN, container name, and runtime ID are non-empty. Its production chooser is `listview.RenderOptions`. Leave the old `Cluster2Task2Container` and its listview callers untouched until the dependency-injected handler migration in plan 2 so every plan-1 commit remains buildable and new handler behavior receives a true RED test first.

- [ ] **Step 7: Run target/view tests and confirm GREEN**

Run: `go test ./internal/target ./internal/view ./internal/listview -count=1`

Expected: PASS; selecting the second duplicate-group row returns its exact ARN and `--wait` polls before task choice.

- [ ] **Step 8: Commit selection and waiting**

```bash
git add internal/target internal/view internal/listview
git commit -m "fix: select exact ready ECS targets"
```

### Task 8: Released automatic ports

**Files:**
- Modify: `pkg/port/available.go`
- Create: `pkg/port/available_test.go`

- [ ] **Step 1: Write failing auto-port rebind test**

```go
func TestAvailablePortIsReleasedBeforeReturn(t *testing.T) {
	port, err := AvailablePort()
	if err != nil { t.Fatal(err) }
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil { t.Fatalf("returned port cannot be rebound: %v", err) }
	l.Close()
}
```

- [ ] **Step 2: Run port test and confirm RED**

Run: `go test ./pkg/port -run TestAvailablePortIsReleasedBeforeReturn -count=1`

Expected: FAIL with an address-in-use error because the current listener is still open.

- [ ] **Step 3: Bind loopback and close before returning**

```go
func AvailablePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { return -1, fmt.Errorf("select local port: %w", err) }
	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil { return -1, fmt.Errorf("release local port %d: %w", port, err) }
	return port, nil
}
```

The close eliminates the confirmed descriptor leak. Do not implement text-based bind-conflict retries: plugin output/exit behavior is not a stable API and the approved design calls that retry optional.

- [ ] **Step 4: Run race and static verification**

Run: `gofmt -w cmd internal pkg && go test -race ./... && go vet ./... && go build ./...`

Expected: all commands exit 0; `gofmt -l cmd internal pkg` prints nothing afterward.

- [ ] **Step 5: Commit the port fix**

```bash
git add pkg/port
git commit -m "fix: release automatic local ports"
```

## Plan completion check

- Duplicate service replicas are keyed by full Task ARN in the new resolver; plan 2 activates it in handlers under dependency-injected tests.
- Empty clusters/tasks/containers return errors before Bubble Tea.
- ListClusters and ListTasks consume every page; DescribeTasks uses 100-item chunks and surfaces `Failures`.
- Eligibility requires task/container/agent RUNNING, ECS Exec enabled, and runtime ID present.
- Wait occurs before task selection and returns fresh task/container data.
- Input precedence is explicit flag > strict file > default, and validation happens before AWS.
- Template generation is non-destructive unless `--force` is explicit.
- Auto-selected listeners bind loopback and close before return.
- Profile/region flags and Terraform remain unchanged.
